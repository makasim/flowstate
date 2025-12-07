package flowstate

import (
	"log/slog"
	"reflect"
	"testing"

	"github.com/thejerf/slogassert"
)

func TestCacheDriver_appendStateLocked(t *testing.T) {
	f := func(ss []State, expMinRev, expMaxRev int64, expSize int) {
		t.Helper()

		l := slog.New(slogassert.New(t, slog.LevelDebug, nil))
		d := newCacheDriver(nil, 10, l)

		for _, s := range ss {
			d.appendStateLocked(&s)
		}

		if d.minRev != expMinRev {
			t.Fatalf("expected minRev %d, got %d", expMinRev, d.minRev)
		}
		if d.maxRev != expMaxRev {
			t.Fatalf("expected maxRev %d, got %d", expMaxRev, d.maxRev)
		}
		if d.size != expSize {
			t.Fatalf("expected size %d, got %d", expSize, d.size)
		}
	}

	// empty
	f(nil, -1, -1, 0)

	// one
	f([]State{
		{ID: `s1`, Rev: 1},
	}, 1, 1, 1)

	// two no gaps
	f([]State{
		{ID: `s1`, Rev: 1},
		{ID: `s1`, Rev: 2},
	}, 1, 2, 2)

	// two with a gap
	f([]State{
		{ID: `s1`, Rev: 1},
		{ID: `s1`, Rev: 3},
	}, 1, 3, 2)

	// two start from three
	f([]State{
		{ID: `s1`, Rev: 3},
		{ID: `s1`, Rev: 4},
	}, 3, 4, 2)

	// five with a gap
	f([]State{
		{ID: `s1`, Rev: 1},
		{ID: `s2`, Rev: 3},
		{ID: `s3`, Rev: 5},
		{ID: `s4`, Rev: 7},
		{ID: `s5`, Rev: 9},
	}, 1, 9, 5)

	// ten max size
	f([]State{
		{ID: `s1`, Rev: 1},
		{ID: `s2`, Rev: 2},
		{ID: `s3`, Rev: 3},
		{ID: `s4`, Rev: 4},
		{ID: `s5`, Rev: 5},
		{ID: `s6`, Rev: 6},
		{ID: `s7`, Rev: 7},
		{ID: `s8`, Rev: 8},
		{ID: `s9`, Rev: 9},
		{ID: `s10`, Rev: 10},
	}, 1, 10, 10)

	// wrap around one
	f([]State{
		{ID: `s1`, Rev: 1},
		{ID: `s2`, Rev: 2},
		{ID: `s3`, Rev: 3},
		{ID: `s4`, Rev: 4},
		{ID: `s5`, Rev: 5},
		{ID: `s6`, Rev: 6},
		{ID: `s7`, Rev: 7},
		{ID: `s8`, Rev: 8},
		{ID: `s9`, Rev: 9},
		{ID: `s10`, Rev: 10},
		{ID: `s11`, Rev: 11},
	}, 2, 11, 10)

	// wrap around whole size
	f([]State{
		{ID: `s1`, Rev: 1},
		{ID: `s2`, Rev: 2},
		{ID: `s3`, Rev: 3},
		{ID: `s4`, Rev: 4},
		{ID: `s5`, Rev: 5},
		{ID: `s6`, Rev: 6},
		{ID: `s7`, Rev: 7},
		{ID: `s8`, Rev: 8},
		{ID: `s9`, Rev: 9},
		{ID: `s10`, Rev: 10},
		{ID: `s11`, Rev: 11},
		{ID: `s12`, Rev: 12},
		{ID: `s13`, Rev: 13},
		{ID: `s14`, Rev: 14},
		{ID: `s15`, Rev: 15},
		{ID: `s16`, Rev: 16},
		{ID: `s17`, Rev: 17},
		{ID: `s18`, Rev: 18},
		{ID: `s19`, Rev: 19},
		{ID: `s20`, Rev: 20},
	}, 11, 20, 10)

	// wrap around whole size plus one
	f([]State{
		{ID: `s1`, Rev: 1},
		{ID: `s2`, Rev: 2},
		{ID: `s3`, Rev: 3},
		{ID: `s4`, Rev: 4},
		{ID: `s5`, Rev: 5},
		{ID: `s6`, Rev: 6},
		{ID: `s7`, Rev: 7},
		{ID: `s8`, Rev: 8},
		{ID: `s9`, Rev: 9},
		{ID: `s10`, Rev: 10},
		{ID: `s11`, Rev: 11},
		{ID: `s12`, Rev: 12},
		{ID: `s13`, Rev: 13},
		{ID: `s14`, Rev: 14},
		{ID: `s15`, Rev: 15},
		{ID: `s16`, Rev: 16},
		{ID: `s17`, Rev: 17},
		{ID: `s18`, Rev: 18},
		{ID: `s19`, Rev: 19},
		{ID: `s20`, Rev: 20},
		{ID: `s21`, Rev: 21},
	}, 12, 21, 10)
}

