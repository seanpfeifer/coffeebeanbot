package coffeebeanbot

import (
	"sync"
	"time"
)

// Pomodoro represents a single Pomodoro instance, which can be started and stopped.
// There were a few options I considered when designing this:
// • A sync.Mutex on each Pomodoro, which is locked and unlocked internally to maintain state.
// • Use channels to notify the Pomodoro of events, which handles the state management on its own goroutine
//
// I chose the channel option.  This was to avoid the risks of issues related to locking, as well as
// generally to make it more idiomatic Go.
type Pomodoro struct {
	WorkDuration time.Duration // The duration for a regular Pomodoro work cycle
	OnWorkEnd    func()

	cancelChan chan bool // A channel to interrupt our wait if this Pomodoro is cancelled first
	cancel     sync.Once // To ensure we only close the cancelChan once
}

// NewPomodoro creates a new Pomodoro and starts it, similar to time.NewTimer. "Start" functionality
// is intentionally omitted to prevent double-starting.
// onWorkEnd is called upon normal Pomodoro ending. NOTE: This does not include cancellation.
func NewPomodoro(workDuration time.Duration, onWorkEnd func()) *Pomodoro {
	pom := &Pomodoro{
		workDuration,
		onWorkEnd,
		make(chan bool),
		sync.Once{},
	}

	go pom.performPom()

	return pom
}

// Cancel is used to cancel a current work cycle. This uses "sync.Once" so we prevent a panic if, for whatever
// reason, the caller is able to call Cancel more than once.
func (pom *Pomodoro) Cancel() {
	pom.cancel.Do(func() {
		close(pom.cancelChan)
	})
}

func (pom *Pomodoro) performPom() {
	workTimer := time.NewTimer(pom.WorkDuration)

	select {
	case <-workTimer.C:
		pom.OnWorkEnd()
	case <-pom.cancelChan:
		workTimer.Stop()
	}
}

// channelPomMap is a map-like structure that has goroutine-safe operations to create Pomodoros on individual channels.
type channelPomMap struct {
	sync.Mutex
	channelToPom map[string]*Pomodoro
}

func newChannelPomMap() channelPomMap {
	return channelPomMap{channelToPom: make(map[string]*Pomodoro)}
}

// CreateIfEmpty will create and start a Pomodoro on the given channel if one does not already exist.
// This method is goroutine-safe.
func (m *channelPomMap) CreateIfEmpty(channel string, duration time.Duration, onWorkEnd func()) bool {
	m.Lock()
	defer m.Unlock()

	wasCreated := false
	if _, exists := m.channelToPom[channel]; !exists {
		m.channelToPom[channel] = NewPomodoro(duration, onWorkEnd)
		wasCreated = true
	}

	return wasCreated
}

// RemoveIfExists will stop and remove a Pomodoro from the given channel if one exists.
// This method is goroutine-safe.
func (m *channelPomMap) RemoveIfExists(channel string) bool {
	m.Lock()
	defer m.Unlock()

	wasRemoved := false
	if p, exists := m.channelToPom[channel]; exists {
		p.Cancel()
		delete(m.channelToPom, channel)
		wasRemoved = true
	}

	return wasRemoved
}
