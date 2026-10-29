package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

// State is a job's lifecycle state.
type State string

const (
	// StateBlocked means the job waits for workflow dependencies.
	StateBlocked State = "blocked"

	// StatePending means the job is ready to be claimed.
	StatePending State = "pending"

	// StateRunning means a worker owns the job under a lease.
	StateRunning State = "running"

	// StateSucceeded means the job finished successfully.
	StateSucceeded State = "succeeded"

	// StateDead means the job exhausted its attempts.
	StateDead State = "dead"

	// StateCancelled means an upstream workflow job failed.
	StateCancelled State = "cancelled"
)

// Job is one row of the jobs table.
type Job struct {
	ID             int64
	Queue          string
	Type           string
	Payload        []byte
	State          State
	RunAt          time.Time
	Attempts       int
	MaxAttempts    int
	LastError      *string
	LeaseOwner     *string
	LeaseExpiresAt *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	CompletedAt    *time.Time
}

// jobColumnNames is the single source of truth for the jobs column order used
// by every query and by scanJob.
var jobColumnNames = []string{
	"id",
	"queue",
	"type",
	"payload",
	"state",
	"run_at",
	"attempts",
	"max_attempts",
	"last_error",
	"lease_owner",
	"lease_expires_at",
	"created_at",
	"updated_at",
	"completed_at",
}

// jobColumns returns the jobs column list, with prefix added to each name for
// queries that join another table.
func jobColumns(prefix string) string {
	columns := make([]string, len(jobColumnNames))
	for i, name := range jobColumnNames {
		columns[i] = prefix + name
	}
	return strings.Join(columns, ", ")
}

// scanJob reads one job row in jobColumnNames order.
func scanJob(row pgx.Row) (Job, error) {
	var job Job
	err := row.Scan(
		&job.ID,
		&job.Queue,
		&job.Type,
		&job.Payload,
		&job.State,
		&job.RunAt,
		&job.Attempts,
		&job.MaxAttempts,
		&job.LastError,
		&job.LeaseOwner,
		&job.LeaseExpiresAt,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.CompletedAt,
	)
	if err != nil {
		return Job{}, err
	}
	return job, nil
}

// scanJobs reads every row and closes rows before returning.
func scanJobs(rows pgx.Rows) ([]Job, error) {
	defer rows.Close()

	var jobs []Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return jobs, nil
}

// EnqueueParams describes a new job. A nil Payload becomes an empty JSON
// object, and MaxAttempts 0 becomes the schema default.
type EnqueueParams struct {
	Queue       string
	Type        string
	Payload     []byte
	MaxAttempts int
}

// Enqueue inserts a new pending job and returns the stored row.
func (s *Store) Enqueue(ctx context.Context, p EnqueueParams) (Job, error) {
	query := `
INSERT INTO jobs (queue, type, payload, max_attempts)
VALUES ($1, $2, COALESCE($3::jsonb, '{}'::jsonb), COALESCE(NULLIF($4::int, 0), 5))
RETURNING ` + jobColumns("")

	job, err := scanJob(s.db.QueryRow(ctx, query, p.Queue, p.Type, p.Payload, p.MaxAttempts))
	if err != nil {
		return Job{}, fmt.Errorf("enqueue job: %w", err)
	}
	return job, nil
}

// GetJob returns the job with the given ID. It returns ErrNotFound when no
// row exists.
func (s *Store) GetJob(ctx context.Context, id int64) (Job, error) {
	query := `SELECT ` + jobColumns("") + ` FROM jobs WHERE id = $1`

	job, err := scanJob(s.db.QueryRow(ctx, query, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Job{}, ErrNotFound
	}
	if err != nil {
		return Job{}, fmt.Errorf("get job %d: %w", id, err)
	}
	return job, nil
}
