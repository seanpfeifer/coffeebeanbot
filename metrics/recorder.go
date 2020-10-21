package metrics

import (
	"context"

	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
)

// Recorder is our backend-independent metrics recorder.
// This should be created with NewRecorder().
type Recorder struct {
	startPomCount   *stats.Int64Measure
	runningPomCount *stats.Int64Measure
	serverCount     *stats.Int64Measure
}

// NewRecorder creates a Recorder with its metrics initialized.
func NewRecorder() (*Recorder, error) {
	recorder := &Recorder{
		startPomCount:   stats.Int64("pomodoros_started", "Count of Pomodoros started", stats.UnitDimensionless),
		runningPomCount: stats.Int64("pomodoros_running", "Current number of Pomodoros running", stats.UnitDimensionless),
		serverCount:     stats.Int64("connected_servers", "Current number of connected servers", stats.UnitDimensionless),
	}

	startView := &view.View{
		Name:        "pomodoros_started_count",
		Measure:     recorder.startPomCount,
		Description: "The number of Pomodoros started",
		Aggregation: view.Count(),
	}

	runningView := &view.View{
		Name:        "pomodoros_running_value",
		Measure:     recorder.runningPomCount,
		Description: "The number of Pomodoros running",
		Aggregation: view.LastValue(),
	}

	serverView := &view.View{
		Name:        "connected_servers_value",
		Measure:     recorder.serverCount,
		Description: "The number of connected servers",
		Aggregation: view.LastValue(),
	}

	return recorder, view.Register(startView, runningView, serverView)
}

// RecordStartPom records the start of a pomodoro.
func (r *Recorder) RecordStartPom() {
	stats.Record(context.Background(), r.startPomCount.M(1))
}

// RecordRunningPoms records the number of currently running pomodoros.
func (r *Recorder) RecordRunningPoms(count int64) {
	stats.Record(context.Background(), r.runningPomCount.M(count))
}

// RecordConnectedServers records the number of currently connected servers (guilds).
func (r *Recorder) RecordConnectedServers(count int64) {
	stats.Record(context.Background(), r.serverCount.M(count))
}
