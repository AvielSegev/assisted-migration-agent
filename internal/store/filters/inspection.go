package filters

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/kubev2v/assisted-migration-agent/internal/models"
)

// Column name constants for vm_inspection_status table
const (
	inspectionColVmID     = `"VM ID"`
	inspectionColState    = "status"
	inspectionColSequence = "sequence"
)

type InspectionFilterFunc func(sq.SelectBuilder) sq.SelectBuilder

type InspectionQueryFilter struct {
	filters []InspectionFilterFunc
}

func NewInspectionQueryFilter() *InspectionQueryFilter {
	return &InspectionQueryFilter{
		filters: make([]InspectionFilterFunc, 0),
	}
}

func (f *InspectionQueryFilter) Add(filter InspectionFilterFunc) *InspectionQueryFilter {
	f.filters = append(f.filters, filter)
	return f
}

func (f *InspectionQueryFilter) ByVmIDs(vmIDs ...string) *InspectionQueryFilter {
	if len(vmIDs) == 0 {
		return f
	}
	return f.Add(func(b sq.SelectBuilder) sq.SelectBuilder {
		return b.Where(sq.Eq{inspectionColVmID: vmIDs})
	})
}

func (f *InspectionQueryFilter) ByState(states ...models.InspectionState) *InspectionQueryFilter {
	if len(states) == 0 {
		return f
	}
	stateStrings := make([]string, len(states))
	for i, s := range states {
		stateStrings[i] = s.Value()
	}
	return f.Add(func(b sq.SelectBuilder) sq.SelectBuilder {
		return b.Where(sq.Eq{inspectionColState: stateStrings})
	})
}

func (f *InspectionQueryFilter) ByStateNot(state models.InspectionState) *InspectionQueryFilter {
	return f.Add(func(b sq.SelectBuilder) sq.SelectBuilder {
		return b.Where(sq.NotEq{
			inspectionColState: state.Value(),
		})
	})
}

func (f *InspectionQueryFilter) Limit(limit int) *InspectionQueryFilter {
	return f.Add(func(b sq.SelectBuilder) sq.SelectBuilder {
		return b.Limit(uint64(limit))
	})
}

func (f *InspectionQueryFilter) OrderBySequence() *InspectionQueryFilter {
	return f.Add(func(b sq.SelectBuilder) sq.SelectBuilder {
		return b.OrderBy(inspectionColSequence + " ASC")
	})
}

func (f *InspectionQueryFilter) Apply(builder sq.SelectBuilder) sq.SelectBuilder {
	for _, filter := range f.filters {
		builder = filter(builder)
	}
	return builder
}

type UpdateFilterFunc func(sq.UpdateBuilder) sq.UpdateBuilder

type InspectionUpdateFilter struct {
	filters []UpdateFilterFunc
}

func NewInspectionUpdateFilter() *InspectionUpdateFilter {
	return &InspectionUpdateFilter{
		filters: make([]UpdateFilterFunc, 0),
	}
}

func (f *InspectionUpdateFilter) ByVmIDs(vmIDs ...string) *InspectionUpdateFilter {
	if len(vmIDs) == 0 {
		return f
	}
	f.filters = append(f.filters, func(b sq.UpdateBuilder) sq.UpdateBuilder {
		return b.Where(sq.Eq{inspectionColVmID: vmIDs})
	})
	return f
}

func (f *InspectionUpdateFilter) ByState(states ...models.InspectionState) *InspectionUpdateFilter {
	if len(states) == 0 {
		return f
	}
	statesStrings := make([]string, len(states))
	for i, s := range states {
		statesStrings[i] = s.Value()
	}
	f.filters = append(f.filters, func(b sq.UpdateBuilder) sq.UpdateBuilder {
		return b.Where(sq.Eq{inspectionColState: statesStrings})
	})
	return f
}

func (f *InspectionUpdateFilter) Apply(builder sq.UpdateBuilder) sq.UpdateBuilder {
	for _, filter := range f.filters {
		builder = filter(builder)
	}
	return builder
}
