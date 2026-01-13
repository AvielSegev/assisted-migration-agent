package store

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubev2v/assisted-migration-agent/internal/models"
)

type VMStore struct {
	db QueryInterceptor
}

func NewVMStore(db QueryInterceptor) *VMStore {
	return &VMStore{db: db}
}

func (s *VMStore) List(ctx context.Context, opts ...ListOption) ([]models.VM, error) {
	builder := sq.Select(
		`vinfo."VM ID"`,
		`vinfo."VM"`,
		`vinfo."Powerstate"`,
		`vinfo."Datacenter"`,
		`vinfo."Cluster"`,
		`vinfo."Total disk capacity MiB"`,
		`vinfo."Memory"`,
		`LIST(concerns."Label") as issues`,
	).From("vinfo").
		LeftJoin(`concerns ON vinfo."VM ID" = concerns."VM_ID"`).
		GroupBy(
			`vinfo."VM ID"`,
			`vinfo."VM"`,
			`vinfo."Powerstate"`,
			`vinfo."Datacenter"`,
			`vinfo."Cluster"`,
			`vinfo."Total disk capacity MiB"`,
			`vinfo."Memory"`,
		)

	for _, opt := range opts {
		builder = opt(builder)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vms []models.VM
	for rows.Next() {
		var vm models.VM
		var issues any
		err := rows.Scan(
			&vm.ID,
			&vm.Name,
			&vm.PowerState,
			&vm.Datacenter,
			&vm.Cluster,
			&vm.DiskSize,
			&vm.MemoryMB,
			&issues,
		)
		if err != nil {
			return nil, err
		}
		vm.Issues = toStringSlice(issues)
		vms = append(vms, vm)
	}

	return vms, rows.Err()
}

func (s *VMStore) Count(ctx context.Context, opts ...ListOption) (int, error) {
	builder := sq.Select("COUNT(*)").From("vinfo")

	for _, opt := range opts {
		builder = opt(builder)
	}

	query, args, err := builder.ToSql()
	if err != nil {
		return 0, err
	}

	var count int
	err = s.db.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

type ListOption func(sq.SelectBuilder) sq.SelectBuilder

func ByDatacenters(datacenters ...string) ListOption {
	return func(b sq.SelectBuilder) sq.SelectBuilder {
		if len(datacenters) == 0 {
			return b
		}
		return b.Where(sq.Eq{`vinfo."Datacenter"`: datacenters})
	}
}

func ByClusters(clusters ...string) ListOption {
	return func(b sq.SelectBuilder) sq.SelectBuilder {
		if len(clusters) == 0 {
			return b
		}
		return b.Where(sq.Eq{`vinfo."Cluster"`: clusters})
	}
}

func ByStatus(statuses ...string) ListOption {
	return func(b sq.SelectBuilder) sq.SelectBuilder {
		if len(statuses) == 0 {
			return b
		}
		return b.Where(sq.Eq{`vinfo."Powerstate"`: statuses})
	}
}

func ByIssues(issues ...string) ListOption {
	return func(b sq.SelectBuilder) sq.SelectBuilder {
		if len(issues) == 0 {
			return b
		}
		return b.Where(sq.Expr(
			`vinfo."VM ID" IN (SELECT "VM_ID" FROM concerns WHERE "Label" IN (?))`,
			issues,
		))
	}
}

func ByDiskSizeRange(min, max int64) ListOption {
	return func(b sq.SelectBuilder) sq.SelectBuilder {
		return b.Where(sq.And{
			sq.GtOrEq{`vinfo."Total disk capacity MiB"`: min},
			sq.Lt{`vinfo."Total disk capacity MiB"`: max},
		})
	}
}

func ByMemorySizeRange(min, max int64) ListOption {
	return func(b sq.SelectBuilder) sq.SelectBuilder {
		return b.Where(sq.And{
			sq.GtOrEq{`vinfo."Memory"`: min},
			sq.Lt{`vinfo."Memory"`: max},
		})
	}
}

func WithLimit(limit uint64) ListOption {
	return func(b sq.SelectBuilder) sq.SelectBuilder {
		return b.Limit(limit)
	}
}

func WithOffset(offset uint64) ListOption {
	return func(b sq.SelectBuilder) sq.SelectBuilder {
		return b.Offset(offset)
	}
}

type SortParam struct {
	Field string
	Desc  bool
}

var apiFieldToDBColumn = map[string]string{
	"name":         `vinfo."VM"`,
	"vCenterState": `vinfo."Powerstate"`,
	"datacenter":   `vinfo."Datacenter"`,
	"cluster":      `vinfo."Cluster"`,
	"diskSize":     `vinfo."Total disk capacity MiB"`,
	"memory":       `vinfo."Memory"`,
}

func WithDefaultSort() ListOption {
	return func(b sq.SelectBuilder) sq.SelectBuilder {
		return b.OrderBy(`vinfo."VM ID"`)
	}
}

func WithSort(sorts []SortParam) ListOption {
	return func(b sq.SelectBuilder) sq.SelectBuilder {
		var orderClauses []string
		for _, s := range sorts {
			col, ok := apiFieldToDBColumn[s.Field]
			if !ok {
				continue
			}
			if s.Desc {
				orderClauses = append(orderClauses, col+" DESC")
			} else {
				orderClauses = append(orderClauses, col+" ASC")
			}
		}
		orderClauses = append(orderClauses, `vinfo."VM ID"`)
		return b.OrderBy(orderClauses...)
	}
}

func toStringSlice(v any) []string {
	if v == nil {
		return nil
	}
	slice, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item == nil {
			continue
		}
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}
