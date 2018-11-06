package eventloop

import (
	"container/list"
	"sync"
)

type EventLoop struct {
	evFunc chan uint32
	evStop chan bool

	highQueue *list.List
	lowQueue  *list.List

	lock sync.Mutex
}

func NewEventLoop() *EventLoop {
	ev := EventLoop{
		evFunc:make(chan uint32, 1024),
		evStop:make(chan bool),
		highQueue:list.New(),
		lowQueue:list.New(),
		lock:sync.Mutex{},
	}
	return &ev
}

func (s *EventLoop) Run()  {
	for {
		select {
		case <-s.evFunc:{
			f := s.pop()
			if f != nil{
				f()
			}
		}
		case <-s.evStop:
			return
		}
	}
}

func (s *EventLoop) pushHigh( f func() ){
	s.lock.Lock()
	defer s.lock.Unlock()

	s.highQueue.PushBack( f )
}

func (s *EventLoop) pushLow( f func() ){
	s.lock.Lock()
	defer s.lock.Unlock()

	s.lowQueue.PushBack( f )
}

func (s *EventLoop) pop() ( func() )  {

	s.lock.Lock()
	defer s.lock.Unlock()

	result := s.highQueue.Front()
	if result != nil {
		s.highQueue.Remove(result)
		return result.Value.(func())
	}

	result = s.lowQueue.Front()
	if result != nil {
		s.lowQueue.Remove(result)
		return result.Value.(func())
	}
	return nil
}

func (s *EventLoop)Stop()  {
	s.evStop <- true
}

func (s *EventLoop) Post( f func() )  {
	s.pushLow( f )
	s.evFunc <- 1
}

func (s *EventLoop) PostHighPri( f func() )  {
	s.pushHigh( f )
	s.evFunc <- 1
}

func (s *EventLoop) SendHighPri( f func() )  {
	done := make(chan bool, 1)

	s.pushHigh(func() {
		f()
		done <- true
	})
	s.evFunc <- 1
	<-done
}

func (s *EventLoop) Send( f func() )  {
	done := make(chan bool, 1)

	s.pushLow(func() {
		f()
		done <- true
	})
	s.evFunc <- 1
	<-done
}