package store

// Configuration queries
const (
	queryGetConfiguration = `
		SELECT agent_mode
		FROM configuration WHERE id = 1`

	queryUpsertConfiguration = `
		INSERT INTO configuration (id, agent_mode)
		VALUES (1, ?)
		ON CONFLICT (id) DO UPDATE SET
			agent_mode = EXCLUDED.agent_mode`
)

// Inventory queries
const (
	queryGetInventory = `
		SELECT data, created_at, updated_at
		FROM inventory WHERE id = 1`

	queryUpsertInventory = `
		INSERT INTO inventory (id, data, updated_at)
		VALUES (1, ?, now())
		ON CONFLICT (id) DO UPDATE SET
			data = EXCLUDED.data,
			updated_at = now()`
)
