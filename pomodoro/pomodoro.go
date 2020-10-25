// Package pomodoro contains functionality for timing work tasks and calling a user-supplied callback on end or cancel.
// This is currently Discord-specific, due to the "NotifyInfo" struct being strongly typed, rather than an interface{}.
// I prefer this for now, until other concrete use cases have been decided upon.
package pomodoro

import (
	"sync"
	"time"
)

// Pomodoro represents a single Pomodoro instance, which can be started and stopped.
// There were a few options I considered when designing this:
//
// • A sync.Mutex on each Pomodoro, which is locked and unlocked internally to maintain state.
//
// • Use channels to notify the Pomodoro of events, which handles the state management on its own goroutine
//
// I chose the channel option.  This was to avoid the risks of issues related to locking if the code is
// expanded upon(user error), as well as generally to make it more idiomatic Go.
type Pomodoro struct {
	workDuration time.Duration // The duration for a regular Pomodoro work cycle
	onWorkEnd    TaskCallback
	notifyInfo   NotifyInfo

	cancelChan chan struct{} // A channel to interrupt our wait if this Pomodoro is cancelled first
	cancel     sync.Once     // To ensure we only close the cancelChan once
}

// TaskCallback is the type of function that will be called upon Pomodoro task completion.  These may be called in a separate
// goroutine, and thus should be made goroutine-safe.
//
// It receives the NotifyInfo and a boolean to tell the receiver whether the task completed (true), or was cancelled (false).
type TaskCallback func(info NotifyInfo, completed bool)

// NotifyInfo contains the necessary information to notify the creating user upon ending the Pomodoro.
type NotifyInfo struct {
	Title     string // The title of the work task
	UserID    string // The UserID to notify
	GuildID   string // The Guild (Discord server) that the user created the Pomodoro on
	ChannelID string // The Channel to notify with the state of the Pomodoro
}

// NewPomodoro creates a new Pomodoro and starts it, similar to time.NewTimer. "Start" functionality
// is intentionally omitted to prevent double-starting.
//
// onWorkEnd is called after the Pomodoro has been completed or cancelled.
func NewPomodoro(workDuration time.Duration, onWorkEnd TaskCallback, notify NotifyInfo) *Pomodoro {
	pom := &Pomodoro{
		workDuration,
		onWorkEnd,
		notify,
		make(chan struct{}),
		sync.Once{},
	}

	go pom.performPom()

	return pom
}

// Cancel is used to cancel a current work cycle. This uses "sync.Once" so we prevent a panic if, for whatever
// reason, the caller is able to call Cancel more than once.
//
// This method is goroutine-safe, and will cancel a Pomodoro only once (multiple calls are OK).
func (pom *Pomodoro) Cancel() {
	pom.cancel.Do(func() {
		close(pom.cancelChan)
	})
}

func (pom *Pomodoro) performPom() {
	workTimer := time.NewTimer(pom.workDuration)

	select {
	case <-workTimer.C:
		go pom.onWorkEnd(pom.notifyInfo, true)
	case <-pom.cancelChan:
		workTimer.Stop()
		go pom.onWorkEnd(pom.notifyInfo, false)
	}
}

// ChannelPomMap is a map-like structure that has goroutine-safe operations to create Pomodoros on individual channels.
type ChannelPomMap struct {
	mutex        sync.Mutex
	channelToPom map[string]*Pomodoro
}

// NewChannelPomMap creates a ChannelPomMap and prepares it to be used.
func NewChannelPomMap() ChannelPomMap {
	return ChannelPomMap{channelToPom: make(map[string]*Pomodoro)}
}

// CreateIfEmpty will create and start a Pomodoro on the given channel if one does not already exist.
// The Pomodoro will be removed from the map when its work is complete, or it is cancelled.
//
// This method is goroutine-safe.
func (m *ChannelPomMap) CreateIfEmpty(duration time.Duration, onWorkEnd TaskCallback, notify NotifyInfo) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	wasCreated := false
	if _, exists := m.channelToPom[notify.ChannelID]; !exists {
		// Ensure we remove the Pomodoro from the map when it completes
		doneInMap := func(notif NotifyInfo, completed bool) {
			// Note that this call is done so we use the mutex. The cancellation will never trigger "onWorkEnd", since the "performPom"
			// goroutine will already be complete by this point.
			m.RemoveIfExists(notif.ChannelID)
			onWorkEnd(notif, completed)
		}

		m.channelToPom[notify.ChannelID] = NewPomodoro(duration, doneInMap, notify)
		wasCreated = true
	}

	return wasCreated
}

// RemoveIfExists will stop and remove a Pomodoro from the given channel if one exists.  Note that this will perform cancellation of
// the Pomodoro if it is running and call the onWorkEnded callback.
//
// It returns a boolean representing whether the Pomodoro was removed.
//
// This method is goroutine-safe.
func (m *ChannelPomMap) RemoveIfExists(channel string) bool {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	wasRemoved := false
	if p, exists := m.channelToPom[channel]; exists {
		delete(m.channelToPom, channel)
		p.Cancel()
		wasRemoved = true
	}

	return wasRemoved
}

// Count returns the number of Pomodoros currently being tracked.
//
// This method is goroutine-safe.
func (m *ChannelPomMap) Count() int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return len(m.channelToPom)
}
