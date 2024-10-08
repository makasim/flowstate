package testpgdriver

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/makasim/flowstate"
	"github.com/stretchr/testify/require"
	"github.com/xo/dburl"
)

type StateRow struct {
	ID     flowstate.StateID
	Rev    int64
	Labels map[string]string
	State  flowstate.State
}

func FindAllStates(t *testing.T, conn *pgx.Conn) []StateRow {
	rows, err := conn.Query(context.Background(), `SELECT rev, id, state, labels FROM flowstate_states ORDER BY rev DESC`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var scannedRows []StateRow
	for rows.Next() {
		r := StateRow{}
		require.NoError(t, rows.Scan(&r.Rev, &r.ID, &r.State, &r.Labels))
		scannedRows = append(scannedRows, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	return scannedRows
}

type LatestStateRow struct {
	ID  flowstate.StateID
	Rev int64
}

func FindAllLatestStates(t *testing.T, conn *pgx.Conn) []LatestStateRow {
	rows, err := conn.Query(context.Background(), `SELECT rev, id FROM flowstate_latest_states ORDER BY rev DESC`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var scannedRows []LatestStateRow
	for rows.Next() {
		r := LatestStateRow{}
		require.NoError(t, rows.Scan(&r.Rev, &r.ID))
		scannedRows = append(scannedRows, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	return scannedRows
}

func OpenFreshDB(t *testing.T, dsn0, dbName string) *pgx.Conn {
	dsn, err := dburl.Parse(dsn0)
	require.NoError(t, err)

	conn0, err := pgx.Connect(context.Background(), dsn.String())
	require.NoError(t, err)
	defer conn0.Close(context.Background())

	if dbName == `` {
		dbName = fmt.Sprintf(`flowstate_testdb_%d`, time.Now().UnixNano())
	}

	_, err = conn0.Exec(context.Background(), fmt.Sprintf(`CREATE DATABASE %s`, dbName))
	require.NoError(t, err)

	dsn.Path = dbName
	conn, err := pgx.Connect(context.Background(), dsn.String())
	require.NoError(t, err)

	t.Cleanup(func() {
		conn.Close(context.Background())
	})

	return conn
}

type DataRow struct {
	ID          flowstate.DataID
	Rev         int64
	Annotations map[string]string
	B           []byte
}

func FindAllData(t *testing.T, conn *pgx.Conn) []DataRow {
	rows, err := conn.Query(context.Background(), `SELECT rev, id, annotations, bytes FROM flowstate_data ORDER BY rev DESC`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var scannedRows []DataRow
	for rows.Next() {
		r := DataRow{}
		require.NoError(t, rows.Scan(&r.Rev, &r.ID, &r.Annotations, &r.B))
		scannedRows = append(scannedRows, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	return scannedRows
}

type DelayedStateRow struct {
	ExecuteAt int64
	State     flowstate.State
}

func FindAllDelayedStates(t *testing.T, conn *pgx.Conn) []DelayedStateRow {
	rows, err := conn.Query(context.Background(), `SELECT execute_at, state FROM flowstate_delayed_states ORDER BY execute_at DESC`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var scannedRows []DelayedStateRow
	for rows.Next() {
		r := DelayedStateRow{}
		require.NoError(t, rows.Scan(&r.ExecuteAt, &r.State))
		scannedRows = append(scannedRows, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	return scannedRows
}

type DelayMetaRow struct {
	Shard int
	Meta  string
}

func FindAllDelayMeta(t *testing.T, conn *pgx.Conn) []DelayMetaRow {
	rows, err := conn.Query(context.Background(), `SELECT shard, meta FROM flowstate_delayer_meta ORDER BY shard ASC`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var scannedRows []DelayMetaRow
	for rows.Next() {
		r := DelayMetaRow{}
		require.NoError(t, rows.Scan(&r.Shard, &r.Meta))
		scannedRows = append(scannedRows, r)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	return scannedRows
}
