package collector

// The forklift library creates its own zap logger (via a package-level Factory)
// that writes to stderr independently of our application's logger. Its Error()
// method bypasses level checks, so setting LOG_LEVEL is not enough to suppress
// noisy error output (e.g. "pool not started" during normal collection).
//
// Replacing the factory with a discard builder silences all forklift-internal
// logging while keeping our own zap output unaffected.

import (
	"github.com/go-logr/logr"
	forkliftlogging "github.com/kubev2v/forklift/pkg/lib/logging"
)

func init() {
	forkliftlogging.Factory = &discardLogBuilder{}
}

type discardLogBuilder struct{}

func (b *discardLogBuilder) New() logr.Logger                   { return logr.Discard() }
func (b *discardLogBuilder) V(_ int, _ logr.Logger) logr.Logger { return logr.Discard() }
