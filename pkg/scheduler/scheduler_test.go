package scheduler

import (
	"context"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

// testEvent
type testEvent struct {
	index int           // identifier for ordering
	delay time.Duration // delay relative to scheduler start time
}

type testEventWithDeadline struct {
	testEvent
}

func (te *testEventWithDeadline) Run(ctx context.Context) error {
	ctx, _ = context.WithTimeout(ctx, 5*time.Second)
	// Signal that this event has run.
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("[%d] event loop stopped for timeout", te.index)
			return nil
		default:
			logrus.Infof("[%d] event loop running ", te.index)
		}
	}
}

func (te *testEvent) ID() string { return strconv.Itoa(te.index) }
func (te *testEvent) Run(ctx context.Context) error {
	logrus.Infof("running event %d", te.index)
	for {
		select {
		case <-ctx.Done():
			logrus.Infof("[%d] event loop stopped by the scheduler ", te.index)
			return nil
		default:
			logrus.Infof("[%d] event running ", te.index)
			time.Sleep(5 * time.Second)
			logrus.Infof("[%d] event return ", te.index)
			return nil
		}
	}
}

func (te *testEvent) After() time.Duration {
	return te.delay
}

func (te *testEvent) Duration() time.Duration {
	return 0
}

// TestSchedulerStartStop verifies that an event scheduled with zero delay is executed.
func TestEvent(t *testing.T) {
	sch := New()
	evt1 := &testEvent{
		index: 1,
		delay: time.Second * 5,
	}
	evt2 := &testEvent{
		index: 2,
		delay: time.Second * 10,
	}
	evt3Deadline := &testEventWithDeadline{
		testEvent: testEvent{
			index: 3,
			delay: 1 * time.Second,
		},
	}

	sch.Schedule(evt1)
	sch.Schedule(evt2)
	sch.Schedule(evt3Deadline)

	go func() {
		err := sch.Start()
		assert.NoError(t, err)
	}()
	time.Sleep(30 * time.Second)
	t.Logf("Stopping scheduler")
	err := sch.Stop()
	assert.NoError(t, err)

}
