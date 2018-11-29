package storage

import (
	"bytes"
	"encoding/gob"
)

type WriteTask struct {
	TaskID     uint64
	DatabaseID string
	Operations []writeOp
}

type WriteAheadLog interface {
	NewTaskID() uint64
	GetTasks() ([]*WriteTask, error)
	GetTask(taskId uint64) (*WriteTask, error)
	PutTask(task *WriteTask) error
	PutTasks(tasks []*WriteTask) error
	DeleteTask(taskId uint64) error
	DeleteTasks(taskIds []uint64) error
}

func EncodeWriteTask(task *WriteTask) []byte {
	buf := new(bytes.Buffer)
	if err := gob.NewEncoder(buf).Encode(task); err == nil {
		return buf.Bytes()
	}
	return nil
}

func DecodeWriteTask(data []byte) *WriteTask {
	task := new(WriteTask)
	if err := gob.NewDecoder(bytes.NewBuffer(data)).Decode(task); err == nil {
		return task
	}
	return nil
}
