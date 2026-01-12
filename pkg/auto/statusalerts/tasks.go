package statusalerts

import (
	"context"
	"errors"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/smart-core-os/sc-bos/pkg/auto/statusalerts/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/alertpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/statuspb"
	"github.com/smart-core-os/sc-bos/pkg/task"
)

func tasksForSource(source config.Source, dest string, statusClient statuspb.StatusApiClient, alertAdminClient alertpb.AlertAdminApiClient, logger *zap.Logger) []task.Task {
	return []task.Task{
		func(ctx context.Context) (task.Next, error) {
			messages := make(chan *statuspb.StatusLog)
			group, ctx := errgroup.WithContext(ctx)
			// fetch data
			group.Go(func() error {
				defer close(messages) // close is "owned" by the send
				return pullFrom(ctx, source.Name, statusClient, messages)
			})
			// process data
			group.Go(func() error {
				return analyseStatusLogs(ctx, source, messages, dest, alertAdminClient, logger)
			})

			err := group.Wait()
			return task.Normal, err
		},
	}
}

type namedTasks struct {
	mu         sync.Mutex
	stopByName map[string]taskRuntime
}

var (
	ErrAlreadyRunning = errors.New("already running")
	ErrNotRunning     = errors.New("not running")
)

func (s *namedTasks) Run(ctx context.Context, name string, tasks []task.Task, opts ...task.Option) error {
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	id := &ctx

	s.mu.Lock()
	if s.stopByName == nil {
		s.stopByName = make(map[string]taskRuntime)
	}

	_, ok := s.stopByName[name]
	if ok {
		s.mu.Unlock()
		return ErrAlreadyRunning
	}
	s.stopByName[name] = taskRuntime{stop, id}
	s.mu.Unlock()

	defer func() {
		// cleanup
		s.mu.Lock()
		defer s.mu.Unlock()
		rt, ok := s.stopByName[name]
		if ok && rt.id == id {
			delete(s.stopByName, name)
		}
	}()

	group, ctx := errgroup.WithContext(ctx)
	for _, t := range tasks {
		t := t
		group.Go(func() error {
			return task.Run(ctx, t, opts...)
		})
	}
	return group.Wait()
}

func (s *namedTasks) Stop(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rt, ok := s.stopByName[name]
	if !ok {
		return ErrNotRunning
	}
	rt.stop()
	delete(s.stopByName, name)
	return nil
}

type taskRuntime struct {
	stop func()
	id   *context.Context
}
