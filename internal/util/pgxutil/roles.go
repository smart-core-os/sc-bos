package pgxutil

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RoleConfig configures per-privilege postgres connections. The embedded
// ConnectConfig is the default for any role not explicitly overridden, so a
// config that only sets uri/passwordFile behaves exactly as a single-pool
// ConnectConfig did (all three roles share one connection).
//
// The Read/Write/Admin overrides let operators connect each role as a distinct
// database user so that, for example, read queries run over a connection that
// only holds SELECT permissions.
type RoleConfig struct {
	ConnectConfig                // default for all roles
	Read          *ConnectConfig `json:"read,omitempty"`
	Write         *ConnectConfig `json:"write,omitempty"`
	Admin         *ConnectConfig `json:"admin,omitempty"`
}

// IsZero reports whether no connection is configured at all.
func (rc RoleConfig) IsZero() bool {
	return rc.ConnectConfig.IsZero() && rc.Read == nil && rc.Write == nil && rc.Admin == nil
}

// resolve returns the effective ConnectConfig for each role, falling back to the
// embedded base config for any role without an explicit override.
func (rc RoleConfig) resolve() (read, write, admin ConnectConfig) {
	read, write, admin = rc.ConnectConfig, rc.ConnectConfig, rc.ConnectConfig
	if rc.Read != nil {
		read = *rc.Read
	}
	if rc.Write != nil {
		write = *rc.Write
	}
	if rc.Admin != nil {
		admin = *rc.Admin
	}
	return read, write, admin
}

// Pools holds the read, write, and admin connection pools for a postgres store.
// Read is used for SELECT-only queries, Write for statements (and transactions)
// that modify data, and Admin for schema DDL and maintenance (VACUUM).
// Two or more of the pools may share the same underlying *pgxpool.Pool when
// their configs are identical.
type Pools struct {
	Read  *pgxpool.Pool
	Write *pgxpool.Pool
	Admin *pgxpool.Pool
}

// SamePool returns a Pools that uses p for all three roles. Use it when only a
// single connection is available (e.g. tools and tests).
func SamePool(p *pgxpool.Pool) Pools {
	return Pools{Read: p, Write: p, Admin: p}
}

// Close closes each distinct underlying pool exactly once, so a Pools built from
// a shared config isn't double-closed.
func (p Pools) Close() {
	seen := make(map[*pgxpool.Pool]struct{}, 3)
	for _, pool := range []*pgxpool.Pool{p.Read, p.Write, p.Admin} {
		if pool == nil {
			continue
		}
		if _, ok := seen[pool]; ok {
			continue
		}
		seen[pool] = struct{}{}
		pool.Close()
	}
}

// ConnectRoles opens the read, write, and admin pools described by rc. Roles
// whose effective config is identical share a single pool, so a config with only
// a base uri opens just one connection pool. If any pool fails to connect, the
// pools already opened are closed and the error is returned.
func ConnectRoles(ctx context.Context, rc RoleConfig) (Pools, error) {
	read, write, admin := rc.resolve()

	cache := make(map[ConnectConfig]*pgxpool.Pool, 3)
	var opened []*pgxpool.Pool

	fail := func(err error) (Pools, error) {
		for _, pool := range opened {
			pool.Close()
		}
		return Pools{}, err
	}

	get := func(cc ConnectConfig) (*pgxpool.Pool, error) {
		if pool, ok := cache[cc]; ok {
			return pool, nil
		}
		pool, err := Connect(ctx, cc)
		if err != nil {
			return nil, err
		}
		cache[cc] = pool
		opened = append(opened, pool)
		return pool, nil
	}

	readPool, err := get(read)
	if err != nil {
		return fail(err)
	}
	writePool, err := get(write)
	if err != nil {
		return fail(err)
	}
	adminPool, err := get(admin)
	if err != nil {
		return fail(err)
	}
	return Pools{Read: readPool, Write: writePool, Admin: adminPool}, nil
}
