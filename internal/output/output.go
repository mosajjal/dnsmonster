package output

import (
	"context"
	"time"

	"github.com/mosajjal/dnsmonster/internal/util"
)

// BaseConfig contains common configuration fields for all outputs
type BaseConfig struct {
	Enabled      bool          `mapstructure:"enabled"`
	BatchSize    uint          `mapstructure:"batch_size"`
	BatchDelay   time.Duration `mapstructure:"batch_delay"`
	FilterMode   string        `mapstructure:"filter_mode"` // none, skipdomains, allowdomains, both
	MaxQueueSize int           `mapstructure:"max_queue_size"`
}

// FilterModeFromString converts string filter mode to integer
func (b *BaseConfig) FilterModeFromString() uint {
	switch b.FilterMode {
	case "skipdomains":
		return 2
	case "allowdomains":
		return 3
	case "both":
		return 4
	default:
		return 1
	}
}

// OutputConfig is a common interface for all output configs.
type OutputConfig interface {
	Initialize(ctx context.Context) error
	OutputChannel() chan util.DNSResult
	Close()
	IsEnabled() bool
}
