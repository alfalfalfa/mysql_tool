# mysql_tool

Mysqlツール群


# サブコマンド
- [conv](#conv)	:	テーブル定義(ドキュメント or データベース)間の変換
- [diff](#diff)	:	テーブル定義の差分出力(マイグレーション用)
- [data](#data)	:	データ定義の変換、mysql入出力
- [gen](#gen)	:	テーブル定義からtemplateを使用してテキスト生成
- [gen-multiple](#gen-multiple)	:	テーブル定義から各テーブル毎にテキスト生成
- [exec](#exec)	:	sql実行(接続成功までリトライ)

## conv
	mysql_tool
		Mysqlツール群
	
	Usage:
		mysql_tool -h | --help
		mysql_tool COMMAND
	
	Arg:
		"conv"           テーブル定義ドキュメント or データベース間の変換
		"diff"           テーブル定義の差分出力(マイグレーション用)
		"data"           データ定義の変換、mysql入出力
		"gen"            テーブル定義からgolangの text/template でテキスト生成
		"gen-multiple"   テーブル定義から各テーブル毎にテキスト生成
		"exec"           sql実行(接続成功までリトライ)
	
	Options:
		-h --help    Show this screen.
例

	# mysqlデータべースからHoge.jsonに出力
	mysql_tool conv -f json -o Hoge.json "root@hoge(127.0.0.1:3306)/hoge"
		
	# -oがあれば-fは拡張子から判別される
	mysql_tool conv -o Hoge.json "root@hoge(127.0.0.1:3306)/hoge"
		
	# -o未指定で標準出力へ (-f xlsxはエラー)
	mysql_tool conv -f json "root@hoge(127.0.0.1:3306)/hoge"
		
	# ExcelファイルからHoge.jsonに出力
	mysql_tool conv -f json -o Hoge.json User.xlsx Master.xlsx
		
	# -oがあれば-fは拡張子から判別される
	mysql_tool conv -o Hoge.json User.xlsx Master.xlsx
		
	# -o未指定で標準出力へ (-f xlsxはエラー)
	mysql_tool conv -f json User.xlsx Master.xlsx


## diff
	mysql_tool diff
		テーブル定義ドキュメント or データベース間の変換
	
	Usage:
		mysql_tool diff -h | --help
		mysql_tool diff -old OLD [-f FORMAT] [-o OUTPUT] [--foreign-key] [--json-comment] [--overwrite] INPUTS...
	
	Arg:
		入力ファイルパス（json,xlsx） | mysql fqdn
	
	Options:
		-h --help                     Show this screen.
		-old OLD                      diff比較元
			ファイルパス
				指定ファイル(xlsx,json)からの差分を出力
			fqdn
				指定データベースからの差分を出力
		-f FORMAT, --format=FORMAT    出力フォーマット [default: sql]
			"sql"
				差分のalter文を出力
			"goose"
				goose-up, goose-downを出力
			"diff"
				create table文のdiffを出力
		-o OUTPUT, --output=OUTPUT    出力先
			ディレクトリ
				日付からファイル名生成
			ファイルパス
				上書き
			none
				標準出力
		--overwrite                   oldファイル上書き
		--foreign-key                 外部キーの出力
		--json-comment                メタデータjsonのコメント埋め込み

例

	# mysqlデータベースと最新のテーブル定義Excelから差分をgooseフォーマットで標準出力に出力
	mysql_tool diff -old "root@hoge(127.0.0.1:3306)/hoge" -f goose User.xlsx Master.xlsx

## data
    mysql_tool data
        データ定義の変換、mysql入出力
    
    Usage:
        mysql_tool data -h | --help
        mysql_tool data [-f FORMAT] [-o OUTPUT] [--tables TABLES...] [--ignore-tables IGNORE_TABLES...] INPUTS...
    
    Arg:
        入力ファイルパス（json,xlsx） | mysql fqdn
    
    Options:
        -h --help                             Show this screen.
        -f FORMAT, --format=FORMAT            出力フォーマット
            "sql"
                insert文を出力
            "json"
                Json出力
            "xlsx"
                Excel出力
            none
                OUTPUTの拡張子から自動判別 （default: sql）
        -o OUTPUT, --output=OUTPUT            出力先
            ファイルパス
                上書き
            none
                標準出力（Excel出力では無効）
        --tables=TABLES...                    対象テーブル
        --ignore-tables=IGNORE_TABLES...      無視テーブル

例

	# 複数のデータ定義ExcelからInsert文を標準出力に出力
	data -f sql master.xlsx user.xlsx
	
	# mysqlデータベースから指定のテーブル(hoge, mage)のデータをExcelに出力(dump)
	data -o out.xlsx --tables hoge --tables mage master.xlsx user.xlsx "root@hoge(127.0.0.1:3306)/hoge"
	
## gen
    mysql_tool gen
        テーブル定義からtemplateを使用してテキスト生成
    
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

## gen-multiple
	mysql_tool gen-multiple
		テーブル定義からテーブル毎にテキスト生成
	
	Usage:
		mysql_tool gen-multiple -h | --help
		mysql_tool gen-multiple [--overwrite=<OVERWRITE_MODE>] [--template-type=<TEMPLATE_TYPE>] [--tables TABLES...] [--ignore-tables IGNORE_TABLES...] <TEMPLATE_PATH> <OUTPUT_PATH_PETTERN> INPUTS...
	
	Arg:
		<TEMPLATE_PATH>        (必須)テンプレートファイルパス
		<OUTPUT_PATH_PETTERN>  (必須)出力ファイルパスパターン
		INPUTS...   入力ファイルパス（json,xlsx） | mysql fqdn
	
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

## exec
    mysql_tool exec
        sql実行(接続成功までリトライ)
        優先度: 標準入力 > SQL > SQLFILE
    
    Usage:
        mysql_tool exec -h | --help
        mysql_tool exec [-q] [-t TIMEOUT] [-e SQL] FQDN [SQLFILE]
    
    Arg:
        FQDN     mysql接続文字列
        SQLFILE  実行するSQLファイル
    
    Options:
        -h --help                        Show this screen.
        -t TIMEOUT, --timeout=TIMEOUT    タイムアウト秒数 [Default: 180]
        -e SQL, --execute=SQL            実行するSQL
        -q, --quiet                      接続状態を出力しない

例

	# DB起動を待ってデータベース作成
    mysql_tool exec -e "create database hoge;" "root@hoge(127.0.0.1:3306)/mysql"
	# パイプで複数insert
    echo "insert into hoge values(\"mage\");insert into hoge values(\"mage\");" | mysql_tool exec "root@hoge(127.0.0.1:3306)/hoge"



# TODO
- DONE in:	Excel
- DONE in:	Json
- DONE in:	Mysql
- DONE out:	Excel
- DONE out:	Json
- DONE out:	SQL
- DONE arg
- TODO diff-in:	Excel
- TODO diff-in:	Mysql
- TODO diff-in:	Json
- TODO diff-out:	sql
- TODO diff-out:	goose
- TODO diff-out:	diff


- TODO JsonComment ?
- TODO Default値
- TODO 詳細な外部キー定義
