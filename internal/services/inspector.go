package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/vmware/govmomi"

	"github.com/kubev2v/assisted-migration-agent/pkg/vmware"

	"go.uber.org/zap"

	"github.com/kubev2v/assisted-migration-agent/internal/models"
	srvErrors "github.com/kubev2v/assisted-migration-agent/pkg/errors"
	"github.com/kubev2v/assisted-migration-agent/pkg/scheduler"
)

type (
	InspectionPipeline    = WorkPipeline[models.InspectionStatus, models.InspectionResult]
	InspectionWorkBuilder func(id string) []models.WorkUnit[models.InspectionStatus, models.InspectionResult]
)

type InspectorService struct {
	scheduler *scheduler.Scheduler[models.InspectionResult]

	buildFn   InspectionWorkBuilder
	pipelines map[string]*InspectionPipeline

	status models.InspectorStatus

	mu sync.Mutex

	done chan any

	vsphereClient *govmomi.Client
	cancel        context.CancelFunc
	cred          *models.Credentials

	operator vmware.VMOperator // needs to be initialized
}

// NewInspectorService creates a new InspectorService with the default vmware builder.
func NewInspectorService() *InspectorService {
	return &InspectorService{
		status: models.InspectorStatus{State: models.InspectorStateReady},
	}
}

// GetStatus returns the current inspector status.
func (c *InspectorService) GetStatus() models.InspectorStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.status
}

// GetVmStatus returns the current vm inspection status.
func (c *InspectorService) GetVmStatus(ctx context.Context, id string) (models.InspectionStatus, error) {
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

func (c *InspectorService) Start(ctx context.Context, vmIDs []string, cred *models.Credentials) error {
	if c.IsBusy() {
		return fmt.Errorf("deep inspector already in progress")
	}

	c.setState(models.InspectorStateInitiating)
	zap.S().Infow("starting inspector", "vmCount", len(vmIDs))

	vClient, err := vmware.NewVsphereClient(ctx, cred.URL, cred.Username, cred.Password, true)
	if err != nil {
		zap.S().Named("inspector_service").Errorw("failed to connect to vSphere", "error", err)
		c.setErrorStatus(err)
		return err
	}

	zap.S().Named("inspector_service").Info("vSphere connection established")

	c.vsphereClient = vClient
	c.cred = cred

	sched := scheduler.NewScheduler[models.InspectionResult](1)
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

	c.setState(models.InspectorStateRunning)

	runCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.done = make(chan any)

	go c.run(runCtx, c.done)

	return nil
}

func (c *InspectorService) startVmPipeline(id string) *InspectionPipeline {
	pipeline := NewWorkPipeline(models.InspectionStatus{State: models.InspectionStatePending}, c.scheduler, c.buildFn(id))
	if err := pipeline.Start(); err != nil {
		c.pipelines[id].state = WorkPipelineStatus[models.InspectionStatus, models.InspectionResult]{
			State: models.InspectionStatus{State: models.InspectionStateError, Error: err},
			Err:   err,
		}
	}

	return pipeline
}

func (c *InspectorService) Add(vmIDs []string) error {
	if !c.IsBusy() {
		return srvErrors.NewInspectorNotRunningError()
	}

	if c.GetStatus().State == models.InspectorStateCanceling {
		return fmt.Errorf("inspector canceling works")
	}

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

func (c *InspectorService) Stop(ctx context.Context) error {
	if !c.IsBusy() {
		return srvErrors.NewInspectorNotRunningError()
	}

	c.setState(models.InspectorStateCanceling)

	// Cancel pending VMs before waiting for the goroutine to finish
	// This ensures VMs are marked as canceled even if the goroutine finishes quickly
	if err := c.CancelVmsInspection(ctx); err != nil {
		return fmt.Errorf("failed to update inspection table: %w", err)
	}

	c.mu.Lock()
	cancel := c.cancel
	done := c.done
	c.mu.Unlock()

	if cancel != nil {
		cancel()
	}

	if done != nil {
		<-done
	}

	c.setState(models.InspectorStateCanceled)
	zap.S().Info("inspector stopped")

	return nil
}

func (c *InspectorService) CancelVmsInspection(ctx context.Context, vmIDs ...string) error {
	if !c.IsBusy() {
		return srvErrors.NewInspectorNotRunningError()
	}

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

func (c *InspectorService) IsBusy() bool {
	switch c.GetStatus().State {
	case models.InspectorStateReady, models.InspectorStateCompleted, models.InspectorStateError, models.InspectorStateCanceled:
		return false
	default:
		return true
	}
}

func (c *InspectorService) WithWorkUnitsBuilder(builder InspectionWorkBuilder) *InspectorService {
	c.buildFn = builder
	return c
}

func (c *InspectorService) buildInspectionWorkUnits(id string) []models.WorkUnit[models.InspectionStatus, models.InspectionResult] {
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

func (c *InspectorService) validate(ctx context.Context, id string) error {
	return c.operator.ValidatePrivileges(ctx, id, models.RequiredPrivileges)
}

func (c *InspectorService) createSnapshot(ctx context.Context, id string) error {
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

func (c *InspectorService) inspect(id string) error {
	return nil
}

func (c *InspectorService) save(ctx context.Context, id string) error {
	return nil
}

func (c *InspectorService) removeSnapshot(ctx context.Context, id string) error {

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

func (c *InspectorService) run(ctx context.Context, done chan any) {
	defer close(done)

	c.mu.Lock()
	pipelines := make([]*InspectionPipeline, 0, len(c.pipelines))
	for _, p := range c.pipelines {
		pipelines = append(pipelines, p)
	}
	c.mu.Unlock()

	for _, p := range pipelines {
		for p.IsRunning() {
			select {
			case <-ctx.Done():
				return
			default:
				time.Sleep(10 * time.Millisecond)
			}
		}
	}

	// All pipelines finished; set completed unless we were canceled
	c.mu.Lock()
	state := c.status.State
	c.mu.Unlock()
	if state == models.InspectorStateRunning {
		c.setState(models.InspectorStateCompleted)
	}
}

//func (c *InspectorService) closeVsphereClient(ctx context.Context) {
//	c.mu.Lock()
//	defer c.mu.Unlock()
//	if c.vsphereClient != nil {
//		_ = c.vsphereClient.Logout(ctx)
//		c.vsphereClient = nil
//	}
//}

func (c *InspectorService) setState(s models.InspectorState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status.State = s
	c.status.Error = nil
}

func (c *InspectorService) setErrorStatus(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.status = models.InspectorStatus{
		State: models.InspectorStateError,
		Error: err,
	}
}
