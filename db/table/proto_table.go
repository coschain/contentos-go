package table

import (
	"fmt"
	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"reflect"
	"strings"
)

type protoColumn struct {
	name string
	typ reflect.Type
	typeId string
	field int
	indexType TableIndexType
}

type protoTableBuilder struct {
	tb *tableBuilder
	typ reflect.Type
	st reflect.Type
	cols []protoColumn
	colsByName map[string]int
	vio *protoTableValueIO
}

func ProtoTableBuilder(x proto.Message) *protoTableBuilder {
	b := &protoTableBuilder{
		tb: TableBuilder(),
		typ: reflect.TypeOf(x),
		colsByName: make(map[string]int),
	}
	b.tb.Name(proto.MessageName(x))
	b.parse()
	return b
}

func (b *protoTableBuilder) Database(db storage.Database) *protoTableBuilder {
	b.tb.Database(db)
	return b
}

func (b *protoTableBuilder) Name(name string) *protoTableBuilder {
	b.tb.Name(name)
	return b
}

func (b *protoTableBuilder) Build() (*Table, error) {
	for _, col := range b.cols {
		b.tb.Column(col.name, col.typeId, col.indexType)
	}
	t, err := b.tb.Build()
	if err != nil {
		return nil, err
	}
	b.vio.primaryCol = t.primaryIndex.column.ordinal
	b.vio.prefix = t.primaryIndex.prefix
	return t, nil
}

func (b *protoTableBuilder) Index(name string, indexType ...TableIndexType) *protoTableBuilder {
	idx, ok := b.colsByName[name]
	if !ok {
		b.error("unknown column name: " + name)
	} else if len(indexType) > 0 {
		b.cols[idx].indexType = indexType[0]
	}
	return b
}

func (b *protoTableBuilder) error(msg string) {
	b.tb.err = errors.New(msg)
}

func (b *protoTableBuilder) parse() {
	b.parseProtoType()
	if b.tb.err == nil {
		b.prepareValueIO()
	}
}

func (b *protoTableBuilder) parseProtoType() {
	t := b.typ
	if t.Kind() != reflect.Ptr {
		b.error("proto.Message must be a struct pointer." + t.String() + " was given.")
		return
	}
	t = t.Elem()
	if t.Kind() != reflect.Struct {
		b.error("proto.Message must be a struct pointer. *" + t.String() + " was given.")
		return
	}
	b.st = t
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if len(f.Name) < 1 || f.Name[0] <= 'A' || f.Name[0] >= 'Z' {
			continue
		}
		protoTag := f.Tag.Get("protobuf")
		if len(protoTag) == 0 {
			continue
		}
		colName := ""
		for _, tag := range strings.Split(protoTag, ",") {
			if strings.HasPrefix(tag, "name=") {
				colName = tag[5:]
				break
			}
		}
		if len(colName) == 0 {
			b.error(fmt.Sprintf("%s.%s has no name in its tag.", t.String(), f.Name))
			return
		}
		typeName := RegisteredName(f.Type)
		if len(typeName) == 0 {
			b.error(fmt.Sprintf("%s.%s's type %s not registered.", t.String(), f.Name, f.Type.String()))
			return
		}
		protoCol := protoColumn{
			name: colName,
			typ: f.Type,
			typeId: typeName,
			field: i,
			indexType: NotIndexed,
		}
		b.colsByName[protoCol.name] = len(b.cols)
		b.cols = append(b.cols, protoCol)
	}
}

func (b *protoTableBuilder) prepareValueIO() {
	b.vio = &protoTableValueIO {
		mt: b.typ,
		st: b.st,
		cols: b.cols,
	}
	b.tb.ValueIO(b.vio)
}

type protoTableValueIO struct {
	mt reflect.Type
	st reflect.Type
	cols []protoColumn
	primaryCol int
	prefix []byte
}

