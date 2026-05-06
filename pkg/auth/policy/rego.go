package policy

import (
	"context"
	"embed"
	"encoding/base32"
	"io/fs"
	"strings"
	"sync"

	"github.com/open-policy-agent/opa/v1/ast"
	"github.com/open-policy-agent/opa/v1/rego"
	"github.com/open-policy-agent/opa/v1/storage"
	"github.com/open-policy-agent/opa/v1/storage/inmem"

	"github.com/smart-core-os/sc-bos/pkg/node/alltraits"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

type static struct {
	compiler *ast.Compiler
}

func newStatic(modules map[string]*ast.Module) (*static, error) {
	compiler, err := compile(modules)
	if err != nil {
		return nil, err
	}
	return &static{compiler: compiler}, nil
}

func (p *static) EvalPolicy(ctx context.Context, query string, input Attributes) (rego.ResultSet, error) {
	return rego.New(
		rego.Compiler(p.compiler),
		rego.Input(input),
		rego.Query(query),
		rego.Store(defaultStore),
	).Eval(ctx)
}

type pool struct {
	modules map[string]*ast.Module
	pool    sync.Pool // of *cache
}

func newPool(modules map[string]*ast.Module) (*pool, error) {
	p := &pool{
		modules: modules,
	}
	p.pool.New = p.factory

	// check that the files provided can be used to construct a valid compiler etc.
	// if this succeeds then all future newPoolEntry calls will also succeed because it's deterministic
	//
	// this also pre-fills the pool, decreasing first-request latency
	first, err := newCache(modules)
	if err != nil {
		return nil, err
	}
	p.pool.Put(first)

	return p, nil
}

func (p *pool) EvalPolicy(ctx context.Context, query string, input Attributes) (rego.ResultSet, error) {
	c := p.pool.Get().(*cache)
	defer p.pool.Put(c)
	return c.EvalPolicy(ctx, query, input)
}

// wraps newCache with correct return type
// panics if newCache fails (after pool has been initialised, it should be impossible)
func (p *pool) factory() any {
	c, err := newCache(p.modules)
	if err != nil {
		panic(err)
	}
	return c
}

// cache implements Policy in a non-concurrency-safe way - it must only be used by one goroutine at a time.
// Therefore, it must not be exposed outside the package - use pool to get a thread-safe pool of caches.
type cache struct {
	compiler *ast.Compiler
	cache    map[string]cacheEntry // keyed by query
}

func newCache(modules map[string]*ast.Module) (*cache, error) {
	compiler, err := compile(modules)
	if err != nil {
		return nil, err
	}

	return &cache{
		compiler: compiler,
		cache:    make(map[string]cacheEntry),
	}, nil
}

func (c *cache) EvalPolicy(ctx context.Context, query string, input Attributes) (rego.ResultSet, error) {
	ce := c.getOrCreateCacheEntry(ctx, query)
	if ce.err != nil {
		return nil, ce.err
	}
	r := ce.partialResult.Rego(
		rego.Input(input),
	)

	return r.Eval(ctx)
}

func (c *cache) getOrCreateCacheEntry(ctx context.Context, query string) cacheEntry {
	ce, ok := c.cache[query]
	if ok {
		return ce
	}

	r := rego.New(
		rego.Query(query),
		rego.Compiler(c.compiler),
		rego.Store(defaultStore),
		// Partial evaluation stores some data inside the compiler.
		// To avoid collisions, each partial evaluation must have a different partial namespace.
		// We run partial evaluation once for each query, so calculate namespace from that.
		rego.PartialNamespace(encodeNamespace(query)),
	)
	ce.partialResult, ce.err = r.PartialResult(ctx)
	if ce.err != nil && ctx.Err() != nil {
		// the partial evaluation failed, but it's probably because the context was cancelled
		// don't save in the cache, we might succeed next time.
		return ce
	}
	c.cache[query] = ce
	return ce
}

type cacheEntry struct {
	partialResult rego.PartialResult
	err           error
}

func parseFS(sources fs.FS) (map[string]*ast.Module, error) {
	modules := make(map[string]*ast.Module)
	err := fs.WalkDir(sources, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !strings.HasSuffix(path, ".rego") {
			return nil
		}

		contents, err := fs.ReadFile(sources, path)
		if err != nil {
			return err
		}

		mod, err := ast.ParseModule(path, string(contents))
		if err != nil {
			return err
		}

		modules[path] = mod
		return nil
	})
	if err != nil {
		return nil, err
	}
	return modules, nil
}

// compile modules in a fresh compiler and check errors
func compile(modules map[string]*ast.Module) (*ast.Compiler, error) {
	compiler := ast.NewCompiler()
	compiler.Compile(modules)
	if compiler.Failed() {
		return nil, compiler.Errors
	}
	return compiler, nil
}

var (
	//go:embed default
	defaultPolicyFS embed.FS
	defaultModules  map[string]*ast.Module
	defaultStore    storage.Store
)

func init() {
	mods, err := parseFS(defaultPolicyFS)
	if err != nil {
		// default (bundled) policies are assumed to be error-free
		panic(err)
	}
	defaultModules = mods
	defaultStore = inmem.NewFromObject(map[string]any{
		"system": systemData{
			KnownTraits: builtinTraits(),
		},
	})
}

// builtinTraits derives the list of known traits from the alltraits registry.
func builtinTraits() []knownTrait {
	names := alltraits.Names()
	knownTraits := make([]knownTrait, 0, len(names))
	for _, name := range names {
		serviceDescs := alltraits.ServiceDesc(name)
		serviceNames := make([]string, 0, len(serviceDescs))
		for _, sd := range serviceDescs {
			serviceNames = append(serviceNames, sd.ServiceName)
		}
		knownTraits = append(knownTraits, knownTrait{
			Name:         name,
			GRPCServices: serviceNames,
		})
	}
	return knownTraits
}

func Default(cached bool) Policy {
	var p Policy
	var err error
	if cached {
		p, err = newPool(defaultModules)
	} else {
		p, err = newStatic(defaultModules)
	}
	if err != nil {
		// default (bundled) modules are assumed to be error-free
		panic(err)
	}
	return p
}

func FromFS(f fs.FS, opts ...FsOpt) (Policy, error) {
	cfg := fsOpts{cached: true}
	for _, opt := range opts {
		opt(&cfg)
	}

	mods, err := parseFS(f)
	if err != nil {
		return nil, err
	}

	if cfg.cached {
		return newPool(mods)
	}
	return newStatic(mods)
}

// FsOpt configures FromFS.
type FsOpt func(*fsOpts)

type fsOpts struct {
	cached bool
}

// WithCached enables or disables caching compiled rego modules. Caching is enabled by default.
func WithCached(cached bool) FsOpt {
	return func(o *fsOpts) {
		o.cached = cached
	}
}

type systemData struct {
	KnownTraits []knownTrait `json:"known_traits"`
}

type knownTrait struct {
	Name         trait.Name `json:"name"`
	GRPCServices []string   `json:"grpc_services"`
}

// encodes an arbitrary string such that it's valid as a Rego namespace component (alphanumeric + underscore)
func encodeNamespace(str string) string {
	enc := base32.StdEncoding.WithPadding('_')
	return enc.EncodeToString([]byte(str))
}
