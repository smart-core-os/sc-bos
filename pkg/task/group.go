package task

import (
	"context"
	"fmt"
	"maps"
	"sync"
	"time"
)

type Group struct {
	m     sync.Mutex
	tasks map[string]*GroupTask
}

func (g *Group) Spawn(ctx context.Context, tag string, task Task, options ...Option) *GroupTask {
	g.m.Lock()
	defer g.m.Unlock()

	if g.tasks == nil {
		g.tasks = make(map[string]*GroupTask)
	}

	if _, present := g.tasks[tag]; present {
		// TODO: autogenerate an alternative name
		panic("tag already present")
	}
	gt := spawnGroupTask(tag, ctx, NewRunner(task, options...))
	g.tasks[tag] = gt
	return gt
}

func (g *Group) Tasks() map[string]*GroupTask {
	g.m.Lock()
	defer g.m.Unlock()

	clonedTasks := make(map[string]*GroupTask, len(g.tasks))
	maps.Copy(clonedTasks, g.tasks)
	return clonedTasks
}

type State int

const (
	StatePending State = iota + 1 // Task is waiting to start.
	StateRunning                  // Task is running now.
	StateDelay                    // Task is in its delay period before the next attempt.
	StateStopped                  // Task is not running, and won't run again.
)

func (s State) String() string {
	switch s {
	case StatePending:
		return "Pending"
	case StateRunning:
		return "Running"
	case StateDelay:
		return "Delay"
	case StateStopped:
		return "StateStopped"
	default:
		return fmt.Sprintf("State(%d)", int(s))
	}
}

type GroupTask struct {
	tag    string
	runner *Runner
	cancel context.CancelFunc
	done   chan struct{}

	m     sync.Mutex
	state State
	err   error
}

func (gt *GroupTask) Tag() string {
	return gt.tag
}

func (gt *GroupTask) Cancel() {
	gt.cancel()
}

func (gt *GroupTask) State() (state State, err error) {
	gt.m.Lock()
	defer gt.m.Unlock()
	return gt.state, gt.err
}

func (gt *GroupTask) Wait(ctx context.Context) (done bool) {
	select {
	case <-ctx.Done():
		return false
	case <-gt.done:
		return true
	}
}

func spawnGroupTask(tag string, ctx context.Context, runner *Runner) *GroupTask {
	ctx, cancel := context.WithCancel(ctx)
	gt := &GroupTask{
		tag:    tag,
		runner: runner,
		cancel: cancel,
		done:   make(chan struct{}),
		state:  StatePending,
	}

	go func() {
		defer close(gt.done)

		var err error
		for {
			gt.m.Lock()
			gt.state = StateRunning
			gt.m.Unlock()

			var (
				again bool
				delay time.Duration
			)
			err, again, delay = runner.Step(ctx)
			if !again {
				gt.m.Lock()
				gt.err = err
				gt.state = StateStopped
				gt.m.Unlock()
				return
			}

			gt.m.Lock()
			gt.state = StateDelay
			gt.m.Unlock()

			select {
			case <-ctx.Done():
				gt.m.Lock()
				gt.err = ctx.Err()
				gt.state = StateStopped
				gt.m.Unlock()
				return
			case <-time.After(delay):
			}
		}
	}()

	return gt
}
