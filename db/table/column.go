package table

import (
	"errors"
	"fmt"
	"reflect"
)

type TableColumn struct {
	name    string
	table   *Table
	ordinal int
	ti      *typeInfo
	index   *TableIndex
}

func (c *TableColumn) checkValueType(value interface{}) error {
	valType := reflect.TypeOf(value)
	if valType != c.ti.typ {
		return errors.New(fmt.Sprintf("column \"%s\" expects %s values. %s was given.", c.name, c.ti.typ.String(), valType.String()))
	}
	return nil
}
