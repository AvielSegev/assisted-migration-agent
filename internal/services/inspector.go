package services

import (
	"context"
	"errors"
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
	InspectorPipeline    = WorkPipeline[models.InspectorStatus, models.InspectorResult]
	InspectorWorkBuilder func(vmIds []string) []models.WorkUnit[models.InspectorStatus, models.InspectorResult]
)

type InspectorService struct {
	scheduler *scheduler.Scheduler[models.InspectorResult]

	buildFn  InspectorWorkBuilder
	pipeline *InspectorPipeline

	mu sync.Mutex

	cred          *models.Credentials
	vsphereClient *govmomi.Client

	inspectionSvc *InspectionService
}

// NewInspectorService creates a new InspectorService with the default vmware builder.
func NewInspectorService() *InspectorService {
	return &InspectorService{}
}

// GetStatus returns the current inspector status.
func (i *InspectorService) GetStatus() models.InspectorStatus {
	i.mu.Lock()
	pipeline := i.pipeline
	i.mu.Unlock()

	if pipeline == nil {
		return models.InspectorStatus{State: models.InspectorStateReady}
	}

	state := pipeline.State()
	if state.Err != nil {
		if !errors.Is(state.Err, errPipelineStopped) {
			return models.InspectorStatus{State: models.InspectorStateCanceled, Error: state.Err}
		}
		return models.InspectorStatus{State: models.InspectorStateError, Error: state.Err}
	}

	if pipeline.IsRunning() {
		return state.State
	}

	return models.InspectorStatus{State: models.InspectorStateCompleted}
}

func (i *InspectorService) Start(ctx context.Context, vmIDs []string, cred *models.Credentials) error {
	if i.IsBusy() {
		return srvErrors.NewInspectionInProgressError()
	}

	zap.S().Infow("starting inspector", "vmCount", len(vmIDs))

	vClient, err := vmware.NewVsphereClient(ctx, cred.URL, cred.Username, cred.Password, true)
	if err != nil {
		zap.S().Named("inspector_service").Errorw("failed to connect to vSphere", "error", err)
		return err
	}

	zap.S().Named("inspector_service").Info("vSphere connection established")

	i.vsphereClient = vClient
	i.cred = cred

	sched := scheduler.NewScheduler[models.InspectorResult](1)
	i.scheduler = sched

	if i.inspectionSvc == nil {
		i.inspectionSvc = NewInspectionService()
	}

	if i.buildFn == nil {
		i.buildFn = i.buildInspectorWorkUnits
	}

	i.pipeline = NewWorkPipeline(models.InspectorStatus{State: models.InspectorStateInitiating}, sched, i.buildFn(vmIDs))

	if err := i.pipeline.Start(); err != nil {
		i.pipeline = nil
		i.scheduler.Close()
		i.scheduler = nil
		return srvErrors.NewInspectionInProgressError()
	}

	return nil
}

func (i *InspectorService) Add(vmIDs []string) error {
	if !i.IsBusy() {
		return srvErrors.NewInspectorNotRunningError()
	}

	return i.inspectionSvc.Add(vmIDs)
}

func (i *InspectorService) Stop(ctx context.Context) error {
	// implement me
	return nil
}

func (i *InspectorService) CancelVmsInspection(ctx context.Context, vmIDs ...string) error {
	if !i.IsBusy() {
		return srvErrors.NewInspectorNotRunningError()
	}

	return i.inspectionSvc.CancelVmsInspection(ctx, vmIDs...)
}

func (i *InspectorService) IsBusy() bool {
	if i.pipeline != nil && i.pipeline.IsRunning() {
		return true
	}

	return false
}

func (i *InspectorService) WithInspectionService(svc *InspectionService) *InspectorService {
	i.inspectionSvc = svc
	return i
}

func (i *InspectorService) WithWorkUnitsBuilder(builder InspectorWorkBuilder) *InspectorService {
	i.buildFn = builder
	return i
}

func (i *InspectorService) buildInspectorWorkUnits(ids []string) []models.WorkUnit[models.InspectorStatus, models.InspectorResult] {
	return []models.WorkUnit[models.InspectorStatus, models.InspectorResult]{
		{
			Status: func() models.InspectorStatus {
				return models.InspectorStatus{State: models.InspectorStateInitiating}
			},
			Work: func(ctx context.Context, result models.InspectorResult) (models.InspectorResult, error) {
				return result, i.initInspectionPipelines(ids)
			},
		},
		{
			Status: func() models.InspectorStatus {
				return models.InspectorStatus{State: models.InspectorStateRunning}
			},
			Work: func(ctx context.Context, result models.InspectorResult) (models.InspectorResult, error) {
				i.run(ctx)
				return result, nil
			},
		},
		{
			Status: func() models.InspectorStatus {
				return models.InspectorStatus{State: models.InspectorStateRunning}
			},
			Work: func(ctx context.Context, result models.InspectorResult) (models.InspectorResult, error) {
				return result, i.closeVsphereClient(ctx)
			},
		},
	}
}

func (i *InspectorService) initInspectionPipelines(ids []string) error {
	return i.inspectionSvc.Start(vmware.NewVMManager(i.vsphereClient, i.cred.Username), ids)
}

func (i *InspectorService) run(ctx context.Context) {
	i.inspectionSvc.mu.Lock()
	pipelines := make([]*InspectionPipeline, 0, len(i.inspectionSvc.pipelines))
	for _, p := range i.inspectionSvc.pipelines {
		pipelines = append(pipelines, p)
	}
	i.inspectionSvc.mu.Unlock()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for _, p := range pipelines {
		for p.IsRunning() {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// wait
			}
		}
	}
}

func (i *InspectorService) closeVsphereClient(ctx context.Context) error {
	i.mu.Lock()
	defer func() {
		i.vsphereClient = nil
		i.mu.Unlock()
	}()

	if i.vsphereClient != nil {
		return i.vsphereClient.Logout(ctx)
	}

	return nil
}
