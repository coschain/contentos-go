package eventloop

type EventLoop2 struct {
	highPrioChan chan func()
	lowPrioChan  chan func()
	evStop       chan bool
}

func NewEventLoop2() *EventLoop2 {
	l := EventLoop2{
		highPrioChan: make(chan func(), 1024),
		lowPrioChan:  make(chan func(), 1024),
		evStop:       make(chan bool),
	}
	return &l
}

func (l *EventLoop2) Run() {

		for {
			select {
			case f := <-l.highPrioChan:
				f()
			case f := <-l.lowPrioChan:
				// process high first
				bk := false
				for !bk {
					select {
					case fHi := <-l.highPrioChan:
						fHi()
					default:
					bk = true
						break
					}
				}
				f()
			case <-l.evStop:
				// need close ?
				close(l.highPrioChan)
				close(l.lowPrioChan)
				return
			}
		}
}

func (l *EventLoop2) Stop(){
	l.evStop <- true
}

func (l *EventLoop2) PushLow(f func()) {
	l.lowPrioChan <- f
}

func (l *EventLoop2) PushHigh(f func()) {
	l.highPrioChan <- f
}

func (l *EventLoop2) SendHigh(f func()) {
	done := make(chan bool, 1)

	l.PushHigh(func() {
		f()
		done <- true
	})
	<-done
}

func (l *EventLoop2) SendLow(f func()) {
	done := make(chan bool, 1)

	l.PushLow(func() {
		f()
		done <- true
	})
	<-done
}