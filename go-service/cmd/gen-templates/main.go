// gen-templates generates the member import Excel template files for both
// Chinese and English locales and writes them to the templates directory.
package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/xuri/excelize/v2"
)

func main() {
	outDir := filepath.Join("internal", "pkg", "excel", "templates")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("mkdir %s: %v", outDir, err)
	}

	// Chinese template
	if err := writeTemplate(
		filepath.Join(outDir, "member_import_zh.xlsx"),
		[]string{"姓名", "用户名", "部门", "角色"},
		[]string{"张三", "zhangsan", "研发部", "开发工程师"},
	); err != nil {
		log.Fatalf("write zh template: %v", err)
	}
	log.Println("generated member_import_zh.xlsx")

	// English template
	if err := writeTemplate(
		filepath.Join(outDir, "member_import_en.xlsx"),
		[]string{"Name", "Username", "Department", "Roles"},
		[]string{"John Doe", "johndoe", "Engineering", "Developer"},
	); err != nil {
		log.Fatalf("write en template: %v", err)
	}
	log.Println("generated member_import_en.xlsx")
}

func writeTemplate(path string, headers, example []string) error {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "Sheet1"

	// Write header row (row 1)
	for col, h := range headers {
		cell, err := excelize.CoordinatesToCellName(col+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, h); err != nil {
			return err
		}
	}

	// Write example row (row 2)
	for col, v := range example {
		cell, err := excelize.CoordinatesToCellName(col+1, 2)
		if err != nil {
			return err
		}
		if err := f.SetCellValue(sheet, cell, v); err != nil {
			return err
		}
	}

	return f.SaveAs(path)
}
