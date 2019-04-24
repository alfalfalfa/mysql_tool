package models

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"github.com/app-studio/mysql_tool/util/null"
)

const SQL_PREFIX = `
BEGIN;
SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;
SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;
SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='STRICT_TRANS_TABLES,STRICT_ALL_TABLES,NO_ENGINE_SUBSTITUTION,ALLOW_INVALID_DATES';

`
const SQL_SUFFIX = `
SET SQL_MODE=@OLD_SQL_MODE;
SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;
SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;
COMMIT;

`

func (this Models) ToCreateSQL(fk bool, jsonComment bool) string {
	//TODO 日付
	//-- Fri Nov 25 15:19:33 2016

	res := bytes.NewBuffer(nil)
	res.WriteString(SQL_PREFIX)

	for _, t := range this.Tables {
		res.WriteString(t.ToCreateSQL(fk, jsonComment))
	}

	res.WriteString(SQL_SUFFIX)
	return res.String()
}

func (this Table) ToCreateSQL(fk bool, jsonComment bool) string {
	res := bytes.NewBuffer(nil)
	res.WriteString("\n")
	res.WriteString("-- -----------------------------------------------------\n")
	res.WriteString(fmt.Sprintf("-- Table `%s`\n", this.Name))
	res.WriteString("-- -----------------------------------------------------\n")

	res.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n", this.Name))

	defs := make([]string, 0)
	//Columns
	for _, c := range this.Columns {
		defs = append(defs, c.ToCreateSQL())
	}

	//PK
	//  PRIMARY KEY (`id`)
	if len(this.PrimaryKeys) > 0 {
		defs = append(defs, "  PRIMARY KEY ("+strings.Join(this.GetPrimaryKeyNames(), ", ")+")")
	}
	//Indexes
	for _, ix := range this.Indexes {
		defs = append(defs, ix.ToCreateSQL())
	}

	res.WriteString(strings.Join(defs, ",\n"))
	res.WriteString(")")

	if this.Engine != "" {
		res.WriteString(fmt.Sprintf("\nENGINE = %s", this.Engine))
	}
	if this.DefaultCharset != "" {
		res.WriteString(fmt.Sprintf("\nDEFAULT CHARACTER SET = %s", this.DefaultCharset))
	}
	if this.Comment != "" {
		res.WriteString(fmt.Sprintf("\nCOMMENT = '%s'", this.Comment))
	}
	res.WriteString(";\n")

	//Ref
	if fk {
		for _, c := range this.Columns {
			//簡易FK
			//ALTER TABLE `user_lock` ADD CONSTRAINT `fk_user_lock_user_id_user_id` FOREIGN KEY (`user_id`) REFERENCES `user` (`id`);
			if c.Reference != "" {
				res.WriteString(c.ToFKAddSQL(this.Name.LowerSnake()))
			}
		}
	}
	res.WriteString("\n")

	return res.String()
}

func (this Table) ToDropSQL() string {
	res := bytes.NewBuffer(nil)
	res.WriteString("DROP TABLE IF EXISTS `")
	res.WriteString(this.Name.LowerSnake())
	res.WriteString("`;\n")

	return res.String()
}

func (this Table) ToAlterSQL() string {
	res := bytes.NewBuffer(nil)
	res.WriteString("ALTER TABLE `")
	res.WriteString(this.Name.LowerSnake())
	res.WriteString("`")
	res.WriteString(" ENGINE=")
	res.WriteString(this.Engine)
	res.WriteString(" DEFAULT CHARSET=")
	res.WriteString(this.DefaultCharset)
	res.WriteString(" COMMENT='")
	res.WriteString(this.Comment)
	res.WriteString("';\n")

	return res.String()
}

func (this Column) ToCreateSQL() string {
	//  `login_bonus_id` INT NOT NULL COMMENT 'ログインボーナスのグルーピングID',
	res := bytes.NewBuffer(nil)
	res.WriteString(" ")
	res.WriteString("`")
	res.WriteString(this.Name.LowerSnake())
	res.WriteString("` ")
	res.WriteString(normalizeMysqlType(this.Type))
	if this.NotNull {
		res.WriteString(" NOT NULL")
	}
	if this.Default.Valid {
		res.WriteString(" DEFAULT ")
		res.WriteString(normalizeDefault(&this))
	}
	if this.Extra != "" {
		res.WriteString(" ")
		res.WriteString(this.Extra)
	}
	if this.Comment != "" {
		res.WriteString(" COMMENT '")
		res.WriteString(this.Comment)
		res.WriteString("'")
	}

	return res.String()
}

func (this Column) ToAddSQLWithDummyDefault(tableName string) string {
	// 時間型であれば、仮のデフォルト設定してAlter、その後デフォルト外す
	if this.IsTime() && this.NotNull && !this.Default.Valid{
		res := bytes.NewBuffer(nil)

		this.Default = null.StringFrom("CURRENT_TIMESTAMP")
		res.WriteString(this.ToAddSQL(tableName))

		this.Default = null.NullString()
		res.WriteString(this.ToModifySQL(tableName, ""))

		return res.String()
	}else{
		return this.ToAddSQL(tableName)
	}
}

