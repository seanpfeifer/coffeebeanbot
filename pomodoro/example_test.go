package pomodoro_test

// Copyright 2017 Sean A. Pfeifer

import (
	"fmt"
	"time"

	"github.com/seanpfeifer/coffeebeanbot/pomodoro"
)

func ExampleNewPomodoro() {
	// This channel will prevent us from exiting the test before our Pomodoro has completed
	c := make(chan bool)
	onTestEnd := func(notify pomodoro.NotifyInfo, completed bool) {
		if completed {
			fmt.Printf("Work '%s' done!\n", notify.Title)
		}
		c <- true
	}

	notify := pomodoro.NotifyInfo{Title: "Create example"}
	pomodoro.NewPomodoro(time.Millisecond*2, onTestEnd, notify)

	<-c
	fmt.Println("Exiting test.")

	// Output:
	// Work 'Create example' done!
	// Exiting test.
}
