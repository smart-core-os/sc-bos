package connect

import (
	"context"
	"crypto/tls"
	"errors"
	"testing"
)

// fakeCredential reports state as its initial State and delivers whatever is
// sent on changes.
type fakeCredential struct {
	state   State
	changes chan State
}

func (f *fakeCredential) GetClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeCredential) State() State { return f.state }

func (f *fakeCredential) PullState(context.Context) (State, <-chan State) {
	return f.state, f.changes
}

func TestWaitConnected(t *testing.T) {
	t.Run("already connected", func(t *testing.T) {
		cred := &fakeCredential{state: State{Connectivity: Connected}}
		if err := WaitConnected(t.Context(), cred); err != nil {
			t.Errorf("WaitConnected() = %v, want nil", err)
		}
	})

	t.Run("connects later", func(t *testing.T) {
		cred := &fakeCredential{changes: make(chan State)}
		errs := make(chan error, 1)
		go func() { errs <- WaitConnected(t.Context(), cred) }()

		cred.changes <- State{Connectivity: Connecting}
		cred.changes <- State{Connectivity: Connected}
		if err := <-errs; err != nil {
			t.Errorf("WaitConnected() = %v, want nil", err)
		}
	})

	t.Run("ctx cancelled", func(t *testing.T) {
		cred := &fakeCredential{changes: make(chan State)}
		ctx, cancel := context.WithCancel(t.Context())
		errs := make(chan error, 1)
		go func() { errs <- WaitConnected(ctx, cred) }()

		cred.changes <- State{Connectivity: Failed}
		cancel()
		if err := <-errs; !errors.Is(err, context.Canceled) {
			t.Errorf("WaitConnected() = %v, want %v", err, context.Canceled)
		}
	})

	t.Run("channel closed early", func(t *testing.T) {
		cred := &fakeCredential{changes: make(chan State)}
		close(cred.changes)
		if err := WaitConnected(t.Context(), cred); err == nil {
			t.Error("WaitConnected() = nil, want an error")
		}
	})
}
