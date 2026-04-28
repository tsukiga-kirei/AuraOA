package excel

import (
	"bytes"
	"errors"
	"io"
	"mime/multipart"
	"strings"

	"github.com/xuri/excelize/v2"
)

// ErrFileTooLarge 是文件超过允许大小时返回的哨兵错误。
var ErrFileTooLarge = errors.New("file too large")

// MemberRow 表示从 Excel 解析出的一条成员记录。
type MemberRow struct {
	// Name 是成员姓名（A 列）。
	Name string
	// Username 是登录用户名（B 列）。
	Username string
	// DepartmentName 是所属部门名称（C 列）。
	DepartmentName string
	// RoleNames 是角色名称列表，由 D 列逗号分隔后拆分而来。
	RoleNames []string
}

// ImportError 表示单行导入失败的详情。
type ImportError struct {
	RowNumber int    `json:"row_number"`
	Reason    string `json:"reason"`
}

// ParseMemberImport 解析上传的 Excel 文件，返回有效行和错误行。
// 从第一个 Sheet 的第二行开始读取（第一行为表头）。
// 列顺序：A=姓名, B=用户名, C=部门, D=角色（逗号分隔）。
// 文件大小超过 maxBytes 时返回 ErrFileTooLarge。
func ParseMemberImport(file multipart.File, maxBytes int64) ([]MemberRow, []ImportError, error) {
	// 读取至多 maxBytes+1 字节，用于判断是否超限
	limited := io.LimitReader(file, maxBytes+1)
	buf, err := io.ReadAll(limited)
	if err != nil {
		return nil, nil, err
	}
	if int64(len(buf)) > maxBytes {
		return nil, nil, ErrFileTooLarge
	}

	// 使用 excelize 解析内存中的 Excel 数据
	f, err := excelize.OpenReader(bytes.NewReader(buf))
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	// 读取第一个 Sheet 的所有行
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return []MemberRow{}, []ImportError{}, nil
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, nil, err
	}

	var members []MemberRow
	var importErrors []ImportError

	// 从第二行（索引 1）开始，跳过表头
	for i := 1; i < len(rows); i++ {
		rowNum := i + 1 // Excel 行号从 1 开始，表头为第 1 行
		row := rows[i]

		// 安全读取各列，不足时视为空字符串
		cellVal := func(col int) string {
			if col < len(row) {
				return strings.TrimSpace(row[col])
			}
			return ""
		}

		name := cellVal(0)
		username := cellVal(1)
		deptName := cellVal(2)
		rolesRaw := cellVal(3)

		// 校验必填字段
		if name == "" || username == "" || deptName == "" {
			var missing []string
			if name == "" {
				missing = append(missing, "姓名")
			}
			if username == "" {
				missing = append(missing, "用户名")
			}
			if deptName == "" {
				missing = append(missing, "部门")
			}
			importErrors = append(importErrors, ImportError{
				RowNumber: rowNum,
				Reason:    "必填字段为空: " + strings.Join(missing, ", "),
			})
			continue
		}

		// 解析角色列表：逗号分隔，去空格，过滤空字符串
		var roleNames []string
		if rolesRaw != "" {
			for _, r := range strings.Split(rolesRaw, ",") {
				r = strings.TrimSpace(r)
				if r != "" {
					roleNames = append(roleNames, r)
				}
			}
		}

		members = append(members, MemberRow{
			Name:           name,
			Username:       username,
			DepartmentName: deptName,
			RoleNames:      roleNames,
		})
	}

	return members, importErrors, nil
}
