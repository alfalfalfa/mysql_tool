package cmd

import (
	"fmt"
	"os"

	"bytes"
	"text/template"

	"path/filepath"

	"go/format"

	"io/ioutil"

	"github.com/docopt/docopt-go"
	"github.com/app-studio/mysql_tool/util/copy"
	"github.com/app-studio/mysql_tool/models"
	"golang.org/x/tools/imports"
)

const usageGen = `mysql_tool gen
    テーブル定義からgolangの text/template でテキスト生成

Usage:
    mysql_tool gen -h | --help
    mysql_tool gen (-t TEMPLATE) (-o OUTPUT) [--tables TABLES...] [--ignore-tables IGNORE_TABLES...] INPUTS...

Arg:
    入力ファイルパス（json,xlsx） | mysql fqdn

Options:
    -h --help                             Show this screen.
    -t TEMPLATE, --template=TEMPLATE      テンプレートファイルパス
    -o OUTPUT, --output=OUTPUT            出力先
        ファイルパス
            上書き
        none
            標準出力
    --tables=TABLES...                    対象テーブル
    --ignore-tables=IGNORE_TABLES...      無視テーブル
`

type GenArg struct {
	Template     string   `arg:"--template"`
	Output       string   `arg:"--output"`
	Inputs       []string `arg:"INPUTS"`
	Tables       []string `arg:"--tables"`
	IgnoreTables []string `arg:"--ignore-tables"`
}

type TemplateData struct {
	PackageName string
	Tables      []*models.Table
}

func RunGen() {
	arguments, err := docopt.Parse(usageGen, os.Args[1:], true, "", false)
	if err != nil {
		panic(err)
	}
	//fmt.Println(json.ToJson(arguments))
	arg := &GenArg{}
	copy.MapToStructWithTag(arguments, arg, "arg")

	m := loadModel(arg.IgnoreTables, arg.Inputs...)

	tables := make([]*models.Table, 0)
	for _, t := range m.Tables {
		if !containsOrEmpty(arg.Tables, t.Name.LowerSnake()) {
			continue
		}
		if contains(arg.IgnoreTables, t.Name.LowerSnake()) {
			continue
		}
		tables = append(tables, t)
	}

	var data TemplateData = TemplateData{
		PackageName: filepath.Base(filepath.Dir(arg.Output)),
		Tables:      tables,
	}

	//fmt.Println(json.ToJson(args))

	funcMap := template.FuncMap{
		// Math functions
		"add":      add,
		"subtract": subtract,
		"multiply": multiply,
		"divide":   divide,
	}
	tmpl := template.Must(template.New(filepath.Base(arg.Template)).Funcs(funcMap).ParseFiles(arg.Template))
	buf := bytes.NewBuffer(nil)
	err = tmpl.Execute(buf, data)
	checkError(err)

	b := buf.Bytes()
	if filepath.Ext(arg.Output) == ".go" {
		writeGoSource(arg.Output, buf.Bytes())
	} else {
		checkError(ioutil.WriteFile(arg.Output, b, os.ModePerm))
	}
}

func writeGoSource(path string, buf []byte) error {
	err := os.MkdirAll(filepath.Dir(path), 0777)
	if err != nil {
		return err
	}
	ofile, err := os.Create(path)
	if err != nil {
		return err
	}
	defer ofile.Close()

	// 整形&importの依存解決 (INFO: optionが効いてない。)
	option := &imports.Options{
		Fragment:  false, // Accept fragment of a source file (no package statement)
		AllErrors: true,  // Report all errors (not just the first 10 on different lines)
		Comments:  true,  // Print comments (true if nil *Options provided)
		TabIndent: false, // Use tabs for indent (true if nil *Options provided)
		TabWidth:  4,     // Tab width (8 if nil *Options provided)
	}

	output, err := imports.Process("", buf, option)
	if err != nil {
		return err
	}

	var bo bytes.Buffer
	var flg bool = false
	for _, c := range output {
		if c == []byte("\n")[0] {
			bo.WriteByte(c)
			flg = true
		} else if flg && c == []byte("\t")[0] {
			bo.WriteString("    ")
		} else {
			flg = false
			bo.WriteByte(c)
		}
	}

	bts, err := format.Source(bo.Bytes())
	if err != nil {
		return err
	}

	_, err = ofile.Write(bts)
	if err != nil {
		return fmt.Errorf("write string err: %s", err)
	}

	return nil
}
