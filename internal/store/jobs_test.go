package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MohammedArab1/jobqueue/internal/store"
	"github.com/MohammedArab1/jobqueue/internal/testutil"
)

func TestEnqueueRoundTrip(t *testing.T) {
	pool := testutil.NewTestDB(t)
	s := store.New(pool)
	ctx := context.Background()

	payload := []byte(`{"to":"a@example.com","subject":"hello"}`)
	job, err := s.Enqueue(ctx, store.EnqueueParams{
		Queue:       "default",
		Type:        "send_email",
		Payload:     payload,
		MaxAttempts: 3,
	})
	require.NoError(t, err)
	require.NotZero(t, job.ID)
	require.Equal(t, "default", job.Queue)
	require.Equal(t, "send_email", job.Type)
	require.JSONEq(t, string(payload), string(job.Payload))
	require.Equal(t, store.StatePending, job.State)
	require.Equal(t, 0, job.Attempts)
	require.Equal(t, 3, job.MaxAttempts)
	require.Nil(t, job.LastError)
	require.Nil(t, job.LeaseOwner)
	require.Nil(t, job.LeaseExpiresAt)
	require.Nil(t, job.CompletedAt)
	require.False(t, job.CreatedAt.IsZero())
	require.False(t, job.UpdatedAt.IsZero())

	got, err := s.GetJob(ctx, job.ID)
	require.NoError(t, err)
	require.Equal(t, job, got)
}

func TestEnqueueDefaults(t *testing.T) {
	pool := testutil.NewTestDB(t)
	s := store.New(pool)

	job, err := s.Enqueue(context.Background(), store.EnqueueParams{
		Queue: "default",
		Type:  "noop",
	})
	require.NoError(t, err)
	require.Equal(t, store.StatePending, job.State)
	require.Equal(t, 0, job.Attempts)
	require.Equal(t, 5, job.MaxAttempts)
	require.JSONEq(t, `{}`, string(job.Payload))
}

func TestGetJobNotFound(t *testing.T) {
	pool := testutil.NewTestDB(t)
	s := store.New(pool)

	_, err := s.GetJob(context.Background(), 999999)
	require.ErrorIs(t, err, store.ErrNotFound)
}
