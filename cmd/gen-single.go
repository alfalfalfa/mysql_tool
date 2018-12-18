package cmd

import (
	"fmt"
	"github.com/flosch/pongo2"
	"os"
	"path"
	"regexp"

	"bytes"
	"text/template"

	"path/filepath"

	"go/format"

	"io/ioutil"

	"github.com/app-studio/mysql_tool/models"
	"github.com/app-studio/mysql_tool/util/copy"
	"github.com/docopt/docopt-go"
	"golang.org/x/tools/imports"
)

const usageGen = `mysql_tool gen-single
    テーブル定義から1テキスト生成

Usage:
    mysql_tool gen-single -h | --help
    mysql_tool gen-single [--template-type=<TEMPLATE_TYPE>] [--tables TABLES...] [--ignore-tables IGNORE_TABLES...] <TEMPLATE_PATH> <OUTPUT_PATH> INPUTS...

Arg:
	<TEMPLATE_PATH>        (必須)テンプレートファイルパス
	<OUTPUT_PATH>  (必須)出力ファイルパス
    INPUTS...				入力ファイルパス（json, yaml, xlsx, dir） | mysql fqdn

Options:
    -h --help                             Show this screen.
    --template-type=<TEMPLATE_TYPE>     テンプレート種別 [default:go]
										    go      text/template
										    pongo2  pongo2
    -t TEMPLATE, --template=TEMPLATE      テンプレートファイルパス
    -o OUTPUT, --output=OUTPUT            出力先
        ファイルパス
            上書き
        none
            標準出力
    --tables=TABLES...                    対象テーブル
    --ignore-tables=IGNORE_TABLES...      無視テーブル
`

type GenSingleArg struct {
	TemplatePath string   `arg:"<TEMPLATE_PATH>"`
	OutputPath   string   `arg:"<OUTPUT_PATH>"`
	TemplateType string   `arg:"--template-type"`
	Inputs       []string `arg:"INPUTS"`
	Tables       []string `arg:"--tables"`
	IgnoreTables []string `arg:"--ignore-tables"`
}

type TemplateData struct {
	Tables []*models.Table
}

func RunGenSingle() {
	arguments, err := docopt.Parse(usageGen, os.Args[1:], true, "", false)
	if err != nil {
		panic(err)
	}
	//fmt.Println(json.ToJson(arguments))
	arg := &GenSingleArg{}
	copy.MapToStructWithTag(arguments, arg, "arg")

	m := models.LoadModel(arg.IgnoreTables, arg.Inputs...)

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

	//fmt.Println(json.ToJson(args))
	if arg.TemplateType == "go" {
		funcMap := template.FuncMap{
			// Math functions
			"add":      add,
			"subtract": subtract,
			"multiply": multiply,
			"divide":   divide,
		}
		tmpl := template.Must(template.New(filepath.Base(arg.TemplatePath)).Funcs(funcMap).ParseFiles(arg.TemplatePath))
		data := TemplateData{
			Tables: tables,
		}
		buf := bytes.NewBuffer(nil)
		err = tmpl.Execute(buf, data)
		e(err)

		b := buf.Bytes()
		if filepath.Ext(arg.OutputPath) == ".go" {
			writeGoSource(arg.OutputPath, buf.Bytes())
		} else {
			e(ioutil.WriteFile(arg.OutputPath, b, os.ModePerm))
		}
	}

	if arg.TemplateType == "pongo2" {
		tpl, err := pongo2.DefaultSet.FromFile(arg.TemplatePath)
		e(err)

		context := pongo2.Context{
			"tables": tables,
		}
		res, err := tpl.Execute(context)
		e(err)

		// 連続した改行を詰める
		re := regexp.MustCompile("\n+")
		res = re.ReplaceAllString(res, "\n")

		os.MkdirAll(path.Dir(arg.OutputPath), os.ModePerm)
		ioutil.WriteFile(arg.OutputPath, []byte(res), os.ModePerm)
		fmt.Println("write:", arg.OutputPath)
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
