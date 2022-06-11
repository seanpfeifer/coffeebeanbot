package pomodoro

import (
	"fmt"
	"sync"
	"testing"
	"time"

	. "github.com/seanpfeifer/rigging/assert"
)

// timeTolerance is the amount of time difference we will tolerate in a test before failing.
// The required value can vary between machines and CI environments, so this may need tweaking,
// but we should keep an eye on it so it stays somewhat reasonable per test.
const timeTolerance = time.Millisecond * 8

func TestPomodoro(t *testing.T) {
	const testDuration = time.Millisecond * 42
	c := make(chan bool)
	testFunc := func(_ NotifyInfo, completed bool) {
		c <- completed
	}

	startTime := time.Now()
	// Intentionally not using the returned Pomodoro - no need for it
	NewPomodoro(testDuration, testFunc, NotifyInfo{})
	completed := <-c
	ExpectedActual(t, true, completed, "Pomodoro completion")

	endDuration := time.Since(startTime)

	ExpectedApprox(t, testDuration, endDuration, timeTolerance, "ending Pomorodoro on time")
}

func TestPomodoroCancel(t *testing.T) {
	const testDuration = time.Millisecond * 100
	const cancelDuration = time.Millisecond * 10
	c := make(chan bool)
	testFunc := func(_ NotifyInfo, completed bool) {
		c <- completed
	}

	startTime := time.Now()
	pom := NewPomodoro(testDuration, testFunc, NotifyInfo{})
	go func() {
		time.Sleep(cancelDuration)
		pom.Cancel()
	}()

	completed := <-c
	ExpectedActual(t, false, completed, "Pomodoro cancellation")
	endDuration := time.Since(startTime)

	ExpectedApprox(t, cancelDuration, endDuration, timeTolerance, "cancelling Pomorodoro on time")
}

func TestPomMapCreate(t *testing.T) {
	cpm := NewChannelPomMap()
	if cpm.channelToPom == nil {
		t.Fatal("Expected non-nil map")
	}

	type pomTestCase struct {
		duration      time.Duration
		notify        NotifyInfo
		shouldSucceed bool
	}
	cases := []pomTestCase{
		{time.Millisecond * 42, NotifyInfo{ChannelID: "TheChannel"}, true},
		{time.Millisecond * 42, NotifyInfo{ChannelID: "TheChannel"}, false},
		{time.Millisecond * 42, NotifyInfo{ChannelID: "TheChannel2"}, true},
		{time.Millisecond * 42, NotifyInfo{ChannelID: "TheChannel2"}, false},
	}
	var wg sync.WaitGroup
	wg.Add(len(cases))

	startTime := time.Now()

	onFinish := func(index int, info NotifyInfo, success bool) {
		defer wg.Done()

		endDuration := time.Since(startTime)

		ExpectedApprox(t, cases[index].duration, endDuration, timeTolerance, fmt.Sprintf("ending Pomorodoro %d on time", index))
		ExpectedActual(t, true, success, fmt.Sprintf("Pomodoro %d completion success", index))
		ExpectedActual(t, cases[index].notify, info, fmt.Sprintf("Pomodoro %d NotifyInfo", index))
	}

	for i := range cases {
		// Local variable to prevent data race issues with the onFinish() call below
		idx := i
		created := cpm.CreateIfEmpty(cases[i].duration, func(info NotifyInfo, completed bool) { onFinish(idx, info, completed) }, cases[i].notify)

		ExpectedActual(t, cases[i].shouldSucceed, created, fmt.Sprintf("Expected creation result for case %d", i))
		// If the task was never created, then remove it from our WaitGroup
		if !created {
			wg.Done()
		}
	}

	wg.Wait()
}

func TestPomMapRemove(t *testing.T) {
	cpm := NewChannelPomMap()
	if cpm.channelToPom == nil {
		t.Fatal("Expected non-nil map")
	}

	ExpectedActual(t, 0, cpm.Count(), "initial count")

	failChan := "Doesn't Exist"
	createdChan := "Does Exist"

	ExpectedActual(t, false, cpm.RemoveIfExists(failChan), "removed unknown after initialize")

	createdInfo := NotifyInfo{
		Title:     "Some title here",
		UserID:    "SomeID",
		GuildID:   "SomeGuild",
		ChannelID: createdChan,
	}
	onFinish := func(info NotifyInfo, completed bool) {
		if completed {
			t.Error("Expected cancellation, received successful completion.")
		}
		if info != createdInfo {
			t.Errorf("Expected correct NotifyInfo %v. Actual %v.", createdInfo, info)
		}
	}

	if !cpm.CreateIfEmpty(time.Millisecond*300, onFinish, createdInfo) {
		t.Fatal("Failed to create valid task")
	}

	ExpectedActual(t, 1, cpm.Count(), "one count")
	// Ensure we still don't have this failChan
	ExpectedActual(t, false, cpm.RemoveIfExists(failChan), "removed unknown after another was added")
	// Remove the one that was added
	ExpectedActual(t, true, cpm.RemoveIfExists(createdChan), "removed created")
	// Ensure it was actually removed prior to here
	ExpectedActual(t, false, cpm.RemoveIfExists(createdChan), "create should not exist")
	ExpectedActual(t, 0, cpm.Count(), "emptied count")
}
