package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/kubev2v/assisted-migration-agent/pkg/vmware"

	"go.uber.org/zap"

	"github.com/kubev2v/assisted-migration-agent/internal/models"
	"github.com/kubev2v/assisted-migration-agent/pkg/scheduler"
)

type (
	InspectionPipeline    = WorkPipeline[models.InspectionStatus, models.InspectionResult]
	InspectionWorkBuilder func(id string) []models.WorkUnit[models.InspectionStatus, models.InspectionResult]
)

type InspectionService struct {
	scheduler *scheduler.Scheduler[models.InspectionResult]

	buildFn   InspectionWorkBuilder
	pipelines map[string]*InspectionPipeline

	mu sync.Mutex

	cancel context.CancelFunc

	operator vmware.VMOperator
}

// NewInspectionService creates a new InspectionService with the default vmware builder.
func NewInspectionService() *InspectionService {
	return &InspectionService{}
}

// GetVmStatus returns the current vm inspectionSvc status.
func (c *InspectionService) GetVmStatus(ctx context.Context, id string) (models.InspectionStatus, error) {
	c.mu.Lock()
	pipeline, found := c.pipelines[id]
	c.mu.Unlock()

	if !found {
		return models.InspectionStatus{State: models.InspectionStateNotFound}, nil
	}

	state := pipeline.State()
	if state.Err != nil {
		if !errors.Is(state.Err, errPipelineStopped) {
			return models.InspectionStatus{State: models.InspectionStateCanceled, Error: state.Err}, nil
		}
		return models.InspectionStatus{State: models.InspectionStateError, Error: state.Err}, nil
	}

	if pipeline.IsRunning() {
		return state.State, nil
	}

	return models.InspectionStatus{State: models.InspectionStateCompleted}, nil
}

func (c *InspectionService) Start(operator *vmware.VMManager, vmIDs []string) error {
	c.operator = operator

	sched := scheduler.NewScheduler[models.InspectionResult](5)
	c.scheduler = sched

	if c.buildFn == nil {
		c.buildFn = c.buildInspectionWorkUnits
	}

	c.pipelines = make(map[string]*InspectionPipeline)
	for _, id := range vmIDs {
		c.mu.Lock()
		c.pipelines[id] = c.startVmPipeline(id)
		c.mu.Unlock()
	}

	return nil
}

func (c *InspectionService) startVmPipeline(id string) *InspectionPipeline {
	pipeline := NewWorkPipeline(models.InspectionStatus{State: models.InspectionStatePending}, c.scheduler, c.buildFn(id))
	if err := pipeline.Start(); err != nil {
		c.pipelines[id].state = WorkPipelineStatus[models.InspectionStatus, models.InspectionResult]{
			State: models.InspectionStatus{State: models.InspectionStateError, Error: err},
			Err:   err,
		}
	}

	return pipeline
}

func (c *InspectionService) Add(vmIDs []string) error {
	if len(vmIDs) == 0 {
		return fmt.Errorf("vmIDs is empty")
	}

	for _, id := range vmIDs {
		c.mu.Lock()
		_, found := c.pipelines[id]
		if !found {
			c.pipelines[id] = c.startVmPipeline(id)
		}
		c.mu.Unlock()
	}

	return nil
}

func (c *InspectionService) Stop(ctx context.Context) error {
	// implement me
	return nil
}

func (c *InspectionService) CancelVmsInspection(ctx context.Context, vmIDs ...string) error {
	c.mu.Lock()
	ids := make([]string, 0, len(c.pipelines))
	if len(vmIDs) == 0 {
		for id := range c.pipelines {
			ids = append(ids, id)
		}
	} else {
		ids = append(ids, vmIDs...)
	}
	pipelines := make([]*InspectionPipeline, 0, len(ids))
	for _, id := range ids {
		if p, ok := c.pipelines[id]; ok {
			pipelines = append(pipelines, p)
		}
	}
	c.mu.Unlock()

	for _, p := range pipelines {
		p.Stop()
	}

	return nil
}

func (c *InspectionService) WithWorkUnitsBuilder(builder InspectionWorkBuilder) *InspectionService {
	c.buildFn = builder
	return c
}

func (c *InspectionService) buildInspectionWorkUnits(id string) []models.WorkUnit[models.InspectionStatus, models.InspectionResult] {
	return []models.WorkUnit[models.InspectionStatus, models.InspectionResult]{
		{
			Status: func() models.InspectionStatus {
				return models.InspectionStatus{State: models.InspectionStateRunning}
			},
			Work: func(ctx context.Context, result models.InspectionResult) (models.InspectionResult, error) {
				err := c.validate(ctx, id)
				return result, err
			},
		},
		{
			Status: func() models.InspectionStatus {
				return models.InspectionStatus{State: models.InspectionStateRunning}
			},
			Work: func(ctx context.Context, result models.InspectionResult) (models.InspectionResult, error) {
				err := c.createSnapshot(ctx, id)
				return result, err
			},
		},
		{
			Status: func() models.InspectionStatus {
				return models.InspectionStatus{State: models.InspectionStateRunning}
			},
			Work: func(ctx context.Context, result models.InspectionResult) (models.InspectionResult, error) {
				err := c.inspect(id)
				return result, err
			},
		},
		{
			Status: func() models.InspectionStatus {
				return models.InspectionStatus{State: models.InspectionStateRunning}
			},
			Work: func(ctx context.Context, result models.InspectionResult) (models.InspectionResult, error) {
				err := c.save(ctx, id)
				return result, err
			},
		},
		{
			Status: func() models.InspectionStatus {
				return models.InspectionStatus{State: models.InspectionStateRunning}
			},
			Work: func(ctx context.Context, result models.InspectionResult) (models.InspectionResult, error) {
				err := c.removeSnapshot(ctx, id)
				return result, err
			},
		},
	}
}

func (c *InspectionService) validate(ctx context.Context, id string) error {
	return c.operator.ValidatePrivileges(ctx, id, models.RequiredPrivileges)
}

func (c *InspectionService) createSnapshot(ctx context.Context, id string) error {
	zap.S().Named("inspector_service").Infow("creating VM snapshot", "vmId", id)
	req := vmware.CreateSnapshotRequest{
		VmId:         id,
		SnapshotName: models.InspectionSnapshotName,
		Description:  "",
		Memory:       false,
		Quiesce:      false,
	}

	if err := c.operator.CreateSnapshot(ctx, req); err != nil {
		zap.S().Named("inspector_service").Errorw("failed to create VM snapshot", "vmId", id, "error", err)
		return err
	}

	zap.S().Named("inspector_service").Infow("VM snapshot created", "vmId", id)

	return nil
}

func (c *InspectionService) inspect(id string) error {
	return nil
}

func (c *InspectionService) save(ctx context.Context, id string) error {
	return nil
}

func (c *InspectionService) removeSnapshot(ctx context.Context, id string) error {

	zap.S().Named("inspector_service").Infow("removing VM snapshot", "vmId", id)

	removeSnapReq := vmware.RemoveSnapshotRequest{
		VmId:         id,
		SnapshotName: models.InspectionSnapshotName,
		Consolidate:  true,
	}

	if err := c.operator.RemoveSnapshot(ctx, removeSnapReq); err != nil {
		zap.S().Named("inspector_service").Errorw("failed to remove VM snapshot", "vmId", id, "error", err)
		return err
	}

	zap.S().Named("inspector_service").Infow("VM snapshot removed", "vmId", id)

	return nil
}
