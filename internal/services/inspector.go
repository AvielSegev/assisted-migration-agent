package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/kubev2v/assisted-migration-agent/internal/models"
	srvErrors "github.com/kubev2v/assisted-migration-agent/pkg/errors"
	"github.com/kubev2v/assisted-migration-agent/pkg/scheduler"
)

type InspectorService struct {
	scheduler *scheduler.Scheduler
	builder   models.InspectorWorkBuilder

	status    models.InspectorStatus
	vmsStatus map[string]models.InspectionStatus

	mu sync.Mutex

	done  chan any
	works chan []models.VmWorkUnit

	cancel context.CancelFunc
}

func NewInspectorService(s *scheduler.Scheduler, bu models.InspectorWorkBuilder) *InspectorService {
	return &InspectorService{
		scheduler: s,
		builder:   bu,
		status:    models.InspectorStatus{State: models.InspectorStateReady},
	}
}

// GetStatus returns the current inspector status.
func (c *InspectorService) GetStatus() models.InspectorStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.status
}

// GetVmStatus returns the current vm inspection status.
func (c *InspectorService) GetVmStatus(moid string) models.InspectionStatus {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.vmsStatus[moid]
}

func (c *InspectorService) Start(ctx context.Context, vmsMoid []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isBusy() {
		return srvErrors.NewInspectionInProgressError()
	}

	zap.S().Infow("starting inspector", "vmCount", len(vmsMoid))

	runCtx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.done = make(chan any)
	c.works = make(chan []models.VmWorkUnit)

	c.status = models.InspectorStatus{State: models.InspectorStateRunning}

	flow := c.builder.WithVms(vmsMoid).Build()
	c.vmsStatus = flow.Inspect.VmsInitialStatus

	go c.run(runCtx, c.done, flow)

	return nil
}

func (c *InspectorService) run(ctx context.Context, done chan any, flow models.InspectorFlow) {
	defer close(done)
	defer close(c.works)
	defer c.finalize()
	defer func() {
		c.mu.Lock()
		if c.done == done {
			c.cancel = nil
			c.done = nil
		}
		c.mu.Unlock()
	}()

	if err := c.DoOneWorkUnit(ctx, flow.Connect); err != nil {
		zap.S().Errorw("inspector failed to connect", "error", err)
		c.setStatus(models.InspectorStatus{
			State: models.InspectorStateError,
			Error: err,
		})
		return
	}

	vmsWorkUnits := flow.Inspect.Works

	for len(vmsWorkUnits) > 0 {
		select {

		case newVmsWork := <-c.works:
			vmsWorkUnits = append(vmsWorkUnits, newVmsWork...)
			for _, vmWorkUnit := range vmsWorkUnits {
				c.setVmStatus(vmWorkUnit.VmMoid,
					models.InspectionStatus{
						State: models.InspectionStatePending,
					})
			}

		default:

			vmMoid := vmsWorkUnits[0].VmMoid
			unit := vmsWorkUnits[0].Work
			vmsWorkUnits = vmsWorkUnits[1:]

			if c.GetVmStatus(vmMoid).State == models.InspectionStateCanceled {
				zap.S().Debugw("skipping canceled VM inspection", "vmMoid", vmMoid)
				continue
			}

			c.setVmStatus(vmMoid, models.InspectionStatus{
				State: models.InspectionStateRunning,
			})

			if err := c.DoOneWorkUnit(ctx, unit); err != nil {
				var e *srvErrors.InspectorWorkError
				switch {
				case errors.As(err, &e):
					zap.S().Warnw("VM inspection failed", "vmMoid", vmMoid, "error", e)
					c.setVmStatus(vmMoid, models.InspectionStatus{
						State: models.InspectionStateError,
						Error: e,
					})
					continue
				default:
					return
				}
			}

			c.setVmStatus(vmMoid, models.InspectionStatus{
				State: models.InspectionStateCompleted,
			})
			zap.S().Debugw("VM inspection completed", "vmMoid", vmMoid)
		}
	}
}

func (c *InspectorService) AddMoreVms(vmsMoid []string) error {
	filtered := make([]string, 0, len(vmsMoid))
	c.mu.Lock()
	for _, moid := range vmsMoid {
		if _, ok := c.vmsStatus[moid]; !ok {
			filtered = append(filtered, moid)
		}
	}
	c.mu.Unlock()

	if len(filtered) == 0 {
		return fmt.Errorf("all vms already sent")
	}

	c.builder.Reset()
	flow := c.builder.WithVms(filtered).Build()

	NewVmsUnits := flow.Inspect.Works

	c.works <- NewVmsUnits

	return nil
}

func (c *InspectorService) finalize() {
	c.builder.Reset()

	doneStatus := models.InspectorStatus{
		State: models.InspectorStateDone,
	}
	c.setStatus(doneStatus)

	zap.S().Info("inspector finished work")
}

func (c *InspectorService) DoOneWorkUnit(ctx context.Context, work models.InspectorWorkUnit) error {
	newStatus := work.Status()

	if newStatus.State != c.GetStatus().State {
		c.setStatus(newStatus)
		zap.S().Debugw("inspector changed state", "state", c.GetStatus().State)
	}

	workFn := work.Work()

	future := c.scheduler.AddWork(func(ctx context.Context) (any, error) {
		return workFn(ctx)
	})

	select {
	case <-ctx.Done():
		future.Stop()
		c.setStatus(models.InspectorStatus{State: models.InspectorStateReady})
		return fmt.Errorf("context done")

	case result := <-future.C():
		if result.Err != nil {
			c.setStatus(models.InspectorStatus{State: models.InspectorStateError, Error: result.Err})
			return srvErrors.NewInspectorWorkError("work finished with error: %s", result.Err.Error())
		}
	}

	return nil
}

func (c *InspectorService) Stop() {
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
}

func (c *InspectorService) setStatus(s models.InspectorStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = s
}

func (c *InspectorService) setVmStatus(vmMoid string, s models.InspectionStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.vmsStatus[vmMoid] = s
}

func (c *InspectorService) CancelVmsInspection(vmsMoid []string) {
	for _, moid := range vmsMoid {
		c.setVmStatus(moid, models.InspectionStatus{
			State: models.InspectionStateCanceled,
		})
	}
}

func (c *InspectorService) CancelAllVmsInspection() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for moid := range c.vmsStatus {
		if c.GetVmStatus(moid).State == models.InspectionStatePending {
			c.setVmStatus(moid, models.InspectionStatus{
				State: models.InspectionStateCanceled,
			})
		}
	}
}

func (c *InspectorService) isBusy() bool {
	// must be protected by the caller
	switch c.status.State {
	case models.InspectorStateReady, models.InspectorStateDone, models.InspectorStateError:
		return false
	default:
		return true
	}
}
