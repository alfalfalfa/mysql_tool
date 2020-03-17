package cmd

import (
	"fmt"
	"os"

	"io/ioutil"

	"strings"

	"encoding/json"

	"bytes"

	"github.com/app-studio/mysql_tool/models"
	"github.com/app-studio/mysql_tool/util/copy"
	jsonutil "github.com/app-studio/mysql_tool/util/json"
	"github.com/docopt/docopt-go"
	"github.com/jinzhu/gorm"
	"github.com/app-studio/xlsx"
)

const usageData = `mysql_tool data
    データ定義の変換、mysql入出力

Usage:
    mysql_tool data -h | --help
    mysql_tool data [-f FORMAT] [-o OUTPUT] [--tables TABLES...] [--ignore-tables IGNORE_TABLES...] [--defines INPUTS...] [--skip-truncate] INPUTS...

Arg:
    入力ファイルパス（json,xlsx） | mysql fqdn

Options:
    -h --help                             Show this screen.
    -f FORMAT, --format=FORMAT            出力フォーマット
        "sql"
            insert文を出力
        "json"
            Json出力
        "xlsx"
            Excel出力
        none
            OUTPUTの拡張子から自動判別 （default: sql）
    -o OUTPUT, --output=OUTPUT            出力先
        ファイルパス
            上書き
        none
            標準出力（Excel出力では無効）
    --tables=TABLES...                    対象テーブル
    --ignore-tables=IGNORE_TABLES...      無視テーブル
    --skip-truncate                       テーブルの削除を実行しない
	--defines=INPUTS...                    DB定義ファイル
`

type DataArg struct {
	Format       string   `arg:"--format"`
	Output       string   `arg:"--output"`
	Inputs       []string `arg:"INPUTS"`
	Tables       []string `arg:"--tables"`
	IgnoreTables []string `arg:"--ignore-tables"`
	SkipTruncate bool     `arg:"--skip-truncate"`
	Defines      []string `arg:"--defines"`
}

func RunData() {
	arguments, err := docopt.Parse(usageData, os.Args[1:], true, "", false)
	checkError(err)

	arg := &DataArg{}
	copy.MapToStructWithTag(arguments, arg, "arg")
	//fmt.Println(json.ToJson(arg))

	d := loadData(arg)

	format := detectOutputFormat(arg.Format, arg.Output)
	b := marshalData(d, format, arg)

	if arg.Output == "" {
		if format == "xlsx" {
			panic("xlsx stdout")
		}
		fmt.Println(string(b))
	} else {
		checkError(ioutil.WriteFile(arg.Output, b, os.ModePerm))
	}
}

func loadData(arg *DataArg) *Data {
	switch models.DetectInputFormat(arg.Inputs[0]) {
	case "xlsx":
		return NewDataFromExcel(arg)
	case "json":
		return NewDataFromJson(arg)
	case "mysql":
		return NewDataFromMysql(arg)
	}
	panic(fmt.Sprint("input format invalid:", arg.Inputs))
}

func marshalData(d *Data, format string, args *DataArg) []byte {
	switch format {
	case "xlsx":
		return d.ToExcelFile()
	case "json":
		return []byte(jsonutil.ToJson(d))
	case "sql":
		return []byte(d.ToSQL(args))
	}
	panic(fmt.Sprint("output format invalid:", format))
}

type tableMaps struct {
	Tables map[string]*models.Table
}

func (this *tableMaps) ContainsTable(tableName string) bool {
	if _, ok := this.Tables[tableName]; ok {
		return true
	}
	return false
}

func (this *tableMaps) ContainsColumn(tableName, columnName string) bool {
	if !this.ContainsTable(tableName) {
		return false
	}

	if col := this.Tables[tableName].GetColumn(columnName); col != nil {
		return true
	}
	return false
}

func (this *tableMaps) GetColumn(tableName, columnName string) *models.Column {
	if !this.ContainsTable(tableName) {
		return nil
	}
	return this.Tables[tableName].GetColumn(columnName)
}

func loadModelToMap(arg *DataArg) *tableMaps {
	m := models.LoadModel(arg.IgnoreTables, arg.Defines...)
	res := &tableMaps{
		Tables: make(map[string]*models.Table),
	}
	for _, table := range m.Tables {
		res.Tables[table.Name.Lower()] = table
	}
	return res
}

