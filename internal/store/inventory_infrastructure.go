package store

import (
	"context"
	"fmt"
	"slices"

	sq "github.com/Masterminds/squirrel"
)

const infrastructureSourceColumn = "source_vcenter_id"

var infrastructureTableNames = []string{"vhostscsi", "vhba", "vnic", "vdatastore", "vhost"}

// SetInfrastructureSource records ownership in a single-source import before
// its rows are appended to a collection containing other vCenters.
func (s *InventoryStore) SetInfrastructureSource(ctx context.Context, vcenterID string) error {
	if vcenterID == "" || vcenterID == "aggregated" {
		return fmt.Errorf("infrastructure requires a vCenter source")
	}

	tables, err := s.infrastructureTables(ctx)
	if err != nil {
		return err
	}

	for _, table := range tables {
		query, args, err := sq.Update(table).
			Set(infrastructureSourceColumn, vcenterID).
			Where(sq.Eq{infrastructureSourceColumn: nil}).ToSql()
		if err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("setting %s source: %w", table, err)
		}
	}
	return nil
}

// DeleteInfrastructure removes source-owned rows before the source inventory is
// deleted. Legacy ownership is recovered only when existing evidence is unique.
func (s *InventoryStore) DeleteInfrastructure(ctx context.Context, vcenterID string) error {
	tables, err := s.infrastructureTables(ctx)
	if err != nil {
		return err
	}

	// Older schemas did not retain the SDK UUID on vhost. Read it as optional
	// JSON so those collections can still use their saved per-source inventories.
	if slices.Contains(tables, "vhost") {
		if _, err := s.db.ExecContext(ctx, `
		UPDATE vhost SET source_vcenter_id = sources.vcenter_id
		FROM (
			SELECT h.rowid AS host_row, lower(trim(to_json(h)->>'VI SDK UUID')) AS vcenter_id
			FROM vhost h
		) sources
		WHERE vhost.rowid = sources.host_row AND vhost.source_vcenter_id IS NULL
		  AND sources.vcenter_id IN (SELECT id FROM inventory WHERE is_subset)
	`); err != nil {
			return fmt.Errorf("recovering host sources: %w", err)
		}
	}

	for _, resource := range []struct {
		table, path, key, column string
	}{
		{"vhost", "hosts", "id", `"Object ID"`},
		{"vdatastore", "datastores", "diskId", `"Name"`},
	} {
		if !slices.Contains(tables, resource.table) {
			continue
		}
		// Inventories are BLOBs. Only subset rows describe a single source;
		// the aggregate must never become a candidate owner.
		query := fmt.Sprintf(`
			UPDATE %s SET source_vcenter_id = sources.vcenter_id
			FROM (
				SELECT resource.value->>'%s' AS resource_id, min(i.id) AS vcenter_id, count(*) AS snapshot_count
				FROM inventory i,
				     json_each(CAST(decode(i.data) AS JSON), '$.vcenter.infra.%s') resource
				WHERE i.is_subset AND nullif(trim(resource.value->>'%s'), '') IS NOT NULL
				GROUP BY resource_id HAVING count(DISTINCT i.id) = 1
			) sources
			WHERE %s.%s = sources.resource_id AND %s.source_vcenter_id IS NULL
			  AND sources.snapshot_count = (SELECT count(*) FROM %s raw WHERE raw.%s = sources.resource_id)
		`, resource.table, resource.key, resource.path, resource.key, resource.table, resource.column, resource.table, resource.table, resource.column)
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("recovering %s sources: %w", resource.table, err)
		}
	}

	for _, table := range tables {
		// Empty means unresolved legacy ownership. Do not infer again after
		// deleting a snapshot, when formerly ambiguous evidence can look unique.
		query, args, err := sq.Update(table).
			Set(infrastructureSourceColumn, "").
			Where(sq.Eq{infrastructureSourceColumn: nil}).ToSql()
		if err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("preserving unknown %s sources: %w", table, err)
		}
		query, args, err = sq.Delete(table).
			Where(sq.Eq{infrastructureSourceColumn: vcenterID}).ToSql()
		if err != nil {
			return err
		}
		if _, err := s.db.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("deleting %s infrastructure: %w", table, err)
		}
	}
	return nil
}

func (s *InventoryStore) infrastructureTables(ctx context.Context) ([]string, error) {
	query, args, err := sq.Select("table_name").From("duckdb_tables()").
		Where(sq.Eq{"schema_name": "main", "table_name": infrastructureTableNames}).
		Where(sq.Expr("database_name = current_database()")).ToSql()
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}
	return tables, rows.Err()
}
