package cmd

import (
	"os"

	"bytes"

	"fmt"

	"io/ioutil"

	"time"

	"path/filepath"

	"github.com/app-studio/mysql_tool/models"
	"github.com/app-studio/mysql_tool/util/copy"
	"github.com/docopt/docopt-go"
	"github.com/sergi/go-diff/diffmatchpatch"
)

const usageDiff = `mysql_tool diff
    テーブル定義ドキュメント or データベース間の変換

Usage:
    mysql_tool diff -h | --help
    mysql_tool diff [--old OLD] [-f FORMAT] [-o OUTPUT] [--foreign-key] [--ignore-tables IGNORE_TABLES...] [--json-comment] INPUTS...

Arg:
    入力ファイルパス（json, yaml, xlsx, dir） | mysql fqdn

Options:
    -h --help                     Show this screen.
    --old OLD                      diff比較元
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
    --foreign-key                 外部キーの出力
    --ignore-tables=IGNORE_TABLES...      無視テーブル
    --json-comment                メタデータjsonのコメント埋め込み
`

//[--overwrite]
//    --overwrite                   oldファイル上書き

type DiffArg struct {
	Format       string   `arg:"--format"`
	Output       string   `arg:"--output"`
	ForeignKey   bool     `arg:"--foreign-key"`
	JsonComment  bool     `arg:"--json-comment"`
	Inputs       []string `arg:"INPUTS"`
	Old          string   `arg:"--old"`
	Overwrite    bool     `arg:"--overwrite"`
	IgnoreTables []string `arg:"--ignore-tables"`
}

func RunDiff() {
	arguments, err := docopt.Parse(usageDiff, os.Args[1:], true, "", false)
	if err != nil {
		panic(err)
	}
	arg := &DiffArg{}
	copy.MapToStructWithTag(arguments, arg, "arg")

	newModel := models.LoadModel(arg.IgnoreTables, arg.Inputs...)
	var output string

	var oldModel *models.Models
	if arg.Old == "" {
		oldModel = &models.Models{}
	} else {
		oldModel = models.LoadModel(arg.IgnoreTables, arg.Old)
	}

	switch arg.Format {
	case "diff":
		n := newModel.ToCreateSQL(arg.ForeignKey, arg.JsonComment)
		o := oldModel.ToCreateSQL(arg.ForeignKey, arg.JsonComment)
		output = diff(o, n)
	case "sql":
		alter, _ := diffDefines(arg, newModel, oldModel)

		if alter != "" {
			output += models.SQL_PREFIX
			output += alter
			output += models.SQL_SUFFIX
		}

	case "goose":
		alter, revert := diffDefines(arg, newModel, oldModel)
		if alter != "" {
			output += `
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
`
			//XXX goose は文中のトランザクション無視する
			output += models.SQL_PREFIX
			output += alter
			output += models.SQL_SUFFIX

			output += `
-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
`

			output += models.SQL_PREFIX
			output += revert
			output += models.SQL_SUFFIX

		}
	}

	if output == "" {
		//fmt.Println("no diff")
		return
	}
	if arg.Output == "" {
		fmt.Println(output)
	} else {
		info, err := os.Stat(arg.Output)

		if err != nil || !info.IsDir() {
			//ファイルパス
			//上書き
			os.MkdirAll(filepath.Dir(arg.Output), os.ModePerm)
			fmt.Println(arg.Output)
			checkError(ioutil.WriteFile(arg.Output, []byte(output), os.ModePerm))
		} else {
			//ディレクトリ
			//日付からファイル名生成
			os.MkdirAll(arg.Output, os.ModePerm)
			name := time.Now().Format("20060102150405") + "_" + arg.Format + ".sql"
			out := filepath.Join(arg.Output, name)
			fmt.Println(out)
			checkError(ioutil.WriteFile(out, []byte(output), os.ModePerm))
		}
	}

	//Overwrite
}

func diff(v1 string, v2 string) string {
	buf := bytes.NewBuffer(nil)
	//fmt.Println(v1)
	result := lineDiff(v1, v2)
	diff := false
	for _, r := range result {
		if r.Type != diffmatchpatch.DiffEqual {
			diff = true
		}
	}
	if diff {
		//	buf.WriteString(k)
	}
	for _, r := range result {
		if r.Type == diffmatchpatch.DiffDelete {
			buf.WriteString("-")
		}
		if r.Type == diffmatchpatch.DiffInsert {
			buf.WriteString("+")
		}

		if r.Type != diffmatchpatch.DiffEqual {
			buf.WriteString(r.Text)
		}
	}
	return buf.String()
}

func lineDiff(src1, src2 string) []diffmatchpatch.Diff {
	dmp := diffmatchpatch.New()
	a, b, c := dmp.DiffLinesToChars(src1, src2)
	diffs := dmp.DiffMain(a, b, false)
	result := dmp.DiffCharsToLines(diffs, c)
	//fmt.Println(result)
	return result
}

