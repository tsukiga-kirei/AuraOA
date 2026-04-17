# Implementation Plan: Redis Cache Optimization

## Overview

本实现计划将 Redis 缓存优化功能分解为可执行的编码任务。采用分阶段实施策略：首先构建缓存基础设施，然后集成到服务层，最后添加管理接口和测试。所有任务使用 Go 语言实现。

## Tasks

- [x] 1. 创建缓存基础设施
  - [x] 1.1 创建 CacheManager 核心组件
    - 创建 `server/internal/cache/manager.go` 文件
    - 实现 CacheManager 结构体，包含 redis.Client、logger、enabled 和 stats 字段
    - 实现 NewCacheManager 构造函数
    - 实现 Get、Set、Delete、Exists 基础操作方法
    - 实现 DeleteByPrefix 批量删除方法
    - 实现 IsEnabled 和 SetEnabled 方法
    - 实现 GetStats 统计查询方法
    - 实现 GetWithFallback 带降级的缓存获取方法
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.6_

  - [ ]* 1.2 编写 CacheManager 属性测试 - 缓存往返一致性
    - **Property 1: 缓存往返一致性**
    - 创建 `server/internal/cache/cache_roundtrip_test.go`
    - 使用 gopter 生成任意可序列化数据对象
    - 验证 Set 后 Get 返回等价数据
    - **Validates: Requirements 1.1, 1.6**

  - [x] 1.3 创建 CacheKeyBuilder 缓存键生成器
    - 创建 `server/internal/cache/key_builder.go` 文件
    - 实现 CacheKeyBuilder 结构体
    - 实现 NewKeyBuilder 构造函数
    - 实现 TodoList、ArchiveList、ProcessConfig、Snapshot、Stats、Dashboard 方法
    - 确保键格式符合 `{module}:{tenant_id}:{resource}:{identifier}` 规范
    - _Requirements: 1.5, 2.3, 2.6, 3.3, 3.6, 4.1, 4.2, 5.1, 5.2, 5.5_

  - [ ]* 1.4 编写 CacheKeyBuilder 属性测试 - 缓存键格式规范
    - **Property 2: 缓存键格式规范**
    - 创建 `server/internal/cache/cache_key_format_test.go`
    - 使用 gopter 生成任意模块名、租户ID、资源类型组合
    - 验证生成的键符合命名规范且唯一
    - **Validates: Requirements 1.5, 2.3, 2.6, 3.3, 3.6, 4.1, 4.2, 5.1, 5.2, 5.5**

  - [x] 1.5 创建 Filter Hash 计算工具
    - 创建 `server/internal/cache/hash.go` 文件
    - 实现 ComputeFilterHash 函数，使用 SHA256 计算筛选条件哈希
    - 取前 8 字节作为哈希值
    - _Requirements: 2.3, 3.3_

  - [x] 1.6 创建 CacheStats 统计组件
    - 创建 `server/internal/cache/stats.go` 文件
    - 实现 CacheStats 结构体，包含 Hits、Misses、Errors 计数器
    - 实现线程安全的计数方法（使用 sync.RWMutex）
    - 实现 GetSnapshot 方法返回 CacheStatsSnapshot
    - 实现命中率计算
    - _Requirements: 7.1, 7.5_

  - [ ]* 1.7 编写 CacheStats 属性测试 - 缓存统计准确性
    - **Property 7: 缓存统计准确性**
    - 创建 `server/internal/cache/cache_stats_test.go`
    - 使用 gopter 生成任意缓存操作序列
    - 验证统计计数与实际操作次数一致
    - **Validates: Requirements 7.1, 7.5**

  - [x] 1.8 创建 InvalidationManager 缓存失效管理器
    - 创建 `server/internal/cache/invalidation.go` 文件
    - 实现 InvalidationManager 结构体
    - 实现 NewInvalidationManager 构造函数
    - 实现 InvalidateUserTodoCache、InvalidateUserArchiveCache 方法
    - 实现 InvalidateSnapshotCache、InvalidateConfigCache、InvalidateStatsCache 方法
    - 实现 InvalidateTenantCache、InvalidateModuleCache 方法
    - _Requirements: 2.5, 3.5, 4.4, 4.5, 6.1, 6.2, 6.3, 6.4, 6.5, 6.6_

  - [ ]* 1.9 编写 InvalidationManager 属性测试 - 前缀批量删除完整性
    - **Property 3: 前缀批量删除完整性**
    - 创建 `server/internal/cache/cache_prefix_delete_test.go`
    - 使用 gopter 生成任意前缀和缓存键集合
    - 验证 DeleteByPrefix 后所有匹配键被删除，其他键保持不变
    - **Validates: Requirements 1.3, 6.4, 6.5**

  - [x] 1.10 创建缓存配置结构
    - 创建 `server/internal/cache/config.go` 文件
    - 实现 Config 结构体，包含 Enabled、DefaultTTL、HitRateThreshold 字段
    - 实现 TTLConfig 结构体，包含各模块的 TTL 配置
    - 定义默认 TTL 常量（待办3分钟、归档5分钟、配置10分钟、统计5分钟、仪表盘2分钟）
    - _Requirements: 1.2, 2.4, 2.7, 3.4, 3.7, 4.3, 5.3, 5.6_

  - [x] 1.11 创建缓存数据模型
    - 创建 `server/internal/cache/models.go` 文件
    - 实现 CachedTodoList、CachedArchiveList 结构体
    - 实现 CachedProcessConfig、CachedSnapshot、CachedStats 结构体
    - 实现 CacheStatsSnapshot 结构体
    - 所有结构体包含 CachedAt 时间戳字段
    - _Requirements: 1.6, 2.1, 3.1_

