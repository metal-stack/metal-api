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
	key           string
	maxblock      time.Duration
	checkinterval time.Duration
	log           *slog.Logger
}

func newSharedMutex(ctx context.Context, log *slog.Logger, session r.QueryExecutor, key string, maxblock time.Duration) *sharedMutex {
	table := r.Table("sharedmutex")

	m := &sharedMutex{
		log:           log.With("key", key),
		session:       session,
		key:           key,
		table:         table,
		maxblock:      maxblock,
		checkinterval: 10 * time.Second,
	}

	go m.expireloop(ctx)

	return m
}

func (m *sharedMutex) lock(ctx context.Context) error {
	_, err := m.table.Insert(m.newMutexDoc(), r.InsertOpts{
		Conflict:      "error",
		Durability:    "soft",
		ReturnChanges: "always",
	}).RunWrite(m.session)
	if err == nil {
		m.log.Debug("mutex acquired")
		return nil
	}

	if !r.IsConflictErr(err) {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, m.maxblock)
	defer cancel()

	m.log.Debug("mutex is already locked, listening for changes")

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
			m.log.Debug("document change received")

			if change.NewValue != nil {
				m.log.Debug("mutex was not yet released")
				continue
			}

			_, err = m.table.Insert(m.newMutexDoc(), r.InsertOpts{
				Conflict:   "error",
				Durability: "soft",
			}).RunWrite(m.session)
			if err != nil && r.IsConflictErr(err) {
				continue
			}
			if err != nil {
				return err
			}

			m.log.Debug("mutex acquired after waiting")

			return nil
		case <-timeoutCtx.Done():
			return fmt.Errorf("unable to acquire %q mutex", m.key)
		}
	}
}

func (m *sharedMutex) unlock() {
	_, err := m.table.Get(m.key).Delete().RunWrite(m.session)
	if err != nil {
		m.log.Error("unable to release shared mutex", "error", err)
	}
}

func (m *sharedMutex) newMutexDoc() *sharedMutexDoc {
	return &sharedMutexDoc{
		ID:       m.key,
		LockedAt: time.Now(),
	}
}

func (m *sharedMutex) expireloop(ctx context.Context) {
	ticker := time.NewTicker(m.checkinterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cursor, err := m.table.Get(m.key).Run(m.session)
			if err != nil {
				if errors.Is(err, r.ErrConnectionClosed) {
					m.log.Error("connection closed unexpectedly, stop shared mutex expiration loop", "error", err)
				}

				m.log.Error("unable to create cursor", "error", err)

				continue
			}

			if cursor.IsNil() {
				continue
			}

			doc := sharedMutexDoc{}
			err = cursor.One(&doc)
			if err != nil {
				m.log.Error("unable to read shared mutex", "error", err)
				continue
			}

			if time.Since(doc.LockedAt) > m.maxblock {
				_, err = m.table.Get(m.key).Delete().RunWrite(m.session)
				if err != nil {
					m.log.Error("unable to release expired shared mutex", "error", err)
					continue
				}

				m.log.Info("cleaned up expired shared mutex")
			}
		case <-ctx.Done():
			m.log.Info("stopped shared mutex expiration loop")
			return
		}
	}
}
