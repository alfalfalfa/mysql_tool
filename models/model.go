package models

import (
	"github.com/alfalfalfa/mysql_tool/util/null"
	"strings"

	"regexp"

	"github.com/alfalfalfa/mysql_tool/util"
)

func checkError(err error) {
	if err != nil {
		panic(err)
	}
}

type Models struct {
	Tables []*Table
}

func (a *Models) Len() int      { return len(a.Tables) }
func (a *Models) Swap(i, j int) { a.Tables[i], a.Tables[j] = a.Tables[j], a.Tables[i] }
func (a *Models) Less(i, j int) bool {
	return a.Tables[i].Name.LowerSnake() < a.Tables[j].Name.LowerSnake()
}

func (this Models) GetTable(name string) *Table {
	for _, t := range this.Tables {
		if t.Name.Lower() == strings.ToLower(name) {
			return t
		}
	}
	return nil
}

type Table struct {
	//LogicalName    string
	Name             util.CaseString
	Engine           string
	DefaultCharset   string
	DefaultCollation string   `json:",omitempty" yaml:",omitempty"`
	DbIndex          int      `json:",omitempty" yaml:",omitempty"`
	ConnectionIndex  int      `json:",omitempty" yaml:",omitempty"`
	Comment          string   `json:",omitempty" yaml:",omitempty"`
	MetaDataJson     string   `json:",omitempty" yaml:",omitempty"`
	Descriptions     []string `json:",omitempty" yaml:",omitempty"`
	Columns          []*Column
	Indexes          []*Index

	PrimaryKeys       []*Column    `json:"-" yaml:"-"`
	References        []*Reference `json:"-" yaml:"-"`
	InverseReferences []*Reference `json:"-" yaml:"-"`
}

func (this Table) GetPrimaryKeyNum() int {
	return len(this.PrimaryKeys)
}

func (this Table) GetPrimaryKeyNames() []string {
	res := make([]string, 0)
	for _, c := range this.PrimaryKeys {
		res = append(res, "`"+c.Name.LowerSnake()+"`")
	}

	return res
}
func (this Table) GetFirstColumn() *Column {
	return this.Columns[0]
}
func (this Table) GetPK() *Column {
	return this.PrimaryKeys[0]
}

func (this Table) getPrimaryKeys() []*Column {
	res := make([]*Column, 0)
	pkOrder := 1
	for {
		pkFound := false
		for _, c := range this.Columns {
			if pkOrder == c.PrimaryKey {
				res = append(res, c)
				pkOrder++
				pkFound = true
				break
			}
		}
		if !pkFound {
			break
		}
	}

	return res
}
func (this Table) GetColumn(name string) *Column {
	for _, c := range this.Columns {
		if c.Name.LowerSnake() == strings.ToLower(name) {
			return c
		}
	}
	return nil
}
func (this Table) GetColumnNames() []string {
	res := make([]string, 0)
	for _, c := range this.Columns {
		res = append(res, c.Name.LowerSnake())
	}
	return res
}

func (this Table) GetIndex(name string) *Index {
	for _, c := range this.Indexes {
		if c.Name == name {
			return c
		}
	}
	return nil
}
func (this *Table) RemoveIndex(removeIndexes ...*Index) {
	newList := make([]*Index, 0)
	for _, in1 := range this.Indexes {
		contain := false
		for _, in2 := range removeIndexes {
			if in1 == in2 {
				contain = true
				break
			}
		}
		if !contain {
			newList = append(newList, in1)
		}
	}
	this.Indexes = newList
}

func (this Table) IsChange(other *Table) bool {
	return this.Engine != other.Engine || this.Comment != other.Comment || this.DefaultCharset != other.DefaultCharset
}

func (this Table) IsBinaryCollation() bool {
	tmp := strings.Split(this.DefaultCollation, "_")
	return tmp[len(tmp)-1] == "bin"
}

type Column struct {
	//LogicalName  string
	Name         util.CaseString
	Type         string
	NotNull      bool        `json:",omitempty" yaml:",omitempty"`
	PrimaryKey   int         `json:",omitempty" yaml:",omitempty"`
	Default      null.String `json:",omitempty" yaml:",omitempty"`
	Extra        string      `json:",omitempty" yaml:",omitempty"`
	Reference    string      `json:",omitempty" yaml:",omitempty"`
	Comment      string      `json:",omitempty" yaml:",omitempty"`
	MetaDataJson string      `json:",omitempty" yaml:",omitempty"`
	Descriptions []string    `json:",omitempty" yaml:",omitempty"`

	Table             *Table       `json:"-" yaml:"-"`
	PreColumn         *Column      `json:"-" yaml:"-"`
	Indexes           []*Index     `json:"-" yaml:"-"`
	References        []*Reference `json:"-" yaml:"-"`
	InverseReferences []*Reference `json:"-" yaml:"-"`
}

