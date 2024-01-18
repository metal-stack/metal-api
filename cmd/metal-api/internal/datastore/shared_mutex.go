package datastore

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

type sharedMutexDoc struct {
	ID       string    `rethinkdb:"id"`
	LockedAt time.Time `rethinkdb:"locked_at"`
}

type sharedMutex struct {
	session       r.QueryExecutor
	table         r.Term
	key           string
	maxblock      time.Duration
	checkinterval time.Duration
	log           *zap.SugaredLogger
}

func newSharedMutex(ctx context.Context, log *zap.SugaredLogger, session r.QueryExecutor, key string, maxblock time.Duration) *sharedMutex {
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
		m.log.Debugw("mutex acquired")
		return nil
	}

	if !r.IsConflictErr(err) {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, m.maxblock)
	defer cancel()

	m.log.Debugw("mutex is already locked, listening for changes")

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
			m.log.Debugw("document change received")

			if change.NewValue != nil {
				m.log.Debugw("mutex was not yet released")
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

			m.log.Debugw("mutex acquired after waiting")

			return nil
		case <-timeoutCtx.Done():
			return fmt.Errorf("unable to acquire %q mutex", m.key)
		}
	}
}

func (m *sharedMutex) unlock() {
	_, err := m.table.Get(m.key).Delete().RunWrite(m.session)
	if err != nil {
		m.log.Errorw("unable to release shared mutex")
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
				m.log.Errorw("unable to create cursor", "error", err)
				continue
			}

			if cursor.IsNil() {
				continue
			}

			doc := sharedMutexDoc{}
			err = cursor.One(&doc)
			if err != nil {
				m.log.Errorw("unable to read shared mutex", "error", err)
				continue
			}

			if time.Since(doc.LockedAt) > m.maxblock {
				_, err = m.table.Get(m.key).Delete().RunWrite(m.session)
				if err != nil {
					m.log.Errorw("unable to release expired shared mutex", "error", err)
					continue
				}

				m.log.Infow("cleaned up expired shared mutex")
			}
		case <-ctx.Done():
			return
		}
	}
}
