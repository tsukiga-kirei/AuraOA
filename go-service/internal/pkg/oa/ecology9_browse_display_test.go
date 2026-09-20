package oa

import "testing"

// 测试数据展现中心（datashowset + datashowparam）确定性解析
func TestBrowseTargetFromDataShowSetSuccess(t *testing.T) {
	target, ok := browseTargetFromDataShowSet(e9DataShowSetDef{
		DataFrom: "1",
		SQLText:  "select htbh, htmc, bz from uf_contract where status=1",
		KeyField: "htbh",
	}, "htmc")
	if !ok {
		t.Fatal("expected valid datashowset to resolve")
	}
	if target.Table != "uf_contract" || target.IDColumn != "htbh" || target.DisplayColumn != "htmc" {
		t.Fatalf("unexpected target: %+v", target)
	}
	if target.NumericID {
		t.Fatal("htbh should not be assumed numeric")
	}
}

// 缺失任一要素时绝不猜测，严格直接返回 false（保留数据库原值）
func TestBrowseTargetFromDataShowSetMissingElements(t *testing.T) {
	// 缺少显示列（datashowparam 无合法 searchname）
	if _, ok := browseTargetFromDataShowSet(e9DataShowSetDef{
		DataFrom: "1",
		SQLText:  "select id, name from uf_item",
		KeyField: "id",
	}, ""); ok {
		t.Fatal("missing titleField should fail and retain raw value")
	}

	// 缺少主键（keyfield 为空）
	if _, ok := browseTargetFromDataShowSet(e9DataShowSetDef{
		DataFrom: "1",
		SQLText:  "select id, name from uf_item",
		KeyField: "",
	}, "name"); ok {
		t.Fatal("missing keyfield should fail and retain raw value")
	}

	// 缺少底表（SQL 无法解析 FROM）
	if _, ok := browseTargetFromDataShowSet(e9DataShowSetDef{
		DataFrom: "1",
		SQLText:  "select 1",
		KeyField: "id",
	}, "name"); ok {
		t.Fatal("missing FROM table should fail and retain raw value")
	}

	// 非数据库数据源（WebService / 自定义页面）
	if _, ok := browseTargetFromDataShowSet(e9DataShowSetDef{
		DataFrom: "0",
		SQLText:  "select id from uf_item",
		KeyField: "id",
	}, "name"); ok {
		t.Fatal("webservice datashowset should fail and retain raw value")
	}
}

// 测试表单建模基础表（mode_browser）解析
func TestModeBrowserTargetFromDef(t *testing.T) {
	// 场景 1：具有明确配置的 keyfield + showfield（优先使用）
	target1, ok := modeBrowserTargetFromDef(e9ModeBrowserDef{
		SQLText:   "select cbzxbm, cbzxmc from uf_cbzx where gsdm='$gc$'",
		KeyField:  "cbzxbm",
		ShowField: "cbzxmc",
	})
	if !ok {
		t.Fatal("expected explicit keyfield and showfield to resolve")
	}
	if target1.Table != "uf_cbzx" || target1.IDColumn != "cbzxbm" || target1.DisplayColumn != "cbzxmc" {
		t.Fatalf("unexpected target: %+v", target1)
	}

	// 场景 2：现场真实常见情况：keyfield 与 showfield 为空，从 searchbyid 语句解析回显定义
	target2, ok := modeBrowserTargetFromDef(e9ModeBrowserDef{
		SearchByID: "select name from uf_customer where id=?",
	})
	if !ok {
		t.Fatal("expected searchbyid to resolve")
	}
	if target2.Table != "uf_customer" || target2.IDColumn != "id" || target2.DisplayColumn != "name" {
		t.Fatalf("unexpected target: %+v", target2)
	}

	// 场景 3：现场真实常见情况：keyfield 与 showfield 为空，从 sqltext 列表语句解析
	target3, ok := modeBrowserTargetFromDef(e9ModeBrowserDef{
		SQLText: "select code, title from uf_project",
	})
	if !ok {
		t.Fatal("expected sqltext to resolve")
	}
	if target3.Table != "uf_project" || target3.IDColumn != "code" || target3.DisplayColumn != "title" {
		t.Fatalf("unexpected target: %+v", target3)
	}

	// 场景 4：无法解析时安全返回 false
	if _, ok := modeBrowserTargetFromDef(e9ModeBrowserDef{}); ok {
		t.Fatal("empty mode_browser should fail and retain raw value")
	}
}
