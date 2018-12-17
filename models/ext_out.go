package models

import (
	"fmt"
	"github.com/app-studio/mysql_tool/util/json"
)

func (m *Models) MarshalModel(format string, fk bool, jsonComment bool) []byte {
	switch format {
	case "xlsx":
		return m.ToExcelFile()
	case "json":
		return []byte(json.ToJson(m.Tables))
	case "yaml":
		return []byte(ToYaml(m.Tables))
	case "yml":
		return []byte(ToYaml(m.Tables))
	case "sql":
		return []byte(m.ToCreateSQL(fk, jsonComment))
	}
	panic(fmt.Sprint("output format invalid:", format))
}
func (t *Table) MarshalTable(format string, fk bool, jsonComment bool) []byte {
	m := &Models{}
	m.Tables = []*Table{t}
	return m.MarshalModel(format, fk, jsonComment)
}