func (vio *protoTableValueIO) NewRow(dbGetter storage.DatabaseGetter, dbScanner storage.DatabaseScanner, dbPutter storage.DatabasePutter, dbDeleter storage.DatabaseDeleter, columnValues...interface{}) (rowKey []byte, err error) {
	if len(columnValues) != len(vio.cols) {
		return nil, errors.New(fmt.Sprintf("column count is %d, but %d values given.", len(vio.cols), len(columnValues)))
	}
	msg := reflect.New(vio.st)
	s := msg.Elem()
	for i, val := range columnValues {
		col := vio.cols[i]
		v := reflect.ValueOf(val)
		if v.Type() != col.typ {
			return nil, errors.New(fmt.Sprintf("type of column %s is %s, %s was given", col.name, col.typ.String(), v.Type().String()))
		}
		s.Field(col.field).Set(v)
	}
	rk := kope.AppendKey(vio.prefix, columnValues[vio.primaryCol])
	if dup, err := dbGetter.Has(rk); err != nil {
		return nil, err
	} else if dup {
		return nil, errors.New("duplicate row")
	}
	err = vio.putMessage(dbPutter, rk, msg)
	if err != nil {
		return nil, err
	}
	return rk, nil
}

func (vio *protoTableValueIO) DeleteRow(dbGetter storage.DatabaseGetter, dbScanner storage.DatabaseScanner, dbPutter storage.DatabasePutter, dbDeleter storage.DatabaseDeleter, rowKey []byte) error {
	return dbDeleter.Delete(rowKey)
}

func (vio *protoTableValueIO) GetCellValue(dbGetter storage.DatabaseGetter, dbScanner storage.DatabaseScanner, rowKey []byte, columnIndex int) (interface{}, error) {
	if columnIndex < 0 || columnIndex >= len(vio.cols) {
		return nil, errors.New(fmt.Sprintf("columnIndex overflow: %d not in [%d, %d]", columnIndex, 0, len(vio.cols) - 1))
	}
	msg, err := vio.getMessage(dbGetter, rowKey)
	if err != nil {
		return nil, err
	}
	return msg.Elem().Field(vio.cols[columnIndex].field).Interface(), nil
}

func (vio *protoTableValueIO) SetCellValue(dbGetter storage.DatabaseGetter, dbScanner storage.DatabaseScanner, dbPutter storage.DatabasePutter, dbDeleter storage.DatabaseDeleter, rowKey []byte, columnIndex int, columnValue interface{}) error {
	if columnIndex < 0 || columnIndex >= len(vio.cols) {
		return errors.New(fmt.Sprintf("columnIndex overflow: %d not in [%d, %d]", columnIndex, 0, len(vio.cols) - 1))
	}
	col := vio.cols[columnIndex]
	valType := reflect.TypeOf(columnValue)
	if valType != col.typ {
		return errors.New(fmt.Sprintf("type of column %s is %s, %s was given", col.name, col.typ.String(), valType.String()))
	}
	msg, err := vio.getMessage(dbGetter, rowKey)
	if err != nil {
		return err
	}
	msg.Elem().Field(col.field).Set(reflect.ValueOf(columnValue))
	return vio.putMessage(dbPutter, rowKey, msg)
}

func (vio *protoTableValueIO) getMessage(dbGetter storage.DatabaseGetter, rowKey []byte) (reflect.Value, error) {
	data, err := dbGetter.Get(rowKey)
	if err != nil {
		return reflect.ValueOf(nil), err
	}
	msg := reflect.New(vio.st)
	if err = proto.Unmarshal(data, msg.Interface().(proto.Message)); err != nil {
		return reflect.ValueOf(nil), err
	}
	return msg, nil
}

func (vio *protoTableValueIO) putMessage(dbPutter storage.DatabasePutter, rowKey []byte, m reflect.Value) error {
	data, err := proto.Marshal(m.Interface().(proto.Message))
	if err != nil {
		return err
	}
	return dbPutter.Put(rowKey, data)
}
