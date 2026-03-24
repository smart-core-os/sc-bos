package cloud

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"testing/synctest"
	"time"
)

// funcClient is a test double for Client that delegates to function fields.
type funcClient struct {
	checkIn         func(ctx context.Context, req CheckInRequest) (CheckInResponse, error)
	downloadPayload func(ctx context.Context, url string) (io.ReadCloser, error)
}

func (f *funcClient) CheckIn(ctx context.Context, req CheckInRequest) (CheckInResponse, error) {
	return f.checkIn(ctx, req)
}

func (f *funcClient) DownloadPayload(ctx context.Context, url string) (io.ReadCloser, error) {
	return f.downloadPayload(ctx, url)
}

func TestAutoPoll(t *testing.T) {
	t.Run("ctx cancellation returns false", func(t *testing.T) {
		storePath := t.TempDir()
		storeDir, err := os.OpenRoot(storePath)
		if err != nil {
			t.Fatalf("open root: %v", err)
		}
		t.Cleanup(func() { _ = storeDir.Close() })

		fc := &funcClient{
			checkIn: func(ctx context.Context, req CheckInRequest) (CheckInResponse, error) {
				return CheckInResponse{}, nil
			},
		}
		updater := NewDeploymentUpdater(storeDir, fc)

		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan bool, 1)
			go func() { done <- updater.AutoPoll(ctx, time.Second) }()
			synctest.Wait()
			cancel()
			synctest.Wait()
			if result := <-done; result != false {
				t.Errorf("want false, got %v", result)
			}
		})
	})

	t.Run("returns true after one interval with new deployment", func(t *testing.T) {
		storePath := t.TempDir()
		storeDir, err := os.OpenRoot(storePath)
		if err != nil {
			t.Fatalf("open root: %v", err)
		}
		t.Cleanup(func() { _ = storeDir.Close() })

		payload := txtarToTarGZ(t, "single.txtar")
		var checkInCalls int
		fc := &funcClient{
			checkIn: func(ctx context.Context, req CheckInRequest) (CheckInResponse, error) {
				checkInCalls++
				if checkInCalls == 1 {
					return CheckInResponse{
						LatestConfig: &LatestConfig{
							Deployment:    Deployment{ID: "1"},
							ConfigVersion: ConfigVersion{ID: "1", PayloadURL: "http://fake.example/payload.tar.gz"},
						},
					}, nil
				}
				return CheckInResponse{}, nil
			},
			downloadPayload: func(ctx context.Context, url string) (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(payload)), nil
			},
		}
		updater := NewDeploymentUpdater(storeDir, fc)

		synctest.Test(t, func(t *testing.T) {
			t0 := time.Now()
			done := make(chan bool, 1)
			go func() { done <- updater.AutoPoll(t.Context(), time.Second) }()
			synctest.Wait()
			time.Sleep(time.Second)
			synctest.Wait()
			if result := <-done; result != true {
				t.Errorf("want true, got %v", result)
			}
			elapsed := time.Since(t0)
			if elapsed != time.Second {
				t.Errorf("elapsed = %v, want %v", elapsed, time.Second)
			}
		})

		// check that the payload is available
		configFS, err := updater.InstallingConfig()
		if err != nil {
			t.Fatalf("InstallingConfig: %v", err)
		}
		if configFS == nil {
			t.Fatal("expected non-nil FS from InstallingConfig after new deployment")
		}
		defer func() { _ = configFS.Close() }()
		assertFileContent(t, configFS, "config.json", `{"key":"value"}`)
	})

	t.Run("continues polling after error", func(t *testing.T) {
		storePath := t.TempDir()
		storeDir, err := os.OpenRoot(storePath)
		if err != nil {
			t.Fatalf("open root: %v", err)
		}
		t.Cleanup(func() { _ = storeDir.Close() })

		var checkInCalls int
		fc := &funcClient{
			checkIn: func(ctx context.Context, req CheckInRequest) (CheckInResponse, error) {
				checkInCalls++
				if checkInCalls == 1 {
					return CheckInResponse{}, errors.New("server unavailable")
				}
				return CheckInResponse{}, nil
			},
		}
		updater := NewDeploymentUpdater(storeDir, fc)

		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan bool, 1)
			go func() { done <- updater.AutoPoll(ctx, time.Second) }()
			synctest.Wait()

			time.Sleep(time.Second) // tick 1: PollOnce errors, loop continues
			synctest.Wait()
			time.Sleep(time.Second) // tick 2: PollOnce returns false/nil, loop continues
			synctest.Wait()

			cancel()
			synctest.Wait()
			if result := <-done; result != false {
				t.Errorf("want false, got %v", result)
			}
		})
	})
}
