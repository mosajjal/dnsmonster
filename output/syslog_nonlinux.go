//go:build !linux
// +build !linux

// This entire file is a dummy one to make sure all our cross platform builds work even if the underlying OS doesn't suppot some of the functionality
// syslog is a Linux-only feature, so we want the relevant function to technically "translate" to something here, which basically returns an error

package output

import (
	"github.com/mosajjal/dnsmonster/types"
	log "github.com/sirupsen/logrus"
)

var syslog struct {
	Writer      bool
	Dial        bool
	LOG_WARNING bool
	LOG_DAEMON  bool
}

func SyslogOutput(sysConfig types.SyslogConfig) {
	log.Error("No Syslog is supported in Windows")
}
