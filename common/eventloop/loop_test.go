package eventloop
/*
import (
	"fmt"
	"sync"
	"testing"
	"time"
)

const (
	loop = 100
)

var wg sync.WaitGroup

func TestEventLoop(t *testing.T) {
	ev := NewEventLoop()
	start := time.Now()
	go func() {
		for i := 0; i < loop; i++ {
			ev.Post(func() {
				//fmt.Println("Execute Low")
			})

			ev.PostHighPri(func() {
				//fmt.Println("Execute High")
			})

			ev.Send(func() {
				//fmt.Println("Send Low")
			})

			ev.Post(func() {
				//fmt.Println("Execute Low")
			})
			ev.PostHighPri(func() {
				//	fmt.Println("Execute High")
			})

			ev.SendHighPri(func() {
				//	fmt.Println("Send High")
			})
		}
		time.Sleep(time.Second)
		ev.Stop()
	}()
	ev.Run()
	fmt.Println("Get Stop Signal")
	fmt.Println("time:", time.Since(start))
}

func TestEventLoop2(t *testing.T) {
	ev := NewEventLoop2()
	start := time.Now()
	go func() {
		for i := 0; i < loop; i++ {
			ev.PushLow(func() {
				//	fmt.Println("Execute Low")
			})

			ev.PushHigh(func() {
				//	fmt.Println("Execute High")
			})

			ev.SendLow(func() {
				//	fmt.Println("Send Low")
			})

			ev.PushLow(func() {
				//		fmt.Println("Execute Low")
			})
			ev.PushHigh(func() {
				//	fmt.Println("Execute High")
			})

			ev.SendHigh(func() {
				//		fmt.Println("Send High")
			})
		}
		time.Sleep(time.Second)
		ev.Stop()
	}()
	ev.Run()
	fmt.Println("Get Stop Signal")
	fmt.Println("time:", time.Since(start))
}

func TestEventLoopPriority(t *testing.T) {
	ev := NewEventLoop()
	start := time.Now()

	go func() {
		fmt.Println("loop:", loop)
		for i := 0; i < loop; i++ {
			ev.Post(func() {
				fmt.Println("Low")
			})
		}

	}()

	go func() {

		for i := 0; i < loop; i++ {

			ev.PostHighPri(func() {
				fmt.Println("High")
			})
		}

	}()

	ev.Run()
	fmt.Println("Get Stop Signal")
	fmt.Println("time:", time.Since(start))
}

func TestEventLoop2Priority(t *testing.T) {
	ev := NewEventLoop2()
	start := time.Now()

	go func() {
		fmt.Println("loop:", loop)
		for i := 0; i < loop; i++ {
			ev.PushLow(func() {
				fmt.Println("Low")
			})
		}

	}()

	go func() {

		for i := 0; i < loop; i++ {

			ev.PushHigh(func() {
				fmt.Println("High")
			})
		}

	}()

	ev.Run()
	fmt.Println("Get Stop Signal")
	fmt.Println("time:", time.Since(start))
}

func startWriteLoop2(t string, ev *EventLoop2) {
	if t == "high" {
		go func() {
			for i := 0; i < loop; i++ {

				ev.PushHigh(func() {
					fmt.Println("<--High-->")
				})
			}
		}()
	} else if t == "low" {
		go func() {
			for i := 0; i < loop; i++ {

				ev.PushLow(func() {
					fmt.Println("<--Low-->")
				})
			}
		}()
	} else {
		panic("invalid option")
	}
}

func startWriteLoop(t string, ev *EventLoop) {
	if t == "high" {
		go func() {
			for i := 0; i < loop; i++ {

				ev.PostHighPri(func() {
					fmt.Println("High")
				})
			}
		}()
	} else if t == "low" {
		go func() {
			for i := 0; i < loop; i++ {

				ev.Post(func() {
					fmt.Println("Low")
				})
			}
		}()
	} else {
		panic("invalid option")
	}
}

func TestMoreRoutine(t *testing.T) {
	ev := NewEventLoop()
	start := time.Now()

	for i := 0; i < 10; i++ {
		startWriteLoop("high", ev)
	}

	for i := 0; i < 10; i++ {
		startWriteLoop("low", ev)
	}

	ev.Run()
	fmt.Println("Get Stop Signal")
	fmt.Println("time:", time.Since(start))
}

func TestMoreRoutine2(t *testing.T) {
	ev := NewEventLoop2()
	start := time.Now()

	for i := 0; i < 10; i++ {
		startWriteLoop2("high", ev)
	}

	for i := 0; i < 10; i++ {
		startWriteLoop2("low", ev)
	}

	ev.Run()
	fmt.Println("Get Stop Signal")
	fmt.Println("time:", time.Since(start))
}
*/