func (this Column) ToAddSQL(tableName string) string {
	res := bytes.NewBuffer(nil)
	res.WriteString("ALTER TABLE `")
	res.WriteString(tableName)
	res.WriteString("`")
	res.WriteString(" ADD COLUMN")
	res.WriteString(this.ToCreateSQL())

	res.WriteString(getColumnOrder(&this))

	res.WriteString(";\n")
	return res.String()
}

func (this Column) ToDropSQL(tableName string) string {
	res := bytes.NewBuffer(nil)
	res.WriteString("ALTER TABLE `")
	res.WriteString(tableName)
	res.WriteString("`")
	res.WriteString(" DROP COLUMN `")
	res.WriteString(this.Name.LowerSnake())
	res.WriteString("`;\n")
	return res.String()
}

func (this Column) ToModifySQL(tableName string, order string) string {
	res := bytes.NewBuffer(nil)
	res.WriteString("ALTER TABLE `")
	res.WriteString(tableName)
	res.WriteString("`")
	res.WriteString(" MODIFY COLUMN")
	res.WriteString(this.ToCreateSQL())
	// NOT NULLでDefaultがないdatetimeを同じ位置にMODIFYしようとするとエラー発生することがあるため、順序に変更がない場合は順序変更クエリを出力しない
	if order != "" {
		res.WriteString(" ")
		res.WriteString(order)
		//res.WriteString(getColumnOrder(&this))
	}
	res.WriteString(";\n")
	return res.String()
}

func (this Column) ToRenameSQL(tableName string, to *Column) string {
	res := bytes.NewBuffer(nil)
	res.WriteString("ALTER TABLE `")
	res.WriteString(tableName)
	res.WriteString("`")
	res.WriteString(" CHANGE")

	res.WriteString(" ")
	res.WriteString("`")
	res.WriteString(this.Name.LowerSnake())
	res.WriteString("`")
	res.WriteString(to.ToCreateSQL())

	//res.WriteString(" ")
	//res.WriteString(getColumnOrder(to))

	res.WriteString(";\n")
	return res.String()
}

func (this Column) ToFKAddSQL(tableName string) string {
	res := bytes.NewBuffer(nil)
	tmp := strings.Split(this.Reference, ".")

	if len(tmp) < 2 {
		log.Fatalf("invalid REF field format. require format:'table.column', actual value:'%s' in table:%s, column:%s", this.Reference, tableName, this.Name)
	}
	refTableName := tmp[0]
	refColumnName := tmp[1]

	res.WriteString(fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `ref_%s`", tableName, generateConstraintSymbol(tableName, this.Name.LowerSnake(), refTableName, refColumnName)))
	res.WriteString(fmt.Sprintf(" FOREIGN KEY (`%s`)", this.Name.LowerSnake()))
	res.WriteString(fmt.Sprintf(" REFERENCES `%s` (`%s`)", refTableName, refColumnName))
	//res.WriteString(" ON DELETE NO ACTION\n  ON UPDATE NO ACTION")
	res.WriteString(";\n")
	return res.String()
}
func generateConstraintSymbol(tableName, columnName, refTableName, refColumnName string) string{
	symbol := fmt.Sprintf("%s_%s_%s_%s", tableName, columnName, refTableName, refColumnName)
	if 60 < len(symbol){
		symbol = fmt.Sprintf("%s_%s_%s", tableName, columnName, refTableName)
	}
	if 60 < len(symbol){
		symbol = fmt.Sprintf("%s_%s", tableName, columnName)
	}
	return symbol
}
func (this Column) ToFKDropSQL(tableName string) string {
	res := bytes.NewBuffer(nil)
	tmp := strings.Split(this.Reference, ".")
	refTableName := tmp[0]
	refColumnName := tmp[1]
	res.WriteString(fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `ref_%s`", tableName, generateConstraintSymbol(tableName, this.Name.LowerSnake(), refTableName, refColumnName)))
	res.WriteString(";\n")
	return res.String()
}

//TODO ASC,DESC
func (this Index) ToCreateSQL() string {
	//  INDEX `user_id_idx` (`user_id` ASC))
	//  INDEX `search_idx` (`user_Id` ASC, `received_at` ASC))
	//  UNIQUE INDEX `store_trans_idx` (`store_transaction_id` ASC),
	res := bytes.NewBuffer(nil)
	res.WriteString(" ")
	if this.Unique {
		res.WriteString(" UNIQUE")
	}
	res.WriteString(" INDEX")

	res.WriteString(" `")
	res.WriteString(this.Name)
	res.WriteString("` (")

	tmp := make([]string, 0)
	for _, colName := range this.ColumnNames {
		tmp = append(tmp, "`"+colName+"`")
	}
	res.WriteString(strings.Join(tmp, ", "))
	res.WriteString(")")
	return res.String()
}
func (this Index) ToAddSQL(tableName string) string {
	res := bytes.NewBuffer(nil)
	res.WriteString("ALTER TABLE `")
	res.WriteString(tableName)
	res.WriteString("`")
	res.WriteString(" ADD")
	res.WriteString(this.ToCreateSQL())
	res.WriteString(";\n")
	return res.String()
}

func (this Index) ToDropSQL(tableName string) string {
	res := bytes.NewBuffer(nil)
	res.WriteString("ALTER TABLE `")
	res.WriteString(tableName)
	res.WriteString("`")
	res.WriteString(" DROP INDEX `")
	res.WriteString(this.Name)
	res.WriteString("`;\n")
	return res.String()
}
