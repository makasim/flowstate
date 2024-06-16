package sqlitedriver

var createRevTableSQL = `
CREATE TABLE IF NOT EXISTS flowstate_rev (
	rev INTEGER AUTO_INCREMENT PRIMARY KEY
);`

var createStateLatestTableSQL = `
CREATE TABLE IF NOT EXISTS flowstate_state_latest (
    id TEXT,
    rev INTEGER,
    PRIMARY KEY (id)
);`

var createStateLogTableSQL = `
CREATE TABLE IF NOT EXISTS flowstate_state_log (
	rev INTEGER AUTO_INCREMENT PRIMARY KEY,
	id TEXT,
	state JSONB
);`