func TestCacheDriver_getStateByIDFromLog(t *testing.T) {
	f := func(ss []State, sID StateID, sRev int64, expFound bool, expID StateID, expRev int64) {
		t.Helper()

		cmd := GetStateByID(&StateCtx{}, sID, sRev)

		l := slog.New(slogassert.New(t, slog.LevelDebug, nil))
		d := newCacheDriver(nil, 10, l)

		for _, s := range ss {
			d.appendStateLocked(&s)
		}

		found := d.getStateByIDFromLog(cmd)
		if found != expFound {
			t.Fatalf("expected found %v, got %v", expFound, found)
		}
		if cmd.StateCtx.Current.ID != expID {
			t.Fatalf("expected id %v, got %v", expID, cmd.StateCtx.Current.ID)
		}
		if cmd.StateCtx.Current.Rev != expRev {
			t.Fatalf("expected rev %v, got %v", expRev, cmd.StateCtx.Current.Rev)
		}
	}

	f(nil, `s1`, 1, false, ``, 0)

	f([]State{
		{ID: `s1`, Rev: 1},
	}, `s1`, 1, true, `s1`, 1)

	f([]State{
		{ID: `s1`, Rev: 1},
		{ID: `s1`, Rev: 2},
	}, `s1`, 1, true, `s1`, 1)

	f([]State{
		{ID: `s1`, Rev: 1},
		{ID: `s1`, Rev: 2},
	}, `s1`, 0, true, `s1`, 2)

	ss := []State{
		{ID: `s1`, Rev: 1},
		{ID: `s2`, Rev: 2},
		{ID: `s3`, Rev: 3},
		{ID: `s4`, Rev: 4},
		{ID: `s5`, Rev: 5},
		// log starts here
		{ID: `s1`, Rev: 6},
		{ID: `s2`, Rev: 7},
		{ID: `s3`, Rev: 8},
		{ID: `s4`, Rev: 9},
		{ID: `s1`, Rev: 10},
		{ID: `s2`, Rev: 11},
		{ID: `s3`, Rev: 12},
		{ID: `s4`, Rev: 13},
		{ID: `s5`, Rev: 14},
		{ID: `s6`, Rev: 15},
		// log ends here
	}

	// get old one
	f(ss, `s1`, 5, false, ``, 0)

	// invalid id
	f(ss, `invalidID`, 9, false, ``, 0)

	// invalid rev
	f(ss, `s4`, 20, false, ``, 0)

	// find latest second available
	f(ss, `s4`, 0, true, `s4`, 13)

	// find latest first available
	f(ss, `s6`, 0, true, `s6`, 15)
}

