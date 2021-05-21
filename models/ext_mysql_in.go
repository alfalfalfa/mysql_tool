package models

import (
	"strings"

	"github.com/alfalfalfa/mysql_tool/util"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
)

func NewModelFromMysql(ignoreTables []string, fqdn string) *Models {
	res := &Models{}
	res.Tables = make([]*Table, 0)
	db, err := gorm.Open("mysql", fqdn)
	checkError(err)

	for _, tableInfo := range LoadMysqlTables(db) {
		// 無視するテーブル名を確認
		if contains(ignoreTables, tableInfo.Name) {
			continue
		}
		t := NewTableFromMysql(db, tableInfo)
		res.Tables = append(res.Tables, t)
	}

	//外部キー取得
	refs := LoadMysqlFK(db, getDBName(fqdn))
	for _, ref := range refs {
		table := res.GetTable(strings.ToLower(ref.TableName))
		column := table.GetColumn(strings.ToLower(ref.ColumnName))
		column.Reference = strings.ToLower(ref.ReferencedTableName) + "." + strings.ToLower(ref.ReferencedColumnName)

		//FKインデックス除外
		deleteIndexes := make([]*Index, 0)
		for _, in := range table.Indexes {
			if in.Name == ref.ConstraintName && in.IsContainColumnName(ref.ColumnName) {
				deleteIndexes = append(deleteIndexes, in)
			}
		}
		table.RemoveIndex(deleteIndexes...)
	}

	res.resolveReferences()
	return res
}

func getDBName(fqdn string) string {
	tmp := strings.Split(fqdn, "/")
	return tmp[len(tmp)-1]
}

func NewTableFromMysql(db *gorm.DB, tableInfo MysqlTable) *Table {
	t := &Table{}

	t.Name = util.NewCaseString(tableInfo.GetName())
	//t.LogicalName = tableInfo.Comment
	t.Engine = tableInfo.Engine
	t.DefaultCharset = tableInfo.GetCharset()
	t.DefaultCollation = tableInfo.Collation

	t.Comment = tableInfo.Comment
	//t.MetaDataJson = strings.TrimSpace(row.Cells[8].Value)

	t.Columns = make([]*Column, 0)
	var pkIndex int
	for _, columnInfo := range LoadMysqlColumns(db, tableInfo.GetName()) {
		c := NewColumnFromMysql(db, t, columnInfo)

		if columnInfo.Key == "PRI" {
			pkIndex++
			c.PrimaryKey = pkIndex
			c.Extra = columnInfo.Extra
		}

		t.Columns = append(t.Columns, c)
	}

	t.Indexes = make([]*Index, 0)

	indexGroup := make(map[string][]MysqlIndex)
	indexNames := make([]string, 0)
	for _, indexInfo := range LoadMysqlIndexes(db, tableInfo.GetName()) {
		//PRIMARY KEYインデックスは除外
		if indexInfo.KeyName == "PRIMARY" {
			continue
		}

		list, ok := indexGroup[indexInfo.GetName()]
		if !ok {
			list = make([]MysqlIndex, 0)
			indexNames = append(indexNames, indexInfo.GetName())
		}
		list = append(list, indexInfo)
		indexGroup[indexInfo.GetName()] = list
	}

	for _, indexName := range indexNames {
		in := NewIndexFromMysql(db, indexGroup[indexName])
		t.Indexes = append(t.Indexes, in)
	}

	return t
}

func NewColumnFromMysql(db *gorm.DB, t *Table, columnInfo MysqlColumn) *Column {
	c := &Column{}

	c.Name = util.NewCaseString(columnInfo.GetName())
	//c.LogicalName = strings.TrimSpace(row.Cells[2].Value)
	c.Type = columnInfo.Type

	// merge collation to type
	if columnInfo.Collation.Valid {
		if columnInfo.GetCharset() != t.DefaultCharset {
			c.Type += " CHARACTER SET " + columnInfo.GetCharset() + " COLLATE " + columnInfo.Collation.ValueOrZero()
		} else if columnInfo.Collation.ValueOrZero() != t.DefaultCollation {
			if columnInfo.IsBinaryCollation() && !t.IsBinaryCollation() {
				c.Type += " binary"
			} else {
				c.Type += " COLLATE " + columnInfo.Collation.ValueOrZero()
			}
		}
	}

	c.NotNull = !columnInfo.isNull()

	c.Comment = columnInfo.Comment
	//c.MetaDataJson = strings.TrimSpace(row.Cells[8].Value)

	c.Default = columnInfo.Default

	return c
}

func NewIndexFromMysql(db *gorm.DB, indexInfos []MysqlIndex) *Index {
	in := &Index{}

	in.ColumnNames = make([]string, len(indexInfos))
	for _, indexInfo := range indexInfos {
		in.Name = indexInfo.GetName()
		in.Unique = !indexInfo.NonUnique
		if indexInfo.Type != "BTREE" {
			in.Type = indexInfo.Type
		}
		in.Comment = indexInfo.Comment
		in.ColumnNames[indexInfo.SeqInIndex-1] = strings.ToLower(indexInfo.ColumnName)
	}

	return in
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
