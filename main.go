package main

import (
	"fmt"

	"os"

	"github.com/app-studio/mysql_tool/cmd"
	"github.com/docopt/docopt-go"
)

const usageRoot = `mysql_tool
    Mysqlツール群

Usage:
    mysql_tool -h | --help
    mysql_tool COMMAND

Arg:
    "conv"           テーブル定義ドキュメント or データベース間の変換
    "diff"           テーブル定義の差分出力(マイグレーション用)
    "data"           データ定義の変換、mysql入出力
    "gen-single"     テーブル定義から1テキスト生成
    "gen-multiple"   テーブル定義から各テーブル毎にテキスト生成
    "exec"           sql実行(接続成功までリトライ)

Options:
    -h --help    Show this screen.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usageRoot)
		return
	}
	arguments, err := docopt.Parse(usageRoot, os.Args[1:2], true, "", false)
	//fmt.Println(err)
	//fmt.Println(json.ToJson(arguments))

	if err != nil {
		panic(err)
	}

	switch arguments["COMMAND"] {
	default:
		fmt.Println(usageRoot)
	case "conv":
		cmd.RunConv()
	case "diff":
		cmd.RunDiff()
	case "data":
		cmd.RunData()
	case "gen-single":
		cmd.RunGenSingle()
	case "gen-multiple":
		cmd.RunGenMultiple()
	case "exec":
		cmd.RunExec()
	}
}
