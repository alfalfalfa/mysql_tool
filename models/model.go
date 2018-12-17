package models

import (
	"strings"

	"regexp"

	"sort"

	"github.com/app-studio/mysql_tool/util"
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

func (this *Models) resolveReferences() {
	//a-z table order
	sort.Sort(this)

	//column order
	for _, t := range this.Tables {
		var pre *Column
		for _, c := range t.Columns {
			c.PreColumn = pre
			pre = c
		}
	}

	//TODO diff,codegen用Ref解決, Table並び替え、IndexのColumns解決
	for _, t := range this.Tables {
		t.PrimaryKeys = t.getPrimaryKeys()
	}

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
	Name util.CaseString
	//LogicalName    string
	Engine          string
	DefaultCharset  string
	DbIndex         int
	ConnectionIndex int
	Comment         string
	MetaDataJson    string
	Descriptions    []string

	Columns []*Column
	Indexes []*Index

	PrimaryKeys []*Column `json:"-"`
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

type Column struct {
	Name util.CaseString
	//LogicalName  string
	Type       string
	NotNull    bool
	PrimaryKey int

	Default      string
	Extra        string
	Reference    string
	Comment      string
	MetaDataJson string
	Descriptions []string

	PreColumn *Column `json:"-"`

	//ReferenceTable  *Table  `json:"-"`
	//ReferenceColumn *Column `json:"-"`
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
		return " AFTER " + c.PreColumn.Name.LowerSnake()
	} else {
		return " FIRST"
	}
}

func normalizeDefault(c *Column) string {
	if c == nil {
		return ""
	}
	d := strings.TrimSpace(c.Default)

	//文字列の末端quote処理
	if strings.HasSuffix(d, "'") || strings.HasPrefix(d, "'") {
		d = strings.Trim(d, "'")
		//TODO escape quote
		d = "'" + d + "'"
	}

	return d
}

type Index struct {
	Name         string
	ColumnNames  []string
	Unique       bool
	Comment      string
	Descriptions []string

	//Columns []*Column `json:"-"`
}

func (this Index) IsContainColumnName(name string) bool {
	for _, n := range this.ColumnNames {
		if n == name {
			return true
		}
	}
	return false
}
