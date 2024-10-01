package pgdriver

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/makasim/flowstate"
	"github.com/makasim/flowstate/pgdriver/testpgdriver"
	"github.com/stretchr/testify/require"
)

func TestQuery_UpdateState(main *testing.T) {
	openDB := func(t *testing.T, dsn0, dbName string) *pgxpool.Pool {
		conn := testpgdriver.OpenFreshDB(t, dsn0, dbName)

		for i, m := range Migrations {
			_, err := conn.Exec(context.Background(), m.SQL)
			require.NoError(t, err, fmt.Sprintf("Migration #%d (%s) failed ", i, m.Desc))
		}

		return conn
	}

	main.Run("IDEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s := flowstate.State{ID: ``}
		err := q.UpdateState(context.Background(), conn, &s)
		require.EqualError(t, err, `id is empty`)
	})

	main.Run("RevEmpty", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s := flowstate.State{ID: `anID`, Rev: 0}
		err := q.UpdateState(context.Background(), conn, &s)
		require.EqualError(t, err, `rev is empty`)
	})

	main.Run("ConflictNotInserted", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s := flowstate.State{ID: `anID`, Rev: 1}
		err := q.UpdateState(context.Background(), conn, &s)
		require.EqualError(t, err, `no rows in result set`)

		require.Equal(t, []testpgdriver.StateRow(nil), testpgdriver.FindAllStates(t, conn))
		require.Equal(t, []testpgdriver.LatestStateRow(nil), testpgdriver.FindAllLatestStates(t, conn))
	})

	main.Run("ConflictRevMismatch", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s0 := flowstate.State{ID: `anID`}
		err := q.InsertState(context.Background(), conn, &s0)
		require.NoError(t, err)
		require.Greater(t, s0.Rev, int64(0))

		s1 := flowstate.State{ID: `anID`, Rev: 123}
		err = q.UpdateState(context.Background(), conn, &s1)
		require.EqualError(t, err, `no rows in result set`)

		require.Equal(t, []testpgdriver.StateRow{
			{
				ID:     "anID",
				Rev:    1,
				Labels: nil,
				State: flowstate.State{
					ID:  "anID",
					Rev: 0, // rev is not updated in json
				},
			},
		}, testpgdriver.FindAllStates(t, conn))

		require.Equal(t, []testpgdriver.LatestStateRow{
			{
				ID:  "anID",
				Rev: 1,
			},
		}, testpgdriver.FindAllLatestStates(t, conn))
	})

	main.Run("OK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s := flowstate.State{ID: `anID`}
		err := q.InsertState(context.Background(), conn, &s)
		require.NoError(t, err)
		require.Greater(t, s.Rev, int64(0))
		insertRev := s.Rev

		err = q.UpdateState(context.Background(), conn, &s)
		require.NoError(t, err)
		require.Greater(t, s.Rev, insertRev)

		require.Equal(t, []testpgdriver.StateRow{
			{
				ID:     "anID",
				Rev:    2,
				Labels: nil,
				State: flowstate.State{
					ID:  "anID",
					Rev: 1, // rev is not updated in json
				},
			},
			{
				ID:     "anID",
				Rev:    1,
				Labels: nil,
				State: flowstate.State{
					ID:  "anID",
					Rev: 0, // rev is not updated in json
				},
			},
		}, testpgdriver.FindAllStates(t, conn))

		require.Equal(t, []testpgdriver.LatestStateRow{
			{
				ID:  "anID",
				Rev: 2,
			},
		}, testpgdriver.FindAllLatestStates(t, conn))
	})

	main.Run("TxConflictNotInserted", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		q := &queries{}

		s := flowstate.State{ID: `anID`, Rev: 1}
		err = q.UpdateState(context.Background(), tx, &s)
		require.EqualError(t, err, `no rows in result set`)

		require.NoError(t, tx.Rollback(context.Background()))

		require.Equal(t, []testpgdriver.StateRow(nil), testpgdriver.FindAllStates(t, conn))
		require.Equal(t, []testpgdriver.LatestStateRow(nil), testpgdriver.FindAllLatestStates(t, conn))
	})

	main.Run("TxConflictRevMismatch", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s0 := flowstate.State{ID: `anID`}
		err := q.InsertState(context.Background(), conn, &s0)
		require.NoError(t, err)
		require.Greater(t, s0.Rev, int64(0))

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		s1 := flowstate.State{ID: `anID`, Rev: 123}
		err = q.UpdateState(context.Background(), tx, &s1)
		require.EqualError(t, err, `no rows in result set`)

		require.NoError(t, tx.Rollback(context.Background()))

		require.Equal(t, []testpgdriver.StateRow{
			{
				ID:     "anID",
				Rev:    1,
				Labels: nil,
				State: flowstate.State{
					ID:  "anID",
					Rev: 0, // rev is not updated in json
				},
			},
		}, testpgdriver.FindAllStates(t, conn))

		require.Equal(t, []testpgdriver.LatestStateRow{
			{
				ID:  "anID",
				Rev: 1,
			},
		}, testpgdriver.FindAllLatestStates(t, conn))
	})

	main.Run("TxOK", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s := flowstate.State{ID: `anID`}
		err := q.InsertState(context.Background(), conn, &s)
		require.NoError(t, err)
		require.Greater(t, s.Rev, int64(0))
		insertRev := s.Rev

		tx, err := conn.Begin(context.Background())
		require.NoError(t, err)
		defer tx.Rollback(context.Background())

		err = q.UpdateState(context.Background(), tx, &s)
		require.NoError(t, err)
		require.Greater(t, s.Rev, insertRev)

		require.NoError(t, tx.Commit(context.Background()))

		require.Equal(t, []testpgdriver.StateRow{
			{
				ID:     "anID",
				Rev:    2,
				Labels: nil,
				State: flowstate.State{
					ID:  "anID",
					Rev: 1, // rev is not updated in json
				},
			},
			{
				ID:     "anID",
				Rev:    1,
				Labels: nil,
				State: flowstate.State{
					ID:  "anID",
					Rev: 0, // rev is not updated in json
				},
			},
		}, testpgdriver.FindAllStates(t, conn))

		require.Equal(t, []testpgdriver.LatestStateRow{
			{
				ID:  "anID",
				Rev: 2,
			},
		}, testpgdriver.FindAllLatestStates(t, conn))
	})

	main.Run("FullState", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s := flowstate.State{ID: `anID`}
		err := q.InsertState(context.Background(), conn, &s)
		require.NoError(t, err)
		require.Greater(t, s.Rev, int64(0))
		insertRev := s.Rev

		s1 := flowstate.State{
			ID:  s.ID,
			Rev: s.Rev,
			Annotations: map[string]string{
				`fooAnnotKey`: `fooAnnotVal`,
				`barAnnotKey`: `barAnnotVal`,
			},
			Labels: map[string]string{
				`fooLabelKey`: `fooLabelVal`,
				`barLabelKey`: `barLabelVal`,
			},
			CommittedAtUnixMilli: 123,
			Transition: flowstate.Transition{
				FromID: "fromFlowID",
				ToID:   "toFlowID",
				Annotations: map[string]string{
					`fooTsAnnotKey`: `fooTsAnnotVal`,
					`barTsAnnotKey`: `barTsAnnotVal`,
				},
			},
		}

		err = q.UpdateState(context.Background(), conn, &s1)
		require.NoError(t, err)
		require.Greater(t, s1.Rev, insertRev)

		require.Equal(t, []testpgdriver.StateRow{
			{
				ID:  "anID",
				Rev: 2,
				Labels: map[string]string{
					`fooLabelKey`: `fooLabelVal`,
					`barLabelKey`: `barLabelVal`,
				},
				State: flowstate.State{
					ID:  "anID",
					Rev: 1, // rev is not updated in json
					Annotations: map[string]string{
						`fooAnnotKey`: `fooAnnotVal`,
						`barAnnotKey`: `barAnnotVal`,
					},
					Labels: map[string]string{
						`fooLabelKey`: `fooLabelVal`,
						`barLabelKey`: `barLabelVal`,
					},
					CommittedAtUnixMilli: 123,
					Transition: flowstate.Transition{
						FromID: "fromFlowID",
						ToID:   "toFlowID",
						Annotations: map[string]string{
							`fooTsAnnotKey`: `fooTsAnnotVal`,
							`barTsAnnotKey`: `barTsAnnotVal`,
						},
					},
				},
			},
			{
				ID:     "anID",
				Rev:    1,
				Labels: nil,
				State: flowstate.State{
					ID:  "anID",
					Rev: 0, // rev is not updated in json
				},
			},
		}, testpgdriver.FindAllStates(t, conn))

		require.Equal(t, []testpgdriver.LatestStateRow{
			{
				ID:  "anID",
				Rev: 2,
			},
		}, testpgdriver.FindAllLatestStates(t, conn))
	})

	main.Run("Several", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s0 := flowstate.State{ID: `aFooID`}
		err := q.InsertState(context.Background(), conn, &s0)
		require.NoError(t, err)

		s1 := flowstate.State{ID: `aBarID`}
		err = q.InsertState(context.Background(), conn, &s1)
		require.NoError(t, err)

		err = q.UpdateState(context.Background(), conn, &s0)
		require.NoError(t, err)
		require.Greater(t, s0.Rev, int64(1))

		err = q.UpdateState(context.Background(), conn, &s1)
		require.NoError(t, err)
		require.Greater(t, s1.Rev, int64(1))

		require.Equal(t, []testpgdriver.StateRow{
			{
				ID:     "aBarID",
				Rev:    4,
				Labels: nil,
				State: flowstate.State{
					ID:  "aBarID",
					Rev: 2, // rev is not updated in json
				},
			},
			{
				ID:     "aFooID",
				Rev:    3,
				Labels: nil,
				State: flowstate.State{
					ID:  "aFooID",
					Rev: 1, // rev is not updated in json
				},
			},
			{
				ID:     "aBarID",
				Rev:    2,
				Labels: nil,
				State: flowstate.State{
					ID:  "aBarID",
					Rev: 0, // rev is not updated in json
				},
			},
			{
				ID:     "aFooID",
				Rev:    1,
				Labels: nil,
				State: flowstate.State{
					ID:  "aFooID",
					Rev: 0, // rev is not updated in json
				},
			},
		}, testpgdriver.FindAllStates(t, conn))

		require.Equal(t, []testpgdriver.LatestStateRow{
			{
				ID:  "aBarID",
				Rev: 4,
			},
			{
				ID:  "aFooID",
				Rev: 3,
			},
		}, testpgdriver.FindAllLatestStates(t, conn))
	})

	main.Run("Deadlock", func(t *testing.T) {
		conn := openDB(t, `postgres://postgres:postgres@localhost:5432/postgres`, ``)

		q := &queries{}

		s0 := flowstate.State{ID: `0`}
		err := q.InsertState(context.Background(), conn, &s0)
		require.NoError(t, err)

		s1 := flowstate.State{ID: `1`}
		err = q.InsertState(context.Background(), conn, &s1)
		require.NoError(t, err)

		wg := &sync.WaitGroup{}
		doneCh := make(chan struct{})

		go func() {
			time.Sleep(time.Second * 5)
			close(doneCh)
		}()

		ok := atomic.Int64{}
		noRowsErr := atomic.Int64{}

		s0Rev := atomic.Int64{}
		s0Rev.Store(s0.Rev)
		s1Rev := atomic.Int64{}
		s1Rev.Store(s1.Rev)

		testTx := func(s0, s1 *flowstate.State, zeroFirst bool) {
			tx, err := conn.Begin(context.Background())
			require.NoError(t, err)
			defer tx.Rollback(context.Background())

			s0PrevRev := s0.Rev
			s1PrevRev := s1.Rev

			var firstS *flowstate.State
			var secondS *flowstate.State
			if zeroFirst {
				firstS = s0
				secondS = s1
			} else {
				firstS = s1
				secondS = s0
			}

			err = q.UpdateState(context.Background(), tx, firstS)
			if err != nil && err.Error() == "no rows in result set" {
				noRowsErr.Add(1)
				return
			} else if err != nil {
				require.NoError(t, err)
			}

			err = q.UpdateState(context.Background(), tx, secondS)
			if err != nil && err.Error() == "no rows in result set" {
				noRowsErr.Add(1)
				return
			} else if err != nil {
				require.NoError(t, err)
			}

			_, err = tx.Exec(context.Background(), `SELECT pg_sleep(0.1)`)
			require.NoError(t, err)

			require.NoError(t, tx.Commit(context.Background()))

			s0Rev.CompareAndSwap(s0PrevRev, s0.Rev)
			s1Rev.CompareAndSwap(s1PrevRev, s1.Rev)

			ok.Add(1)
		}

		for i := 0; i < 10; i++ {
			wg.Add(1)

			i := i

			go func() {
				defer wg.Done()

				s0 := s0.CopyTo(&flowstate.State{})
				s1 := s1.CopyTo(&flowstate.State{})

				for {
					select {
					case <-doneCh:
						return
					default:
					}

					s0.Rev = s0Rev.Load()
					s1.Rev = s1Rev.Load()

					testTx(&s0, &s1, i%2 == 0)
					testTx(&s0, &s1, i%2 == 0)
				}
			}()
		}

		<-doneCh
		wg.Wait()

		log.Println("ok", ok.Load())
		log.Println("noRowsErr", noRowsErr.Load())
	})
}
