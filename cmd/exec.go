package cmd

import (
	"os"

	"fmt"

	"io/ioutil"

	"bytes"
	"strings"

	"encoding/json"

	"time"

	"github.com/app-studio/mysql_tool/util/copy"
	"github.com/docopt/docopt-go"
	"github.com/jinzhu/gorm"
	"golang.org/x/crypto/ssh/terminal"
	"syscall"
)

const usageExec = `mysql_tool exec
    sql実行(接続成功までリトライ)
    優先度: 標準入力 > SQL > SQLFILE

Usage:
    mysql_tool exec -h | --help
    mysql_tool exec [-q] [-t TIMEOUT] [-e SQL] DSN [SQLFILE]

Arg:
    DSN      mysql接続文字列(https://github.com/go-sql-driver/mysql#dsn-data-source-name)
    SQLFILE  実行するSQLファイル

Options:
    -h --help                        Show this screen.
    -t TIMEOUT, --timeout=TIMEOUT    タイムアウト秒数 [Default: 180]
    -e SQL, --execute=SQL            実行するSQL
    -q, --quiet                      接続状態を出力しない
`

type ExecArg struct {
	DSN     string `arg:"DSN"`
	SQLFILE string `arg:"SQLFILE"`
	SQL     string `arg:"--execute"`
	Timeout int    `arg:"--timeout"`
	Quiet   bool   `arg:"--quiet"`
}

func RunExec() {
	arguments, err := docopt.Parse(usageExec, os.Args[1:], true, "", false)
	if err != nil {
		panic(err)
	}
	//fmt.Println(json.ToJson(arguments))

	arg := &ExecArg{}
	copy.MapToStructWithTag(arguments, arg, "arg")

	//fmt.Println(json.Marshal(arg))

	sql := getSQL(arg)

	if sql == "" {
		return
	}

	//fmt.Println(strings.Join(separateSQL(sql), "\n-\n"))
	//return

	var db *gorm.DB
	second := 0
	for {
		second++
		db, err = gorm.Open("mysql", arg.DSN)

		if err != nil {
			if !arg.Quiet {
				fmt.Println("gorm.Open failed. retry...", err)
			}
			if second > arg.Timeout {
				if !arg.Quiet {
					fmt.Println("timeout.")
				}
				return
			}
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}

	defer db.Close()
	checkError(err)

	for _, sql := range separateSQL(sql) {
		res := execSQL(db, sql)
		res.printJson()
	}
}

type ResultData struct {
	SQL     string
	Columns []string
	Values  [][]string
}

func (this ResultData) printJson() {
	if len(this.Values) > 0 {
		fmt.Println("{")
		b, _ := json.Marshal(this.SQL)
		fmt.Print(`  "SQL":`)
		fmt.Print(string(b))
		fmt.Println(",")

		fmt.Print(`  "Columns":`)
		b, _ = json.Marshal(this.Columns)
		fmt.Print(string(b))
		fmt.Println(",")

		fmt.Println(`  "Values":[`)
		values := make([]string, 0)
		for _, v := range this.Values {
			buf := bytes.NewBuffer(nil)
			buf.WriteString("    ")
			b, _ := json.Marshal(v)
			buf.Write(b)
			values = append(values, buf.String())
		}
		fmt.Println(strings.Join(values, ",\n"))
		fmt.Println(`  ]`)

		fmt.Println("}")
	}
}

func execSQL(db *gorm.DB, sql string) *ResultData {
	res := &ResultData{
		SQL:     sql,
		Columns: make([]string, 0),
		Values:  make([][]string, 0),
	}

	rows, err := db.DB().Query(sql)
	checkError(err)
	defer rows.Close()
	var columns []string
	for rows.Next() {
		if columns == nil {
			columns, err := rows.Columns()
			checkError(err)
			res.Columns = columns
		}

		valueRefs := make([]*string, 0)
		valueRefsForScan := make([]interface{}, 0)
		for range res.Columns {
			var v string
			valueRefs = append(valueRefs, &v)
			valueRefsForScan = append(valueRefsForScan, &v)
		}

		rows.Scan(valueRefsForScan...)
		values := make([]string, 0)
		for _, v := range valueRefs {
			values = append(values, *v)
		}
		//fmt.Println(values)
		res.Values = append(res.Values, values)
	}
	checkError(rows.Err())

	return res
}

func getSQL(arg *ExecArg) string {
	res := ""
	if arg.SQL != "" {
		res = arg.SQL
	} else if arg.SQLFILE != "" {
		b, err := ioutil.ReadFile(arg.SQLFILE)
		checkError(err)
		res = string(b)
	} else if !terminal.IsTerminal(syscall.Stdin) {
		b, err := ioutil.ReadAll(os.Stdin)
		checkError(err)
		res = string(b)
	}
	return strings.TrimSpace(res)
}

func separateSQL(sql string) []string {
	var current, pre, prepre rune

	lineComment := false
	blockComment := false

	singleQuote := false
	doubleQuote := false
	backQuote := false
	escape := false

	res := make([]string, 0)
	buf := bytes.NewBuffer(nil)

	for _, c := range sql {
		prepre = pre
		pre = current
		current = c
		buf.WriteRune(c)

		//comment out
		if lineComment {
			if c == '\n' || c == '\r' {
				lineComment = false
			}
			continue
		}
		if blockComment {
			if c == '/' || pre == '*' {
				blockComment = false
			}
			continue
		}

		//escape in/out
		if escape {
			escape = false
			continue
		}
		if c == '\\' {
			escape = true
			continue
		}

		//quote in/out
		if !singleQuote && !doubleQuote && !backQuote {
			switch c {
			case '\'':
				singleQuote = true
			case '"':
				doubleQuote = true
			case '`':
				backQuote = true
			}
		} else if singleQuote {
			if c == '\'' {
				singleQuote = false
			}
		} else if doubleQuote {
			if c == '"' {
				doubleQuote = false
			}
		} else if backQuote {
			if c == '`' {
				backQuote = false
			}
		}
		if singleQuote || doubleQuote || backQuote {
			continue
		}

		//comment in
		if lineComment || blockComment {
			continue
		}
		if c == '#' {
			lineComment = true
			continue
		}
		if c == ' ' && pre == '-' && prepre == '-' {
			lineComment = true
			continue
		}
		if c == '*' && pre == '/' {
			blockComment = true
			continue
		}

		//separate
		if c == ';' {
			res = append(res, buf.String())
			buf = bytes.NewBuffer(nil)
		}
	}

	rest := buf.String()
	if strings.TrimSpace(rest) != "" {
		res = append(res, buf.String())
	}

	return res
}