/*
TODO カラム削除時に関連するIndex, FKのAlterクエリは吐かない

*/
func diffDefines(arg *DiffArg, newModel, oldModel *models.Models) (alter, revert string) {
	alterBuf := bytes.NewBuffer(nil)
	revertBuf := bytes.NewBuffer(nil)

	//テーブル追加/削除
	addTables, dropTables, remainTableNames := diffTableByName(newModel, oldModel)
	for _, t := range dropTables {
		alterBuf.WriteString(t.ToDropSQL())
	}
	for _, t := range addTables {
		revertBuf.WriteString(t.ToDropSQL())
	}
	for _, t := range dropTables {
		revertBuf.WriteString(t.ToCreateSQL(arg.ForeignKey, arg.JsonComment))
	}
	for _, t := range addTables {
		alterBuf.WriteString(t.ToCreateSQL(arg.ForeignKey, arg.JsonComment))
	}
	//テーブル変更
	for _, name := range remainTableNames {
		newTable := newModel.GetTable(name)
		oldTable := oldModel.GetTable(name)
		if oldTable.IsChange(newTable) {
			alterBuf.WriteString(newTable.ToAlterSQL())
			revertBuf.WriteString(oldTable.ToAlterSQL())
		}
	}

	//カラム追加/削除 AddSQL
	for _, tableName := range remainTableNames {
		newTable := newModel.GetTable(tableName)
		oldTable := oldModel.GetTable(tableName)
		//カラム追加/削除
		adds, drops, _, renames := diffColumnByDefine(newTable, oldTable)

		for _, r := range renames{
			alterBuf.WriteString(r.Old.ToRenameSQL(tableName, r.New))
			revertBuf.WriteString(r.New.ToRenameSQL(tableName, r.Old))
		}
		for _, c := range drops {
			// 日付型, NOT NULLの場合の仮のデフォルト値を自動で設定する TODO オプションで切り替える？
			//revertBuf.WriteString(c.ToAddSQLWithDummyDefault(tableName))
			revertBuf.WriteString(c.ToAddSQL(tableName))
		}
		for _, c := range adds {
			// 日付型, NOT NULLの場合の仮のデフォルト値を自動で設定する TODO オプションで切り替える？
			//alterBuf.WriteString(c.ToAddSQLWithDummyDefault(tableName))
			alterBuf.WriteString(c.ToAddSQL(tableName))
		}
	}

	//インデックス追加/削除
	for _, tableName := range remainTableNames {
		newTable := newModel.GetTable(tableName)
		oldTable := oldModel.GetTable(tableName)
		adds, drops, modifyNames := diffIndexBySQL(newTable, oldTable)
		for _, indexName := range modifyNames {
			newIndex := newTable.GetIndex(indexName)
			oldIndex := oldTable.GetIndex(indexName)
			alterBuf.WriteString(newIndex.ToDropSQL(tableName))
			alterBuf.WriteString(newIndex.ToAddSQL(tableName))
			revertBuf.WriteString(oldIndex.ToDropSQL(tableName))
			revertBuf.WriteString(oldIndex.ToAddSQL(tableName))
		}
		for _, in := range drops {
			alterBuf.WriteString(in.ToDropSQL(tableName))
		}
		for _, in := range adds {
			revertBuf.WriteString(in.ToDropSQL(tableName))
		}
		for _, in := range drops {
			revertBuf.WriteString(in.ToAddSQL(tableName))
		}
		for _, in := range adds {
			alterBuf.WriteString(in.ToAddSQL(tableName))
		}
	}

	//Ref
	if arg.ForeignKey {
		for _, tableName := range remainTableNames {
			newTable := newModel.GetTable(tableName)
			oldTable := oldModel.GetTable(tableName)
			adds, drops := diffRef(newTable, oldTable)

			for _, c := range drops {
				alterBuf.WriteString(c.ToFKDropSQL(tableName))
			}
			for _, c := range adds {
				revertBuf.WriteString(c.ToFKDropSQL(tableName))
			}
			for _, c := range drops {
				revertBuf.WriteString(c.ToFKAddSQL(tableName))
			}
			for _, c := range adds {
				alterBuf.WriteString(c.ToFKAddSQL(tableName))
			}
		}
	}

	//カラム追加/削除 DropSQL
	for _, tableName := range remainTableNames {
		newTable := newModel.GetTable(tableName)
		oldTable := oldModel.GetTable(tableName)

		//カラム追加/削除
		adds, drops, _, _ := diffColumnByDefine(newTable, oldTable)
		for _, c := range drops {
			if c.Reference != "" {
				alterBuf.WriteString(c.ToFKDropSQL(tableName))
			}
			alterBuf.WriteString(c.ToDropSQL(tableName))
		}
		for _, c := range adds {
			revertBuf.WriteString(c.ToDropSQL(tableName))
		}
	}

	//カラム定義変更 ModifySQL
	for _, tableName := range remainTableNames {
		newTable := newModel.GetTable(tableName)
		oldTable := oldModel.GetTable(tableName)

		// 順序変更無しのカラム定義変更を適用
		_, _, commonNames := diffColumnByName(newTable, oldTable)
		for _, columnName := range commonNames {
			newColumn := newTable.GetColumn(columnName)
			oldColumn := oldTable.GetColumn(columnName)

			changeRes := oldColumn.IsChange(newColumn)
			if changeRes != models.ColumnChangeType_Same {
				//fmt.Println("column chnaged:", tableName, columnName, changeRes)
				alterBuf.WriteString(newColumn.ToModifySQL(tableName, ""))
				revertBuf.WriteString(oldColumn.ToModifySQL(tableName, ""))
			}
		}

		// カラム並び順の変更適用
		for _, moveOp := range getMoveOps(newTable, oldTable) {
			if moveOp.After != "" {
				alterBuf.WriteString(newTable.GetColumn(moveOp.Column).ToModifySQL(tableName, "AFTER "+moveOp.After))
			} else {
				alterBuf.WriteString(newTable.GetColumn(moveOp.Column).ToModifySQL(tableName, "FIRST"))
			}
		}
		for _, moveOp := range getMoveOps(oldTable, newTable) {
			if moveOp.After != "" {
				revertBuf.WriteString(oldTable.GetColumn(moveOp.Column).ToModifySQL(tableName, "AFTER "+moveOp.After))

			} else {
				revertBuf.WriteString(oldTable.GetColumn(moveOp.Column).ToModifySQL(tableName, "FIRST"))
			}
		}

	}

	alter = alterBuf.String()
	revert = revertBuf.String()
	return
}

