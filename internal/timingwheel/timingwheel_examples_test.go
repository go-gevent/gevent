package timingwheel_test

import (
	"fmt"
	"github.com/go-gevent/gevent/internal/timingwheel"
	"time"
)

func Example_startTimer() {
	tw := timingwheel.NewTimingWheel(time.Millisecond, 20)
	tw.Start()
	defer tw.Stop()

	exitC := make(chan time.Time, 1)
	tw.AfterFunc(time.Second, func() {
		fmt.Println("The timer fires")
		exitC <- time.Now()
	})

	<-exitC

	// Output:
	// The timer fires
}

func Example_stopTimer() {
	tw := timingwheel.NewTimingWheel(time.Millisecond, 20)
	tw.Start()
	defer tw.Stop()

	t := tw.AfterFunc(time.Second, func() {
		fmt.Println("The timer fires")
	})

	<-time.After(900 * time.Millisecond)
	// Stop the timer before it fires
	t.Stop()

	// Output:
	//
}
