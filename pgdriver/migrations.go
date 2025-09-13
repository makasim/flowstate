package pgdriver

type Migration struct {
	Desc string
	SQL  string
}

var Migrations = []Migration{
	{
		Desc: "create flowstate_latest_states table",
		SQL: `
	CREATE TABLE IF NOT EXISTS flowstate_latest_states (
	   id TEXT NOT NULL,
	   rev bigint  NOT NULL,
	   PRIMARY KEY (id)
	);`,
	},
	{
		Desc: "create flowstate_states table",
		SQL: `
CREATE SEQUENCE IF NOT EXISTS flowstate_states_rev_seq;

CREATE TABLE IF NOT EXISTS flowstate_states (
	rev bigint  NOT NULL,
	id TEXT  NOT NULL,
	state JSONB  NOT NULL,
	labels JSONB,
	PRIMARY KEY (rev, id)
);

CREATE INDEX flowstate_states_id_idx ON flowstate_states(id);
CREATE INDEX flowstate_states_labels_idx ON flowstate_states USING GIN ("labels") WHERE labels IS NOT NULL;
`,
	},
	{
		Desc: "create flowstate_delayed_states table",
		SQL: `
CREATE TABLE IF NOT EXISTS flowstate_delayed_states (
	execute_at bigint NOT NULL,
	state JSONB  NOT NULL
);
CREATE INDEX IF NOT EXISTS flowstate_delayed_states_execute_at ON flowstate_delayed_states(execute_at);
`,
	},
	{
		Desc: "create flowstate_data",
		SQL: `
CREATE SEQUENCE IF NOT EXISTS flowstate_data_rev_seq;

CREATE TABLE IF NOT EXISTS flowstate_data (
	rev bigint NOT NULL,
	annotations JSONB,
	b TEXT,
	PRIMARY KEY (rev)
);`,
	},
	{
		Desc: "add state id to flowstate_delayed_states table, recreate index",
		SQL: `
CREATE SEQUENCE IF NOT EXISTS flowstate_delayed_states_pos;

ALTER TABLE flowstate_delayed_states ADD COLUMN pos BIGINT NOT NULL;

UPDATE flowstate_delayed_states SET pos = nextval('flowstate_delayed_states_pos');

DROP INDEX IF EXISTS flowstate_delayed_states_execute_at;

CREATE UNIQUE INDEX IF NOT EXISTS flowstate_delayed_states_execute_at_pos ON flowstate_delayed_states(execute_at,pos);
`,
	},
}
