package table

import (
	"fmt"
	"reflect"
	"sync"
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

var (
	sTableWatchers = make(map[uint32]map[reflect.Type][]*tableWatcher)	// db -> { record_type -> []watchers }
	sTableWatchersLock sync.RWMutex

	sTableWatcherChangeCallbacks = make(map[reflect.Type]func(uint32))
	sTableWatcherChangeCallbacksLock sync.RWMutex
)

func RegisterTableWatcherChangedCallback(recordType reflect.Type, callback func(uint32)) {
	sTableWatcherChangeCallbacksLock.Lock()
	defer sTableWatcherChangeCallbacksLock.Unlock()

	sTableWatcherChangeCallbacks[recordType] = callback
}

func notifyWatcherChanged(dbSvcId uint32, recordType reflect.Type) {
	sTableWatcherChangeCallbacksLock.RLock()
	defer sTableWatcherChangeCallbacksLock.RUnlock()

	if callback, ok := sTableWatcherChangeCallbacks[recordType]; ok {
		callback(dbSvcId)
	}
}

func AddTableRecordFieldWatcher(dbSvcId uint32, recordType reflect.Type, primaryField, watchingField string, fn interface{}) {
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

	sTableWatchersLock.Lock()
	dbWatchers := sTableWatchers[dbSvcId]
	if dbWatchers == nil {
		dbWatchers = make(map[reflect.Type][]*tableWatcher)
		sTableWatchers[dbSvcId] = dbWatchers
	}
	dbWatchers[recordType] = append(dbWatchers[recordType], &tableWatcher{
		field: watchingField,
		keyType: keyType,
		dataType: valType,
		callback: watcher,
	})
	sTableWatchersLock.Unlock()

	notifyWatcherChanged(dbSvcId, recordType)
}

func getStructMemberType(recordType reflect.Type, member string) reflect.Type {
	if sf, found := recordType.FieldByName(member); !found {
		panic(fmt.Sprintf("record type %v has no member called %s", recordType, member))
	} else {
		return sf.Type
	}
}

func AddTableRecordWatcher(dbSvcId uint32, recordType reflect.Type, primaryField string, fn interface{}) {
	AddTableRecordFieldWatcher(dbSvcId, recordType, primaryField, "", fn)
}

func HasTableRecordWatcher(dbSvcId uint32, recordType reflect.Type, field string) (result bool) {
	sTableWatchersLock.RLock()
	defer sTableWatchersLock.RUnlock()

	if dbTableWatchers := sTableWatchers[dbSvcId]; len(dbTableWatchers) > 0 {
		if watchers := dbTableWatchers[recordType]; len(watchers) > 0 {
			for _, w := range watchers {
				if len(w.field) == 0 || w.field == field {
					result = true
					break
				}
			}
		}
	}
	return
}

func ReportTableRecordInsert(dbSvcId uint32, key interface{}, record interface{}) {
	sTableWatchersLock.RLock()
	defer sTableWatchersLock.RUnlock()

	if dbWatchers := sTableWatchers[dbSvcId]; len(dbWatchers) > 0 {
		vRec := reflect.ValueOf(record)
		if watchers, ok := dbWatchers[vRec.Type().Elem()]; ok {
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
}

func ReportTableRecordUpdate(dbSvcId uint32, key interface{}, oldRecord, newRecord interface{}) {
	sTableWatchersLock.RLock()
	defer sTableWatchersLock.RUnlock()

	if dbWatchers := sTableWatchers[dbSvcId]; len(dbWatchers) > 0 {
		oldRec, newRec := reflect.ValueOf(oldRecord), reflect.ValueOf(newRecord)
		if watchers, ok := dbWatchers[newRec.Type().Elem()]; ok {
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
}

func ReportTableRecordDelete(dbSvcId uint32, key interface{}, record interface{}) {
	sTableWatchersLock.RLock()
	defer sTableWatchersLock.RUnlock()

	if dbWatchers := sTableWatchers[dbSvcId]; len(dbWatchers) > 0 {
		vRec := reflect.ValueOf(record)
		if watchers, ok := dbWatchers[vRec.Type().Elem()]; ok {
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
}
