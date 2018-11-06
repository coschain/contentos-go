package eventloop

import "sync"

type EventLoop struct {
	evFunc chan uint32
	evStop chan bool

	highQueue []func()
	lowQueue  []func()

	lock sync.Mutex
}

func NewEventLoop() *EventLoop {
	ev := EventLoop{
		evFunc:make(chan uint32, 1024),
		evStop:make(chan bool),
		highQueue:make([]func(),0),
		lowQueue:make([]func(),0),
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
	s.highQueue = append( s.highQueue, f)
}

func (s *EventLoop) pushLow( f func() ){
	s.lock.Lock()
	defer s.lock.Unlock()
	s.lowQueue = append( s.lowQueue, f)
}

func (s *EventLoop) pop() ( func() )  {

	s.lock.Lock()
	defer s.lock.Unlock()

	if len(s.highQueue) > 0{
		result := s.highQueue[0]
		s.highQueue = s.highQueue[1:]
		return result
	}
	if len(s.lowQueue) > 0{
		result := s.lowQueue[0]
		s.lowQueue = s.lowQueue[1:]
		return result
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