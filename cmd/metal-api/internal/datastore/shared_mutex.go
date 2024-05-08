package datastore

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

type sharedMutexDoc struct {
	ID       string    `rethinkdb:"id"`
	LockedAt time.Time `rethinkdb:"locked_at"`
}

// sharedMutex constructs a mutex using the rethinkdb to guarantee atomic operations.
// this can be helpful because in RethinkDB there are no transactions but sometimes you want
// to prevent concurrency issues over multiple metal-api replicas.
// the performance of this is remarkably worse than running code without this mutex, so
// only make use of this when it really makes sense.
type sharedMutex struct {
	session       r.QueryExecutor
	table         r.Term
	maxblock      time.Duration
	checkinterval time.Duration
	log           *slog.Logger
}

func newSharedMutex(ctx context.Context, log *slog.Logger, session r.QueryExecutor, maxblock time.Duration) *sharedMutex {
	table := r.Table("sharedmutex")

	m := &sharedMutex{
		log:           log,
		session:       session,
		table:         table,
		maxblock:      maxblock,
		checkinterval: 10 * time.Second,
	}

	go m.expireloop(ctx)

	return m
}

type lockOpt interface{}

type lockOptAcquireTimeout struct {
	timeout time.Duration
}

func newLockOptAcquireTimeout(t time.Duration) *lockOptAcquireTimeout {
	return &lockOptAcquireTimeout{timeout: t}
}

func (m *sharedMutex) lock(ctx context.Context, key string, opts ...lockOpt) error {
	for _, opt := range opts {
		switch o := opt.(type) {
		case *lockOptAcquireTimeout:
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, o.timeout)
			defer cancel()
		default:
			return fmt.Errorf("unknown option: %T", opt)
		}
	}

	_, err := m.table.Insert(m.newMutexDoc(key), r.InsertOpts{
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

	timeoutCtx, cancel := context.WithTimeout(ctx, m.maxblock)
	defer cancel()

	m.log.Debug("mutex is already locked, listening for changes", "key", key)

	cursor, err := m.table.Changes(r.ChangesOpts{
		Squash: false,
	}).Run(m.session, r.RunOpts{
		Context: timeoutCtx,
	})
	if err != nil {
		return err
	}

	changes := make(chan r.ChangeResponse)
	go cursor.Listen(changes)

	for {
		select {
		case change := <-changes:
			m.log.Debug("document change received", "key", key)

			if change.NewValue != nil {
				m.log.Debug("mutex was not yet released", "key", key)
				continue
			}

			_, err = m.table.Insert(m.newMutexDoc(key), r.InsertOpts{
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

func (m *sharedMutex) newMutexDoc(key string) *sharedMutexDoc {
	return &sharedMutexDoc{
		ID:       key,
		LockedAt: time.Now(),
	}
}

func (m *sharedMutex) expireloop(ctx context.Context) {
	ticker := time.NewTicker(m.checkinterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.log.Debug("checking for expired mutexes")

			cursor, err := m.table.Run(m.session, r.RunOpts{Context: ctx})
			if err != nil {
				if errors.Is(err, r.ErrConnectionClosed) {
					m.log.Error("connection closed unexpectedly, stop shared mutex expiration loop", "error", err)
				} else {
					m.log.Error("unable to create cursor", "error", err)
				}

				continue
			}

			if cursor.IsNil() {
				continue
			}

			docs := []sharedMutexDoc{}

			err = cursor.All(&docs)
			if err != nil {
				m.log.Error("unable to read shared mutexes", "error", err)
				continue
			}

			m.log.Debug("searched for expiring mutexes in database", "mutex-count", len(docs))

			for _, doc := range docs {
				if time.Since(doc.LockedAt) > m.maxblock {
					_, err = m.table.Get(doc.ID).Delete().RunWrite(m.session, r.RunOpts{Context: ctx})
					if err != nil {
						m.log.Error("unable to release expired shared mutex", "key", doc.ID, "error", err)
						continue
					}

					m.log.Info("cleaned up expired shared mutex", "key", doc.ID)
				}
			}
		case <-ctx.Done():
			m.log.Info("stopped shared mutex expiration loop")
			return
		}
	}
}
