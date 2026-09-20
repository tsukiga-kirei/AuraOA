package oa

import "testing"

func TestParseModeBrowserSQLTextTarget(t *testing.T) {
	target, ok := parseModeBrowserSQLTextTarget("select cbzxbm,cbzxbm,cbzxbm from uf_cbzx where gsdm='$gc$'")
	if !ok {
		t.Fatal("expected sqltext to parse")
	}
	if target.Table != "uf_cbzx" || target.IDColumn != "cbzxbm" || target.DisplayColumn != "cbzxbm" {
		t.Fatalf("got %+v", target)
	}
}

func TestParseModeBrowserSearchByIDTarget(t *testing.T) {
	target, ok := parseModeBrowserSearchByIDTarget("select cbzxbm,cbzxbm from uf_cbzx where cbzxbm=?")
	if !ok {
		t.Fatal("expected searchbyid to parse")
	}
	if target.Table != "uf_cbzx" || target.IDColumn != "cbzxbm" || target.DisplayColumn != "cbzxbm" {
		t.Fatalf("got %+v", target)
	}
	if target.NumericID {
		t.Fatal("cbzxbm should not be treated as numeric id")
	}
}

func TestBrowseTargetFromDataShowSetPrefersEchoSQL(t *testing.T) {
	target, ok := browseTargetFromDataShowSet(e9DataShowSetDef{
		DataFrom:   "1",
		SQLText:    "select id,name from uf_item where status=1",
		SearchByID: "select itemname from uf_item where itemcode=?",
		KeyField:   "id",
		ShowField:  "",
	}, "")
	if !ok {
		t.Fatal("expected datashowset echo sql to parse")
	}
	if target.Table != "uf_item" || target.IDColumn != "itemcode" || target.DisplayColumn != "itemname" {
		t.Fatalf("echo sql should win over list sql and keyfield, got %+v", target)
	}
}

func TestBrowseTargetFromDataShowSetUsesKeyAndTitle(t *testing.T) {
	target, ok := browseTargetFromDataShowSet(e9DataShowSetDef{
		DataFrom:  "1",
		SQLText:   "select id,name,code from uf_cbzx where gsdm='$gc$'",
		KeyField:  "cbzxbm",
		ShowField: "",
	}, "cbzxmc")
	if !ok {
		t.Fatal("expected keyfield + title field to build target")
	}
	if target.Table != "uf_cbzx" || target.IDColumn != "cbzxbm" || target.DisplayColumn != "cbzxmc" {
		t.Fatalf("got %+v", target)
	}
	if target.NumericID {
		t.Fatal("cbzxbm should not be treated as numeric id")
	}
}

func TestBrowseTargetFromDataShowSetSkipsWebService(t *testing.T) {
	if _, ok := browseTargetFromDataShowSet(e9DataShowSetDef{
		DataFrom:   "0",
		SearchByID: "select name from uf_item where id=?",
	}, "name"); ok {
		t.Fatal("webservice datashowset should not be queried from OA db")
	}
	if _, ok := browseTargetFromDataShowSet(e9DataShowSetDef{
		DataFrom:   "2",
		SearchByID: "select name from uf_item where id=?",
	}, "name"); ok {
		t.Fatal("custom page datashowset should not be queried from OA db")
	}
}

func TestBrowseTargetFromDataShowSetOverridesListSQLKey(t *testing.T) {
	target, ok := browseTargetFromDataShowSet(e9DataShowSetDef{
		DataFrom:  "1",
		SQLText:   "select id,name from uf_item",
		KeyField:  "code",
		ShowField: "itemname",
	}, "")
	if !ok {
		t.Fatal("expected list sql with key/show override")
	}
	if target.Table != "uf_item" || target.IDColumn != "code" || target.DisplayColumn != "itemname" {
		t.Fatalf("got %+v", target)
	}
}
