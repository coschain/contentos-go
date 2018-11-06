package eventloop

import (
	"fmt"
	"testing"
	"time"
)

func TestEventLoop(t *testing.T) {
	ev := NewEventLoop()

	go func() {

		ev.Post(func() {
			fmt.Println("Execute Low")
		})

		ev.PostHighPri(func() {
			fmt.Println("Execute High")
		})

		ev.Send(func() {
			fmt.Println("Send Low")
		})

		ev.Post(func() {
			fmt.Println("Execute Low")
		})
		ev.PostHighPri(func() {
			fmt.Println("Execute High")
		})

		ev.SendHighPri(func() {
			fmt.Println("Send High")
		})

		time.Sleep(time.Second)
		ev.Stop()
	}()
	ev.Run()
	fmt.Println("Get Stop Signal")
}