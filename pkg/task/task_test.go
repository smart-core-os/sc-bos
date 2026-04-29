package task

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

var errTest = errors.New("test error")

func TestRun(t *testing.T) {
	task := Task(func(ctx context.Context) (next Next, err error) {
		err = errTest
		return
	})

	err := Run(context.Background(), task)

	if err != errTest {
		t.Errorf("got the wrong error: %v", err)
	}
}

func TestWithRetry(t *testing.T) {
	var actual int
	task := Task(func(ctx context.Context) (Next, error) {
		actual++
		return Normal, errTest
	})

	err := Run(context.Background(), task, WithRetry(10))
	if err != errTest {
		t.Errorf("got the wrong error: %v", err)
	}
	if actual != 10 {
		t.Errorf("unexpected retry count %d", actual)
	}
}

func TestWithRetry_StopNow(t *testing.T) {
	var actual int
	task := Task(func(ctx context.Context) (next Next, err error) {
		err = errTest
		actual++
		if actual >= 5 {
			next = StopNow
		}
		return
	})

	err := Run(context.Background(), task, WithRetry(10))
	if err != errTest {
		t.Errorf("got the wrong error: %v", err)
	}
	if actual != 5 {
		t.Errorf("unexpected retry count %d", actual)
	}
}

func TestWithTimeout(t *testing.T) {
	task := Task(func(ctx context.Context) (Next, error) {
		<-ctx.Done()
		return Normal, ctx.Err()
	})

	err := Run(context.Background(), task, WithTimeout(time.Second))

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("unexpected error %v", err)
	}
}

func ExampleRunner_Step() {
	n := 0
	t := Task(func(ctx context.Context) (Next, error) {
		n++
		return Normal, errors.New("an error")
	})

	runner := NewRunner(t, WithRetry(3))
	for {
		_, again, delay := runner.Step(context.Background())
		if !again {
			break
		}
		time.Sleep(delay)
	}
	fmt.Println(n)
	// Output: 3
}
