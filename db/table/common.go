package table

import "github.com/coschain/contentos-go/db/storage"

type TableValueIO interface {
	NewRow(dbGetter storage.DatabaseGetter, dbScanner storage.DatabaseScanner, dbPutter storage.DatabasePutter, dbDeleter storage.DatabaseDeleter, columnValues ...interface{}) (rowKey []byte, err error)
	DeleteRow(dbGetter storage.DatabaseGetter, dbScanner storage.DatabaseScanner, dbPutter storage.DatabasePutter, dbDeleter storage.DatabaseDeleter, rowKey []byte) error
	GetCellValue(dbGetter storage.DatabaseGetter, dbScanner storage.DatabaseScanner, rowKey []byte, columnIndex int) (interface{}, error)
	SetCellValue(dbGetter storage.DatabaseGetter, dbScanner storage.DatabaseScanner, dbPutter storage.DatabasePutter, dbDeleter storage.DatabaseDeleter, rowKey []byte, columnIndex int, columnValue interface{}) error
}
