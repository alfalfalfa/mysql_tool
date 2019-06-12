package models

import (
	"github.com/app-studio/mysql_tool/util/null"
	"strings"

	"database/sql"

	"github.com/jinzhu/gorm"
)

type MysqlTable struct {
	Name      string `gorm:"column:Name"`
	Engine    string `gorm:"column:Engine"`
	Collation string `gorm:"column:Collation"`
	Comment   string `gorm:"column:Comment"`
}

func (this MysqlTable) GetName() string {
	return strings.ToLower(this.Name)
}

func (this MysqlTable) GetCharset() string {
	tmp := strings.Split(this.Collation, "_")
	return tmp[0]
}

func LoadMysqlTables(db *gorm.DB) []MysqlTable {
	var fields []MysqlTable
	db.Raw("SHOW TABLE STATUS").Find(&fields)

	return fields
}

type MysqlColumn struct {
	Field      string      `gorm:"column:Field"`
	Type       string      `gorm:"column:Type"`
	Null       string      `gorm:"column:Null"`
	Key        string      `gorm:"column:Key"`
	Comment    string      `gorm:"column:Comment"`
	Default    null.String `gorm:"column:Default"`
	Collation  string      `gorm:"column:Collation"`
	Extra      string      `gorm:"column:Extra"`
	Privileges string      `gorm:"column:Privileges"`
}

func (this MysqlColumn) isNull() bool {
	return this.Null == "YES"
}

func (this MysqlColumn) GetName() string {
	return strings.ToLower(this.Field)
}

func LoadMysqlColumns(db *gorm.DB, table string) []MysqlColumn {
	var fields []MysqlColumn
	db.Raw("SHOW FULL COLUMNS FROM `" + table + "`").Find(&fields)
	return fields
}

type MysqlIndex struct {
	Table       string        `gorm:"column:Table"`
	NonUnique   bool          `gorm:"column:Non_unique"`
	KeyName     string        `gorm:"column:Key_name"`
	SeqInIndex  int           `gorm:"column:Seq_in_index"`
	ColumnName  string        `gorm:"column:Column_name"`
	Collation   string        `gorm:"column:Collation"`
	Cardinality sql.NullInt64 `gorm:"column:Cardinality"`
	Type        string        `gorm:"column:Index_type"`
	Comment     string        `gorm:"column:Index_comment"`
}

func (this MysqlIndex) GetName() string {
	return strings.ToLower(this.KeyName)
}

func LoadMysqlIndexes(db *gorm.DB, table string) []MysqlIndex {
	var fields []MysqlIndex
	db.Raw("SHOW INDEX FROM `" + table + "`").Find(&fields)
	return fields
}

type MysqlFK struct {
	TableSchema          string `gorm:"column:TABLE_SCHEMA"`
	TableName            string `gorm:"column:TABLE_NAME"`
	ColumnName           string `gorm:"column:COLUMN_NAME"`
	ConstraintType       string `gorm:"column:CONSTRAINT_TYPE"`
	ConstraintName       string `gorm:"column:CONSTRAINT_NAME"`
	ReferencedTableName  string `gorm:"column:REFERENCED_TABLE_NAME"`
	ReferencedColumnName string `gorm:"column:REFERENCED_COLUMN_NAME"`
	UpdateRule           string `gorm:"column:UPDATE_RULE"`
	DeleteRule           string `gorm:"column:DELETE_RULE"`
}

func LoadMysqlFK(db *gorm.DB, dbName string) []MysqlFK {
	var fields []MysqlFK
	db.Raw(`
SELECT
F1.TABLE_SCHEMA AS TABLE_SCHEMA
,F1.TABLE_NAME AS TABLE_NAME
,F1.COLUMN_NAME AS COLUMN_NAME
,F2.CONSTRAINT_TYPE AS CONSTRAINT_TYPE
,F2.CONSTRAINT_NAME AS CONSTRAINT_NAME
,F1.REFERENCED_TABLE_NAME AS REFERENCED_TABLE_NAME
,F1.REFERENCED_COLUMN_NAME AS REFERENCED_COLUMN_NAME
,F3.UPDATE_RULE
,F3.DELETE_RULE
FROM
information_schema.KEY_COLUMN_USAGE F1
LEFT JOIN information_schema.TABLE_CONSTRAINTS F2 ON F1.TABLE_SCHEMA = F2.TABLE_SCHEMA AND F1.CONSTRAINT_NAME = F2.CONSTRAINT_NAME
LEFT JOIN information_schema.REFERENTIAL_CONSTRAINTS F3 ON F2.CONSTRAINT_SCHEMA = F3.CONSTRAINT_SCHEMA AND F2.CONSTRAINT_NAME = F3.CONSTRAINT_NAME
WHERE F2.CONSTRAINT_TYPE = 'FOREIGN KEY'
AND F1.TABLE_SCHEMA = '` + dbName + `'
;
	`).Find(&fields)
	return fields
}
