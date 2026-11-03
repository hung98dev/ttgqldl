-- +goose Up
-- Core village, session and participant state (docs/plan/04_DATABASE.md).
-- Every authoritative row is owned by schema `ttg`; no object is created in
-- `public`. Foreign keys use ON DELETE RESTRICT so that removing authoritative
-- history is only possible through an explicit audited purge.

CREATE SCHEMA IF NOT EXISTS ttg;

CREATE TABLE ttg.villages (
    id UUID PRIMARY KEY,
    broadcaster_key CHAR(64) NOT NULL UNIQUE,
    level SMALLINT NOT NULL DEFAULT 1 CHECK (level BETWEEN 1 AND 50),
    wall_tier SMALLINT NOT NULL DEFAULT 1 CHECK (wall_tier BETWEEN 1 AND 20),
    wall_hp BIGINT NOT NULL DEFAULT 1000 CHECK (wall_hp >= 0),
    townhall_hp BIGINT NOT NULL DEFAULT 500 CHECK (townhall_hp BETWEEN 0 AND 500),
    wood BIGINT NOT NULL DEFAULT 400 CHECK (wood BETWEEN 0 AND 1000000),
    stone BIGINT NOT NULL DEFAULT 250 CHECK (stone BETWEEN 0 AND 1000000),
    food BIGINT NOT NULL DEFAULT 300 CHECK (food BETWEEN 0 AND 1000000),
    construction_work BIGINT NOT NULL DEFAULT 0 CHECK (construction_work >= 0),
    total_wins BIGINT NOT NULL DEFAULT 0 CHECK (total_wins >= 0),
    consecutive_wins SMALLINT NOT NULL DEFAULT 0 CHECK (consecutive_wins BETWEEN 0 AND 10),
    consecutive_losses SMALLINT NOT NULL DEFAULT 0 CHECK (consecutive_losses BETWEEN 0 AND 3),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT villages_wall_hp_within_tier CHECK (wall_hp <= 1000 + 200 * (wall_tier - 1))
);

CREATE TABLE ttg.sessions (
    id UUID PRIMARY KEY,
    village_id UUID NOT NULL REFERENCES ttg.villages (id) ON DELETE RESTRICT,
    mode TEXT NOT NULL CHECK (mode IN ('mock', 'replay', 'authorized_live')),
    lifecycle TEXT NOT NULL CHECK (lifecycle IN
        ('waiting', 'running', 'paused', 'source_disconnected', 'storage_unavailable', 'stopped')),
    wave_no BIGINT NOT NULL DEFAULT 1 CHECK (wave_no >= 1),
    wave_tick INT NOT NULL DEFAULT -1 CHECK (wave_tick BETWEEN -1 AND 3000),
    privacy_epoch BIGINT NOT NULL DEFAULT 0 CHECK (privacy_epoch >= 0),
    rules_version TEXT NOT NULL,
    writer_epoch BIGINT NOT NULL DEFAULT 0,
    input_sequence BIGINT NOT NULL DEFAULT 0,
    stream_sequence BIGINT NOT NULL DEFAULT 0,
    last_reduced_tick INT NOT NULL DEFAULT -1,
    last_state_hash CHAR(64),
    runtime_state JSONB,
    pause_reasons JSONB NOT NULL DEFAULT '[]'::jsonb,
    started_at TIMESTAMPTZ,
    stopped_at TIMESTAMPTZ,
    last_heartbeat_at TIMESTAMPTZ
);

-- At most one non-stopped session per village.
CREATE UNIQUE INDEX sessions_one_active_per_village
    ON ttg.sessions (village_id)
    WHERE lifecycle <> 'stopped';

CREATE TABLE ttg.players (
    id UUID PRIMARY KEY,
    village_id UUID NOT NULL REFERENCES ttg.villages (id) ON DELETE RESTRICT,
    actor_key CHAR(64) NOT NULL,
    ephemeral BOOLEAN NOT NULL,
    alias TEXT CHECK (alias IS NULL OR octet_length(alias) <= 256),
    hide_alias BOOLEAN NOT NULL DEFAULT FALSE,
    lifetime_xp BIGINT NOT NULL DEFAULT 0 CHECK (lifetime_xp >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ,
    CONSTRAINT players_village_actor_key UNIQUE (village_id, actor_key)
);

CREATE TABLE ttg.session_player_progress (
    session_id UUID NOT NULL REFERENCES ttg.sessions (id) ON DELETE RESTRICT,
    actor_key CHAR(64) NOT NULL,
    session_xp SMALLINT NOT NULL DEFAULT 0 CHECK (session_xp BETWEEN 0 AND 300),
    PRIMARY KEY (session_id, actor_key)
);

CREATE TABLE ttg.wave_members (
    session_id UUID NOT NULL REFERENCES ttg.sessions (id) ON DELETE RESTRICT,
    wave_no BIGINT NOT NULL CHECK (wave_no >= 1),
    actor_key CHAR(64) NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('builder', 'guard', 'fighter')),
    joined_tick INT NOT NULL CHECK (joined_tick >= 0),
    last_change_tick INT,
    first_switch_used BOOLEAN NOT NULL DEFAULT FALSE,
    last_update_seq BIGINT,
    xp_awarded BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (session_id, wave_no, actor_key)
);

CREATE TABLE ttg.ballots (
    session_id UUID NOT NULL REFERENCES ttg.sessions (id) ON DELETE RESTRICT,
    wave_no BIGINT NOT NULL CHECK (wave_no >= 1),
    actor_key CHAR(64) NOT NULL,
    option CHAR(1) NOT NULL CHECK (option IN ('A', 'B', 'C')),
    input_seq BIGINT NOT NULL,
    PRIMARY KEY (session_id, wave_no, actor_key)
);

-- +goose Down
DROP TABLE IF EXISTS ttg.ballots;
DROP TABLE IF EXISTS ttg.wave_members;
DROP TABLE IF EXISTS ttg.session_player_progress;
DROP TABLE IF EXISTS ttg.players;
DROP TABLE IF EXISTS ttg.sessions;
DROP TABLE IF EXISTS ttg.villages;
