package v1

import (
	"fmt"

	"github.com/kubev2v/assisted-migration-agent/internal/models"
	"github.com/kubev2v/assisted-migration-agent/internal/services"
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
		DiskSize:     fmt.Sprintf("%dTB", vm.DiskSize),
		Memory:       fmt.Sprintf("%dGB", vm.Memory),
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

// ParseDiskSizeRanges converts API disk size params to service ranges.
func ParseDiskSizeRanges(ranges []GetVMsParamsDisksize) []services.SizeRange {
	var result []services.SizeRange
	for _, r := range ranges {
		switch r {
		case N010TB:
			result = append(result, services.SizeRange{Min: 0, Max: 10})
		case N1120TB:
			result = append(result, services.SizeRange{Min: 11, Max: 20})
		case N2150TB:
			result = append(result, services.SizeRange{Min: 21, Max: 50})
		case N50TB:
			result = append(result, services.SizeRange{Min: 50, Max: 0})
		}
	}
	return result
}

// ParseMemorySizeRanges converts API memory size params to service ranges.
func ParseMemorySizeRanges(ranges []GetVMsParamsMemorysize) []services.SizeRange {
	var result []services.SizeRange
	for _, r := range ranges {
		switch r {
		case N04GB:
			result = append(result, services.SizeRange{Min: 0, Max: 4})
		case N516GB:
			result = append(result, services.SizeRange{Min: 5, Max: 16})
		case N1732GB:
			result = append(result, services.SizeRange{Min: 17, Max: 32})
		case N3364GB:
			result = append(result, services.SizeRange{Min: 33, Max: 64})
		case N65128GB:
			result = append(result, services.SizeRange{Min: 65, Max: 128})
		case N129256GB:
			result = append(result, services.SizeRange{Min: 129, Max: 256})
		case N256GB:
			result = append(result, services.SizeRange{Min: 256, Max: 0})
		}
	}
	return result
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
