-- GENERATED FILE - do not edit by hand.
-- Snapshot of the schema produced by backend/db/migrations, used only by sqlc:
-- the goose files carry down migrations and markers that sqlc cannot parse.
-- Regenerate with tooling/schema-dump.ps1 (PowerShell) or schema-dump.sh after
-- changing a migration; CI fails if the snapshot drifts from the migrations.

--
-- PostgreSQL database dump
--

\restrict eedgZmBoMbuFx8TLufjyTrl2JgaM5q3TYgukOL8F8QGRbWhStp2doWk8FkJxfgp

-- Dumped from database version 18.6 (Debian 18.6-1.pgdg13+2)
-- Dumped by pg_dump version 18.6 (Debian 18.6-1.pgdg13+2)


--
-- Name: ttg; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA ttg;




--
-- Name: actor_identity_aliases; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.actor_identity_aliases (
    village_id uuid NOT NULL,
    actor_key character(64) NOT NULL,
    key_version text NOT NULL,
    player_id uuid NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT actor_identity_aliases_actor_key_is_hmac_hex CHECK ((actor_key ~ '^[0-9a-f]{64}$'::text)),
    CONSTRAINT actor_identity_aliases_key_version_shape CHECK ((key_version ~ '^[A-Za-z0-9._:-]{1,32}$'::text))
);


--
-- Name: admin_idempotency_keys; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.admin_idempotency_keys (
    account_id uuid NOT NULL,
    route text NOT NULL,
    key text NOT NULL,
    request_hash text NOT NULL,
    response_status integer NOT NULL,
    response_json jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL
);


--
-- Name: ballots; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.ballots (
    session_id uuid NOT NULL,
    wave_no bigint NOT NULL,
    actor_key character(64) NOT NULL,
    option character(1) NOT NULL,
    input_seq bigint NOT NULL,
    CONSTRAINT ballots_option_check CHECK ((option = ANY (ARRAY['A'::bpchar, 'B'::bpchar, 'C'::bpchar]))),
    CONSTRAINT ballots_wave_no_check CHECK ((wave_no >= 1))
);


--
-- Name: broadcaster_source_bindings; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.broadcaster_source_bindings (
    village_id uuid NOT NULL,
    provider text NOT NULL,
    broadcaster_key character(64) NOT NULL,
    approval_ref text,
    permission_expires_at timestamp with time zone
);


--
-- Name: checkpoints; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.checkpoints (
    session_id uuid NOT NULL,
    revision bigint NOT NULL,
    wave_no bigint NOT NULL,
    wave_tick integer NOT NULL,
    input_seq bigint NOT NULL,
    state_json jsonb NOT NULL,
    state_hash character(64) NOT NULL,
    rules_version text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    privacy_epoch bigint DEFAULT 0 NOT NULL
);


