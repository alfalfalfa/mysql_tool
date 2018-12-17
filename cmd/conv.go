package cmd

import (
	"fmt"
	"os"

	"path/filepath"

	"io/ioutil"

	"github.com/docopt/docopt-go"
	"github.com/app-studio/mysql_tool/util/copy"
	"github.com/app-studio/mysql_tool/util/json"
	"github.com/app-studio/mysql_tool/models"
)

const usageConv = `mysql_tool conv
    テーブル定義ドキュメント or データベース間の変換

Usage:
    mysql_tool conv -h | --help
    mysql_tool conv [-f FORMAT] [-o OUTPUT] [--foreign-key] [--ignore-tables IGNORE_TABLES...] [--json-comment] INPUTS...

Arg:
    入力ファイルパス（json,xlsx） | mysql fqdn

Options:
    -h --help                     Show this screen.
    -f FORMAT, --format=FORMAT    出力フォーマット
        "sql"
            create table文を出力
        "json"
            Json出力
        "xlsx"
            Excel出力
        none
            OUTPUTの拡張子から自動判別 （default: sql）
    -o OUTPUT, --output=OUTPUT    出力先
        ファイルパス
            上書き
        none
            標準出力（Excel出力では無効）
    --foreign-key                 外部キーの出力（sql出力のみ）
    --ignore-tables=IGNORE_TABLES...      無視テーブル
    --json-comment                メタデータjsonのコメント埋め込み（sql出力のみ）
`

type ConvArg struct {
	Format       string   `arg:"--format"`
	Output       string   `arg:"--output"`
	ForeignKey   bool     `arg:"--foreign-key"`
	JsonComment  bool     `arg:"--json-comment"`
	IgnoreTables []string `arg:"--ignore-tables"`
	Inputs       []string `arg:"INPUTS"`
}

func RunConv() {
	arguments, err := docopt.Parse(usageConv, os.Args[1:], true, "", false)
	checkError(err)
	//fmt.Println(json.ToJson(arguments))
	arg := &ConvArg{}
	copy.MapToStructWithTag(arguments, arg, "arg")
	//fmt.Println(json.ToJson(arg))

	m := loadModel(arg.IgnoreTables, arg.Inputs...)

	format := detectOutputFormat(arg.Format, arg.Output)
	b := marshalModel(m, format, arg.ForeignKey, arg.JsonComment)

	if arg.Output == "" {
		if format == "xlsx" {
			panic("xlsx stdout")
		}
		fmt.Println(string(b))
	} else {
		checkError(ioutil.WriteFile(arg.Output, b, os.ModePerm))
	}
}

func loadModel(ignoreTables[]string, inputs ...string) *models.Models {
	switch detectInputFormat(inputs[0]) {
	case "xlsx":
		return models.NewModelFromExcel(ignoreTables, inputs...)
	case "json":
		return models.NewModelFromJson(ignoreTables, inputs...)
	case "mysql":
		return models.NewModelFromMysql(ignoreTables, inputs[0])
	}
	panic(fmt.Sprint("input format invalid:", inputs))
}

func marshalModel(m *models.Models, format string, fk bool, jsonComment bool) []byte {
	switch format {
	case "xlsx":
		return m.ToExcelFile()
	case "json":
		return []byte(json.ToJson(m.Tables))
	case "sql":
		return []byte(m.ToCreateSQL(fk, jsonComment))
	}
	panic(fmt.Sprint("output format invalid:", format))
}

func detectInputFormat(input string) string {
	if filepath.Ext(input) == ".xlsx" {
		return "xlsx"
	}
	if filepath.Ext(input) == ".json" {
		return "json"
	}
	return "mysql"
}

func detectOutputFormat(format string, output string) string {
	if format != "" {
		return format
	}
	if filepath.Ext(output) == ".sql" {
		return "sql"
	}
	if filepath.Ext(output) == ".xlsx" {
		return "xlsx"
	}
	if filepath.Ext(output) == ".json" {
		return "json"
	}
	return "sql"
}