- [x] 2. Checkpoint - 确保缓存基础设施测试通过
  - 运行 `go test ./internal/cache/...` 确保所有测试通过
  - 如有问题请询问用户

- [x] 3. 集成 AuditExecuteService 缓存
  - [x] 3.1 修改 AuditExecuteService 添加缓存依赖
    - 修改 `server/internal/service/audit_review_service.go`
    - 添加 cache.CacheManager 和 cache.InvalidationManager 字段
    - 修改构造函数接受缓存依赖
    - _Requirements: 2.1, 2.2_

  - [x] 3.2 实现待办列表缓存逻辑
    - 修改 ListPendingProcesses 方法
    - 构建缓存键：`audit:todo:{tenant_id}:{user_id}:{filter_hash}`
    - 优先从缓存读取，未命中时查询 OA 并写入缓存
    - 设置 TTL 为 3 分钟
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [x] 3.3 实现审核配置缓存逻辑
    - 修改获取流程配置的方法
    - 构建缓存键：`audit:config:{tenant_id}:{process_type}`
    - 设置 TTL 为 10 分钟
    - _Requirements: 2.6, 2.7_

  - [x] 3.4 实现审核操作后缓存失效
    - 修改 Execute 方法
    - 审核完成后清除用户待办列表缓存
    - 清除相关快照缓存和统计缓存
    - _Requirements: 2.5, 4.4_

  - [ ]* 3.5 编写审核缓存失效属性测试
    - **Property 4: 审核操作后缓存失效**
    - 创建 `server/internal/service/audit_cache_invalidation_test.go`
    - 验证审核操作后相关缓存被清除
    - **Validates: Requirements 2.5, 4.4, 6.2**