// =============================================
func diffTableByName(new, old *models.Models) (addTables, dropTables []*models.Table, remainNames []string) {
	addTables = make([]*models.Table, 0)
	dropTables = make([]*models.Table, 0)
	remainNames = make([]string, 0)

	for _, newTable := range new.Tables {
		oldTable := old.GetTable(newTable.Name.LowerSnake())
		if oldTable == nil {
			addTables = append(addTables, newTable)
		} else {
			remainNames = append(remainNames, newTable.Name.LowerSnake())
		}
	}

	for _, oldTable := range old.Tables {
		newTable := new.GetTable(oldTable.Name.LowerSnake())
		if newTable == nil {
			dropTables = append(dropTables, oldTable)
		}
	}
	return
}

func diffColumnByName(newTable, oldTable *models.Table) (addColumns, dropColumns []*models.Column, remainNames []string) {
	// 旧テーブル定義にカラム定義がないもの
	addColumns = make([]*models.Column, 0)
	// 新テーブル定義にカラム定義がないもの
	dropColumns = make([]*models.Column, 0)
	remainNames = make([]string, 0)

	for _, newColumn := range newTable.Columns {
		oldColumn := oldTable.GetColumn(newColumn.Name.LowerSnake())
		if oldColumn == nil {
			addColumns = append(addColumns, newColumn)
		} else {
			remainNames = append(remainNames, newColumn.Name.LowerSnake())
		}
	}

	for _, oldColumn := range oldTable.Columns {
		newColumn := newTable.GetColumn(oldColumn.Name.LowerSnake())
		if newColumn == nil {
			dropColumns = append(dropColumns, oldColumn)
		}
	}
	return
}

type renameOperation struct {
	Old *models.Column
	New *models.Column
}

func diffColumnByDefine(newTable, oldTable *models.Table) (addColumns, dropColumns []*models.Column, remainNames []string, renameOperations []renameOperation) {
	missingNewColumns, missingOldColumns, remainNames := diffColumnByName(newTable, oldTable)

	addColumns = make([]*models.Column, 0)
	dropColumns = make([]*models.Column, 0)

	renameOperations = make([]renameOperation, 0)
	for _, addColumn := range missingNewColumns {
		similarColumn := getSimilarColumn(missingOldColumns, addColumn, renameOperations)
		if similarColumn != nil{
			renameOperations = append(renameOperations, renameOperation {
				Old: similarColumn,
				New: addColumn,
			})
		}else{
			addColumns = append(addColumns, addColumn)
		}
	}
	for _, dropColumn := range missingOldColumns {
		if !containsOld(renameOperations, dropColumn){
			dropColumns = append(dropColumns, dropColumn)
		}
	}
	return
}

