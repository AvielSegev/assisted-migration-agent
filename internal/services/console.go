package services

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/cenkalti/backoff/v5"
	"github.com/google/uuid"

	"github.com/kubev2v/assisted-migration-agent/internal/config"
	"github.com/kubev2v/assisted-migration-agent/internal/models"
	"github.com/kubev2v/assisted-migration-agent/internal/store"
	"github.com/kubev2v/assisted-migration-agent/pkg/console"
	"github.com/kubev2v/assisted-migration-agent/pkg/errors"
	"github.com/kubev2v/assisted-migration-agent/pkg/scheduler"
)

// errInventoryNotChanged is a sentinel error used to signal that no inventory
// request was made because the inventory hasn't changed.
var errInventoryNotChanged = stderrors.New("inventory not changed")

type Collector interface {
	GetStatus() models.CollectorStatus
}

type Console struct {
	updateInterval      time.Duration
	agentID             uuid.UUID
	sourceID            uuid.UUID
	version             string
	state               models.ConsoleStatus
	mu                  sync.Mutex
	scheduler           *scheduler.Scheduler
	client              *console.Client
	close               chan any
	collector           Collector
	inventoryLastHash   string // holds the hash of the last sent inventory
	store               *store.Store
	legacyStatusEnabled bool
	fatalStopped        bool
}

func NewConsoleService(cfg config.Agent, s *scheduler.Scheduler, client *console.Client, collector Collector, st *store.Store) *Console {
	targetStatus, err := models.ParseConsoleStatusType(cfg.Mode)
	if err != nil {
		targetStatus = models.ConsoleStatusDisconnected
	}

	defaultStatus := models.ConsoleStatus{
		Current: models.ConsoleStatusDisconnected,
		Target:  targetStatus,
	}

	config, err := st.Configuration().Get(context.Background())
	if err == nil && config.AgentMode == models.AgentModeConnected {
		defaultStatus.Target = models.ConsoleStatusConnected
	}
	c := newConsoleService(cfg, s, client, collector, st, defaultStatus)

	if defaultStatus.Target == models.ConsoleStatusConnected {
		go c.run()
	}

	zap.S().Named("console_service").Infow("agent mode", "current", defaultStatus.Current, "target", defaultStatus.Target)

	return c
}

func newConsoleService(cfg config.Agent, s *scheduler.Scheduler, client *console.Client, collector Collector, store *store.Store, defaultStatus models.ConsoleStatus) *Console {
	return &Console{
		updateInterval:      cfg.UpdateInterval,
		agentID:             uuid.MustParse(cfg.ID),
		sourceID:            uuid.MustParse(cfg.SourceID),
		version:             cfg.Version,
		scheduler:           s,
		state:               defaultStatus,
		client:              client,
		close:               make(chan any),
		store:               store,
		collector:           collector,
		legacyStatusEnabled: cfg.LegacyStatusEnabled,
	}
}

func (c *Console) GetMode(ctx context.Context) (models.AgentMode, error) {
	config, err := c.store.Configuration().Get(ctx)
	if err != nil {
		return models.AgentModeDisconnected, err
	}
	return config.AgentMode, nil
}

func (c *Console) SetMode(ctx context.Context, mode models.AgentMode) error {
	prevMode, _ := c.GetMode(ctx)

	if prevMode == mode {
		return nil
	}

	if c.isFatalStopped() {
		// TODO: Should we change the status in db to disconnected in this case to prevent user
		// from restarting the agent and try to connect to console again?
		// If we don't save the disconnected state, at each restart,if the mode is connected, the agent
		// will do another request and exit again.
		return errors.NewModeConflictError("console reporting stopped after receiving 401/410 from the server")
	}

	err := c.store.Configuration().Save(ctx, &models.Configuration{AgentMode: mode})
	if err != nil {
		return err
	}

	switch mode {
	case models.AgentModeConnected:
		c.mu.Lock()
		c.state.Target = models.ConsoleStatusConnected
		c.mu.Unlock()
		if prevMode != models.AgentModeConnected {
			zap.S().Debugw("starting run loop for connected mode")
			go c.run()
		}
	case models.AgentModeDisconnected:
		c.mu.Lock()
		c.state.Target = models.ConsoleStatusDisconnected
		c.mu.Unlock()
		if prevMode == models.AgentModeConnected {
			zap.S().Debugw("stopping run loop for disconnected mode")
			c.close <- struct{}{}
		}
	}

	zap.S().Named("console_service").Infow("agent mode changed", "mode", mode)
	return nil
}

func (c *Console) Status() models.ConsoleStatus {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

func (c *Console) setError(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Error = err
}

func (c *Console) clearError() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.Error = nil
}

