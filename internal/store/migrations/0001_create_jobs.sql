-- +goose Up
CREATE TABLE jobs (
    id               BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    queue            TEXT        NOT NULL,
    type             TEXT        NOT NULL,
    payload          JSONB       NOT NULL DEFAULT '{}'::jsonb,
    state            TEXT        NOT NULL DEFAULT 'pending',
    run_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    attempts         INT         NOT NULL DEFAULT 0,
    max_attempts     INT         NOT NULL DEFAULT 5,
    last_error       TEXT,
    lease_owner      TEXT,
    lease_expires_at TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at     TIMESTAMPTZ,

    CONSTRAINT jobs_state_check
        CHECK (state IN ('blocked', 'pending', 'running', 'succeeded', 'dead', 'cancelled')),
    CONSTRAINT jobs_attempts_check     CHECK (attempts >= 0),
    CONSTRAINT jobs_max_attempts_check CHECK (max_attempts >= 1),
    -- Invariant 1: a job has a lease if and only if it is running.
    CONSTRAINT jobs_lease_check
        CHECK ((state = 'running') = (lease_owner IS NOT NULL AND lease_expires_at IS NOT NULL)),
    -- Invariant 2: completed_at is set if and only if the job is in a terminal state.
    CONSTRAINT jobs_completed_check
        CHECK ((state IN ('succeeded', 'dead', 'cancelled')) = (completed_at IS NOT NULL))
);

-- The "ready" index: only pending rows, in the order workers should take them.
CREATE INDEX jobs_ready_idx ON jobs (queue, run_at, id) WHERE state = 'pending';

-- +goose Down
DROP TABLE jobs;
