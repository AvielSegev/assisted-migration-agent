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
	var inspectionState InspectionStatusState
	switch vm.InspectionState {
	case "completed":
		inspectionState = InspectionStatusStateCompleted
	case "running":
		inspectionState = InspectionStatusStateRunning
	case "error":
		inspectionState = InspectionStatusStateError
	default:
		inspectionState = InspectionStatusStatePending
	}

	apiVM := VM{
		Id:           vm.ID,
		Name:         vm.Name,
		Cluster:      vm.Cluster,
		Datacenter:   vm.Datacenter,
		DiskSize:     vm.DiskSize,
		Memory:       vm.Memory,
		VCenterState: vm.State,
		Issues:       vm.Issues,
		Inspection: InspectionStatus{
			State: inspectionState,
		},
	}

	if vm.InspectionError != "" {
		apiVM.Inspection.Error = &vm.InspectionError
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
