package models

import (
	"encoding/json"
	"io/ioutil"
)

func loadTablesFromJson(ignoreTables[]string, path string) []*Table {
	tables := make([]*Table, 0)
	m := &Models{}
	b, err := ioutil.ReadFile(path)
	checkError(err)
	err = json.Unmarshal(b, &m.Tables)
	checkError(err)

	for _, t := range m.Tables {
		// 無視するテーブル名を判定
		if contains(ignoreTables, t.Name.LowerSnake()) {
			continue
		}
		tables = append(tables, t)
	}
	return tables
}