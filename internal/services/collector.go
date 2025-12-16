package services

import (
	"context"
	"errors"

	"github.com/kubev2v/assisted-migration-agent/internal/models"
	"github.com/kubev2v/assisted-migration-agent/internal/store"
	"github.com/kubev2v/assisted-migration-agent/pkg/scheduler"
)

type Collector struct {
	scheduler *scheduler.Scheduler
	store     *store.Store
}

func NewCollectorService(s *scheduler.Scheduler, st *store.Store) *Collector {
	return &Collector{
		scheduler: s,
		store:     st,
	}
}

// SaveCredentials stores vCenter credentials.
func (c *Collector) SaveCredentials(ctx context.Context, creds *models.Credentials) error {
	return c.store.Credentials().Save(ctx, creds)
}

// DeleteCredentials removes stored credentials.
func (c *Collector) DeleteCredentials(ctx context.Context) error {
	return c.store.Credentials().Delete(ctx)
}

// HasCredentials checks if credentials exist.
func (c *Collector) HasCredentials(ctx context.Context) (bool, error) {
	_, err := c.store.Credentials().Get(ctx)
	if errors.Is(err, store.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetCredentials retrieves stored credentials.
func (c *Collector) GetCredentials(ctx context.Context) (*models.Credentials, error) {
	return c.store.Credentials().Get(ctx)
}
