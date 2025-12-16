package store

import "database/sql"

// Store provides access to all storage repositories.
type Store struct {
	db          *sql.DB
	credentials *CredentialsStore
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:          db,
		credentials: NewCredentialsStore(db),
	}
}

func (s *Store) Credentials() *CredentialsStore {
	return s.credentials
}

func (s *Store) Close() error {
	return s.db.Close()
}