- [x] 4. 集成 ArchiveReviewService 缓存
  - [x] 4.1 修改 ArchiveReviewService 添加缓存依赖
    - 修改 `server/internal/service/archive_review_service.go`
    - 添加 cache.CacheManager 和 cache.InvalidationManager 字段
    - 修改构造函数接受缓存依赖
    - _Requirements: 3.1, 3.2_

  - [x] 4.2 实现归档列表缓存逻辑
    - 修改 ListProcessesPaged 方法
    - 构建缓存键：`archive:list:{tenant_id}:{user_id}:{filter_hash}`
    - 优先从缓存读取，未命中时查询并写入缓存
    - 设置 TTL 为 5 分钟
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

  - [x] 4.3 实现归档配置缓存逻辑
    - 修改获取归档配置的方法
    - 构建缓存键：`archive:config:{tenant_id}:{process_type}`
    - 设置 TTL 为 10 分钟
    - _Requirements: 3.6, 3.7_

  - [x] 4.4 实现复盘操作后缓存失效
    - 修改复盘执行方法
    - 复盘完成后清除用户归档列表缓存
    - 清除相关快照缓存和统计缓存
    - _Requirements: 3.5, 4.5_

  - [ ]* 4.5 编写归档缓存失效属性测试
    - **Property 5: 复盘操作后缓存失效**
    - 创建 `server/internal/service/archive_cache_invalidation_test.go`
    - 验证复盘操作后相关缓存被清除
    - **Validates: Requirements 3.5, 4.5, 6.3**

- [x] 5. 实现快照和统计数据缓存
  - [x] 5.1 实现审核快照缓存
    - 修改快照查询方法
    - 构建缓存键：`audit:snapshot:{tenant_id}:{process_ids_hash}`
    - 设置 TTL 为 5 分钟
    - 支持批量查询优化
    - _Requirements: 4.1, 4.3, 4.6_

  - [x] 5.2 实现归档快照缓存
    - 修改归档快照查询方法
    - 构建缓存键：`archive:snapshot:{tenant_id}:{process_ids_hash}`
    - 设置 TTL 为 5 分钟
    - _Requirements: 4.2, 4.3_

  - [x] 5.3 实现统计数据缓存
    - 修改统计查询方法
    - 构建缓存键：`audit:stats:{tenant_id}:{user_id}:{date_range_hash}` 和 `archive:stats:{tenant_id}:{user_id}:{date_range_hash}`
    - 设置 TTL 为 5 分钟
    - _Requirements: 5.1, 5.2, 5.3_

  - [x] 5.4 实现仪表盘缓存
    - 修改 DashboardOverviewService
    - 构建缓存键：`dashboard:{tenant_id}:{user_id}:{role}`
    - 设置 TTL 为 2 分钟
    - _Requirements: 5.5, 5.6_

  - [ ]* 5.5 编写分页缓存属性测试
    - **Property 8: 分页缓存一致性**
    - 创建 `server/internal/cache/paged_cache_test.go`
    - 验证分页缓存数据与原始数据切片一致
    - **Validates: Requirements 8.4**

- [x] 6. Checkpoint - 确保服务层集成测试通过
  - 运行服务层相关测试确保缓存集成正确
  - 如有问题请询问用户

- [x] 7. 实现缓存失效策略
  - [x] 7.1 实现租户配置变更缓存失效
    - 在租户配置更新时调用 InvalidateTenantCache
    - 清除该租户的所有配置相关缓存
    - _Requirements: 6.1_

  - [x] 7.2 实现审核规则变更缓存失效
    - 在审核规则更新时调用 InvalidateConfigCache
    - 清除该租户的审核配置缓存
    - _Requirements: 6.2_

  - [x] 7.3 实现归档规则变更缓存失效
    - 在归档规则更新时调用 InvalidateConfigCache
    - 清除该租户的归档配置缓存
    - _Requirements: 6.3_

  - [x] 7.4 实现 OA 连接配置变更缓存失效
    - 在 OA 连接配置更新时清除相关缓存
    - 清除该租户的所有 OA 数据相关缓存
    - _Requirements: 6.6_

  - [ ]* 7.5 编写租户缓存失效属性测试
    - **Property 6: 租户配置变更后缓存失效**
    - 创建 `server/internal/cache/tenant_cache_invalidation_test.go`
    - 验证租户配置变更后该租户缓存被清除，其他租户不受影响
    - **Validates: Requirements 6.1, 6.6**

