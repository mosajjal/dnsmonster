// dnsmonster implements a packet sniffer for DNS traffic. It can accept traffic from a pcap file or a live interface,
// and can be used to index and store hundreds of thousands of queries per second. It aims to be scalable and easy to use, and help
// security teams to understand the details about an enterprise's DNS traffic. It does not aim to breach
// the privacy of the end-users, with the ability to mask source IP, making the data potentially untraceable.
// the project has been developed as a monolith, but now it has been split into multiple modules.
// the modules will be expanded and improved over time, and will have a robust API to be used by other projects.
package main

import (
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/mosajjal/dnsmonster/capture"
	"github.com/mosajjal/dnsmonster/util"
	"github.com/pkg/profile"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

func handleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			for {
				log.Infof("SIGINT Received. Stopping capture...")
				util.GeneralFlags.GetExit() <- true
				<-time.After(2 * time.Second)
				log.Fatal("emergency exit")
				return
			}
		}
	}()
}

func main() {

	// process and handle flags
	util.ProcessFlags()

	// debug and profile options
	runtime.GOMAXPROCS(util.GeneralFlags.Gomaxprocs)
	if util.GeneralFlags.Cpuprofile != "" {
		defer profile.Start(profile.CPUProfile).Stop()
	}
	// Setup the memory profile if reuqested
	if util.GeneralFlags.Memprofile != "" {
		defer profile.Start(profile.MemProfile).Stop()
	}

	// Setup SIGINT handling
	handleInterrupt()

	// todo: this needs to be its own file with configurable output formats and endpoints (stdout, file, syslog, prometheus, etc)
	go metrics.Log(metrics.DefaultRegistry, util.GeneralFlags.PrintStatsDelay, log.StandardLogger())

	// set up capture
	capture.GlobalCaptureConfig.CheckFlagsAndStart()

	// Set up output dispatch
	setupOutputs(capture.GlobalCaptureConfig.GetResultChannel())

	// block until capture and output finish their loop, in order to exit cleanly
	util.GeneralFlags.GetWg().Wait()
	<-time.After(2 * time.Second)
	// print metrics for one last time before exiting the program
	metrics.WriteOnce(metrics.DefaultRegistry, log.StandardLogger().Writer())
}
