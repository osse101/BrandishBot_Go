package cooldown

import (
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
)

func TestNewPostgresService(t *testing.T) {
	pool := &pgxpool.Pool{}
	config := Config{}
	mockProg := &mockProgressionService{}

	svc := NewPostgresService(pool, config, mockProg)

	assert.NotNil(t, svc)
	backend, ok := svc.(*postgresBackend)
	assert.True(t, ok)
	assert.Equal(t, pool, backend.db)
	assert.Equal(t, config, backend.config)
	assert.Equal(t, mockProg, backend.progressionSvc)
}
