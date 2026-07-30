package oa

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
)

type noopConnector struct{}

func (noopConnector) Connect(context.Context) (driver.Conn, error) {
	return noopConn{}, nil
}

func (noopConnector) Driver() driver.Driver {
	return noopDriver{}
}

type noopDriver struct{}

func (noopDriver) Open(string) (driver.Conn, error) {
	return noopConn{}, nil
}

type noopConn struct{}

func (noopConn) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (noopConn) Close() error {
	return nil
}

func (noopConn) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

type trackedPoolOpener struct {
	calls atomic.Int32
	mu    sync.Mutex
	dbs   []*sql.DB
}

func (o *trackedPoolOpener) open(
	context.Context,
	string,
	*model.OADatabaseConnection,
	bool,
) (*gorm.DB, *sql.DB, error) {
	o.calls.Add(1)
	sqlDB := sql.OpenDB(noopConnector{})
	o.mu.Lock()
	o.dbs = append(o.dbs, sqlDB)
	o.mu.Unlock()
	return &gorm.DB{}, sqlDB, nil
}

func (o *trackedPoolOpener) database(index int) *sql.DB {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.dbs[index]
}

func testOAConnection(id uuid.UUID) *model.OADatabaseConnection {
	return &model.OADatabaseConnection{
		ID:                id,
		OAType:            "weaver_e9",
		Driver:            "mysql",
		Host:              "127.0.0.1",
		Port:              3306,
		DatabaseName:      "ecology",
		Username:          "readonly",
		Password:          "secret",
		PoolSize:          10,
		ConnectionTimeout: 30,
		Enabled:           true,
	}
}

func TestConnectionManagerReusesPool(t *testing.T) {
	opener := &trackedPoolOpener{}
	manager := newConnectionManager(nil, opener.open)
	conn := testOAConnection(uuid.New())

	first, err := manager.GetAdapter(context.Background(), conn.OAType, conn)
	if err != nil {
		t.Fatalf("首次获取适配器失败: %v", err)
	}
	second, err := manager.GetAdapter(context.Background(), conn.OAType, conn)
	if err != nil {
		t.Fatalf("再次获取适配器失败: %v", err)
	}
	if first == second {
		t.Fatal("适配器应保持轻量并按调用创建，不应缓存带业务状态的适配器对象")
	}
	if got := opener.calls.Load(); got != 1 {
		t.Fatalf("相同配置应只创建一个连接池，实际创建 %d 次", got)
	}
}

func TestConnectionManagerConcurrentAcquireOpensOnce(t *testing.T) {
	opener := &trackedPoolOpener{}
	manager := newConnectionManager(nil, opener.open)
	conn := testOAConnection(uuid.New())

	const goroutines = 32
	var wg sync.WaitGroup
	errs := make(chan error, goroutines)
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := manager.GetAdapter(context.Background(), conn.OAType, conn)
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("并发获取适配器失败: %v", err)
		}
	}
	if got := opener.calls.Load(); got != 1 {
		t.Fatalf("并发首次访问应只创建一个连接池，实际创建 %d 次", got)
	}
}

func TestConnectionManagerReplacesPoolWhenConfigChanges(t *testing.T) {
	opener := &trackedPoolOpener{}
	manager := newConnectionManager(nil, opener.open)
	conn := testOAConnection(uuid.New())

	if _, err := manager.GetAdapter(context.Background(), conn.OAType, conn); err != nil {
		t.Fatalf("首次获取适配器失败: %v", err)
	}
	oldDB := opener.database(0)

	changed := *conn
	changed.PoolSize = 20
	if _, err := manager.GetAdapter(context.Background(), changed.OAType, &changed); err != nil {
		t.Fatalf("配置变化后获取适配器失败: %v", err)
	}
	if got := opener.calls.Load(); got != 2 {
		t.Fatalf("配置变化后应替换连接池，实际创建 %d 次", got)
	}
	if err := oldDB.PingContext(context.Background()); err == nil {
		t.Fatalf("旧连接池应已关闭，实际错误: %v", err)
	}
}

func TestConnectionManagerTransientAdapterAlwaysCloses(t *testing.T) {
	opener := &trackedPoolOpener{}
	manager := newConnectionManager(nil, opener.open)
	conn := testOAConnection(uuid.Nil)

	_, closeFn, err := manager.OpenTransientAdapter(context.Background(), conn.OAType, conn)
	if err != nil {
		t.Fatalf("创建短生命周期适配器失败: %v", err)
	}
	transientDB := opener.database(0)
	if err := closeFn(); err != nil {
		t.Fatalf("关闭短生命周期连接失败: %v", err)
	}
	if err := transientDB.PingContext(context.Background()); err == nil {
		t.Fatalf("短生命周期连接应已关闭，实际错误: %v", err)
	}
	if stats := manager.Stats(); len(stats) != 0 {
		t.Fatalf("短生命周期连接不应进入共享池，实际池数量: %d", len(stats))
	}
}

