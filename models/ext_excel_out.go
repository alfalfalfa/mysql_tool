package models

import (
	"bytes"
	"strings"

	"github.com/tealeg/xlsx"
)

var headerStyle *xlsx.Style

func init() {
	var style = xlsx.NewStyle()
	style.Fill = *xlsx.NewFill("solid", "00606060", "00FFFFFF")
	style.ApplyFill = true
	style.Font.Color = "00FFFFFF"
	headerStyle = style
}

func (this Models) ToExcelFile() []byte {
	file := xlsx.NewFile()

	for _, t := range this.Tables {
		sheet, err := file.AddSheet(t.Name.LowerSnake())
		checkError(err)
		t.ToExcelSheet(sheet)
	}

	buf := bytes.NewBuffer(nil)
	file.Write(buf)
	return buf.Bytes()
}

func (this Table) ToExcelSheet(sheet *xlsx.Sheet) {
	//テーブルタイトル行
	tableHeaderRow := sheet.AddRow()
	SetHeaderStyle(tableHeaderRow.AddCell()).SetValue("Table")
	SetHeaderStyle(tableHeaderRow.AddCell()).SetValue("物理名")
	SetHeaderStyle(tableHeaderRow.AddCell()).SetValue("ENGINE")
	SetHeaderStyle(tableHeaderRow.AddCell()).SetValue("DEFAULT CHARSET")
	SetHeaderStyle(tableHeaderRow.AddCell()).SetValue("DB INDEX")
	SetHeaderStyle(tableHeaderRow.AddCell()).SetValue("CONNECTION INDEX")
	SetHeaderStyle(tableHeaderRow.AddCell()).SetValue("")
	SetHeaderStyle(tableHeaderRow.AddCell()).SetValue("")
	SetHeaderStyle(tableHeaderRow.AddCell()).SetValue("COMMENT")
	SetHeaderStyle(tableHeaderRow.AddCell()).SetValue("メタデータ(JSON)")
	SetHeaderStyle(tableHeaderRow.AddCell()).SetValue("備考")
	//sheet.Col(0).SetStyle(getHeaderStyle())

	//テーブル行
	tableRow := sheet.AddRow()
	SetHeaderStyle(tableRow.AddCell()).SetValue("")
	tableRow.AddCell().SetValue(this.Name)
	tableRow.AddCell().SetValue(this.Engine)
	tableRow.AddCell().SetValue(this.DefaultCharset)
	tableRow.AddCell().SetValue(this.DbIndex)
	tableRow.AddCell().SetValue(this.ConnectionIndex)
	tableRow.AddCell().SetValue("")
	tableRow.AddCell().SetValue("")
	tableRow.AddCell().SetValue(this.Comment)
	//TODO metadata
	tableRow.AddCell().SetValue("")
	for _, v := range this.Descriptions{
		tableRow.AddCell().SetValue(v)
	}

	//カラムヘッダー行
	columnHeaderRow := sheet.AddRow()
	SetHeaderStyle(columnHeaderRow.AddCell()).SetValue("Columns")
	SetHeaderStyle(columnHeaderRow.AddCell()).SetValue("物理名")
	SetHeaderStyle(columnHeaderRow.AddCell()).SetValue("型")
	SetHeaderStyle(columnHeaderRow.AddCell()).SetValue("Nullable")
	SetHeaderStyle(columnHeaderRow.AddCell()).SetValue("PK")
	SetHeaderStyle(columnHeaderRow.AddCell()).SetValue("Default")
	SetHeaderStyle(columnHeaderRow.AddCell()).SetValue("Extra")
	SetHeaderStyle(columnHeaderRow.AddCell()).SetValue("REF")
	SetHeaderStyle(columnHeaderRow.AddCell()).SetValue("COMMENT")
	SetHeaderStyle(columnHeaderRow.AddCell()).SetValue("メタデータ(JSON)")
	SetHeaderStyle(columnHeaderRow.AddCell()).SetValue("備考")

	//Columns
	for _, c := range this.Columns {
		row := sheet.AddRow()
		c.ToExcelRow(row)
	}
	//インデックスヘッダー行
	indexHeaderRow := sheet.AddRow()
	SetHeaderStyle(indexHeaderRow.AddCell()).SetValue("Indexes")
	SetHeaderStyle(indexHeaderRow.AddCell()).SetValue("Index名")
	SetHeaderStyle(indexHeaderRow.AddCell()).SetValue("対象カラム名")
	SetHeaderStyle(indexHeaderRow.AddCell()).SetValue("UNIQUE KEY")
	SetHeaderStyle(indexHeaderRow.AddCell()).SetValue("")
	SetHeaderStyle(indexHeaderRow.AddCell()).SetValue("")
	SetHeaderStyle(indexHeaderRow.AddCell()).SetValue("")
	SetHeaderStyle(indexHeaderRow.AddCell()).SetValue("")
	SetHeaderStyle(indexHeaderRow.AddCell()).SetValue("COMMENT")
	SetHeaderStyle(indexHeaderRow.AddCell()).SetValue("")
	SetHeaderStyle(indexHeaderRow.AddCell()).SetValue("備考")

	//Indexes
	for _, ix := range this.Indexes {
		row := sheet.AddRow()
		ix.ToExcelRow(row)
	}

}

func (this Column) ToExcelRow(row *xlsx.Row) {
	SetHeaderStyle(row.AddCell()).SetValue("")
	row.AddCell().SetValue(this.Name)
	row.AddCell().SetValue(this.Type)
	if this.NotNull {
		row.AddCell().SetValue("")
	} else {
		row.AddCell().SetValue("1")
	}
	if this.PrimaryKey == 0 {
		row.AddCell().SetValue("")
	} else {
		row.AddCell().SetValue(this.PrimaryKey)
	}

	row.AddCell().SetValue(this.Default)

	row.AddCell().SetValue(this.Extra)
	row.AddCell().SetValue(this.Reference)
	row.AddCell().SetValue(this.Comment)
	//TODO metadata
	row.AddCell().SetValue("")
	for _, v := range this.Descriptions{
		row.AddCell().SetValue(v)
	}
}

//TODO ASC,DESC
func (this Index) ToExcelRow(row *xlsx.Row) {
	SetHeaderStyle(row.AddCell()).SetValue("")
	row.AddCell().SetValue(this.Name)
	row.AddCell().SetValue(strings.Join(this.ColumnNames, ","))
	if this.Unique {
		row.AddCell().SetValue("1")
	} else {
		row.AddCell().SetValue("")
	}
	row.AddCell().SetValue("")
	row.AddCell().SetValue("")
	row.AddCell().SetValue("")
	row.AddCell().SetValue("")
	row.AddCell().SetValue(this.Comment)
	row.AddCell().SetValue("")
	for _, v := range this.Descriptions{
		row.AddCell().SetValue(v)
	}
}

func SetHeaderStyle(cell *xlsx.Cell) *xlsx.Cell {
	cell.SetStyle(headerStyle)
	return cell
}
