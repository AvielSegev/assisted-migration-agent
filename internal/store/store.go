package store

import "database/sql"

// Store provides access to all storage repositories.
type Store struct {
	db            *sql.DB
	configuration *ConfigurationStore
	inventory     *InventoryStore
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:            db,
		configuration: NewConfigurationStore(db),
		inventory:     NewInventoryStore(db),
	}
}

func (s *Store) Configuration() *ConfigurationStore {
	return s.configuration
}

func (s *Store) Inventory() *InventoryStore {
	return s.inventory
}

func (s *Store) Close() error {
	return s.db.Close()
}
