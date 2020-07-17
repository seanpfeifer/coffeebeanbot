package pomodoro

import (
	"sync"
	"testing"
	"time"
)

// timeTolerance is the amount of time difference we will tolerate in a test before failing.
// The required value can vary between machines and CI environments, so this may need tweaking,
// but we should keep an eye on it so it stays somewhat reasonable per test.
const timeTolerance = time.Millisecond * 8

// isDurationTolerable will return true if the two times are within timeTolerance of each other.
func isDurationTolerable(expected, actual time.Duration) bool {
	delta := actual - expected
	return delta <= timeTolerance && delta >= -timeTolerance
}

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
	if !completed {
		t.Error("Expected successful completion, received cancellation.")
	}
	endDuration := time.Since(startTime)

	if !isDurationTolerable(testDuration, endDuration) {
		t.Errorf("Failed to end Pomodoro in time. Expected '%s'. Received '%s'", testDuration, endDuration)
	}
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
	if completed {
		t.Error("Expected cancellation, received successful completion.")
	}
	endDuration := time.Since(startTime)

	if !isDurationTolerable(cancelDuration, endDuration) {
		t.Errorf("Failed to end Pomodoro in time. Expected '%s'. Received '%s'", cancelDuration, endDuration)
	}
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

		if !isDurationTolerable(cases[index].duration, endDuration) {
			t.Errorf("[%d] Failed to end Pomodoro in time. Expected '%s'. Received '%s'",
				index, cases[index].duration, endDuration)
		}

		if !success {
			t.Errorf("[%d] Expected success, received cancellation.", index)
		}

		if info != cases[index].notify {
			t.Errorf("[%d] Expected correct NotifyInfo %v. Actual %v.", index, cases[index].notify, info)
		}
	}

	for i := range cases {
		// Local variable to prevent data race issues with the onFinish() call below
		idx := i
		created := cpm.CreateIfEmpty(cases[i].duration, func(info NotifyInfo, completed bool) { onFinish(idx, info, completed) }, cases[i].notify)

		if created != cases[i].shouldSucceed {
			t.Errorf("Did not receive expected creation result for test case %d: Expected %t. Actual %t.",
				i, cases[i].shouldSucceed, created)
		}
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

	failChan := "Doesn't Exist"
	createdChan := "Does Exist"

	if exists := cpm.RemoveIfExists(failChan); exists {
		t.Errorf("Expected false. Actual true.")
	}

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

	// Ensure we still don't have this failChan
	if exists := cpm.RemoveIfExists(failChan); exists {
		t.Errorf("Expected false. Actual true.")
	}

	// Remove the one that was added
	if exists := cpm.RemoveIfExists(createdChan); !exists {
		t.Error("Expected removed Pomodoro.")
	}

	// Ensure it was removed
	if exists := cpm.RemoveIfExists(createdChan); exists {
		t.Errorf("Expected false. Actual true.")
	}
}
