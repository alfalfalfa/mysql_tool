package models

import (
	"encoding/json"
	"io/ioutil"
)

func NewModelFromJson(ignoreTables[]string, pathes ...string) *Models {
	tables := make([]*Table, 0)
	for _, path := range pathes {
		m := &Models{}
		b, err := ioutil.ReadFile(path)
		checkError(err)
		err = json.Unmarshal(b, m)
		checkError(err)

		for _, t := range m.Tables {
			// 無視するテーブル名を判定
			if contains(ignoreTables, t.Name.LowerSnake()) {
				continue
			}
			tables = append(tables, t)
		}
	}

	res := &Models{}
	res.Tables = tables
	res.resolveReferences()
	return res
}