func TestCacheDriver_getStateByLabelsFromLog(t *testing.T) {
	f := func(ss []State, labels map[string]string, expFound bool, expID StateID, expRev int64) {
		t.Helper()

		cmd := GetStateByLabels(&StateCtx{}, labels)

		l := slog.New(slogassert.New(t, slog.LevelDebug, nil))
		d := newCacheDriver(nil, 10, l)

		for _, s := range ss {
			d.appendStateLocked(&s)
		}

		found := d.getStateByLabelsFromLog(cmd)
		if found != expFound {
			t.Fatalf("expected found %v, got %v", expFound, found)
		}
		if cmd.StateCtx.Current.ID != expID {
			t.Fatalf("expected id %v, got %v", expID, cmd.StateCtx.Current.ID)
		}
		if cmd.StateCtx.Current.Rev != expRev {
			t.Fatalf("expected rev %v, got %v", expRev, cmd.StateCtx.Current.Rev)
		}
	}

	f(nil, nil, false, ``, 0)

	f(nil, map[string]string{
		"foo": "fooVal",
	}, false, ``, 0)

	f([]State{
		{ID: `s1`, Rev: 1, Labels: map[string]string{"foo": "fooVal"}},
	}, map[string]string{
		"foo": "fooVal",
	}, true, `s1`, 1)

	f([]State{
		{ID: `s1`, Rev: 1, Labels: map[string]string{"foo": "fooVal"}},
		{ID: `s2`, Rev: 2, Labels: map[string]string{"foo": "fooVal"}},
	}, map[string]string{
		"foo": "fooVal",
	}, true, `s2`, 2)

	f([]State{
		{ID: `s1`, Rev: 1, Labels: map[string]string{"foo": "fooVal"}},
		{ID: `s2`, Rev: 2, Labels: map[string]string{"foo": "fooVal"}},
		{ID: `s3`, Rev: 3, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
		{ID: `s4`, Rev: 4, Labels: map[string]string{"bar": "barVal"}},
	}, map[string]string{
		"foo": "fooVal",
		"bar": "barVal",
	}, true, `s3`, 3)

	f([]State{
		{ID: `s1`, Rev: 1, Labels: map[string]string{"foo": "fooVal"}},
		{ID: `s2`, Rev: 2, Labels: map[string]string{"foo": "fooVal"}},
		{ID: `s3`, Rev: 3, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
		{ID: `s4`, Rev: 4, Labels: map[string]string{"bar": "barVal"}},
	}, map[string]string{
		"bar": "barVal",
	}, true, `s4`, 4)

	f([]State{
		{ID: `s1`, Rev: 1, Labels: map[string]string{"foo": "fooVal"}},
		{ID: `s2`, Rev: 2, Labels: map[string]string{"foo": "fooVal"}},
		{ID: `s3`, Rev: 3, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
		{ID: `s4`, Rev: 4, Labels: map[string]string{"bar": "barVal"}},
	}, nil, true, `s4`, 4)
}

func TestCacheDriver_getStatesFromLog(t *testing.T) {
	f := func(ss []State, cmd *GetStatesCommand, expFound bool, expRes *GetStatesResult) {
		t.Helper()

		l := slog.New(slogassert.New(t, slog.LevelDebug, nil))
		d := newCacheDriver(nil, 10, l)

		for _, s := range ss {
			d.appendStateLocked(&s)
		}

		found := d.getStatesFromLog(cmd)
		if found != expFound {
			t.Fatalf("expected found %v, got %v", expFound, found)
		}
		if !reflect.DeepEqual(expRes, cmd.Result) {
			t.Fatalf("expected result %+v, got %+v", expRes, cmd.Result)
		}
	}

	ss := []State{
		{ID: `s1`, Rev: 6, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
		{ID: `s2`, Rev: 7, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
		{ID: `s3`, Rev: 8, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
		{ID: `s4`, Rev: 9},
		{ID: `s1`, Rev: 10, Labels: map[string]string{"bar": "barVal", "baz": "bazVal"}},
		{ID: `s2`, Rev: 11},
		{ID: `s3`, Rev: 12, Labels: map[string]string{"bar": "barVal", "baz": "bazVal"}},
		{ID: `s4`, Rev: 13},
		{ID: `s5`, Rev: 14, Labels: map[string]string{"foo": "fooVal", "baz": "bazVal"}},
		{ID: `s6`, Rev: 15, Labels: map[string]string{"foo": "fooVal", "baz": "bazVal"}},
	}

	// since rev not set
	f(ss, GetStatesByLabels(nil), false, nil)

	// since rev lower than min rev
	f(ss, GetStatesByLabels(nil).WithSinceRev(5), false, nil)

	// since rev greater than max rev
	f(ss, GetStatesByLabels(nil).WithSinceRev(16), false, nil)

	f(ss, GetStatesByLabels(nil).WithSinceRev(6), true, &GetStatesResult{
		States: ss,
		More:   false,
	})

	f(ss, GetStatesByLabels(nil).WithSinceRev(6).WithLimit(5), true, &GetStatesResult{
		States: []State{
			{ID: `s1`, Rev: 6, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
			{ID: `s2`, Rev: 7, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
			{ID: `s3`, Rev: 8, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
			{ID: `s4`, Rev: 9},
			{ID: `s1`, Rev: 10, Labels: map[string]string{"bar": "barVal", "baz": "bazVal"}},
		},
		More: true,
	})

	f(ss, GetStatesByLabels(nil).WithSinceRev(13), true, &GetStatesResult{
		States: []State{
			{ID: `s4`, Rev: 13},
			{ID: `s5`, Rev: 14, Labels: map[string]string{"foo": "fooVal", "baz": "bazVal"}},
			{ID: `s6`, Rev: 15, Labels: map[string]string{"foo": "fooVal", "baz": "bazVal"}},
		},
		More: false,
	})

	f(ss, GetStatesByLabels(map[string]string{"foo": "fooVal"}).WithSinceRev(6), true, &GetStatesResult{
		States: []State{
			{ID: `s1`, Rev: 6, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
			{ID: `s2`, Rev: 7, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
			{ID: `s3`, Rev: 8, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
			{ID: `s5`, Rev: 14, Labels: map[string]string{"foo": "fooVal", "baz": "bazVal"}},
			{ID: `s6`, Rev: 15, Labels: map[string]string{"foo": "fooVal", "baz": "bazVal"}},
		},
		More: false,
	})

	f(ss, GetStatesByLabels(map[string]string{"foo": "fooVal", "bar": "barVal"}).WithSinceRev(6), true, &GetStatesResult{
		States: []State{
			{ID: `s1`, Rev: 6, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
			{ID: `s2`, Rev: 7, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
			{ID: `s3`, Rev: 8, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
		},
		More: false,
	})

	cmd := GetStatesByLabels(map[string]string{"foo": "fooVal", "bar": "barVal"}).
		WithORLabels(map[string]string{"bar": "barVal", "baz": "bazVal"}).
		WithSinceRev(6)
	f(ss, cmd, true, &GetStatesResult{
		States: []State{
			{ID: `s1`, Rev: 6, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
			{ID: `s2`, Rev: 7, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
			{ID: `s3`, Rev: 8, Labels: map[string]string{"foo": "fooVal", "bar": "barVal"}},
			{ID: `s1`, Rev: 10, Labels: map[string]string{"bar": "barVal", "baz": "bazVal"}},
			{ID: `s3`, Rev: 12, Labels: map[string]string{"bar": "barVal", "baz": "bazVal"}},
		},
		More: false,
	})
}

func TestCacheDriver_commitRevMismatch(t *testing.T) {
	f := func(cmd *CommitCommand, expRevMismatch bool) {
		t.Helper()

		l := slog.New(slogassert.New(t, slog.LevelDebug, nil))
		d := newCacheDriver(nil, 10, l)

		ss := []State{
			{ID: `s1`, Rev: 1},
			{ID: `s2`, Rev: 2},
			{ID: `s3`, Rev: 3},
			{ID: `s4`, Rev: 4},
			{ID: `s5`, Rev: 5},
		}
		for _, s := range ss {
			d.appendStateLocked(&s)
		}

		revMismatch := d.commitRevMismatch(cmd) != nil
		if revMismatch != expRevMismatch {
			t.Fatalf("expected revMismatch %v, got %v", expRevMismatch, revMismatch)
		}
	}

	f(Commit(), false)
	f(Commit(GetData(&StateCtx{}, `foo`)), false)
	f(Commit(
		Park(&StateCtx{Current: State{ID: `otherS`}}),
	), false)
	f(Commit(
		Park(&StateCtx{Current: State{ID: `s1`, Rev: 1}}),
	), false)
	f(Commit(
		Park(&StateCtx{Current: State{ID: `s3`, Rev: 3}}),
		Park(&StateCtx{Current: State{ID: `s4`, Rev: 4}}),
	), false)

	f(Commit(
		Park(&StateCtx{Current: State{ID: `s5`, Rev: 4}}),
	), true)

	f(Commit(
		Park(&StateCtx{Current: State{ID: `s3`, Rev: 3}}),
		Park(&StateCtx{Current: State{ID: `s4`, Rev: 2}}),
	), true)
}

func TestCacheDriver_commitAppendLog(t *testing.T) {
	newD := func() *cacheDriver {
		l := slog.New(slogassert.New(t, slog.LevelDebug, nil))
		return newCacheDriver(nil, 10, l)
	}

	f := func(d *cacheDriver, cmd *CommitCommand, expMaxRev int64) {
		t.Helper()

		ss := []State{
			{ID: `s1`, Rev: 1},
			{ID: `s2`, Rev: 2},
			{ID: `s3`, Rev: 3},
			{ID: `s4`, Rev: 4},
			{ID: `s5`, Rev: 5},
		}
		for _, s := range ss {
			d.appendStateLocked(&s)
		}

		d.commitAppendLog(cmd)

		if d.maxRev != expMaxRev {
			t.Fatalf("expected maxRev %d, got %d", expMaxRev, d.maxRev)
		}
	}

	f(newD(), Commit(), 5)
	f(newD(), Commit(GetData(&StateCtx{}, `foo`)), 5)

	f(newD(), Commit(
		Park(&StateCtx{Current: State{ID: `s6`, Rev: 6}}),
	), 6)

	f(newD(), Commit(
		Park(&StateCtx{Current: State{ID: `s6`, Rev: 6}}),
		Park(&StateCtx{Current: State{ID: `s7`, Rev: 7}}),
	), 7)

	f(newD(), Commit(
		Park(&StateCtx{Current: State{ID: `s7`, Rev: 7}}),
		Park(&StateCtx{Current: State{ID: `s6`, Rev: 6}}),
	), 7)

	f(newD(), Commit(
		Park(&StateCtx{Current: State{ID: `s7`, Rev: 7}}),
	), 5)

	d := newD()
	f(d, Commit(
		Park(&StateCtx{Current: State{ID: `s7`, Rev: 7}}),
	), 5)
	f(d, Commit(
		Park(&StateCtx{Current: State{ID: `s6`, Rev: 6}}),
	), 7)

	d = newD()
	f(d, Commit(
		Park(&StateCtx{Current: State{ID: `s7`, Rev: 7}}),
		Park(&StateCtx{Current: State{ID: `s7`, Rev: 8}}),
		Park(&StateCtx{Current: State{ID: `s7`, Rev: 9}}),
	), 5)
	f(d, Commit(
		Park(&StateCtx{Current: State{ID: `s6`, Rev: 6}}),
	), 9)
}
