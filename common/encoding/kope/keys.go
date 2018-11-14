package kope

import "bytes"

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


func validKey(k Key) bool {
	if size := len(k); size >= 3 && (k[0] & ^byte(typeReversedFlag) == typeList) {
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

func ComplementKey(k Key) Key {
	if ck, err := Complement(k); err == nil {
		return ck
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
		ik = NewKey([]byte(ComplementKey(ik)))
	}
	return ConcatKey(prefix, ik, NewKey(primaryKey))
}

func SingleIndexRange(prefix Key, indexStart interface{}, indexLimit interface{}, reversed bool) (Key, Key) {
	start, limit := indexStart, indexLimit
	if start == nil {
		start = MinimalKey
	}
	if limit == nil {
		limit = MaximumKey
	}
	return SingleIndexKey(prefix, start, MinimalKey, reversed), SingleIndexKey(prefix, limit, MaximumKey, reversed)
}

func IndexedPrimaryKey(indexKey Key) Key {
	if validKey(indexKey) {
		data := indexKey[1: len(indexKey) - 2]
		pos := bytes.LastIndex(data, separator)
		if pos >= 0 {
			data = data[pos + len(separator):]
		}
		if validKey(data) {
			return data
		}
	}
	return nil
}
