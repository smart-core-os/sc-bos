package jwks

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4"

	"github.com/smart-core-os/sc-bos/internal/util/fetch"
)

const minimumUpdateInterval = time.Minute

func NewRemoteKeySet(background context.Context, url string, permittedSignatureAlgorithms []jose.SignatureAlgorithm) *RemoteKeySet {
	return &RemoteKeySet{
		url:                          url,
		background:                   background,
		permittedSignatureAlgorithms: permittedSignatureAlgorithms,
	}
}

// RemoteKeySet handles verification of JSON Web Signatures based on public keys from a remote JWKS URL.
// If a verification of a signature fails because the signing key is unknown, the RemoteKeySet will automatically
// query the remote JWKS url for new keys.
type RemoteKeySet struct {
	url        string
	background context.Context

	m               sync.RWMutex
	cache           jose.JSONWebKeySet
	pending         *keySetFetchJob
	noUpdatesBefore time.Time

	permittedSignatureAlgorithms []jose.SignatureAlgorithm
}

// VerifySignature will check that the provided JWS has a valid signature from a key included in this
// RemoteKeySet. Returns nil if the signature is valid, or a non-nil error otherwise.
// This function may make a network request to refresh the local cache of the remote key set, if the local cache
// cannot verify the token.
// It verifies only the signature - it does not verify any claims in the payload, or inspect the payload in any way!
func (ks *RemoteKeySet) VerifySignature(ctx context.Context, jws string) (payload []byte, err error) {
	sig, err := jose.ParseSigned(jws, ks.permittedSignatureAlgorithms)
	if err != nil {
		return nil, err
	}

	ks.m.RLock()
	keySet := ks.cache
	ks.m.RUnlock()

	payload, err = verifyJWSWithKeySet(sig, keySet)
	if err == nil {
		return payload, nil
	} else if !errors.Is(err, ErrKeyNotFound) {
		// the JWS failed to verify, and it wasn't because of a missing key
		return nil, err
	}

	// The JWS failed to verify because the key we need is not cached; try getting it
	keySet, err = ks.updateKeys(ctx)
	if errors.Is(err, ErrUpdateTooSoon) {
		// We can't update at the moment, so just report that the key wasn't found.
		return nil, ErrKeyNotFound
	} else if err != nil {
		return nil, err
	}

	// try verification again
	return verifyJWSWithKeySet(sig, keySet)
}

// updates the key cache, or waits for the update to complete if one is already in progress.
func (ks *RemoteKeySet) updateKeys(ctx context.Context) (jose.JSONWebKeySet, error) {
	// get the current fetch job, starting one if necessary
	ks.m.Lock()
	if time.Now().Before(ks.noUpdatesBefore) {
		ks.m.Unlock()
		return jose.JSONWebKeySet{}, ErrUpdateTooSoon
	}
	if ks.pending == nil {
		ks.pending = ks.startKeyFetchJob()
	}
	pending := ks.pending
	ks.m.Unlock()

	// wait for the job to complete
	return pending.wait(ctx)
}

func (ks *RemoteKeySet) startKeyFetchJob() *keySetFetchJob {
	done := make(chan struct{})
	job := &keySetFetchJob{done: done}

	go func() {
		var keySet jose.JSONWebKeySet
		err := fetch.JSON(ks.background, ks.url, &keySet)
		job.complete(keySet, err)

		ks.m.Lock()
		defer ks.m.Unlock()
		if err == nil {
			// save results in the cache
			ks.cache = keySet
		}
		// job is no longer pending
		ks.pending = nil
		ks.noUpdatesBefore = time.Now().Add(minimumUpdateInterval)
	}()

	return job
}

type keySetFetchJob struct {
	done   chan struct{}
	result jose.JSONWebKeySet
	err    error
}

func (job *keySetFetchJob) wait(ctx context.Context) (jose.JSONWebKeySet, error) {
	select {
	case <-ctx.Done():
		return jose.JSONWebKeySet{}, ctx.Err()
	case <-job.done:
		return job.result, job.err
	}
}

// must only be called once!
func (job *keySetFetchJob) complete(result jose.JSONWebKeySet, err error) {
	job.result = result
	job.err = err
	close(job.done)
}
