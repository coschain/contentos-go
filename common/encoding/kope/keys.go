package kope

import (
	"bytes"
	"reflect"
)

type Key []byte
type Keys []Key

func (keys Keys) Len() int {
	return len(keys)
}

func (keys Keys) Less(i, j int) bool {
	return bytes.Compare(keys[i], keys[j]) < 0
}

func (keys Keys) Swap(i, j int) {
	keys[i], keys[j] = keys[j], keys[i]
}

var (
	keyType = reflect.TypeOf(Key(nil))
	keyTypePkgName = keyType.PkgPath() + "." + keyType.Name()
	keysType = reflect.TypeOf(Keys(nil))
	keysTypePkgName = keysType.PkgPath() + "." + keysType.Name()
)

func validKey(k Key) bool {
	if size := len(k); size >= 3 && (k[0] == typeList) {
		return true
	}
	return false
}

func NewKey(values...interface{}) Key {
	if data, err := Encode(values); err == nil {
		return data
	}
	return nil
}

func ConcatKey(keys...Key) Key {
	var data [][]byte
	for _, k := range keys {
		if validKey(k) && len(k) > 3 {
			_, _, kd, _ := unpack(k)
			data = append(data, kd)
		}
	}
	k, _ := pack(typeList, true, bytes.Join(data, separator))
	return k
}

func AppendKey(prefix Key, values...interface{}) Key {
	return ConcatKey(prefix, NewKey(values...))
}

func MinKey(prefix Key) Key {
	return AppendKey(prefix, MinimalKey)
}

func MaxKey(prefix Key) Key {
	return AppendKey(prefix, MaximumKey)
}

func nestKey(k Key) Key {
	nk := make([]byte, 0, len(k) + 3)
	nk = append(nk, typeList)
	nk = append(nk, k...)
	nk = append(nk, 0, extEnding)
	return nk
}

func complementKey(k Key) Key {
	if validKey(k) {
		if ck, err := Complement(k); err == nil {
			return nestKey(ck)
		}
	}
	return nil
}

func DecodeKey(k Key) interface{} {
	if d, err := Decode(k); err == nil {
		return d
	}
	return nil
}

func SingleIndexKey(prefix Key, indexKey interface{}, primaryKey interface{}, reversed bool) Key {
	ik := NewKey(indexKey)
	if reversed {
		ik = complementKey(ik)
	}
	return ConcatKey(prefix, ik, nestKey(NewKey(primaryKey)))
}

func SingleIndexRange(prefix Key, indexStart interface{}, indexLimit interface{}, reversed bool) (Key, Key) {
	start, limit := indexStart, indexLimit
	if start == nil {
		start = MinimalKey
	}
	if limit == nil {
		limit = MaximumKey
	}
	return SingleIndexKey(prefix, start, MinimalKey, reversed), SingleIndexKey(prefix, limit, MinimalKey, reversed)
}

func IndexedPrimaryValue(indexKey Key) interface{} {
	if pk := IndexedPrimaryKey(indexKey); pk != nil {
		if v, err := Decode(pk); err == nil {
			if s, ok := v.([]interface{}); ok {
				return s[0]
			}
		}
	}
	return nil
}

func IndexedPrimaryKey(indexKey Key) Key {
	if validKey(indexKey) {
		_, _, data, err := unpack(indexKey)
		if err == nil {
			pos := bytes.LastIndex(data, separator)
			if pos >= 0 {
				return data[pos + len(separator):]
			}
		}
	}
	return nil
}