func TestConnectionManagerDisabledConfigInvalidatesPool(t *testing.T) {
	opener := &trackedPoolOpener{}
	manager := newConnectionManager(nil, opener.open)
	conn := testOAConnection(uuid.New())

	if _, err := manager.GetAdapter(context.Background(), conn.OAType, conn); err != nil {
		t.Fatalf("建立共享连接池失败: %v", err)
	}
	sqlDB := opener.database(0)

	disabled := *conn
	disabled.Enabled = false
	if _, err := manager.GetAdapter(context.Background(), disabled.OAType, &disabled); err == nil {
		t.Fatal("停用的 OA 连接不应继续返回适配器")
	}
	if err := sqlDB.PingContext(context.Background()); err == nil {
		t.Fatal("OA 连接停用后共享连接池应立即关闭")
	}
	if stats := manager.Stats(); len(stats) != 0 {
		t.Fatalf("OA 连接停用后不应保留共享池，实际数量: %d", len(stats))
	}
}

func TestConnectionManagerInvalidateAndClose(t *testing.T) {
	opener := &trackedPoolOpener{}
	manager := newConnectionManager(nil, opener.open)
	firstConn := testOAConnection(uuid.New())
	secondConn := testOAConnection(uuid.New())

	if _, err := manager.GetAdapter(context.Background(), firstConn.OAType, firstConn); err != nil {
		t.Fatalf("建立第一个连接池失败: %v", err)
	}
	if _, err := manager.GetAdapter(context.Background(), secondConn.OAType, secondConn); err != nil {
		t.Fatalf("建立第二个连接池失败: %v", err)
	}
	firstDB := opener.database(0)
	secondDB := opener.database(1)

	manager.Invalidate(firstConn.ID)
	if err := firstDB.PingContext(context.Background()); err == nil {
		t.Fatalf("失效连接池应已关闭，实际错误: %v", err)
	}
	if stats := manager.Stats(); len(stats) != 1 {
		t.Fatalf("失效后应保留一个连接池，实际数量: %d", len(stats))
	}

	if err := manager.Close(); err != nil {
		t.Fatalf("关闭全部连接池失败: %v", err)
	}
	if err := secondDB.PingContext(context.Background()); err == nil {
		t.Fatalf("进程关闭时连接池应已关闭，实际错误: %v", err)
	}
	if stats := manager.Stats(); len(stats) != 0 {
		t.Fatalf("关闭后不应保留连接池，实际数量: %d", len(stats))
	}
}

func TestConnectionManagerInvalidationDuringOpenClosesNewPool(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	opened := make(chan *sql.DB, 1)
	opener := func(
		context.Context,
		string,
		*model.OADatabaseConnection,
		bool,
	) (*gorm.DB, *sql.DB, error) {
		close(started)
		<-release
		sqlDB := sql.OpenDB(noopConnector{})
		opened <- sqlDB
		return &gorm.DB{}, sqlDB, nil
	}
	manager := newConnectionManager(nil, opener)
	conn := testOAConnection(uuid.New())

	result := make(chan error, 1)
	go func() {
		_, err := manager.GetAdapter(context.Background(), conn.OAType, conn)
		result <- err
	}()

	<-started
	manager.Invalidate(conn.ID)
	close(release)

	if err := <-result; !errors.Is(err, errConnectionPoolInvalidated) {
		t.Fatalf("建池期间失效应返回配置已变化错误，实际错误: %v", err)
	}
	sqlDB := <-opened
	if err := sqlDB.PingContext(context.Background()); err == nil {
		t.Fatal("建池期间失效的新连接池应立即关闭")
	}
	if stats := manager.Stats(); len(stats) != 0 {
		t.Fatalf("失效竞态后不应残留连接池，实际数量: %d", len(stats))
	}
}

func TestConnectionPoolNormalization(t *testing.T) {
	if got := normalizePoolSize(0); got != defaultOAPoolSize {
		t.Fatalf("默认连接池大小错误: %d", got)
	}
	if got := normalizePoolSize(maxOAPoolSize + 1); got != maxOAPoolSize {
		t.Fatalf("连接池上限未生效: %d", got)
	}
	if got := normalizeMaxIdleConns(1); got != 1 {
		t.Fatalf("共享池至少应保留一个空闲连接: %d", got)
	}
	if got := normalizeConnectionTimeout(1); got != minConnectionTimeout {
		t.Fatalf("连接超时下限未生效: %s", got)
	}
	if got := normalizeConnectionTimeout(301); got != maxConnectionTimeout {
		t.Fatalf("连接超时上限未生效: %s", got)
	}
}
