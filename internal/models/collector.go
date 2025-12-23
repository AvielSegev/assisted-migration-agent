package models

// CollectorState represents the current state of the collector.
type CollectorState string

const (
	// CollectorStateReady - credentials saved, waiting for collection request
	CollectorStateReady CollectorState = "ready"
	// CollectorStateConnecting - verifying credentials with vCenter
	CollectorStateConnecting CollectorState = "connecting"
	// CollectorStateConnected - credentials verified
	CollectorStateConnected CollectorState = "connected"
	// CollectorStateCollecting - async collection in progress
	CollectorStateCollecting CollectorState = "collecting"
	// CollectorStateCollected - collection complete (auto-transitions to ready)
	CollectorStateCollected CollectorState = "collected"
	// CollectorStateError - error during connecting or collecting
	CollectorStateError CollectorState = "error"
)

// CollectorStatus holds the current collector state and metadata.
type CollectorStatus struct {
	State          CollectorState
	Error          string
	HasCredentials bool
}
