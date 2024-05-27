package datastore

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

// sharedMutex constructs a mutex using the rethinkdb to guarantee atomic operations.
// this can be helpful because in RethinkDB there are no transactions but sometimes you want
// to prevent concurrency issues over multiple metal-api replicas.
// the performance of this is remarkably worse than running code without this mutex, so
// only make use of this when it really makes sense.
type sharedMutex struct {
	session       r.QueryExecutor
	table         r.Term
	checkinterval time.Duration
	log           *slog.Logger
}

type sharedMutexDoc struct {
	ID        string    `rethinkdb:"id"`
	LockedAt  time.Time `rethinkdb:"locked_at"`
	ExpiresAt time.Time `rethinkdb:"expires_at"`
}

const (
	// defaultSharedMutexCheckInterval is the interval in which it checked whether mutexes have expired. if they have expired, they will be released.
	// this is a safety mechanism in case a mutex was forgotten to be released to prevent the whole machinery to lock up forever.
	defaultSharedMutexCheckInterval = 30 * time.Second
	// defaultSharedMutexAcquireTimeout defines a timeout for the context for the acquisition of the mutex.
	defaultSharedMutexAcquireTimeout = 10 * time.Second
)

type mutexOpt any

type mutexOptCheckInterval struct {
	timeout time.Duration
}

func newMutexOptCheckInterval(t time.Duration) *mutexOptCheckInterval {
	return &mutexOptCheckInterval{timeout: t}
}

func newSharedMutex(ctx context.Context, log *slog.Logger, session r.QueryExecutor, opts ...mutexOpt) (*sharedMutex, error) {
	table := r.Table("sharedmutex")
	timeout := defaultSharedMutexCheckInterval

	for _, opt := range opts {
		switch o := opt.(type) {
		case *mutexOptCheckInterval:
			timeout = o.timeout
		default:
			return nil, fmt.Errorf("unknown option: %T", opt)
		}
	}

	m := &sharedMutex{
		log:           log,
		session:       session,
		table:         table,
		checkinterval: timeout,
	}

	go m.expireloop(ctx)

	return m, nil
}

type lockOpt any

type lockOptAcquireTimeout struct {
	timeout time.Duration
}

func newLockOptAcquireTimeout(t time.Duration) *lockOptAcquireTimeout {
	return &lockOptAcquireTimeout{timeout: t}
}

func (m *sharedMutex) lock(ctx context.Context, key string, expiration time.Duration, opts ...lockOpt) error {
	timeout := defaultSharedMutexAcquireTimeout
	for _, opt := range opts {
		switch o := opt.(type) {
		case *lockOptAcquireTimeout:
			timeout = o.timeout
		default:
			return fmt.Errorf("unknown option: %T", opt)
		}
	}

	_, err := m.table.Insert(m.newMutexDoc(key, expiration), r.InsertOpts{
		Conflict:      "error",
		Durability:    "soft",
		ReturnChanges: "always",
	}).RunWrite(m.session, r.RunOpts{Context: ctx})
	if err == nil {
		m.log.Debug("mutex acquired", "key", key)
		return nil
	}

	if !r.IsConflictErr(err) {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	m.log.Debug("mutex is already locked, listening for changes", "key", key)

	cursor, err := m.table.Get(key).Changes(r.ChangesOpts{
		Squash: false,
	}).Run(m.session, r.RunOpts{
		Context: timeoutCtx,
	})
	if err != nil {
		return err
	}

	changes := make(chan r.ChangeResponse)
	cursor.Listen(changes)

	for {
		select {
		case change := <-changes:
			m.log.Debug("document change received", "key", key)

			if change.NewValue != nil {
				m.log.Debug("mutex was not yet released", "key", key)
				continue
			}

			_, err = m.table.Insert(m.newMutexDoc(key, expiration), r.InsertOpts{
				Conflict:   "error",
				Durability: "soft",
			}).RunWrite(m.session, r.RunOpts{Context: timeoutCtx})
			if err != nil && r.IsConflictErr(err) {
				continue
			}
			if err != nil {
				return err
			}

			m.log.Debug("mutex acquired after waiting", "key", key)

			return nil
		case <-timeoutCtx.Done():
			return fmt.Errorf("unable to acquire mutex: %s", key)
		}
	}
}

func (m *sharedMutex) unlock(ctx context.Context, key string) {
	_, err := m.table.Get(key).Delete().RunWrite(m.session, r.RunOpts{Context: ctx})
	if err != nil {
		m.log.Error("unable to release shared mutex", "key", key, "error", err)
	}
}

func (m *sharedMutex) newMutexDoc(key string, expiration time.Duration) *sharedMutexDoc {
	now := time.Now()
	return &sharedMutexDoc{
		ID:        key,
		LockedAt:  now,
		ExpiresAt: now.Add(expiration),
	}
}

func (m *sharedMutex) expireloop(ctx context.Context) {
	ticker := time.NewTicker(m.checkinterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.log.Debug("checking for expired mutexes")

			resp, err := m.table.Filter(func(row r.Term) r.Term {
				return row.Field("expires_at").Lt(time.Now())
			}).Delete().RunWrite(m.session, r.RunOpts{Context: ctx})
			if err != nil {
				m.log.Error("unable to release shared mutexes", "error", err)
				continue
			}

			m.log.Debug("searched for expiring mutexes in database", "deletion-count", resp.Deleted)
		case <-ctx.Done():
			m.log.Info("stopped shared mutex expiration loop")
			return
		}
	}
}
