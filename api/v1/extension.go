package v1

import (
	"github.com/kubev2v/assisted-migration-agent/internal/models"
)

func (a *AgentStatus) FromModel(m models.AgentStatus) {
	a.ConsoleConnection = AgentStatusConsoleConnection(m.Console.Current)
	a.Mode = AgentStatusMode(m.Console.Target)
}

// NewVMFromModel converts a models.VM to an API VM.
func NewVMFromModel(vm models.VM) VM {
	apiVM := VM{
		Id:           vm.ID,
		Name:         vm.Name,
		Cluster:      vm.Cluster,
		Datacenter:   vm.Datacenter,
		DiskSize:     vm.DiskSize,
		Memory:       vm.Memory,
		VCenterState: vm.State,
		Issues:       vm.Issues,
		Inspection:   NewInspectionStatus(vm.InspectionState, vm.InspectionError),
	}

	return apiVM
}

func NewCollectorStatus(status models.CollectorStatus) CollectorStatus {
	var c CollectorStatus

	switch status.State {
	case models.CollectorStateReady:
		c.Status = CollectorStatusStatusReady
	case models.CollectorStateConnecting:
		c.Status = CollectorStatusStatusConnecting
	case models.CollectorStateConnected:
		c.Status = CollectorStatusStatusConnected
	case models.CollectorStateCollecting:
		c.Status = CollectorStatusStatusCollecting
	case models.CollectorStateCollected:
		c.Status = CollectorStatusStatusCollected
	case models.CollectorStateError:
		c.Status = CollectorStatusStatusError
	default:
		c.Status = CollectorStatusStatusReady
	}

	if status.Error != nil {
		e := status.Error.Error()
		c.Error = &e
	}

	return c
}

func NewCollectorStatusWithError(status models.CollectorStatus, err error) CollectorStatus {
	c := NewCollectorStatus(status)
	if err != nil {
		errStr := err.Error()
		c.Error = &errStr
	}
	return c
}

func NewInspectorStatus(status models.InspectorStatus) InspectorStatus {
	var c InspectorStatus

	switch status.State {
	case models.InspectorStateReady:
		c.State = InspectorStatusStateReady
	case models.InspectorStateRunning, models.InspectorStateConnecting:
		c.State = InspectorStatusStateRunning
	case models.InspectorStateCancelled:
		c.State = InspectorStatusStateCanceled
	case models.InspectorStateDone:
		c.State = InspectorStatusStateDone
	case models.InspectorStateError:
		c.State = InspectorStatusStateError
	default:
		c.State = InspectorStatusStateReady
	}

	if status.Error != nil {
		e := status.Error.Error()
		c.Error = &e
	}

	return c
}

func NewInspectionStatus(state string, err string) InspectionStatus {
	var c InspectionStatus
	switch state {
	case models.InspectionStatePending.Value():
		c.State = InspectionStatusStatePending
	case models.InspectionStateRunning.Value():
		c.State = InspectionStatusStateRunning
	case models.InspectionStateCanceled.Value():
		c.State = InspectionStatusStateCanceled
	case models.InspectionStateCompleted.Value():
		c.State = InspectionStatusStateCompleted
	case models.InspectionStateError.Value():
		c.State = InspectionStatusStateError
	default:
		c.State = InspectionStatusStateNotFound
	}

	if err != "" {
		c.Error = &err
	}

	return c
}

func FlatStatus(s models.InspectionStatus) (string, string) {
	var errStr string

	if s.Error != nil {
		errStr = s.Error.Error()
	}

	return s.State.Value(), errStr
}