func (c *Console) setFatalStopped() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fatalStopped = true
}

func (c *Console) isFatalStopped() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.fatalStopped
}

// run is the main loop that sends status and inventory updates to the console.
//
// On each iteration:
//  1. Dispatch status and inventory updates (combined in single call) and block until complete.
//  2. Handle errors (fatal errors stop the loop, transient errors trigger backoff).
//  3. Wait for next tick or close signal.
//
// Fatal errors (stop the loop, no retry):
//   - SourceGoneError (410): The source was deleted from the console. No point in sending updates.
//   - AgentUnauthorizedError (401): Invalid or expired JWT. Agent cannot authenticate.
//
// Transient errors are logged and stored in status.Error, but the loop continues.
// If inventory hasn't changed, the error state is preserved (not cleared or set).
//
// Backoff:
// When the server is unreachable (transient errors), exponential backoff is used to avoid
// hammering the server. On error, requests are skipped until the backoff interval expires.
// The interval grows exponentially from updateInterval up to 60 seconds max. On success,
// the backoff resets to allow immediate requests on the next tick.
func (c *Console) run() {
	tick := time.NewTicker(c.updateInterval)
	defer func() {
		tick.Stop()
		zap.S().Named("console_service").Info("service stopped sending requests to console.rh.com")
	}()

	// use exponential backoff if server is unreachable.
	nextAllowedTime := time.Time{}
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = c.updateInterval
	b.MaxInterval = 60 * time.Second // Don't wait longer than 60s

	for {
		select {
		case <-tick.C:
		case <-c.close:
			return
		}

		now := time.Now()

		if !now.After(nextAllowedTime) {
			continue
		}

		future := c.dispatch()

		select {
		case result := <-future.C():
			if result.Err != nil {
				if stderrors.Is(result.Err, errInventoryNotChanged) {
					goto backoff
				}
				c.setError(result.Err)
				switch result.Err.(type) {
				case *errors.SourceGoneError:
					zap.S().Named("console_service").Error("source is gone..stop sending requests")
					c.setFatalStopped()
					return
				case *errors.AgentUnauthorizedError:
					zap.S().Named("console_service").Error("agent not authenticated..stop sending requests")
					c.setFatalStopped()
					return
				default:
					zap.S().Named("console_service").Errorw("failed to dispatch to console", "error", result.Err)
				}
			} else {
				c.clearError()
			}
		case <-c.close:
			future.Stop()
			return
		}

	backoff:
		// if there's an error activate backoff, otherwise reset it
		if c.Status().Error != nil {
			nextAllowedTime = now.Add(b.NextBackOff())
			zap.S().Debugw("set backoff", "next-allowed-time", nextAllowedTime)
		} else {
			b.Reset()
			nextAllowedTime = time.Time{}
		}
	}
}

func (c *Console) dispatch() *models.Future[models.Result[any]] {
	return c.scheduler.AddWork(func(ctx context.Context) (any, error) {
		collectorStatus := models.CollectorStatusType(c.collector.GetStatus().State)
		if c.legacyStatusEnabled {
			switch c.collector.GetStatus().State {
			case models.CollectorStateReady:
				collectorStatus = models.CollectorLegacyStatusWaitingForCredentials
			case models.CollectorStateConnecting, models.CollectorStateCollecting:
				collectorStatus = models.CollectorLegacyStatusCollecting
			case models.CollectorStateCollected:
				collectorStatus = models.CollectorLegacyStatusCollected
			}
		}

		if err := c.client.UpdateAgentStatus(ctx, c.agentID, c.sourceID, c.version, collectorStatus); err != nil {
			return nil, err
		}

		// dispatch inventory
		data, changed, err := c.getInventoryIfChanged(ctx)
		if err != nil {
			return nil, err
		}

		if !changed {
			return nil, errInventoryNotChanged
		}

		inventory := models.Inventory{}
		if err := json.Unmarshal(data, &inventory); err != nil {
			return nil, err
		}

		if err := c.client.UpdateSourceStatus(ctx, c.sourceID, c.agentID, inventory); err != nil {
			return nil, err
		}

		zap.S().Named("console_service").Debugw("inventory updated", "hash", c.inventoryLastHash)

		return struct{}{}, nil
	})
}

func (c *Console) getInventoryIfChanged(ctx context.Context) ([]byte, bool, error) {
	inventory, err := c.store.Inventory().Get(ctx)
	if err != nil {
		return nil, false, err
	}

	data, err := json.Marshal(inventory)
	if err != nil {
		return nil, false, fmt.Errorf("failed to marshal inventory %v", err)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256(data))
	if hash == c.inventoryLastHash {
		return nil, false, nil
	}

	c.inventoryLastHash = hash
	return data, true, nil
}