func getSimilarColumn(columns []*models.Column, column *models.Column, renameOperations []renameOperation) *models.Column{
	for _, c := range columns {
		if containsOld(renameOperations, column){
			continue
		}
		changeType := c.IsChange(column)
		// 追加/削除されたカラムの中で、型とコメント(論理名)、NotNull制約, Default, Extraが同一であればリネームとみなす
		if changeType == models.ColumnChangeType_Same {
			// 追加/削除されたカラムの中で、型とコメント(論理名)が同一であればリネームとみなす
			//if changeType != models.ColumnChangeType_Type && changeType != models.ColumnChangeType_Comment {
			return c
		}
	}

	return nil
}
func containsOld(renameOperations []renameOperation, column *models.Column) bool{
	for _, r := range renameOperations {
		if r.Old == column{
			return true
		}
	}

	return false
}

func diffIndexBySQL(new, old *models.Table) (adds, drops []*models.Index, modifyNames []string) {
	adds = make([]*models.Index, 0)
	drops = make([]*models.Index, 0)
	modifyNames = make([]string, 0)

	news := make(map[string]*models.Index)
	olds := make(map[string]*models.Index)

	for _, newIndex := range new.Indexes {
		news[newIndex.ToCreateSQL()] = newIndex
	}
	for _, oldIndex := range old.Indexes {
		olds[oldIndex.ToCreateSQL()] = oldIndex
	}

	for newSQL, newIndex := range news {
		if _, ok := olds[newSQL]; !ok {
			if old.GetIndex(newIndex.Name) == nil {
				adds = append(adds, newIndex)
			} else {
				modifyNames = append(modifyNames, newIndex.Name)
			}
		}
	}

	for oldSQL, oldIndex := range olds {
		if _, ok := news[oldSQL]; !ok {
			if new.GetIndex(oldIndex.Name) == nil {
				drops = append(drops, oldIndex)
			}
		}
	}
	return
}

func diffRef(new, old *models.Table) (adds, drops []*models.Column) {
	adds = make([]*models.Column, 0)
	drops = make([]*models.Column, 0)

	for _, newColumn := range new.Columns {
		if newColumn.Reference == "" {
			continue
		}
		oldColumn := old.GetColumn(newColumn.Name.LowerSnake())
		if oldColumn == nil {
			continue
		}
		if newColumn.Reference != oldColumn.Reference {
			adds = append(adds, newColumn)
		}
	}

	for _, oldColumn := range old.Columns {
		if oldColumn.Reference == "" {
			continue
		}
		newColumn := new.GetColumn(oldColumn.Name.LowerSnake())
		if newColumn == nil {
			continue
		}
		if newColumn.Reference != oldColumn.Reference {
			drops = append(drops, oldColumn)
		}
	}
	return
}

// カラムの位置変更=============================================
type stringSlice []string

func (this stringSlice) index(value string) int {
	for p, v := range this {
		if v == value {
			return p
		}
	}
	return -1
}
func (this stringSlice) insert(i int, value string) stringSlice {
	return append(this[:i], append([]string{value}, this[i:]...)...)
}
func (this stringSlice) delete(value string) stringSlice {
	i := this.index(value)
	return append(this[:i], this[i+1:]...)
}
func (this stringSlice) equals(other stringSlice) bool {
	if len(this) != len(other) {
		return false
	}
	for i, v := range this {
		if v != other[i] {
			return false
		}
	}
	return true
}

type moveOperation struct {
	Column string
	After  string
}

func getMoveOps(newTable, oldTable *models.Table) []moveOperation {
	res := make([]moveOperation, 0)

	//from := stringSlice(oldTable.GetColumnNames())
	to := stringSlice(newTable.GetColumnNames())
	cache := stringSlice(oldTable.GetColumnNames())

	adds, drops, _ := diffColumnByName(newTable, oldTable)

	// 追加
	for _, addColumn := range adds {
		v := addColumn.Name.LowerSnake()
		i := to.index(v)
		if i == 0 {
			cache = cache.insert(0, v)
		} else {
			after := to[i-1]
			cache = cache.insert(cache.index(after)+1, v)
		}
	}

	// 削除
	for _, dropColumn := range drops {
		v := dropColumn.Name.LowerSnake()
		cache = cache.delete(v)
	}

	// 移動
	for !cache.equals(to) {
		for _, v := range cache {
			from_i := cache.index(v)
			from_prev := ""
			if from_i != 0 {
				from_prev = cache[from_i-1]
			}
			to_i := to.index(v)
			to_prev := ""
			if to_i != 0 {
				to_prev = to[to_i-1]
			}

			if from_prev != to_prev {
				res = append(res, moveOperation{
					Column: v,
					After:  to_prev,
				})
				cache = cache.delete(v)
				if to_prev != "" {
					cache = cache.insert(cache.index(to_prev)+1, v)
				} else {
					cache = cache.insert(0, v)
				}
			}
		}
	}

	return res
}
