package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kubev2v/assisted-migration-agent/internal/models"
	srvErrors "github.com/kubev2v/assisted-migration-agent/pkg/errors"
)

// ConfigurationStore handles configuration storage using DuckDB.
type ConfigurationStore struct {
	db *sql.DB
}

// NewConfigurationStore creates a new configuration store.
func NewConfigurationStore(db *sql.DB) *ConfigurationStore {
	return &ConfigurationStore{db: db}
}

// Get retrieves the stored configuration.
func (s *ConfigurationStore) Get(ctx context.Context) (*models.Configuration, error) {
	row := s.db.QueryRowContext(ctx, queryGetConfiguration)

	var agentMode string
	err := row.Scan(&agentMode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, srvErrors.NewConfigurationNotFoundError()
	}
	if err != nil {
		return nil, err
	}
	return &models.Configuration{
		AgentMode: models.AgentMode(agentMode),
	}, nil
}

// Save stores or updates the configuration.
func (s *ConfigurationStore) Save(ctx context.Context, cfg *models.Configuration) error {
	_, err := s.db.ExecContext(ctx, queryUpsertConfiguration, string(cfg.AgentMode))
	return err
}
