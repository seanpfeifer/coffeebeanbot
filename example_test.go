package coffeebeanbot_test

import (
	"fmt"
	"time"

	"github.com/seanpfeifer/coffeebeanbot"
)

func ExampleNewPomodoro() {
	// This channel will prevent us from exiting the test before our Pomodoro has completed
	c := make(chan bool)
	onTestEnd := func(notify coffeebeanbot.NotifyInfo, completed bool) {
		if completed {
			fmt.Printf("Work '%s' done!\n", notify.Title)
		}
		c <- true
	}

	notify := coffeebeanbot.NotifyInfo{Title: "Create example"}
	coffeebeanbot.NewPomodoro(time.Millisecond*2, onTestEnd, notify)

	<-c
	fmt.Println("Exiting test.")

	// Output:
	// Work 'Create example' done!
	// Exiting test.
}
