package models

func (t Table) FirstPKName() string {
	return t.GetPK().Name.LowerSnake()
}

// 単一のUnique Indexを持つか
func (c Column) IsUnique() bool {
	if c.PrimaryKey != 0 && c.Table.GetPrimaryKeyNum() == 1 {
		return true
	}
	for _, ix := range c.Indexes {
		if ix.Unique && len(ix.Columns) == 1 {
			return true
		}
	}
	return false
}

func (c Column) IsAutoIncrement() bool {
	return c.Extra == "AUTO_INCREMENT"
}
