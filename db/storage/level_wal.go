package storage

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"github.com/syndtr/goleveldb/leveldb/filter"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"strconv"
	"sync"
)

var (
	walKeyNextTaskId = []byte("next_task_id")
	walKeyTaskPrefix = []byte("task_id_")
)

type LevelWriteAheadLog struct {
	file string
	db *leveldb.DB
	nextTaskId uint64
	lock sync.RWMutex
}

func NewLevelWriteAheadLog(file string) (*LevelWriteAheadLog, error) {
	db, err := leveldb.OpenFile(file, &opt.Options{
		Filter: filter.NewBloomFilter(10),
	})
	if _, corrupted := err.(*errors.ErrCorrupted); corrupted {
		db, err = leveldb.RecoverFile(file, nil)
	}
	if err != nil {
		return nil, err
	}
	nextTaskId := uint64(0)
	if data, err := db.Get(walKeyNextTaskId, nil); err == nil {
		if id, err := strconv.ParseUint(string(data), 10, 64); err == nil {
			nextTaskId = id
		}
	}
	return &LevelWriteAheadLog{
		file: file,
		db: db,
		nextTaskId: nextTaskId,
	}, nil
}

func (wal *LevelWriteAheadLog) Close() {
	wal.db.Close()
}

func (wal *LevelWriteAheadLog) NewTaskID() uint64 {
	wal.lock.Lock()
	defer wal.lock.Unlock()

	wal.nextTaskId++
	wal.db.Put(walKeyNextTaskId, []byte(strconv.FormatUint(wal.nextTaskId, 10)), nil)
	return wal.nextTaskId
}

func (wal *LevelWriteAheadLog) GetTasks() ([]*WriteTask, error) {
	it := wal.db.NewIterator(util.BytesPrefix(walKeyTaskPrefix), nil)
	defer it.Release()

	var result []*WriteTask
	for it.Next() {
		task := DecodeWriteTask(it.Value())
		if task == nil {
			return nil, errors.New("failed decoding wal data.")
		}
		result = append(result, task)
	}
	return result, nil
}

func (wal *LevelWriteAheadLog) GetTask(taskId uint64) (*WriteTask, error) {
	k := keyOfTask(taskId)
	v, err := wal.db.Get(k, nil)
	if err != nil {
		return nil, err
	}
	task := DecodeWriteTask(v)
	if task == nil {
		return nil, errors.New("failed decoding wal data.")
	}
	return task, nil
}

func (wal *LevelWriteAheadLog) PutTask(task *WriteTask) error {
	k := keyOfTask(task.TaskID)
	v := EncodeWriteTask(task)
	if len(v) == 0 {
		return errors.New("failed encoding wal data.")
	}
	return wal.db.Put(k, v, nil)
}

func (wal *LevelWriteAheadLog) PutTasks(tasks []*WriteTask) error {
	b := new(leveldb.Batch)
	for _, task := range tasks {
		k := keyOfTask(task.TaskID)
		v := EncodeWriteTask(task)
		if len(v) == 0 {
			return errors.New("failed encoding wal data.")
		}
		b.Put(k, v)
	}
	return wal.db.Write(b, nil)
}

func (wal *LevelWriteAheadLog) DeleteTask(taskId uint64) error {
	return wal.db.Delete(keyOfTask(taskId), nil)
}

func (wal *LevelWriteAheadLog) DeleteTasks(taskIds []uint64) error {
	b := new(leveldb.Batch)
	for _, taskId := range taskIds {
		b.Delete(keyOfTask(taskId))
	}
	return wal.db.Write(b, nil)
}

func keyOfTask(taskId uint64) []byte {
	return []byte(fmt.Sprintf("%s%016x", string(walKeyTaskPrefix), taskId))
}
