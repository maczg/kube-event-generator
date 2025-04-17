package scheduler

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// testEvent
type testEvent struct {
	index      string        // identifier for ordering
	delay      time.Duration // delay relative to scheduler start time
	actualTime map[string]time.Time
}

func (te *testEvent) ID() string { return te.index }

func (te *testEvent) ExecuteAfterDuration() time.Duration { return te.delay }

func (te *testEvent) ExecuteForDuration() time.Duration { return 0 }

func (te *testEvent) ComparePriority(other Schedulable) bool {
	return te.delay < other.ExecuteAfterDuration()
}

func (te *testEvent) Execute(ctx context.Context) error {
	te.actualTime[te.index] = time.Now()
	logrus.Infof("running event %s", te.index)
	return nil
}

type schedulerTestCase struct {
	name   string
	events []struct {
		id       string
		after    time.Duration
		deadline time.Duration
	}
	testDeadline time.Duration
}

func TestScheduler_Start(t *testing.T) {
	setupLogging()
	tests := []schedulerTestCase{
		{
			name: "basic scheduling",
			events: []struct {
				id       string
				after    time.Duration
				deadline time.Duration
			}{
				{id: "ev0", after: 5 * time.Second, deadline: 0},
				{id: "ev1", after: 10 * time.Second, deadline: 0},
			},
			testDeadline: 20 * time.Second,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := New()
			actualTime := make(map[string]time.Time)
			expected := make(map[string]time.Duration)
			for _, ev := range tc.events {
				expected[ev.id] = ev.after
				event := &testEvent{index: ev.id, delay: ev.after, actualTime: actualTime}
				s.Schedule(event)
			}
			ctx, cancelFn := context.WithCancelCause(context.Background())

			go func() {
				err := s.Start(ctx)
				assert.NoError(t, err, "scheduler should start without error")
			}()
			<-time.After(tc.testDeadline)
			cancelFn(nil)
			verifyExecutionTimes(t, actualTime, expected, s.StartedAt())
		})
	}

}

func setupLogging() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05.0",
	})
}

func verifyExecutionTimes(t *testing.T, actualTime map[string]time.Time, expected map[string]time.Duration, startTime time.Time) {
	for id, expectedTime := range expected {
		assert.Containsf(t, actualTime, id, "event %s should be executed", id)
		actualTimeDiff := actualTime[id].Sub(startTime).Round(time.Second)
		assert.Equalf(t, expectedTime, actualTimeDiff, "event %s should be executed after %v", id, expectedTime)
	}
}
