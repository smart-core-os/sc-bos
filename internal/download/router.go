package download

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"
)

// contextKey is the unexported type used to attach the verified download
// payload to a request's context. The single payloadContextKey instance is
// the only value of this type, so it cannot collide with keys from other
// packages.
type contextKey int

const payloadContextKey contextKey = 0

// PayloadFromContext returns the verified download payload associated with
// ctx, or nil if ctx is not derived from a download.Router request. Handlers
// registered via Router.Handle / Router.HandleFunc read their bytes through
// this accessor instead of receiving them as a method parameter.
func PayloadFromContext(ctx context.Context) []byte {
	b, _ := ctx.Value(payloadContextKey).([]byte)
	return b
}

// ContextWithPayload returns a copy of ctx carrying payload as the value
// retrievable by PayloadFromContext. Useful in tests that exercise a
// handler's ServeHTTP directly without driving traffic through Router.
func ContextWithPayload(ctx context.Context, payload []byte) context.Context {
	return context.WithValue(ctx, payloadContextKey, payload)
}

// Option configures a Router.
type Option func(*Router)

// WithBaseURL configures the URL prefix the router is mounted at. Both
// root-relative ("/download") and fully-qualified
// ("https://example.com/download") forms are accepted. The path portion is
// used to strip incoming request paths; the full string is used as the prefix
// when forming URLs in Router.GenerateURL. The trailing "/" is normalised —
// "/download" and "/download/" behave the same.
func WithBaseURL(base string) Option {
	return func(rt *Router) {
		rt.setBaseURL(base)
	}
}

// WithTTL configures how long URLs returned by Router.GenerateURL remain
// valid. The default is 5 minutes. A zero or negative value (re)sets the
// TTL to the default value.
func WithTTL(ttl time.Duration) Option {
	return func(rt *Router) {
		if ttl > 0 {
			rt.ttl = ttl
		} else {
			rt.ttl = defaultTTL
		}
	}
}

const defaultTTL = 5 * time.Minute

// Router signs download URLs, verifies them on serve, and dispatches to
// handlers registered by type. It is itself an http.Handler.
type Router struct {
	signer Signer
	ttl    time.Duration

	// base is the configured base URL, normalised so base.Path always ends
	// in "/" (and is at least "/"). Its String() form (which may include
	// scheme + host) is used when building outbound URLs in GenerateURL;
	// its Path is used as the prefix to strip from incoming request paths
	// in ServeHTTP.
	base url.URL

	mu       sync.RWMutex
	handlers map[string]http.Handler
}

// NewRouter builds a Router that uses signer to sign and verify tokens.
// Typically signer is NewHMACSigner(...).
func NewRouter(signer Signer, opts ...Option) *Router {
	rt := &Router{
		signer:   signer,
		ttl:      defaultTTL,
		handlers: make(map[string]http.Handler),
	}
	for _, opt := range opts {
		opt(rt)
	}
	return rt
}

func (rt *Router) setBaseURL(base string) {
	u, err := url.Parse(base)
	if err != nil {
		// Fall back to treating the whole string as a path prefix.
		u = &url.URL{Path: base}
	}
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}
	rt.base = *u
}

// Handle registers h under typ, replacing any previous handler for the same
// type. Panics if typ is empty or h is nil. Type strings are part of the wire
// format and should be stable identifiers (e.g. "devices-csv", "log-file")
// so URLs can be recognised later. The verified payload bytes are placed on
// the request context; the handler reads them via PayloadFromContext.
func (rt *Router) Handle(typ string, h http.Handler) {
	if typ == "" {
		panic("download: empty type")
	}
	if h == nil {
		panic("download: nil handler")
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.handlers[typ] = h
}

// HandleFunc registers the function f as the handler for typ.
func (rt *Router) HandleFunc(typ string, f func(http.ResponseWriter, *http.Request)) {
	rt.Handle(typ, http.HandlerFunc(f))
}

// GenerateURL produces the URL a client should use to download, along with
// the absolute time after which the URL ceases to be valid. The URL is the
// configured base URL with the opaque token appended as the final path
// segment; with no WithBaseURL it is root-relative ("/<token>"). The expiry
// is time.Now() + the configured TTL (see WithTTL).
func (rt *Router) GenerateURL(typ string, payload []byte) (downloadURL string, expiresAt time.Time, err error) {
	expiresAt = time.Now().Add(rt.ttl)
	token, err := signToken(rt.signer, typ, payload, expiresAt)
	if err != nil {
		return "", time.Time{}, err
	}
	target := rt.base
	target.Path = path.Join(rt.base.Path, token)
	return target.String(), expiresAt, nil
}

// ServeHTTP extracts the token from r.URL.Path, verifies it, and dispatches
// to the registered handler. It strips the path portion of the configured
// base URL from r.URL.Path itself — mount with mux.Handle, no StripPrefix
// wrapper needed.
func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token, ok := strings.CutPrefix(r.URL.Path, rt.base.Path)
	if !ok || token == "" {
		http.NotFound(w, r)
		return
	}

	env, err := verifyToken(rt.signer, token)
	if err != nil {
		switch {
		case errors.Is(err, errTokenFormat),
			errors.Is(err, errInvalidSignature),
			errors.Is(err, errEnvelope),
			errors.Is(err, errExpired):
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
		default:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	rt.mu.RLock()
	h, ok := rt.handlers[env.GetType()]
	rt.mu.RUnlock()
	if !ok {
		// Do not disclose validity to unregistered types.
		http.NotFound(w, r)
		return
	}
	h.ServeHTTP(w, r.WithContext(ContextWithPayload(r.Context(), env.GetPayload())))
}