- [x] 8. 创建缓存管理接口
  - [x] 8.1 创建 CacheAdminHandler
    - 创建 `server/internal/handler/cache_admin_handler.go` 文件
    - 实现 NewCacheAdminHandler 构造函数
    - 实现 GetStats 方法（GET /api/admin/cache/stats）
    - 实现 ClearTenantCache 方法（DELETE /api/admin/cache/tenant/:tenant_id）
    - 实现 ClearModuleCache 方法（DELETE /api/admin/cache/module/:module）
    - 实现 ToggleCache 方法（POST /api/admin/cache/toggle）
    - _Requirements: 6.4, 6.5, 7.3, 7.5_

  - [x] 8.2 注册缓存管理路由
    - 修改 `server/internal/router/router.go`
    - 在 admin 路由组下注册缓存管理接口
    - 添加超级管理员权限验证中间件
    - _Requirements: 6.4, 6.5_

  - [ ]* 8.3 编写缓存管理接口单元测试
    - 创建 `server/internal/handler/cache_admin_handler_test.go`
    - 测试各管理接口的正确性
    - _Requirements: 6.4, 6.5, 7.5_

- [x] 9. 实现缓存监控与日志
  - [x] 9.1 实现缓存操作日志记录
    - 在 CacheManager 中添加操作日志
    - 记录缓存命中/未命中信息
    - 记录缓存操作失败的错误详情
    - _Requirements: 7.1, 7.2_

  - [x] 9.2 实现命中率告警
    - 在 GetStats 中计算命中率
    - 当命中率低于阈值时记录警告日志
    - _Requirements: 7.4_

  - [ ]* 9.3 编写日志记录单元测试
    - 创建 `server/internal/cache/cache_logging_test.go`
    - 验证各场景下日志记录的完整性
    - _Requirements: 7.2_

- [x] 10. 更新依赖注入和配置
  - [x] 10.1 修改主程序初始化缓存组件
    - 修改 `server/cmd/server/main.go`
    - 初始化 CacheManager 和 InvalidationManager
    - 注入到各服务构造函数
    - _Requirements: 1.1_

  - [x] 10.2 扩展配置文件
    - 修改 `server/config/config.go` 添加缓存配置结构
    - 修改 `server/config.yaml` 添加缓存配置项
    - 包含 enabled、default_ttl、hit_rate_threshold 和各模块 TTL
    - _Requirements: 1.2, 7.3_

  - [x] 10.3 添加错误码定义
    - 修改 `server/internal/errcode/errcode.go`
    - 添加缓存相关错误码（50001-50006）
    - _Requirements: 7.2_

- [x] 11. Checkpoint - 确保所有测试通过
  - 运行 `go test ./...` 确保所有测试通过
  - 如有问题请询问用户

- [ ] 12. 性能验证
  - [ ]* 12.1 编写性能测试
    - 创建 `server/internal/cache/cache_performance_test.go`
    - 验证缓存命中时响应时间 < 200ms
    - 验证缓存读写操作延迟 < 50ms
    - _Requirements: 8.1, 8.2, 8.3_

  - [ ]* 12.2 编写降级机制测试
    - 创建 `server/internal/cache/cache_fallback_test.go`
    - 验证 Redis 不可用时的降级行为
    - 验证降级时记录警告日志
    - _Requirements: 1.4_

- [x] 13. Final Checkpoint - 确保所有测试通过
  - 运行完整测试套件确保功能正确
  - 如有问题请询问用户

## Notes

- 任务标记 `*` 为可选任务，可跳过以加快 MVP 开发
- 每个任务引用具体需求以确保可追溯性
- Checkpoint 任务用于增量验证
- 属性测试验证通用正确性属性
- 单元测试验证具体示例和边界条件
- 使用 Go 语言和 `github.com/leanovate/gopter` 进行属性测试
