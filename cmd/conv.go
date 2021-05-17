package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"path/filepath"

	"github.com/app-studio/mysql_tool/models"
	"github.com/app-studio/mysql_tool/util/copy"
	"github.com/docopt/docopt-go"
)

const usageConv = `mysql_tool conv
    テーブル定義ドキュメント or データベース間の変換

Usage:
    mysql_tool conv -h | --help
    mysql_tool conv [-f FORMAT] [-o OUTPUT] [--foreign-key] [--ignore-tables IGNORE_TABLES...] [--json-comment] INPUTS...

Arg:
    入力ファイルパス（json, yaml, xlsx, dir） | mysql dsn(https://github.com/go-sql-driver/mysql#dsn-data-source-name)

Options:
    -h --help                     Show this screen.
    -f FORMAT, --format=FORMAT    出力フォーマット
        "sql"
            create table文を出力
        "json"
            Json出力
        "yaml"
            yaml出力
        "xlsx"
            Excel出力
        none
            OUTPUTの拡張子から自動判別 （default: sql）
    -o OUTPUT, --output=OUTPUT    出力先
        ディレクトリパス
            1テーブル1ファイルで出力
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

	m := models.LoadModel(arg.IgnoreTables, arg.Inputs...)

	format := detectOutputFormat(arg.Format, arg.Output)

	if isDirOutput(arg.Output) {
		// output per-table files
		os.MkdirAll(arg.Output, os.ModePerm)
		for _, table := range m.Tables {
			b := table.MarshalTable(format, arg.ForeignKey, arg.JsonComment)
			checkError(ioutil.WriteFile(filepath.Join(arg.Output, table.Name.LowerSnake()+"."+format), b, os.ModePerm))
		}
	} else {
		// output single file
		b := m.MarshalModel(format, arg.ForeignKey, arg.JsonComment)
		if arg.Output == "" {
			if format == "xlsx" {
				panic("xlsx stdout")
			}
			fmt.Println(string(b))
		} else {
			checkError(ioutil.WriteFile(arg.Output, b, os.ModePerm))
		}
	}
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
	if filepath.Ext(output) == ".yaml" {
		return "yaml"
	}
	if filepath.Ext(output) == ".yml" {
		return "yml"
	}
	return "sql"
}

func isDirOutput(output string) bool {
	if output == "" {
		return false
	}
	if filepath.Ext(output) == ".sql" {
		return false
	}
	if filepath.Ext(output) == ".xlsx" {
		return false
	}
	if filepath.Ext(output) == ".json" {
		return false
	}
	if filepath.Ext(output) == ".yaml" {
		return false
	}
	if filepath.Ext(output) == ".yml" {
		return false
	}
	return true
}
