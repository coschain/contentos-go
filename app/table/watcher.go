package table

import (
	"fmt"
	"reflect"
)

const (
	EventTableRecordInsert = iota
	EventTableRecordUpdate
	EventTableRecordDelete
)
var (
	vTableRecordInsert = reflect.ValueOf(EventTableRecordInsert)
	vTableRecordUpdate = reflect.ValueOf(EventTableRecordUpdate)
	vTableRecordDelete = reflect.ValueOf(EventTableRecordDelete)
)

type tableWatcher struct {
	field string
	keyType  reflect.Type
	dataType reflect.Type
	callback reflect.Value
}

var sTableWatchers = make(map[reflect.Type][]*tableWatcher)
var sTableWatcherChangeCallbacks = make(map[reflect.Type]func())

func RegisterTableWatcherChangedCallback(recordType reflect.Type, callback func()) {
	sTableWatcherChangeCallbacks[recordType] = callback
}

func notifyWatcherChanged(recordType reflect.Type) {
	if callback, ok := sTableWatcherChangeCallbacks[recordType]; ok {
		callback()
	}
}

func AddTableRecordFieldWatcher(recordType reflect.Type, primaryField, watchingField string, fn interface{}) {
	if recordType == nil || recordType.Kind() != reflect.Struct {
		panic(fmt.Sprintf("record type %v is not a struct", recordType))
	}
	keyType := getStructMemberType(recordType, primaryField)
	valType := reflect.PtrTo(recordType)
	if len(watchingField) > 0 {
		valType = getStructMemberType(recordType, watchingField)
	}
	if fn == nil {
		panic("watcher is nil")
	}
	watcher := reflect.ValueOf(fn)
	if watcher.Kind() != reflect.Func {
		panic("watcher is not a function")
	}
	watcherType := watcher.Type()
	if watcherType.NumIn() != 4 ||
		watcherType.In(0).Kind() != reflect.Int ||
		watcherType.In(1) != keyType ||
		watcherType.In(2) != valType ||
		watcherType.In(3) != valType {
		panic(fmt.Sprintf("watcher function type must be func(event int, key %v, before, after %v)", keyType, valType))
	}
	sTableWatchers[recordType] = append(sTableWatchers[recordType], &tableWatcher{
		field: watchingField,
		keyType: keyType,
		dataType: valType,
		callback: watcher,
	})
	notifyWatcherChanged(recordType)
}

func AddTableRecordWatcher(recordType reflect.Type, primaryField string, fn interface{}) {
	AddTableRecordFieldWatcher(recordType, primaryField, "", fn)
}

func HasTableRecordWatcher(recordType reflect.Type, field string) (result bool) {
	if watchers, ok := sTableWatchers[recordType]; ok {
		for _, w := range watchers {
			if len(w.field) == 0 || w.field == field {
				result = true
				break
			}
		}
	}
	return
}

func getStructMemberType(recordType reflect.Type, member string) reflect.Type {
	if sf, found := recordType.FieldByName(member); !found {
		panic(fmt.Sprintf("record type %v has no member called %s", recordType, member))
	} else {
		return sf.Type
	}
}

func ReportTableRecordInsert(key interface{}, record interface{}) {
	vRec := reflect.ValueOf(record)
	if watchers, ok := sTableWatchers[vRec.Type().Elem()]; ok {
		vKey := reflect.ValueOf(key)
		for _, w := range watchers {
			vData := vRec
			if len(w.field) > 0 {
				vData = vRec.Elem().FieldByName(w.field)
			}
			w.callback.Call([]reflect.Value{
				vTableRecordInsert,
				vKey,
				reflect.Zero(w.dataType),
				vData,
			})
		}
	}
}

func ReportTableRecordUpdate(key interface{}, oldRecord, newRecord interface{}) {
	oldRec, newRec := reflect.ValueOf(oldRecord), reflect.ValueOf(newRecord)
	if watchers, ok := sTableWatchers[newRec.Type().Elem()]; ok {
		vKey := reflect.ValueOf(key)
		for _, w := range watchers {
			oldData, newData := oldRec, newRec
			if len(w.field) > 0 {
				oldData, newData = oldRec.Elem().FieldByName(w.field), newRec.Elem().FieldByName(w.field)
			}
			w.callback.Call([]reflect.Value{
				vTableRecordUpdate,
				vKey,
				oldData,
				newData,
			})
		}
	}
}

func ReportTableRecordDelete(key interface{}, record interface{}) {
	vRec := reflect.ValueOf(record)
	if watchers, ok := sTableWatchers[vRec.Type().Elem()]; ok {
		vKey := reflect.ValueOf(key)
		for _, w := range watchers {
			vData := vRec
			if len(w.field) > 0 {
				vData = vRec.Elem().FieldByName(w.field)
			}
			w.callback.Call([]reflect.Value{
				vTableRecordDelete,
				vKey,
				vData,
				reflect.Zero(w.dataType),
			})
		}
	}
}

