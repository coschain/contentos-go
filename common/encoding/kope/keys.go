package kope

type Key = []byte

func NewKey(values ...interface{}) Key {
	if data, err := Encode(values); err == nil {
		return data
	}
	return nil
}

func ConcatKey(keys ...Key) Key {
	count := len(keys)
	if count < 1 {
		return nil
	} else if count == 1 {
		return keys[0]
	}
	return catLists(keys)
}

func AppendKey(prefix Key, values ...interface{}) Key {
	if len(values) == 0 {
		return prefix
	}
	return ConcatKey(prefix, NewKey(values...))
}

func MinKey(prefix Key) Key {
	return AppendKey(prefix, MinimalKey)
}

func MaxKey(prefix Key) Key {
	return AppendKey(prefix, MaximumKey)
}

func IndexKey(prefix Key, primaryKey Key, idxValues ...interface{}) Key {
	return ConcatKey(AppendKey(prefix, idxValues...), packList([]Key{primaryKey}))
}

func IndexedPrimaryKey(indexKey Key) Key {
	lists := unpackList(indexKey)
	return lists[len(lists)-1]
}

type ValueRange struct {
	start, limit interface{}
}

func IndexKeyRange(prefix Key, idxValues ...interface{}) (start Key, limit Key) {
	valueCount := len(idxValues)
	var startVal, limitVal interface{}
	ik, startVal, limitVal := prefix, MinimalKey, MaximumKey
	count := valueCount
	if valueCount > 0 {
		for i := 0; i < valueCount; i++ {
			if r, ok := idxValues[i].(ValueRange); ok {
				if r.start != nil {
					startVal = r.start
				}
				if r.limit != nil {
					limitVal = r.limit
				}
				count = i
				break
			}
		}
	}
	ik = AppendKey(ik, idxValues[:count])
	start, limit = AppendKey(ik, startVal), AppendKey(ik, limitVal)
	return
}
