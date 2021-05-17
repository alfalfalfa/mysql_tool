package cmd

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"github.com/app-studio/mysql_tool/models"
	"github.com/app-studio/mysql_tool/util/copy"
	"github.com/docopt/docopt-go"
	"github.com/flosch/pongo2"
)

const usageGenMultiple = `mysql_tool gen-multiple
    テーブル定義からテーブル毎にテキスト生成

Usage:
    mysql_tool gen-multiple -h | --help
    mysql_tool gen-multiple [--overwrite=<OVERWRITE_MODE>] [--template-type=<TEMPLATE_TYPE>] [--tables TABLES...] [--ignore-tables IGNORE_TABLES...] <TEMPLATE_PATH> <OUTPUT_PATH_PETTERN> INPUTS...

Arg:
	<TEMPLATE_PATH>        (必須)テンプレートファイルパス
	<OUTPUT_PATH_PETTERN>  (必須)出力ファイルパスパターン
    INPUTS...				入力ファイルパス（json, yaml, xlsx, dir） | mysql dsn(https://github.com/go-sql-driver/mysql#dsn-data-source-name)

Options:
    -h --help                           Show this screen.
    --template-type=<TEMPLATE_TYPE>     テンプレート種別 [default:go]
										    go      text/template
										    pongo2  pongo2
    --tables=TABLES...                  対象テーブル
    --ignore-tables=IGNORE_TABLES...    無視テーブル
	--overwrite=<OVERWRITE_MODE>        Overwrite behavior [default:force]
							                force 上書き
							                skip  存在していたらスキップ
							                clear 出力ディレクトリを削除
`

type GenMultipleArg struct {
	TemplatePath      string   `arg:"<TEMPLATE_PATH>"`
	OutputPathPettern string   `arg:"<OUTPUT_PATH_PETTERN>"`
	TemplateType      string   `arg:"--template-type"`
	OverwriteMode     string   `arg:"--overwrite"`
	Inputs            []string `arg:"INPUTS"`
	Tables            []string `arg:"--tables"`
	IgnoreTables      []string `arg:"--ignore-tables"`
}

func (this GenMultipleArg) IsClear() bool {
	return this.OverwriteMode == "clear"
}
func (this GenMultipleArg) IsForce() bool {
	return this.OverwriteMode == "force"
}
func (this GenMultipleArg) IsSkip() bool {
	return this.OverwriteMode == "skip"
}

type MultipleTemplateData struct {
	Tables []*models.Table
	Table  *models.Table
}

func RunGenMultiple() {
	arguments, err := docopt.Parse(usageGenMultiple, os.Args[1:], true, "", false)
	if err != nil {
		panic(err)
	}
	//fmt.Println(json.ToJson(arguments))
	arg := &GenMultipleArg{}
	copy.MapToStructWithTag(arguments, arg, "arg")
	//dump(arg)

	m := models.LoadModel(arg.IgnoreTables, arg.Inputs...)

	// filter tables
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

	if arg.TemplateType == "go" {
		funcMap := template.FuncMap{
			// Math functions
			"add":      add,
			"subtract": subtract,
			"multiply": multiply,
			"divide":   divide,
		}
		tmpl := template.Must(template.New(filepath.Base(arg.TemplatePath)).Funcs(funcMap).ParseFiles(arg.TemplatePath))
		outputPathTmpl := template.Must(template.New("outputPath").Funcs(funcMap).Parse(arg.OutputPathPettern))
		for _, table := range tables {
			// create template data
			data := MultipleTemplateData{
				Tables: tables,
				Table:  table,
			}

			//fmt.Println(json.ToJson(args))

			buf := bytes.NewBuffer(nil)
			err = tmpl.Execute(buf, data)
			e(err)
			outputPathBuf := bytes.NewBuffer(nil)
			err = outputPathTmpl.Execute(outputPathBuf, data)
			e(err)
			outputPath := outputPathBuf.String()

			// TODO OverwriteMode
			b := buf.Bytes()
			if filepath.Ext(outputPath) == ".go" {
				writeGoSource(outputPath, buf.Bytes())
			} else {
				e(ioutil.WriteFile(outputPath, b, os.ModePerm))
			}
		}
	}

	if arg.TemplateType == "pongo2" {
		outputPathTpl, err := pongo2.DefaultSet.FromString(arg.OutputPathPettern)
		e(err)
		tpl, err := pongo2.DefaultSet.FromFile(arg.TemplatePath)
		e(err)

		outputs := make(map[string]string)
		for _, table := range tables {
			context := pongo2.Context{
				"tables": tables,
				"table":  table,
			}
			outputPath, res := renderPongo2Template(context, outputPathTpl, tpl)
			outputs[outputPath] = res
		}
		dirs := make(map[string]bool)
		for outputPath, _ := range outputs {
			dirs[path.Dir(outputPath)] = true
		}

		for outputDir, _ := range dirs {
			if arg.IsClear() {
				os.RemoveAll(outputDir)
			}
			os.MkdirAll(outputDir, os.ModePerm)
		}

		for outputPath, res := range outputs {
			if arg.IsSkip() {
				_, err := os.Stat(outputPath)
				if err != nil {
					ioutil.WriteFile(outputPath, []byte(res), os.ModePerm)
					fmt.Println("write:", outputPath)
				} else {
					fmt.Println("skip:", outputPath)
				}
			} else {
				ioutil.WriteFile(outputPath, []byte(res), os.ModePerm)
				fmt.Println("write:", outputPath)
			}
		}
	}
}

func renderPongo2Template(context pongo2.Context, outputPathTpl, tpl *pongo2.Template) (string, string) {
	outputPath, err := outputPathTpl.Execute(context)
	e(err)

	res, err := tpl.Execute(context)
	e(err)

	// 連続した改行を詰める
	re := regexp.MustCompile("\n+")
	res = re.ReplaceAllString(res, "\n")

	return outputPath, res
}
