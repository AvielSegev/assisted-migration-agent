package v1

import (
	"github.com/kubev2v/assisted-migration-agent/internal/models"
)

func (a *AgentStatus) FromModel(m models.AgentStatus) {
	a.ConsoleConnection = AgentStatusConsoleConnection(m.Console.Current)
	a.Mode = AgentStatusMode(m.Console.Target)
}

func (c *CollectorStatus) FromModel(m models.CollectorStatus) {
	switch m.State {
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
	c.HasCredentials = m.HasCredentials
	if m.Error != "" {
		c.Error = &m.Error
	}
}