func (this Column) isNotNull(notNull string, null string) string {
	if this.NotNull {
		return notNull
	} else {
		return null
	}
}

func (this Column) isNotUnsign(notUnsign string, unsign string) string {
	switch {
	case regexp.MustCompile("unsigned").MatchString(this.Type):
		return unsign
	default:
		return notUnsign
	}
}

func (this Column) GetGoType() string {
	switch {

	case regexp.MustCompile("^(bool|boolean|tinyint\\(1\\))").MatchString(this.Type):
		return this.isNotNull("bool", "sql.NullBool")

	case regexp.MustCompile("^tinyint").MatchString(this.Type):
		return this.isNotNull(this.isNotUnsign("int8", "uint8"), "sql.NullInt64")

	case regexp.MustCompile("^smallint").MatchString(this.Type):
		return this.isNotNull(this.isNotUnsign("int16", "uint16"), "sql.NullInt64")

	case regexp.MustCompile("^mediumint").MatchString(this.Type):
		return this.isNotNull(this.isNotUnsign("int", "uint"), "sql.NullInt64")

	case regexp.MustCompile("^int").MatchString(this.Type):
		return this.isNotNull(this.isNotUnsign("int", "uint"), "sql.NullInt64")

	case regexp.MustCompile("^bigint").MatchString(this.Type):
		return this.isNotNull(this.isNotUnsign("int64", "uint64"), "sql.NullInt64")

	case regexp.MustCompile("^float").MatchString(this.Type):
		return this.isNotNull("float32", "sql.NullFloat64")

	case regexp.MustCompile("^double").MatchString(this.Type):
		return this.isNotNull("float64", "sql.NullFloat64")

	case regexp.MustCompile("^(tinytext|text|mediumtext|longtext|varchar|char)").MatchString(this.Type):
		return this.isNotNull("string", "sql.NullString")

	case regexp.MustCompile("^(date|datetime|timestamp|time|year)").MatchString(this.Type):
		return this.isNotNull("time.Time", "*time.Time")

	case regexp.MustCompile("^(tinyblob|blob|mediumblob|longblob)").MatchString(this.Type):
		return "[]byte"

	default:
		println(this.Type)
		panic("not found type")

	}
}

func (this Column) GetCSType() string {
	switch {

	case regexp.MustCompile("^(bool|boolean|tinyint\\(1\\))").MatchString(this.Type):
		return "bool"

	case regexp.MustCompile("^tinyint").MatchString(this.Type):
		return "int"

	case regexp.MustCompile("^smallint").MatchString(this.Type):
		return "int"

	case regexp.MustCompile("^mediumint").MatchString(this.Type):
		return "int"

	case regexp.MustCompile("^int").MatchString(this.Type):
		return "int"

	case regexp.MustCompile("^bigint").MatchString(this.Type):
		return "long"

	case regexp.MustCompile("^float").MatchString(this.Type):
		return "float"

	case regexp.MustCompile("^double").MatchString(this.Type):
		return "double"

	case regexp.MustCompile("^(tinytext|text|mediumtext|longtext|varchar|char)").MatchString(this.Type):
		return "string"

	case regexp.MustCompile("^(date|datetime|timestamp|time|year)").MatchString(this.Type):
		return "DateTime"

	case regexp.MustCompile("^(tinyblob|blob|mediumblob|longblob)").MatchString(this.Type):
		return "byte[]"

	default:
		println(this.Type)
		panic("not found type")

	}
}

/*
unused type(ruby)
	primary_key
	decimal
*/
func (this Column) GetActiveRecordType() string {
	switch {

	case regexp.MustCompile("^(bool|boolean|tinyint\\(1\\))").MatchString(this.Type):
		return "boolean"

	case regexp.MustCompile("^tinyint").MatchString(this.Type):
		return "integer"

	case regexp.MustCompile("^smallint").MatchString(this.Type):
		return "integer"

	case regexp.MustCompile("^mediumint").MatchString(this.Type):
		return "integer"

	case regexp.MustCompile("^int").MatchString(this.Type):
		return "integer"

	case regexp.MustCompile("^bigint").MatchString(this.Type):
		return "integer"

	case regexp.MustCompile("^year").MatchString(this.Type):
		return "integer"

	case regexp.MustCompile("^float").MatchString(this.Type):
		return "float"

	case regexp.MustCompile("^double").MatchString(this.Type):
		return "float"

	case regexp.MustCompile("^(varchar|char)").MatchString(this.Type):
		return "string"

	case regexp.MustCompile("^(tinytext|text|mediumtext|longtext)").MatchString(this.Type):
		return "text"

	case regexp.MustCompile("^(date)").MatchString(this.Type):
		return "date"

	case regexp.MustCompile("^(datetime)").MatchString(this.Type):
		return "datetime"

	case regexp.MustCompile("^(time)").MatchString(this.Type):
		return "time"

	case regexp.MustCompile("^(timestamp)").MatchString(this.Type):
		return "timestamp"

	case regexp.MustCompile("^(tinyblob|blob|mediumblob|longblob)").MatchString(this.Type):
		return "binary"

	default:
		println(this.Type)
		panic("not found type")

	}
}

