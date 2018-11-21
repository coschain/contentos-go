package table

import (
	"errors"
	"github.com/coschain/contentos-go/db/storage"
)

type TableCell struct {
	row *TableRow
	column *TableColumn
}

func (c *TableCell) get() (interface{}, error) {
	table := c.row.table
	db := table.db
	return table.valueIO.GetCellValue(db, db, c.row.key, c.column.ordinal)
}

func (c *TableCell) modify(dbPutter storage.DatabasePutter, dbDeleter storage.DatabaseDeleter, modifer func(interface{})(interface{}, error)) error {
	index := c.column.index
	if index != nil && index.typ == Primary {
		return errors.New("primary key modification not supported.")
	}
	table := c.row.table
	db := table.db
	oldValue, err := table.valueIO.GetCellValue(db, db, c.row.key, c.column.ordinal)
	if err != nil {
		return err
	}
	newValue, err := modifer(oldValue)
	if err != nil {
		return err
	}
	if err = table.valueIO.SetCellValue(db, db, dbPutter, dbDeleter, c.row.key, c.column.ordinal, newValue); err != nil {
		return err
	}
	if index != nil {
		if err = index.removeIndex(dbDeleter, oldValue, c.row.key); err != nil {
			return err
		}
		if err = index.addIndex(dbPutter, newValue, c.row.key); err != nil {
			return err
		}
	}
	return nil
}

type TableCells struct {
	table *Table
	cells [][]*TableCell
	rows, cols int
	err error
}

func errorTableCells(err error) *TableCells {
	return &TableCells{err:err}
}

func (c *TableCells) Size() (rows int, cols int) {
	return c.rows, c.cols
}

func (c *TableCells) Get() (values [][]interface{}, err error) {
	if c.err != nil {
		return nil, c.err
	}
	for _, row := range c.cells {
		rowValues := make([]interface{}, len(row))
		for _, cell := range row {
			if value, err := cell.get(); err == nil {
				rowValues = append(rowValues, value)
			} else {
				return nil, err
			}
		}
		values = append(values, rowValues)
	}
	return values, nil
}

func (c *TableCells) Modify(modifer func(interface{})(interface{}, error)) error {
	if c.err != nil {
		return c.err
	}

	batch := c.table.db.NewBatch()
	defer c.table.db.DeleteBatch(batch)

	for _, row := range c.cells {
		for _, cell := range row {
			if err := cell.modify(batch, batch, modifer); err != nil {
				return err
			}
		}
	}
	return batch.Write()
}
