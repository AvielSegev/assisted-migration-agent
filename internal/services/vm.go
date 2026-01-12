package services

import (
	"context"

	"github.com/kubev2v/assisted-migration-agent/internal/models"
	"github.com/kubev2v/assisted-migration-agent/internal/store"
)

type VMService struct {
	store *store.Store
}

func NewVMService(st *store.Store) *VMService {
	return &VMService{store: st}
}

type VMListParams struct {
	Datacenters []string
	Clusters    []string
	Statuses    []string
	Issues      []string
	DiskRanges  []SizeRange
	MemRanges   []SizeRange
	Limit       uint64
	Offset      uint64
}

type SizeRange struct {
	Min int64
	Max int64 // 0 means no upper limit
}

type VMListResult struct {
	VMs   []models.VM
	Total int
}

func (s *VMService) List(ctx context.Context, params VMListParams) (*VMListResult, error) {
	opts := s.buildListOptions(params)

	vms, err := s.store.VM().List(ctx, opts...)
	if err != nil {
		return nil, err
	}

	// Get total count without pagination
	countOpts := s.buildListOptions(VMListParams{
		Datacenters: params.Datacenters,
		Clusters:    params.Clusters,
		Statuses:    params.Statuses,
		Issues:      params.Issues,
		DiskRanges:  params.DiskRanges,
		MemRanges:   params.MemRanges,
	})
	total, err := s.store.VM().Count(ctx, countOpts...)
	if err != nil {
		return nil, err
	}

	return &VMListResult{
		VMs:   vms,
		Total: total,
	}, nil
}

func (s *VMService) buildListOptions(params VMListParams) []store.ListOption {
	var opts []store.ListOption

	if len(params.Datacenters) > 0 {
		opts = append(opts, store.ByDatacenters(params.Datacenters...))
	}
	if len(params.Clusters) > 0 {
		opts = append(opts, store.ByClusters(params.Clusters...))
	}
	if len(params.Statuses) > 0 {
		opts = append(opts, store.ByStatus(params.Statuses...))
	}
	if len(params.Issues) > 0 {
		opts = append(opts, store.ByIssues(params.Issues...))
	}

	// Handle disk size ranges (OR logic between ranges)
	for _, r := range params.DiskRanges {
		if r.Max == 0 {
			// No upper limit
			opts = append(opts, store.ByDiskSizeRange(r.Min, 1<<62))
		} else {
			opts = append(opts, store.ByDiskSizeRange(r.Min, r.Max))
		}
	}

	// Handle memory size ranges (OR logic between ranges)
	for _, r := range params.MemRanges {
		if r.Max == 0 {
			// No upper limit
			opts = append(opts, store.ByMemorySizeRange(r.Min, 1<<62))
		} else {
			opts = append(opts, store.ByMemorySizeRange(r.Min, r.Max))
		}
	}

	if params.Limit > 0 {
		opts = append(opts, store.WithLimit(params.Limit))
	}
	if params.Offset > 0 {
		opts = append(opts, store.WithOffset(params.Offset))
	}

	return opts
}