func (this Column) IsNumeric() bool {
	switch {

	case regexp.MustCompile("^(bool|boolean|tinyint\\(1\\))").MatchString(this.Type):
		return true

	case regexp.MustCompile("^tinyint").MatchString(this.Type):
		return true

	case regexp.MustCompile("^smallint").MatchString(this.Type):
		return true

	case regexp.MustCompile("^mediumint").MatchString(this.Type):
		return true

	case regexp.MustCompile("^int").MatchString(this.Type):
		return true

	case regexp.MustCompile("^bigint").MatchString(this.Type):
		return true

	case regexp.MustCompile("^float").MatchString(this.Type):
		return true

	case regexp.MustCompile("^double").MatchString(this.Type):
		return true

	case regexp.MustCompile("^(tinytext|text|mediumtext|longtext|varchar|char)").MatchString(this.Type):
		return false

	case regexp.MustCompile("^(date|datetime|timestamp|time|year)").MatchString(this.Type):
		return false

	case regexp.MustCompile("^(tinyblob|blob|mediumblob|longblob)").MatchString(this.Type):
		return false

	default:
		println(this.Type)
		panic("not found type")
	}
}
func (this Column) IsTime() bool {
	switch {
	case regexp.MustCompile("^(date|datetime|timestamp|time|year)").MatchString(this.Type):
		return true
	default:
		return false
	}

}

//go:generate stringer -type=ColumnChangeType
type ColumnChangeType int

const (
	ColumnChangeType_Same ColumnChangeType = iota
	ColumnChangeType_Type
	ColumnChangeType_Comment
	ColumnChangeType_NotNull
	ColumnChangeType_Default
	ColumnChangeType_Extra
)

/**
カラム定義の変更を検査する。
*/
func (this Column) IsChange(other *Column) ColumnChangeType {
	// 型の変更チェック
	if !isSameMysqlType(this.Type, other.Type) {
		return ColumnChangeType_Type
	}
	// コメントの変更チェック
	if this.Comment != other.Comment {
		return ColumnChangeType_Comment
	}
	// NotNull制約の変更チェック
	if this.NotNull != other.NotNull {
		return ColumnChangeType_NotNull
	}
	// Defaultの変更チェック
	if normalizeDefault(&this) != normalizeDefault(other) {
		//fmt.Println("default changed:", "'"+this.Default+"'", "'"+other.Default+"'")
		return ColumnChangeType_Default
	}
	// Extraの変更チェック
	if this.Extra != other.Extra {
		return ColumnChangeType_Extra
	}
	return ColumnChangeType_Same
}

/**
型の同一性を検査する
意味的に同一なものは同じ型とみなす
*/
func isSameMysqlType(type1, type2 string) bool {
	return normalizeMysqlType(type1) == normalizeMysqlType(type2)
}

/**
カラム型名を正規化する
*/
func normalizeMysqlType(t string) string {

	switch t {
	case "bigint":
		return "bigint(20)"
	case "int":
		return "int(11)"
	case "mediumint":
		return "mediumint(9)"
	case "smallint":
		return "smallint(6)"
	case "tinyint":
		return "tinyint(4)"
	case "boolean":
		return "tinyint(1)"
	case "bool":
		return "tinyint(1)"
	case "year":
		return "year(4)"
	}
	return t
}

func getColumnOrder(c *Column) string {
	if c.PreColumn != nil {
		return " AFTER `" + c.PreColumn.Name.LowerSnake() + "`"
	} else {
		return " FIRST"
	}
}

func normalizeDefault(c *Column) string {
	if c == nil {
		return ""
	}
	if !c.Default.Valid {
		return ""
	}
	d := strings.TrimSpace(c.Default.ValueOrZero())
	d = strings.Trim(d, "'")

	if !c.IsNumeric() &&
		!(c.IsTime() && (strings.ToUpper(d) == "CURRENT_TIMESTAMP" || strings.ToUpper(d) == "NOW()")) {
		//TODO escape inner quote
		d = "'" + d + "'"
		return d
	}
	return d
}

type Index struct {
	Name         string
	ColumnNames  []string
	Unique       bool     `json:",omitempty" yaml:",omitempty"`
	Type         string   `json:",omitempty" yaml:",omitempty"`
	Options      string   `json:",omitempty" yaml:",omitempty"`
	Comment      string   `json:",omitempty" yaml:",omitempty"`
	Descriptions []string `json:",omitempty" yaml:",omitempty"`

	Columns []*Column `json:"-" yaml:"-"`
}

func (this Index) IsContainColumnName(name string) bool {
	for _, n := range this.ColumnNames {
		if n == name {
			return true
		}
	}
	return false
}
