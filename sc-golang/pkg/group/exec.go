package group

import (
	"context"
	"errors"
	"math"
	"sync"

	"google.golang.org/protobuf/proto"
)

// ExecutionStrategy defines the possible methods for combining a collection of operations into one result.
type ExecutionStrategy int

const (
	ExecutionStrategyUnspecified ExecutionStrategy = iota // Implementation will chose a strategy

	ExecutionStrategyAll  // Execute all, return first error if any fail
	ExecutionStrategyMost // Execute all, return first error if most fail
	ExecutionStrategyAny  // Execute all, return first error if all fail
	ExecutionStrategyOne  // Execute one, if fail, try the next, return first error if all fail
	ExecutionStrategyFast // Execute all, return first to successfully respond or first error if all fail
	ExecutionStrategyRace // Execute all, return first to respond even if it errors
)

// Member defines some action to be taken as part of a group.
type Member func(context.Context) (proto.Message, error)

// Execute runs all members according to the provided strategy.
func Execute(ctx context.Context, strategy ExecutionStrategy, members []Member) ([]proto.Message, error) {
	switch strategy {
	default:
		fallthrough
	case ExecutionStrategyUnspecified, ExecutionStrategyAll:
		return ExecuteAll(ctx, members)
	case ExecutionStrategyMost:
		return ExecuteMost(ctx, members)
	case ExecutionStrategyAny:
		return ExecuteAny(ctx, members)
	case ExecutionStrategyOne:
		res, i, err := ExecuteOne(ctx, members)
		allRes := make([]proto.Message, len(members))
		allRes[i] = res
		return allRes, err
	case ExecutionStrategyFast:
		res, i, err := ExecuteFast(ctx, members)
		allRes := make([]proto.Message, len(members))
		allRes[i] = res
		return allRes, err
	case ExecutionStrategyRace:
		res, i, err := ExecuteRace(ctx, members)
		allRes := make([]proto.Message, len(members))
		allRes[i] = res
		return allRes, err
	}
}

// ExecuteAll executes all the member functions in parallel,
// if any return errors then this will return the first reported error.
// Incomplete executions will have their context cancelled on error,
// but this function will not return until all execution have completed.
func ExecuteAll(ctx context.Context, members []Member) ([]proto.Message, error) {
	return ExecuteUpTo(ctx, 0, members)
}

// ExecuteMost executes all the member functions in parallel,
// if more than half of the members error then this will return the first reported error.
// Incomplete executions will have their context cancelled on error,
// but this function will not return until all execution have completed.
func ExecuteMost(ctx context.Context, members []Member) ([]proto.Message, error) {
	noMoreThanHalf := int(math.Floor(float64(len(members)) / 2))
	return ExecuteUpTo(ctx, noMoreThanHalf, members)
}

// ExecuteAny executes all the member functions in parallel,
// if all the members error then this will return the first reported error.
// Incomplete executions will have their context cancelled on error,
// but this function will not return until all execution have completed.
func ExecuteAny(ctx context.Context, members []Member) ([]proto.Message, error) {
	allBut1 := len(members) - 1
	return ExecuteUpTo(ctx, allBut1, members)
}

// ExecuteUpTo executes all the member functions in parallel,
// if more than allowedErrors members return errors then this will return the first reported error.
// Incomplete executions will have their context cancelled on error,
// but this function will not return until all execution have completed.
func ExecuteUpTo(ctx context.Context, allowedErrors int, members []Member) ([]proto.Message, error) {
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	errCount := 0
	var firstError error
	results := make([]proto.Message, len(members))
	for response := range executeEach(cancelCtx, members) {
		results[response.i] = response.msg
		if response.err != nil {
			if firstError == nil {
				firstError = response.err
			}

			errCount++
			if errCount > allowedErrors {
				cancelFunc()
			}
		}
	}
	if errCount > allowedErrors {
		return results, firstError
	}
	return results, nil
}

// ExecuteOne executes the first member, if it fails, executes the next, etc.
// Returns the first successful response and the member index that succeeded.
// If all fail, the first error recorded will be returned.
func ExecuteOne(ctx context.Context, members []Member) (proto.Message, int, error) {
	var firstErr error
	for i, member := range members {
		res, err := member(ctx)
		if err == nil {
			return res, i, nil
		}
		if i == 0 {
			firstErr = err
		}
	}
	return nil, 0, firstErr
}

// ExecuteFast executes all the member functions in parallel,
// the first non-err response from a member is returned.
// If all members error then the first error response is returned.
func ExecuteFast(ctx context.Context, members []Member) (proto.Message, int, error) {
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	var firstErrResponse *memberResponse
	for response := range executeEach(cancelCtx, members) {
		if response.err == nil {
			// success
			return response.msg, response.i, nil
		}
		if firstErrResponse == nil {
			firstErrResponse = &response
		}
	}

	if firstErrResponse == nil {
		return nil, 0, errors.New("no members returned a response")
	}
	return nil, firstErrResponse.i, firstErrResponse.err
}

// ExecuteRace executes all the member functions in parallel,
// the first response from a member is returned.
func ExecuteRace(ctx context.Context, members []Member) (proto.Message, int, error) {
	cancelCtx, cancelFunc := context.WithCancel(ctx)
	defer cancelFunc()

	for response := range executeEach(cancelCtx, members) {
		return response.msg, response.i, response.err
	}

	return nil, 0, errors.New("no members returned a response")
}

// executeEach runs each of the members in their own goroutine.
// The returned chan will contain the responses in completion order.
// The chan will be closed once all members have returned a result.
func executeEach(ctx context.Context, members []Member) <-chan memberResponse {
	responses := make(chan memberResponse)
	var all sync.WaitGroup
	all.Add(len(members))

	for i, member := range members {
		member := member
		i := i
		go func() {
			defer all.Done()
			result, err := member(ctx)
			responses <- memberResponse{i, result, err}
		}()
	}

	go func() {
		all.Wait()
		close(responses)
	}()

	return responses
}

type memberResponse struct {
	i   int
	msg proto.Message
	err error
}
