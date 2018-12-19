package models

import (
	"fmt"
	"sort"
	"strings"
)

type Reference struct {
	From *TableColumn
	To   *TableColumn
}

type TableColumn struct {
	Table  *Table
	Column *Column
}

// diff,codegen用Ref解決, IndexのColumns解決
func (this *Models) resolveReferences() {
	//a-z table order
	sort.Sort(this)

	for _, t := range this.Tables {
		// set pk ref
		t.PrimaryKeys = t.getPrimaryKeys()

		//set column order
		var pre *Column
		for _, c := range t.Columns {
			c.PreColumn = pre
			pre = c

			// set Table ref
			c.Table = t

			// fix Extra to upper case
			c.Extra = strings.ToUpper(c.Extra)
		}

		// set index:column ref
		for _, ix := range t.Indexes {
			for _, name := range ix.ColumnNames {
				column := t.findColumn(name)
				if column == nil {
					panic(fmt.Sprintf("column %s not found. table:%s, index:%s, columns:%v", name, t.Name.LowerSnake(), ix.Name, ix.ColumnNames))
				}
				ix.addRefColumn(column)
			}
		}
	}

	//for _, t := range this.Tables {
	//	t.References = make([]Reference, 0)
	//	t.InverseReferences = make([]Reference, 0)
	//	for _, c := range t.Columns {
	//		c.References = make([]Reference, 0)
	//		c.InverseReferences = make([]Reference, 0)
	//		c.Indexes = make([]*Index, 0)
	//	}
	//	for _, ix := range t.Indexes {
	//		ix.Columns = make([]*Column, 0)
	//	}
	//}

	// set table associations
	for _, t := range this.Tables {
		for _, c := range t.Columns {
			if c.Reference == "" {
				continue
			}
			c.addRef(this.findReferringColumn(c))
		}
	}
}

func (this Models) findReferringColumn(column *Column) *Column {
	names := strings.Split(column.Reference, ".")
	if len(names) != 2 {
		panic(fmt.Sprintf("ref name must be 'table_name.column_name'. table:%s, column:%s, ref:%s", column.Table.Name.LowerSnake(), column.Name.LowerSnake(), column.Reference))
	}
	toTableName := strings.TrimSpace(names[0])
	toColumnName := strings.TrimSpace(names[1])
	for _, t := range this.Tables {
		if t.Name.LowerSnake() != toTableName {
			continue
		}
		for _, c := range t.Columns {
			if c.Name.LowerSnake() != toColumnName {
				continue
			}
			return c
		}
	}
	panic(fmt.Sprintf("column not found. table:%s, column:%s, ref:%s", column.Table.Name.LowerSnake(), column.Name.LowerSnake(), column.Reference))
}

func (c *Column) addRef(referenced *Column) {
	if c.References == nil {
		c.References = make([]*Reference, 0)
	}
	if referenced.InverseReferences == nil {
		referenced.InverseReferences = make([]*Reference, 0)
	}
	if c.Table.References == nil {
		c.Table.References = make([]*Reference, 0)
	}
	if referenced.Table.InverseReferences == nil {
		referenced.Table.InverseReferences = make([]*Reference, 0)
	}
	ref := &Reference{
		From: &TableColumn{
			Table:  c.Table,
			Column: c,
		},
		To: &TableColumn{
			Table:  referenced.Table,
			Column: referenced,
		},
	}

	c.References = append(c.References, ref)
	c.Table.References = append(c.Table.References, ref)
	referenced.InverseReferences = append(referenced.InverseReferences, ref)
	referenced.Table.InverseReferences = append(referenced.Table.InverseReferences, ref)
}

func (t Table) findColumn(columnName string) *Column {
	for _, c := range t.Columns {
		if c.Name.LowerSnake() == columnName {
			return c
		}
	}
	return nil
}

func (ix *Index) addRefColumn(c *Column) {
	if ix.Columns == nil {
		ix.Columns = make([]*Column, 0)
	}
	if c.Indexes == nil {
		c.Indexes = make([]*Index, 0)
	}
	ix.Columns = append(ix.Columns, c)
	c.Indexes = append(c.Indexes, ix)
}
