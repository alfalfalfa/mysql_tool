package models

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func LoadModel(ignoreTables []string, inputs ...string) *Models {
	if isDsnString(inputs[0]) {
		if 1 < len(inputs) {
			panic("multiple dsn")
		}
		return NewModelFromMysql(ignoreTables, inputs[0])
	}

	tables := make([]*Table, 0)
	for _, path := range resolvFilePathes(inputs...) {
		switch DetectInputFormat(path) {
		case "xlsx":
			tables = append(tables, loadTablesFromExcel(ignoreTables, path)...)
		case "json":
			tables = append(tables, loadTablesFromJson(ignoreTables, path)...)
		case "yaml":
			tables = append(tables, loadTablesFromYaml(ignoreTables, path)...)
		default:
			panic(fmt.Sprint("input path must be [.json, .yaml, .yml, .xlsx] inputs:", inputs))
		}
	}

	res := &Models{}
	res.Tables = tables
	res.resolveReferences()
	return res
}

func DetectInputFormat(input string) string {
	if filepath.Ext(input) == ".xlsx" {
		return "xlsx"
	}
	if filepath.Ext(input) == ".json" {
		return "json"
	}
	if filepath.Ext(input) == ".yaml" {
		return "yaml"
	}
	return "mysql"
}

func isDsnString(input string) bool {
	// コロンが含まれていて存在しないパスであればmysql接続文字列とする
	if !strings.Contains(input, ":") {
		return false
	}
	_, err := os.Stat(input)
	return err != nil
}

func resolvFilePathes(pathes ...string) []string {
	res := make([]string, 0)
	for _, path := range pathes {
		fileInfo, err := os.Stat(path)
		checkError(err)
		if fileInfo.IsDir() {
			res = append(res, readDir(path)...)
		} else {
			res = append(res, path)
		}
	}
	return res
}

func readDir(dirPath string) []string {
	res := make([]string, 0)
	fileInfos, err := ioutil.ReadDir(dirPath)
	checkError(err)
	for _, info := range fileInfos {
		path := filepath.Join(dirPath, info.Name())
		if info.IsDir() {
			res = append(res, readDir(path)...)
		} else {
			res = append(res, path)
		}
	}
	return res
}