--
-- Name: domain_events; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.domain_events (
    session_id uuid NOT NULL,
    revision bigint NOT NULL,
    event_index integer NOT NULL,
    event_type text NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: input_events; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.input_events (
    session_id uuid NOT NULL,
    input_seq bigint NOT NULL,
    accepted_tick integer NOT NULL,
    scheduled_tick integer NOT NULL,
    applied_at_tick integer,
    wave_no bigint NOT NULL,
    actor_key character(64) NOT NULL,
    kind text NOT NULL,
    canonical_payload jsonb NOT NULL,
    rules_version text NOT NULL,
    inserted_at timestamp with time zone DEFAULT now() NOT NULL,
    accepted_wave bigint,
    CONSTRAINT input_events_kind_check CHECK ((kind = ANY (ARRAY['role'::text, 'vote'::text]))),
    CONSTRAINT input_events_wave_no_check CHECK ((wave_no >= 1))
);


--
-- Name: input_receipts; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.input_receipts (
    session_id uuid NOT NULL,
    provider_source text NOT NULL,
    provider_event_id text NOT NULL,
    inserted_at timestamp with time zone DEFAULT now() NOT NULL,
    payload_fingerprint character(64),
    input_seq bigint,
    CONSTRAINT input_receipts_fingerprint_shape CHECK (((payload_fingerprint IS NULL) OR (payload_fingerprint ~ '^[0-9a-f]{64}$'::text))),
    CONSTRAINT input_receipts_input_seq_positive CHECK (((input_seq IS NULL) OR (input_seq > 0)))
);


--
-- Name: operator_accounts; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.operator_accounts (
    id uuid NOT NULL,
    username text NOT NULL,
    password_hash text NOT NULL,
    role text NOT NULL,
    disabled_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT operator_accounts_role_check CHECK ((role = ANY (ARRAY['admin'::text, 'operator'::text, 'viewer'::text])))
);


--
-- Name: operator_audit; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.operator_audit (
    id bigint NOT NULL,
    actor_id uuid,
    action text NOT NULL,
    target text,
    reason text,
    ip_hash text,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: operator_audit_id_seq; Type: SEQUENCE; Schema: ttg; Owner: -
--

ALTER TABLE ttg.operator_audit ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME ttg.operator_audit_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: operator_sessions; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.operator_sessions (
    id uuid NOT NULL,
    account_id uuid NOT NULL,
    secret_hash character(64) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    revoked_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: players; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.players (
    id uuid NOT NULL,
    village_id uuid NOT NULL,
    actor_key character(64) NOT NULL,
    ephemeral boolean NOT NULL,
    alias text,
    hide_alias boolean DEFAULT false NOT NULL,
    lifetime_xp bigint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_seen_at timestamp with time zone,
    CONSTRAINT players_alias_check CHECK (((alias IS NULL) OR (octet_length(alias) <= 256))),
    CONSTRAINT players_lifetime_xp_check CHECK ((lifetime_xp >= 0))
);


--
-- Name: privacy_erase_requests; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.privacy_erase_requests (
    id uuid NOT NULL,
    session_id uuid NOT NULL,
    actor_ref_protected text NOT NULL,
    reason text NOT NULL,
    status text NOT NULL,
    requested_at timestamp with time zone DEFAULT now() NOT NULL,
    applied_at timestamp with time zone,
    privacy_epoch bigint NOT NULL,
    requested_by uuid,
    scope text DEFAULT 'session'::text NOT NULL,
    tombstone_id uuid,
    parent_id uuid,
    CONSTRAINT privacy_erase_requests_status_check CHECK ((status = ANY (ARRAY['queued'::text, 'applied'::text, 'blocked'::text, 'failed'::text])))
);


--
-- Name: privacy_ledger_outbox; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.privacy_ledger_outbox (
    id uuid NOT NULL,
    seq bigint NOT NULL,
    tombstone_id uuid NOT NULL,
    payload jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    exported_at timestamp with time zone,
    ledger_ack_ref text,
    attempts integer DEFAULT 0 NOT NULL,
    last_error text,
    CONSTRAINT privacy_ledger_outbox_ack_consistency CHECK (((exported_at IS NULL) = (ledger_ack_ref IS NULL))),
    CONSTRAINT privacy_ledger_outbox_ack_ref_length CHECK (((ledger_ack_ref IS NULL) OR ((length(ledger_ack_ref) >= 1) AND (length(ledger_ack_ref) <= 128)))),
    CONSTRAINT privacy_ledger_outbox_attempts_non_negative CHECK ((attempts >= 0)),
    CONSTRAINT privacy_ledger_outbox_payload_is_object CHECK ((jsonb_typeof(payload) = 'object'::text))
);


--
-- Name: privacy_ledger_outbox_seq_seq; Type: SEQUENCE; Schema: ttg; Owner: -
--

ALTER TABLE ttg.privacy_ledger_outbox ALTER COLUMN seq ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME ttg.privacy_ledger_outbox_seq_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: privacy_restore_erasures; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.privacy_restore_erasures (
    run_id uuid NOT NULL,
    tombstone_id uuid NOT NULL,
    session_id uuid NOT NULL,
    actor_ref_protected text NOT NULL,
    applied_revision bigint NOT NULL,
    applied_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: privacy_restore_runs; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.privacy_restore_runs (
    id uuid NOT NULL,
    ledger_id text NOT NULL,
    ledger_last_seq bigint NOT NULL,
    ledger_head_hash character(64) NOT NULL,
    target_identity text NOT NULL,
    applied_at timestamp with time zone DEFAULT now() NOT NULL,
    result_hash character(64) NOT NULL
);


--
-- Name: privacy_scope_requests; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.privacy_scope_requests (
    id uuid NOT NULL,
    village_id uuid NOT NULL,
    actor_ref_protected text NOT NULL,
    scope text NOT NULL,
    status text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    completed_at timestamp with time zone,
    CONSTRAINT privacy_scope_requests_scope_check CHECK ((scope = 'village'::text)),
    CONSTRAINT privacy_scope_requests_status_check CHECK ((status = ANY (ARRAY['queued'::text, 'applied'::text, 'blocked'::text, 'failed'::text])))
);


--
-- Name: privacy_tombstones; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.privacy_tombstones (
    id uuid NOT NULL,
    actor_ref_protected text NOT NULL,
    scope text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    restore_applied_at timestamp with time zone
);


--
-- Name: reducer_transitions; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.reducer_transitions (
    session_id uuid NOT NULL,
    revision bigint NOT NULL,
    operation text NOT NULL,
    wave_no bigint NOT NULL,
    wave_tick integer NOT NULL,
    state_hash character(64) NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    input_seq_high bigint DEFAULT 0 NOT NULL,
    CONSTRAINT reducer_transitions_operation_check CHECK ((operation = ANY (ARRAY['start'::text, 'advance'::text, 'rollover'::text, 'privacy'::text]))),
    CONSTRAINT reducer_transitions_revision_check CHECK ((revision > 0))
);


--
-- Name: session_player_progress; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.session_player_progress (
    session_id uuid NOT NULL,
    actor_key character(64) NOT NULL,
    session_xp smallint DEFAULT 0 NOT NULL,
    CONSTRAINT session_player_progress_session_xp_check CHECK (((session_xp >= 0) AND (session_xp <= 300)))
);


--
-- Name: sessions; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.sessions (
    id uuid NOT NULL,
    village_id uuid NOT NULL,
    mode text NOT NULL,
    lifecycle text NOT NULL,
    wave_no bigint DEFAULT 1 NOT NULL,
    wave_tick integer DEFAULT '-1'::integer NOT NULL,
    privacy_epoch bigint DEFAULT 0 NOT NULL,
    rules_version text NOT NULL,
    writer_epoch bigint DEFAULT 0 NOT NULL,
    input_sequence bigint DEFAULT 0 NOT NULL,
    stream_sequence bigint DEFAULT 0 NOT NULL,
    last_reduced_tick integer DEFAULT '-1'::integer NOT NULL,
    last_state_hash character(64),
    runtime_state jsonb,
    pause_reasons jsonb DEFAULT '[]'::jsonb NOT NULL,
    started_at timestamp with time zone,
    stopped_at timestamp with time zone,
    last_heartbeat_at timestamp with time zone,
    reducer_revision bigint DEFAULT 0 NOT NULL,
    CONSTRAINT sessions_lifecycle_check CHECK ((lifecycle = ANY (ARRAY['waiting'::text, 'running'::text, 'paused'::text, 'source_disconnected'::text, 'storage_unavailable'::text, 'stopped'::text]))),
    CONSTRAINT sessions_mode_check CHECK ((mode = ANY (ARRAY['mock'::text, 'replay'::text, 'authorized_live'::text]))),
    CONSTRAINT sessions_privacy_epoch_check CHECK ((privacy_epoch >= 0)),
    CONSTRAINT sessions_reducer_revision_check CHECK ((reducer_revision >= 0)),
    CONSTRAINT sessions_wave_no_check CHECK ((wave_no >= 1)),
    CONSTRAINT sessions_wave_tick_check CHECK (((wave_tick >= '-1'::integer) AND (wave_tick <= 3000)))
);


--
-- Name: stream_outbox; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.stream_outbox (
    session_id uuid NOT NULL,
    stream_seq bigint NOT NULL,
    revision bigint NOT NULL,
    type text NOT NULL,
    payload jsonb NOT NULL,
    checkpoint_ref bigint,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: villages; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.villages (
    id uuid NOT NULL,
    broadcaster_key character(64) NOT NULL,
    level smallint DEFAULT 1 NOT NULL,
    wall_tier smallint DEFAULT 1 NOT NULL,
    wall_hp bigint DEFAULT 1000 NOT NULL,
    townhall_hp bigint DEFAULT 500 NOT NULL,
    wood bigint DEFAULT 400 NOT NULL,
    stone bigint DEFAULT 250 NOT NULL,
    food bigint DEFAULT 300 NOT NULL,
    construction_work bigint DEFAULT 0 NOT NULL,
    total_wins bigint DEFAULT 0 NOT NULL,
    consecutive_wins smallint DEFAULT 0 NOT NULL,
    consecutive_losses smallint DEFAULT 0 NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT villages_consecutive_losses_check CHECK (((consecutive_losses >= 0) AND (consecutive_losses <= 3))),
    CONSTRAINT villages_consecutive_wins_check CHECK (((consecutive_wins >= 0) AND (consecutive_wins <= 10))),
    CONSTRAINT villages_construction_work_check CHECK ((construction_work >= 0)),
    CONSTRAINT villages_food_check CHECK (((food >= 0) AND (food <= 1000000))),
    CONSTRAINT villages_level_check CHECK (((level >= 1) AND (level <= 50))),
    CONSTRAINT villages_stone_check CHECK (((stone >= 0) AND (stone <= 1000000))),
    CONSTRAINT villages_total_wins_check CHECK ((total_wins >= 0)),
    CONSTRAINT villages_townhall_hp_check CHECK (((townhall_hp >= 0) AND (townhall_hp <= 500))),
    CONSTRAINT villages_wall_hp_check CHECK ((wall_hp >= 0)),
    CONSTRAINT villages_wall_hp_within_tier CHECK ((wall_hp <= (1000 + (200 * (wall_tier - 1))))),
    CONSTRAINT villages_wall_tier_check CHECK (((wall_tier >= 1) AND (wall_tier <= 20))),
    CONSTRAINT villages_wood_check CHECK (((wood >= 0) AND (wood <= 1000000)))
);


--
-- Name: wave_members; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.wave_members (
    session_id uuid NOT NULL,
    wave_no bigint NOT NULL,
    actor_key character(64) NOT NULL,
    role text NOT NULL,
    joined_tick integer NOT NULL,
    last_change_tick integer,
    first_switch_used boolean DEFAULT false NOT NULL,
    last_update_seq bigint,
    xp_awarded boolean DEFAULT false NOT NULL,
    CONSTRAINT wave_members_joined_tick_check CHECK ((joined_tick >= 0)),
    CONSTRAINT wave_members_role_check CHECK ((role = ANY (ARRAY['builder'::text, 'guard'::text, 'fighter'::text]))),
    CONSTRAINT wave_members_wave_no_check CHECK ((wave_no >= 1))
);


--
-- Name: writer_leases; Type: TABLE; Schema: ttg; Owner: -
--

CREATE TABLE ttg.writer_leases (
    village_id uuid NOT NULL,
    writer_epoch bigint DEFAULT 0 NOT NULL,
    owner_instance text,
    leased_until timestamp with time zone
);


--
-- Name: actor_identity_aliases actor_identity_aliases_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.actor_identity_aliases
    ADD CONSTRAINT actor_identity_aliases_pkey PRIMARY KEY (village_id, actor_key, key_version);


--
-- Name: admin_idempotency_keys admin_idempotency_keys_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.admin_idempotency_keys
    ADD CONSTRAINT admin_idempotency_keys_pkey PRIMARY KEY (account_id, route, key);


--
-- Name: ballots ballots_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.ballots
    ADD CONSTRAINT ballots_pkey PRIMARY KEY (session_id, wave_no, actor_key);


--
-- Name: broadcaster_source_bindings broadcaster_source_bindings_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.broadcaster_source_bindings
    ADD CONSTRAINT broadcaster_source_bindings_pkey PRIMARY KEY (village_id, provider);


--
-- Name: checkpoints checkpoints_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.checkpoints
    ADD CONSTRAINT checkpoints_pkey PRIMARY KEY (session_id, revision);


--
-- Name: domain_events domain_events_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.domain_events
    ADD CONSTRAINT domain_events_pkey PRIMARY KEY (session_id, revision, event_index);


--
-- Name: input_events input_events_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.input_events
    ADD CONSTRAINT input_events_pkey PRIMARY KEY (session_id, input_seq);


--
-- Name: input_receipts input_receipts_unique_event; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.input_receipts
    ADD CONSTRAINT input_receipts_unique_event UNIQUE (session_id, provider_source, provider_event_id);


--
-- Name: operator_accounts operator_accounts_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.operator_accounts
    ADD CONSTRAINT operator_accounts_pkey PRIMARY KEY (id);


--
-- Name: operator_audit operator_audit_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.operator_audit
    ADD CONSTRAINT operator_audit_pkey PRIMARY KEY (id);


--
-- Name: operator_sessions operator_sessions_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.operator_sessions
    ADD CONSTRAINT operator_sessions_pkey PRIMARY KEY (id);


--
-- Name: players players_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.players
    ADD CONSTRAINT players_pkey PRIMARY KEY (id);


--
-- Name: players players_village_actor_key; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.players
    ADD CONSTRAINT players_village_actor_key UNIQUE (village_id, actor_key);


--
-- Name: players players_village_id_id_key; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.players
    ADD CONSTRAINT players_village_id_id_key UNIQUE (village_id, id);


--
-- Name: privacy_erase_requests privacy_erase_requests_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_erase_requests
    ADD CONSTRAINT privacy_erase_requests_pkey PRIMARY KEY (id);


--
-- Name: privacy_ledger_outbox privacy_ledger_outbox_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_ledger_outbox
    ADD CONSTRAINT privacy_ledger_outbox_pkey PRIMARY KEY (id);


--
-- Name: privacy_ledger_outbox privacy_ledger_outbox_seq_key; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_ledger_outbox
    ADD CONSTRAINT privacy_ledger_outbox_seq_key UNIQUE (seq);


--
-- Name: privacy_restore_erasures privacy_restore_erasures_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_restore_erasures
    ADD CONSTRAINT privacy_restore_erasures_pkey PRIMARY KEY (run_id, tombstone_id, session_id);


--
-- Name: privacy_restore_runs privacy_restore_runs_ledger_id_ledger_last_seq_target_ident_key; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_restore_runs
    ADD CONSTRAINT privacy_restore_runs_ledger_id_ledger_last_seq_target_ident_key UNIQUE (ledger_id, ledger_last_seq, target_identity);


--
-- Name: privacy_restore_runs privacy_restore_runs_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_restore_runs
    ADD CONSTRAINT privacy_restore_runs_pkey PRIMARY KEY (id);


--
-- Name: privacy_scope_requests privacy_scope_requests_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_scope_requests
    ADD CONSTRAINT privacy_scope_requests_pkey PRIMARY KEY (id);


--
-- Name: privacy_scope_requests privacy_scope_requests_village_id_actor_ref_protected_scope_key; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_scope_requests
    ADD CONSTRAINT privacy_scope_requests_village_id_actor_ref_protected_scope_key UNIQUE (village_id, actor_ref_protected, scope);


--
-- Name: privacy_tombstones privacy_tombstones_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_tombstones
    ADD CONSTRAINT privacy_tombstones_pkey PRIMARY KEY (id);


--
-- Name: reducer_transitions reducer_transitions_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.reducer_transitions
    ADD CONSTRAINT reducer_transitions_pkey PRIMARY KEY (session_id, revision);


--
-- Name: session_player_progress session_player_progress_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.session_player_progress
    ADD CONSTRAINT session_player_progress_pkey PRIMARY KEY (session_id, actor_key);


--
-- Name: sessions sessions_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.sessions
    ADD CONSTRAINT sessions_pkey PRIMARY KEY (id);


--
-- Name: stream_outbox stream_outbox_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.stream_outbox
    ADD CONSTRAINT stream_outbox_pkey PRIMARY KEY (session_id, stream_seq);


--
-- Name: villages villages_broadcaster_key_key; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.villages
    ADD CONSTRAINT villages_broadcaster_key_key UNIQUE (broadcaster_key);


--
-- Name: villages villages_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.villages
    ADD CONSTRAINT villages_pkey PRIMARY KEY (id);


--
-- Name: wave_members wave_members_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.wave_members
    ADD CONSTRAINT wave_members_pkey PRIMARY KEY (session_id, wave_no, actor_key);


--
-- Name: writer_leases writer_leases_pkey; Type: CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.writer_leases
    ADD CONSTRAINT writer_leases_pkey PRIMARY KEY (village_id);


--
-- Name: actor_identity_aliases_player; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX actor_identity_aliases_player ON ttg.actor_identity_aliases USING btree (player_id);


--
-- Name: admin_idempotency_keys_expires_at; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX admin_idempotency_keys_expires_at ON ttg.admin_idempotency_keys USING btree (expires_at);


--
-- Name: input_events_pending; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX input_events_pending ON ttg.input_events USING btree (session_id, wave_no, scheduled_tick, input_seq) WHERE (applied_at_tick IS NULL);


--
-- Name: input_events_replay_boundary; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX input_events_replay_boundary ON ttg.input_events USING btree (session_id, accepted_wave, accepted_tick, input_seq);


--
-- Name: input_receipts_inserted_at; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX input_receipts_inserted_at ON ttg.input_receipts USING btree (inserted_at);


--
-- Name: input_receipts_unassociated; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX input_receipts_unassociated ON ttg.input_receipts USING btree (session_id) WHERE (input_seq IS NULL);


--
-- Name: operator_accounts_username_lower; Type: INDEX; Schema: ttg; Owner: -
--

CREATE UNIQUE INDEX operator_accounts_username_lower ON ttg.operator_accounts USING btree (lower(username));


--
-- Name: operator_audit_created_at; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX operator_audit_created_at ON ttg.operator_audit USING btree (created_at);


--
-- Name: operator_sessions_account; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX operator_sessions_account ON ttg.operator_sessions USING btree (account_id);


--
-- Name: privacy_erase_requests_idempotency; Type: INDEX; Schema: ttg; Owner: -
--

CREATE UNIQUE INDEX privacy_erase_requests_idempotency ON ttg.privacy_erase_requests USING btree (session_id, actor_ref_protected, scope);


--
-- Name: privacy_erase_requests_parent; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX privacy_erase_requests_parent ON ttg.privacy_erase_requests USING btree (parent_id);


--
-- Name: privacy_erase_requests_session; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX privacy_erase_requests_session ON ttg.privacy_erase_requests USING btree (session_id);


--
-- Name: privacy_erase_requests_tombstone_unique; Type: INDEX; Schema: ttg; Owner: -
--

CREATE UNIQUE INDEX privacy_erase_requests_tombstone_unique ON ttg.privacy_erase_requests USING btree (tombstone_id) WHERE (tombstone_id IS NOT NULL);


--
-- Name: privacy_ledger_outbox_pending; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX privacy_ledger_outbox_pending ON ttg.privacy_ledger_outbox USING btree (seq) WHERE (exported_at IS NULL);


--
-- Name: privacy_ledger_outbox_tombstone; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX privacy_ledger_outbox_tombstone ON ttg.privacy_ledger_outbox USING btree (tombstone_id);


--
-- Name: privacy_tombstones_actor; Type: INDEX; Schema: ttg; Owner: -
--

CREATE INDEX privacy_tombstones_actor ON ttg.privacy_tombstones USING btree (actor_ref_protected);


--
-- Name: sessions_one_active_per_village; Type: INDEX; Schema: ttg; Owner: -
--

CREATE UNIQUE INDEX sessions_one_active_per_village ON ttg.sessions USING btree (village_id) WHERE (lifecycle <> 'stopped'::text);


--
-- Name: actor_identity_aliases actor_identity_aliases_same_village_player_fk; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.actor_identity_aliases
    ADD CONSTRAINT actor_identity_aliases_same_village_player_fk FOREIGN KEY (village_id, player_id) REFERENCES ttg.players(village_id, id) ON DELETE CASCADE;


--
-- Name: actor_identity_aliases actor_identity_aliases_village_fk; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.actor_identity_aliases
    ADD CONSTRAINT actor_identity_aliases_village_fk FOREIGN KEY (village_id) REFERENCES ttg.villages(id) ON DELETE RESTRICT;


--
-- Name: ballots ballots_session_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.ballots
    ADD CONSTRAINT ballots_session_id_fkey FOREIGN KEY (session_id) REFERENCES ttg.sessions(id) ON DELETE RESTRICT;


--
-- Name: broadcaster_source_bindings broadcaster_source_bindings_village_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.broadcaster_source_bindings
    ADD CONSTRAINT broadcaster_source_bindings_village_id_fkey FOREIGN KEY (village_id) REFERENCES ttg.villages(id) ON DELETE RESTRICT;


--
-- Name: checkpoints checkpoints_session_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.checkpoints
    ADD CONSTRAINT checkpoints_session_id_fkey FOREIGN KEY (session_id) REFERENCES ttg.sessions(id) ON DELETE RESTRICT;


--
-- Name: domain_events domain_events_session_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.domain_events
    ADD CONSTRAINT domain_events_session_id_fkey FOREIGN KEY (session_id) REFERENCES ttg.sessions(id) ON DELETE RESTRICT;


--
-- Name: input_events input_events_session_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.input_events
    ADD CONSTRAINT input_events_session_id_fkey FOREIGN KEY (session_id) REFERENCES ttg.sessions(id) ON DELETE RESTRICT;


--
-- Name: input_receipts input_receipts_input_event_fk; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.input_receipts
    ADD CONSTRAINT input_receipts_input_event_fk FOREIGN KEY (session_id, input_seq) REFERENCES ttg.input_events(session_id, input_seq) ON DELETE RESTRICT;


--
-- Name: input_receipts input_receipts_session_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.input_receipts
    ADD CONSTRAINT input_receipts_session_id_fkey FOREIGN KEY (session_id) REFERENCES ttg.sessions(id) ON DELETE RESTRICT;


--
-- Name: operator_sessions operator_sessions_account_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.operator_sessions
    ADD CONSTRAINT operator_sessions_account_id_fkey FOREIGN KEY (account_id) REFERENCES ttg.operator_accounts(id) ON DELETE RESTRICT;


--
-- Name: players players_village_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.players
    ADD CONSTRAINT players_village_id_fkey FOREIGN KEY (village_id) REFERENCES ttg.villages(id) ON DELETE RESTRICT;


--
-- Name: privacy_erase_requests privacy_erase_requests_parent_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_erase_requests
    ADD CONSTRAINT privacy_erase_requests_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES ttg.privacy_scope_requests(id) ON DELETE RESTRICT;


--
-- Name: privacy_erase_requests privacy_erase_requests_session_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_erase_requests
    ADD CONSTRAINT privacy_erase_requests_session_id_fkey FOREIGN KEY (session_id) REFERENCES ttg.sessions(id) ON DELETE RESTRICT;


--
-- Name: privacy_erase_requests privacy_erase_requests_tombstone_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_erase_requests
    ADD CONSTRAINT privacy_erase_requests_tombstone_id_fkey FOREIGN KEY (tombstone_id) REFERENCES ttg.privacy_tombstones(id) ON DELETE RESTRICT;


--
-- Name: privacy_ledger_outbox privacy_ledger_outbox_tombstone_fk; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_ledger_outbox
    ADD CONSTRAINT privacy_ledger_outbox_tombstone_fk FOREIGN KEY (tombstone_id) REFERENCES ttg.privacy_tombstones(id) ON DELETE RESTRICT;


--
-- Name: privacy_restore_erasures privacy_restore_erasures_run_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_restore_erasures
    ADD CONSTRAINT privacy_restore_erasures_run_id_fkey FOREIGN KEY (run_id) REFERENCES ttg.privacy_restore_runs(id) ON DELETE RESTRICT;


--
-- Name: privacy_restore_erasures privacy_restore_erasures_tombstone_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_restore_erasures
    ADD CONSTRAINT privacy_restore_erasures_tombstone_id_fkey FOREIGN KEY (tombstone_id) REFERENCES ttg.privacy_tombstones(id) ON DELETE RESTRICT;


--
-- Name: privacy_scope_requests privacy_scope_requests_village_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.privacy_scope_requests
    ADD CONSTRAINT privacy_scope_requests_village_id_fkey FOREIGN KEY (village_id) REFERENCES ttg.villages(id) ON DELETE RESTRICT;


--
-- Name: reducer_transitions reducer_transitions_session_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.reducer_transitions
    ADD CONSTRAINT reducer_transitions_session_id_fkey FOREIGN KEY (session_id) REFERENCES ttg.sessions(id) ON DELETE RESTRICT;


--
-- Name: session_player_progress session_player_progress_session_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.session_player_progress
    ADD CONSTRAINT session_player_progress_session_id_fkey FOREIGN KEY (session_id) REFERENCES ttg.sessions(id) ON DELETE RESTRICT;


--
-- Name: sessions sessions_village_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.sessions
    ADD CONSTRAINT sessions_village_id_fkey FOREIGN KEY (village_id) REFERENCES ttg.villages(id) ON DELETE RESTRICT;


--
-- Name: stream_outbox stream_outbox_session_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.stream_outbox
    ADD CONSTRAINT stream_outbox_session_id_fkey FOREIGN KEY (session_id) REFERENCES ttg.sessions(id) ON DELETE RESTRICT;


--
-- Name: wave_members wave_members_session_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.wave_members
    ADD CONSTRAINT wave_members_session_id_fkey FOREIGN KEY (session_id) REFERENCES ttg.sessions(id) ON DELETE RESTRICT;


--
-- Name: writer_leases writer_leases_village_id_fkey; Type: FK CONSTRAINT; Schema: ttg; Owner: -
--

ALTER TABLE ONLY ttg.writer_leases
    ADD CONSTRAINT writer_leases_village_id_fkey FOREIGN KEY (village_id) REFERENCES ttg.villages(id) ON DELETE RESTRICT;


--
-- PostgreSQL database dump complete
--

\unrestrict eedgZmBoMbuFx8TLufjyTrl2JgaM5q3TYgukOL8F8QGRbWhStp2doWk8FkJxfgp

