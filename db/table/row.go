package table

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/db/storage"
)

type TableRows struct {
	index    *TableIndex
	key      []byte
	keyStart []byte
	keyLimit []byte
	err      error
}

func errorTableRows(err error) *TableRows {
	return &TableRows{err:err}
}

func (rows *TableRows) Col(names...string) *TableCells {
	if rows.err != nil {
		return errorTableCells(rows.err)
	}
	table := rows.index.table
	cols := table.columns
	if len(names) > 0 {
		cols = []*TableColumn{}
		dups := make(map[string]bool)
		for _, name := range names {
			if dups[name] {
				return errorTableCells(errors.New(fmt.Sprintf("duplicated column names found: %s", name)))
			}
			if i, ok := table.columnByName[name]; ok {
				cols = append(cols, table.columns[i])
			} else {
				return errorTableCells(errors.New(fmt.Sprintf("unknown column name: %s", name)))
			}
			dups[name] = true
		}
	}
	var cells [][]*TableCell
	for _, r := range rows.rows() {
		line := make([]*TableCell, len(cols))
		for i, c := range cols {
			line[i] = &TableCell{ row: r, column: c }
		}
		cells = append(cells, line)
	}
	return &TableCells{table: table, cells: cells, rows: len(cells), cols: len(cols)}
}

func (rows *TableRows) Delete() error {
	if rows.err != nil {
		return rows.err
	}

	batch := rows.index.table.db.NewBatch()
	defer rows.index.table.db.DeleteBatch(batch)

	for _, r := range rows.rows() {
		if err := r.delete(rows.index.table.db, rows.index.table.db, batch, batch); err != nil {
			return err
		}
	}
	return batch.Write()
}

func (rows *TableRows) rows() (result []*TableRow) {
	if len(rows.key) > 0 {
		if rk, err := rows.index.rowKey(rows.key); err == nil {
			result = append(result, &TableRow{ table: rows.index.table, key: common.CopyBytes(rk)})
		}
	} else {
		if rks, err := rows.index.rowKeyScan(rows.keyStart, rows.keyLimit); err == nil {
			for _, rk := range rks {
				result = append(result, &TableRow{ table: rows.index.table, key: rk})
			}
		}
	}
	return
}


type TableRow struct {
	table *Table
	key []byte
}

func (r *TableRow) delete(dbGetter storage.DatabaseGetter, dbScanner storage.DatabaseScanner, dbPutter storage.DatabasePutter, dbDeleter storage.DatabaseDeleter) error {
	vio := r.table.valueIO
	for _, index := range r.table.indices {
		if index.typ == Primary {
			continue
		}
		colVal, err := vio.GetCellValue(dbGetter, dbScanner, r.key, index.column.ordinal)
		if err != nil {
			return err
		}
		err = index.removeIndex(dbDeleter, colVal, r.key)
		if err != nil {
			return err
		}
	}
	return vio.DeleteRow(dbGetter, dbScanner, dbPutter, dbDeleter, r.key)
}