//==========================================================================
type Data struct {
	Tables []*TableData
}
type TableData struct {
	Name   string
	Keys   []string
	Values [][]string
}

func NewDataFromExcel(arg *DataArg) *Data {
	res := &Data{
		Tables: make([]*TableData, 0),
	}
	for _, path := range arg.Inputs {
		file, err := xlsx.OpenFile(path)
		checkError(err)
		for _, sheet := range file.Sheets {
			if strings.HasPrefix(sheet.Name, "_") {
				continue
			}
			if !containsOrEmpty(arg.Tables, sheet.Name) {
				continue
			}
			if contains(arg.IgnoreTables, sheet.Name) {
				continue
			}
			//fmt.Println(path, sheet.Name)
			t := NewDataFromExcelSheet(sheet)
			if t == nil || len(t.Values) == 0 {
				continue
			}
			// 同名テーブルはマージ(重複チェックはしない)
			var exist *TableData = nil
			for _, et := range res.Tables {
				if et.Name == t.Name {
					exist = et

					// 新規分を追加
					newRow := make([][]string, 0)
					for _, ov := range et.Values {
						newRow = append(newRow, ov)
					}
					for _, nv := range t.Values {
						newRow = append(newRow, nv)
					}
					et.Values = newRow
				}
			}
			// 既存が見つからなかった場合は追加
			if exist == nil {
				res.Tables = append(res.Tables, t)
			}
		}
	}
	return res
}

func NewDataFromExcelSheet(sheet *xlsx.Sheet) *TableData {
	res := &TableData{
		Name:   sheet.Name,
		Keys:   make([]string, 0),
		Values: make([][]string, 0),
	}
	// keyのcell位置を保持
	keyIndexMap := make(map[string]int)

	for i, row := range sheet.Rows {
		if i == 0 {
			//Keys
			for n, _ := range row.Cells {
				v := getCellValue(row, n)
				if v == "" {
					continue
				}
				if strings.HasPrefix(v, "__") {
					continue
				}
				res.Keys = append(res.Keys, v)
				keyIndexMap[v] = n
			}
		} else {
			//Values
			v := getCellValue(row, 0)
			if v == "" {
				continue
			}
			dataRow := make([]string, 0)
			for _, key := range res.Keys {
				n := keyIndexMap[key]
				dataRow = append(dataRow, escapeValue(getCellValue(row, n)))
			}
			res.Values = append(res.Values, dataRow)
		}
	}
	return res
}

func NewDataFromJson(arg *DataArg) *Data {
	tableDatas := make([]*TableData, 0)
	for _, path := range arg.Inputs {
		m := &Data{}
		b, err := ioutil.ReadFile(path)
		checkError(err)
		err = json.Unmarshal(b, m)
		checkError(err)

		targets := make([]*TableData, 0)
		for _, t := range m.Tables {
			if !containsOrEmpty(arg.Tables, t.Name) {
				continue
			}
			if contains(arg.IgnoreTables, t.Name) {
				continue
			}
			targets = append(targets, t)
		}

		tableDatas = append(tableDatas, targets...)
	}

	res := &Data{}
	res.Tables = tableDatas
	return res
}

func NewDataFromMysql(arg *DataArg) *Data {
	fqdn := arg.Inputs[0]
	db, err := gorm.Open("mysql", fqdn)
	checkError(err)
	res := &Data{
		Tables: make([]*TableData, 0),
	}
	for _, tableInfo := range models.LoadMysqlTables(db) {
		if !containsOrEmpty(arg.Tables, tableInfo.Name) {
			continue
		}
		if contains(arg.IgnoreTables, tableInfo.Name) {
			continue
		}
		t := NewTableDataFromMysql(db, tableInfo)
		if len(t.Values) == 0 {
			continue
		}
		res.Tables = append(res.Tables, t)
	}

	return res
}

