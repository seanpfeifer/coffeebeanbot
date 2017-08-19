package coffeebeanbot

// Copyright 2017 Sean A. Pfeifer

import (
	"sync"
	"testing"
	"time"
)

func TestPomodoro(t *testing.T) {
	const testDuration = time.Millisecond * 42
	c := make(chan bool)
	testFunc := func() {
		c <- true
	}

	startTime := time.Now()
	// Intentionally not using the returned Pomodoro - no need for it
	NewPomodoro(testDuration, testFunc, NotifyInfo{})
	<-c
	endDuration := time.Since(startTime)

	const tolerance = time.Millisecond * 2
	delta := endDuration - testDuration
	if delta > tolerance {
		t.Errorf("Failed to end Pomodoro in time. Expected '%s'. Received '%s'", testDuration, endDuration)
	}
}

func TestPomodoroCancel(t *testing.T) {
	const testDuration = time.Millisecond * 100
	const cancelDuration = time.Millisecond * 10
	testFunc := func() {
		t.Error("Executed testFunc when Pomodoro should have been cancelled!")
	}

	startTime := time.Now()
	pom := NewPomodoro(testDuration, testFunc, NotifyInfo{})
	go func() {
		time.Sleep(cancelDuration)
		pom.Cancel()
	}()
	// Cancel SHOULD cause this to close, which would stop our timer.
	// Note: At this point we're dealing with the internals of the Pomodoro.
	<-pom.cancelChan
	endDuration := time.Since(startTime)

	const tolerance = time.Millisecond * 2
	delta := endDuration - cancelDuration
	if delta > tolerance {
		t.Errorf("Failed to end Pomodoro in time. Expected '%s'. Received '%s'", cancelDuration, endDuration)
	}
}

func TestPomMapCreate(t *testing.T) {
	cpm := newChannelPomMap()
	if cpm.channelToPom == nil {
		t.Fatal("Expected non-nil map")
	}

	type pomTestCase struct {
		channel       string
		duration      time.Duration
		notify        NotifyInfo
		shouldSucceed bool
	}
	cases := []pomTestCase{
		{"TheChannel", time.Millisecond * 42, NotifyInfo{}, true},
		{"TheChannel", time.Millisecond * 42, NotifyInfo{}, false},
		{"TheChannel2", time.Millisecond * 42, NotifyInfo{}, true},
		{"TheChannel2", time.Millisecond * 42, NotifyInfo{}, false},
	}
	var wg sync.WaitGroup
	wg.Add(len(cases))

	startTime := time.Now()

	onFinish := func(index int) {
		defer wg.Done()

		endDuration := time.Since(startTime)

		const tolerance = time.Millisecond * 2
		delta := endDuration - cases[index].duration
		if delta > tolerance {
			t.Errorf("Failed to end Pomodoro %d in time. Expected '%s'. Received '%s'",
				index, cases[index].duration, endDuration)
		}
	}

	for i := range cases {
		// Local variable to prevent data race issues with the onFinish() call below
		idx := i
		created := cpm.CreateIfEmpty(cases[i].channel, cases[i].duration, func() { onFinish(idx) }, cases[i].notify)

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
	cpm := newChannelPomMap()
	if cpm.channelToPom == nil {
		t.Fatal("Expected non-nil map")
	}

	failChan := "Doesn't Exist"
	createdChan := "Does Exist"

	if info, exists := cpm.RemoveIfExists(failChan); exists {
		t.Errorf("Expected nil NotifyInfo, actual %v", info)
	}

	createdInfo := NotifyInfo{
		Title:   "Some title here",
		UserID:  "SomeID",
		GuildID: "SomeGuild",
	}
	onFinish := func() {
		t.Error("Should not have called onFinish, as this task should be cancelled.")
	}

	if !cpm.CreateIfEmpty(createdChan, time.Millisecond*300, onFinish, createdInfo) {
		t.Fatal("Failed to create valid task")
	}

	// Ensure we still don't have this failChan
	if info, exists := cpm.RemoveIfExists(failChan); exists {
		t.Errorf("Expected nil NotifyInfo, actual %v", info)
	}

	// Remove the one that was added
	if info, exists := cpm.RemoveIfExists(createdChan); exists {
		if createdInfo != info {
			t.Errorf("Did not receive expected NotifyInfo. Expected %v. Actual %v.", createdInfo, info)
		}
	} else {
		t.Errorf("Expected non-nil NotifyInfo")
	}

	// Ensure it was removed
	if info, exists := cpm.RemoveIfExists(createdChan); exists {
		t.Errorf("Expected nil NotifyInfo, actual %v", info)
	}
}
