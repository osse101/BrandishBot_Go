package cooldown

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostgresBackend_CheckCooldown(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()

	tests := []struct{
		name string
		devMode bool
		mockSetup func()
		wantOnCooldown bool
		wantRemaining time.Duration
		wantErr bool
	}{
		{
			name: "dev mode bypass",
			devMode: true,
			mockSetup: func() {},
			wantOnCooldown: false,
			wantRemaining: 0,
			wantErr: false,
		},
		{
			name: "no previous cooldown",
			devMode: false,
			mockSetup: func() {
				mock.ExpectQuery("SELECT last_used_at FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
					WithArgs("user1", "action1").
					WillReturnError(pgx.ErrNoRows)
			},
			wantOnCooldown: false,
			wantRemaining: 0,
			wantErr: false,
		},
		{
			name: "db error",
			devMode: false,
			mockSetup: func() {
				mock.ExpectQuery("SELECT last_used_at FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
					WithArgs("user1", "action1").
					WillReturnError(errors.New("db error"))
			},
			wantOnCooldown: false,
			wantRemaining: 0,
			wantErr: true,
		},
		{
			name: "active cooldown",
			devMode: false,
			mockSetup: func() {
				// Setting last used to 1 minute ago, default cooldown is 5 minutes
				rows := pgxmock.NewRows([]string{"last_used"}).AddRow(now.Add(-time.Minute))
				mock.ExpectQuery("SELECT last_used_at FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
					WithArgs("user1", "action1").
					WillReturnRows(rows)
			},
			wantOnCooldown: true,
			wantRemaining: 4 * time.Minute, // Approximation since time.Now() varies
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			config := Config{
				DevMode: tt.devMode,
				Cooldowns: map[string]time.Duration{
					"action1": 5 * time.Minute,
				},
			}

			svc := NewPostgresService(mock, config, nil)

			gotOnCooldown, gotRemaining, err := svc.CheckCooldown(context.Background(), "user1", "action1")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantOnCooldown, gotOnCooldown)
				if tt.wantOnCooldown {
					// Check remaining is roughly what we expect (allowing 1 second difference)
					assert.InDelta(t, tt.wantRemaining.Seconds(), gotRemaining.Seconds(), 1.0)
				}
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPostgresBackend_ResetCooldown(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
		WithArgs("user1", "action1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	svc := NewPostgresService(mock, Config{}, nil)
	err = svc.ResetCooldown(context.Background(), "user1", "action1")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresBackend_ResetCooldown_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	mock.ExpectExec("DELETE FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
		WithArgs("user1", "action1").
		WillReturnError(errors.New("db error"))

	svc := NewPostgresService(mock, Config{}, nil)
	err = svc.ResetCooldown(context.Background(), "user1", "action1")
	assert.Error(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresBackend_GetLastUsed(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()
	rows := pgxmock.NewRows([]string{"last_used"}).AddRow(now)

	mock.ExpectQuery("SELECT last_used_at FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
		WithArgs("user1", "action1").
		WillReturnRows(rows)

	svc := NewPostgresService(mock, Config{}, nil)
	lastUsed, err := svc.GetLastUsed(context.Background(), "user1", "action1")

	assert.NoError(t, err)
	assert.NotNil(t, lastUsed)
	if lastUsed != nil {
		// Truncate to second for comparison since pgx returns rounded times
		assert.Equal(t, now.Truncate(time.Second), lastUsed.Truncate(time.Second))
	}

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresBackend_EnforceCooldown_DevMode(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// Dev mode updates the cooldown after execution
	mock.ExpectExec("INSERT INTO user_cooldowns \\(user_id, action_name, last_used_at\\) VALUES \\(\\$1, \\$2, \\$3\\) ON CONFLICT \\(user_id, action_name\\) DO UPDATE SET last_used_at = EXCLUDED.last_used_at").
		WithArgs("user1", "action1", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	svc := NewPostgresService(mock, Config{DevMode: true}, nil)

	called := false
	fn := func() error {
		called = true
		return nil
	}

	err = svc.EnforceCooldown(context.Background(), "user1", "action1", fn)

	assert.NoError(t, err)
	assert.True(t, called)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresBackend_EnforceCooldown_DevMode_FnError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	svc := NewPostgresService(mock, Config{DevMode: true}, nil)

	expectedErr := errors.New("fn error")
	fn := func() error {
		return expectedErr
	}

	err = svc.EnforceCooldown(context.Background(), "user1", "action1", fn)

	assert.ErrorIs(t, err, expectedErr)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresBackend_EnforceCooldown_AlreadyOnCooldown(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()

	// CheckCooldown returns true
	rows := pgxmock.NewRows([]string{"last_used"}).AddRow(now.Add(-time.Minute))
	mock.ExpectQuery("SELECT last_used_at FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
		WithArgs("user1", "action1").
		WillReturnRows(rows)

	svc := NewPostgresService(mock, Config{
		Cooldowns: map[string]time.Duration{
			"action1": 5 * time.Minute,
		},
	}, nil)

	called := false
	fn := func() error {
		called = true
		return nil
	}

	err = svc.EnforceCooldown(context.Background(), "user1", "action1", fn)

	assert.Error(t, err)
	assert.IsType(t, ErrOnCooldown{}, err)
	assert.False(t, called)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresBackend_EnforceCooldown_Success(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// 1. CheckCooldown phase
	mock.ExpectQuery("SELECT last_used_at FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
		WithArgs("user1", "action1").
		WillReturnError(pgx.ErrNoRows)

	// 2. Transaction phase
	mock.ExpectBegin()

	// Advisory lock
	mock.ExpectExec("SELECT pg_advisory_xact_lock\\(\\$1\\)").
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("", 0))

	// GetLastUsedTx
	mock.ExpectQuery("SELECT last_used_at FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
		WithArgs("user1", "action1").
		WillReturnError(pgx.ErrNoRows)

	// Update cooldown
	mock.ExpectExec("INSERT INTO user_cooldowns \\(user_id, action_name, last_used_at\\) VALUES \\(\\$1, \\$2, \\$3\\) ON CONFLICT \\(user_id, action_name\\) DO UPDATE SET last_used_at = EXCLUDED.last_used_at").
		WithArgs("user1", "action1", pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectCommit()

	svc := NewPostgresService(mock, Config{}, nil)

	called := false
	fn := func() error {
		called = true
		return nil
	}

	err = svc.EnforceCooldown(context.Background(), "user1", "action1", fn)

	assert.NoError(t, err)
	assert.True(t, called)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresBackend_EnforceCooldown_RaceConditionDetected(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	now := time.Now()

	// 1. CheckCooldown phase returns false (not on cooldown)
	mock.ExpectQuery("SELECT last_used_at FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
		WithArgs("user1", "action1").
		WillReturnError(pgx.ErrNoRows)

	// 2. Transaction phase
	mock.ExpectBegin()

	// Advisory lock
	mock.ExpectExec("SELECT pg_advisory_xact_lock\\(\\$1\\)").
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("", 0))

	// GetLastUsedTx returns an active cooldown (race condition occurred!)
	rows := pgxmock.NewRows([]string{"last_used"}).AddRow(now)
	mock.ExpectQuery("SELECT last_used_at FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
		WithArgs("user1", "action1").
		WillReturnRows(rows)

	mock.ExpectRollback()

	svc := NewPostgresService(mock, Config{
		Cooldowns: map[string]time.Duration{
			"action1": 5 * time.Minute,
		},
	}, nil)

	called := false
	fn := func() error {
		called = true
		return nil
	}

	err = svc.EnforceCooldown(context.Background(), "user1", "action1", fn)

	assert.Error(t, err)
	assert.IsType(t, ErrOnCooldown{}, err)
	assert.False(t, called)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPostgresBackend_EnforceCooldown_FnError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	require.NoError(t, err)
	defer mock.Close()

	// 1. CheckCooldown phase
	mock.ExpectQuery("SELECT last_used_at FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
		WithArgs("user1", "action1").
		WillReturnError(pgx.ErrNoRows)

	// 2. Transaction phase
	mock.ExpectBegin()

	// Advisory lock
	mock.ExpectExec("SELECT pg_advisory_xact_lock\\(\\$1\\)").
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("", 0))

	// GetLastUsedTx
	mock.ExpectQuery("SELECT last_used_at FROM user_cooldowns WHERE user_id = \\$1 AND action_name = \\$2").
		WithArgs("user1", "action1").
		WillReturnError(pgx.ErrNoRows)

	mock.ExpectRollback()

	svc := NewPostgresService(mock, Config{}, nil)

	expectedErr := errors.New("fn error")
	fn := func() error {
		return expectedErr
	}

	err = svc.EnforceCooldown(context.Background(), "user1", "action1", fn)

	assert.ErrorIs(t, err, expectedErr)

	assert.NoError(t, mock.ExpectationsWereMet())
}
