package sqlitedriver

var createRevTableSQL = `
CREATE TABLE IF NOT EXISTS flowstate_rev (
	rev INTEGER AUTO_INCREMENT,
	PRIMARY KEY (rev)
);`

var createStateLatestTableSQL = `
CREATE TABLE IF NOT EXISTS flowstate_state_latest (
    id TEXT,
    rev INTEGER,
    PRIMARY KEY (id)
);`

var createStateLogTableSQL = `
CREATE TABLE IF NOT EXISTS flowstate_state_log (
	rev INTEGER AUTO_INCREMENT,
	id TEXT,
	state JSONB,
	PRIMARY KEY (rev)
);`

var createDelayLogTableSQL = `
CREATE TABLE IF NOT EXISTS flowstate_delay_log (
	execute_at INTEGER,
    state JSONB
);
CREATE INDEX IF NOT EXISTS flowstate_delay_log_execute_at ON flowstate_delay_log(execute_at);
`

var createDelayMeta = `
CREATE TABLE IF NOT EXISTS flowstate_delay_meta (
    executed_until INTEGER
);`