func NewTableDataFromMysql(db *gorm.DB, tableInfo models.MysqlTable) *TableData {
	res := &TableData{
		Name:   tableInfo.GetName(),
		Keys:   make([]string, 0),
		Values: make([][]string, 0),
	}

	rows, err := db.DB().Query("select * from `" + tableInfo.GetName() + "`")
	defer rows.Close()
	for rows.Next() {
		if len(res.Keys) == 0 {
			columns, err := rows.Columns()
			checkError(err)

			res.Keys = columns
			//fmt.Println(res.Keys)
		}

		valueRefs := make([]*string, 0)
		valueRefsForScan := make([]interface{}, 0)
		for range res.Keys {
			var v string
			valueRefs = append(valueRefs, &v)
			valueRefsForScan = append(valueRefsForScan, &v)
		}

		rows.Scan(valueRefsForScan...)
		values := make([]string, 0)
		for _, v := range valueRefs {
			values = append(values, *v)
		}
		res.Values = append(res.Values, values)
	}
	err = rows.Err()
	checkError(err)

	return res
}

func (this Data) ToExcelFile() []byte {
	file := xlsx.NewFile()

	for _, t := range this.Tables {
		sheet, err := file.AddSheet(t.Name)
		checkError(err)
		t.ToExcelSheet(sheet)
	}

	buf := bytes.NewBuffer(nil)
	file.Write(buf)
	return buf.Bytes()
}

func (this TableData) ToExcelSheet(sheet *xlsx.Sheet) {
	//カラムヘッダー行
	tableHeaderRow := sheet.AddRow()
	for _, v := range this.Keys {
		models.SetHeaderStyle(tableHeaderRow.AddCell()).SetValue(v)
	}

	//データ行
	for _, values := range this.Values {
		tableRow := sheet.AddRow()
		for _, v := range values {
			tableRow.AddCell().SetValue(v)
		}
	}
}

func (this Data) ToSQL(args *DataArg) string {
	var defines *tableMaps = nil
	if 0 < len(args.Defines) {
		defines = loadModelToMap(args)
	}

	buf := bytes.NewBuffer(nil)
	for _, t := range this.Tables {
		// TRUNCATEのフラグ制御
		if !args.SkipTruncate {
			buf.WriteString("TRUNCATE `")
			buf.WriteString(t.Name)
			buf.WriteString("`;\n")
		}

		buf.WriteString("INSERT INTO `")
		buf.WriteString(t.Name)
		buf.WriteString("` ")

		//keys
		keys := make([]string, 0)
		for _, k := range t.Keys {
			keys = append(keys, "`"+k+"`")
		}
		buf.WriteString("(")
		buf.WriteString(strings.Join(keys, ", "))
		buf.WriteString(") ")

		valuesList := make([]string, 0)
		for _, row := range t.Values {
			values := make([]string, 0)
			for i, v := range row {
				if strings.ToLower(v) == "null" {
					values = append(values, "null")

				} else if v == "" && defines != nil {
					// データが空の場合、columnを取得
					if col := defines.GetColumn(t.Name, t.Keys[i]); col != nil {
						// columnがある場合はdefaultを突っ込む
						values = append(values, "'"+col.Default.ValueOrZero()+"'")
					} else {
						// columnがない場合はそのまま突っ込む
						values = append(values, "'"+v+"'")
					}

				} else {
					values = append(values, "'"+v+"'")
				}
			}
			valuesList = append(valuesList, "("+strings.Join(values, ",")+")")
		}

		buf.WriteString("\nVALUES \n")
		buf.WriteString(strings.Join(valuesList, ",\n"))
		buf.WriteString(";\n\n")
	}
	return buf.String()
}

func getCellValue(row *xlsx.Row, num int) string {
	if len(row.Cells) <= num {
		return ""
	}
	formatedValue, err := row.Cells[num].FormattedValue()
	// formatした値が取得できない場合は元の値を返す
	if err != nil {
		return strings.TrimSpace(row.Cells[num].Value)
	}
	return strings.TrimSpace(formatedValue)
}
func escapeValue(v string) string {
	v = strings.Replace(v, "\\", "\\\\", -1)
	v = strings.Replace(v, "'", "\\'", -1)
	return v
}

func containsOrEmpty(s []string, e string) bool {
	if len(s) == 0 {
		return true
	}
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func contains(s []string, e string) bool {
	if s == nil {
		return false
	}
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
