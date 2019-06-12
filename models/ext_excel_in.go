package models

import (
	"strconv"
	"strings"

	"github.com/app-studio/mysql_tool/util"
	"github.com/app-studio/mysql_tool/util/null"
	"github.com/tealeg/xlsx"
)

func loadTablesFromExcel(ignoreTables []string, path string) []*Table {
	tables := make([]*Table, 0)
	file, err := xlsx.OpenFile(path)
	checkError(err)
	for _, sheet := range file.Sheets {
		if strings.HasPrefix(sheet.Name, "_") {
			continue
		}

		// 無視するテーブル名を確認
		if contains(ignoreTables, sheet.Name) {
			continue
		}

		//fmt.Println(path, sheet.Name)
		t := NewTableFromExcelSheet(sheet)
		if t != nil {
			tables = append(tables, t)
		}
	}
	return tables
}

func NewTableFromExcelSheet(sheet *xlsx.Sheet) *Table {
	row := sheet.Rows[1]
	t := &Table{}
	t.Name = util.NewCaseString(getCellValue(row, 1))
	//t.LogicalName = strings.TrimSpace(row.Cells[2].Value)
	t.Engine = getCellValue(row, 2)
	t.DefaultCharset = getCellValue(row, 3)
	t.DbIndex = getCellValueAsInt(row, 4)
	t.ConnectionIndex = getCellValueAsInt(row, 5)
	t.Comment = getCellValue(row, 8)
	t.MetaDataJson = getCellValue(row, 9)
	t.Descriptions = getBelowCellValues(row, 10)

	t.Columns = NewColumnsFromExcelSheet(sheet)
	t.Indexes = NewIndexesFromExcelSheet(sheet)

	return t
}

func NewColumnsFromExcelSheet(sheet *xlsx.Sheet) []*Column {
	res := make([]*Column, 0)
	rownum := 3
	for {
		row := sheet.Rows[rownum]
		//物理名が空かA列に値が入っていれば中断
		if len(row.Cells) < 4 || strings.TrimSpace(row.Cells[0].Value) != "" || strings.TrimSpace(row.Cells[1].Value) == "" {
			break
		}
		column := NewColumnFromExcelRow(row)
		res = append(res, column)
		rownum++
	}
	return res
}

func NewColumnFromExcelRow(row *xlsx.Row) *Column {
	var err error

	c := &Column{}
	c.Name = util.NewCaseString(getCellValue(row, 1))
	//c.LogicalName = strings.TrimSpace(row.Cells[2].Value)
	c.Type = strings.ToLower(getCellValue(row, 2))
	c.NotNull = getCellValue(row, 3) == ""
	pkIndex := getCellValue(row, 4)
	if pkIndex != "" {
		c.PrimaryKey, err = strconv.Atoi(pkIndex)
		checkError(err)
	}
	c.Default = getDefaultCellValue(row, 5)

	c.Extra = getCellValue(row, 6)
	c.Reference = util.CamelToSnake(getCellValue(row, 7))
	c.Comment = getCellValue(row, 8)
	c.MetaDataJson = getCellValue(row, 9)
	c.Descriptions = getBelowCellValues(row, 10)

	return c
}

func NewIndexesFromExcelSheet(sheet *xlsx.Sheet) []*Index {
	res := make([]*Index, 0)
	rownum := 0
	for {
		row := sheet.Rows[rownum]
		rownum++
		if len(row.Cells) > 0 && strings.TrimSpace(row.Cells[0].Value) == "Indexes" {
			break
		}
	}

	for {
		if len(sheet.Rows) <= rownum {
			break
		}

		row := sheet.Rows[rownum]

		//Index名が空かA列に値が入っていれば中断
		if len(row.Cells) < 2 || strings.TrimSpace(row.Cells[0].Value) != "" || strings.TrimSpace(row.Cells[1].Value) == "" {
			break
		}
		index := NewIndexFromExcelRow(row)
		res = append(res, index)
		rownum++
	}
	return res
}

func NewIndexFromExcelRow(row *xlsx.Row) *Index {
	ix := &Index{}
	ix.Name = util.CamelToSnake(getCellValue(row, 1))
	ix.ColumnNames = strings.Split(getCellValue(row, 2), ",")
	for i, v := range ix.ColumnNames {
		ix.ColumnNames[i] = strings.TrimSpace(v)
	}
	ix.Unique = getCellValue(row, 3) != ""
	ix.Comment = getCellValue(row, 8)
	ix.Descriptions = getBelowCellValues(row, 10)
	return ix
}

func getBelowCellValues(row *xlsx.Row, from int) []string {
	column := from
	var res []string
	for {
		v := getCellValue(row, column)
		if v == "" {
			break
		}
		if res == nil {
			res = make([]string, 0)
		}
		res = append(res, v)
		column++
	}
	return res
}
func getCellValue(row *xlsx.Row, num int) string {
	if len(row.Cells) <= num {
		return ""
	}
	return strings.TrimSpace(row.Cells[num].Value)
}
func getNullableCellValue(row *xlsx.Row, num int) null.String {
	if len(row.Cells) <= num {
		return null.NullString()
	}
	v := strings.TrimSpace(row.Cells[num].Value)
	if v == "" {
		return null.NullString()
	}
	return null.StringFrom(v)
}
func getDefaultCellValue(row *xlsx.Row, num int) null.String {
	if len(row.Cells) <= num {
		return null.NullString()
	}
	v := strings.TrimSpace(row.Cells[num].Value)
	if v == "" {
		return null.NullString()
	}
	return null.StringFrom(strings.Trim(v, "'"))
}

func getCellValueAsInt(row *xlsx.Row, num int) int {
	v := getCellValue(row, num)
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return i
}
