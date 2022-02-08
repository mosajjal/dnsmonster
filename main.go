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
	"runtime/pprof"
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
				<-time.After(10 * time.Second)
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
		go func() {
			time.Sleep(120 * time.Second)
			log.Warn("Writing memory profile")
			f, err := os.Create(util.GeneralFlags.Memprofile)
			if err != nil {
				log.Fatal(err)
			}
			runtime.GC() // get up-to-date statistics

			err = pprof.Lookup("heap").WriteTo(f, 0)
			if err != nil {
				log.Fatal(err)
			}
			f.Close()
		}()
	}
	// Setup SIGINT handling
	handleInterrupt()

	// todo: this needs to be its own file with configurable output formats and endpoints (stdout, file, syslog, prometheus, etc)
	go metrics.Log(metrics.DefaultRegistry, util.GeneralFlags.PrintStatsDelay, log.StandardLogger())

	// set up captures
	capture.GlobalCaptureConfig.CheckFlagsAndStart()

	// Setup our output channels
	setupOutputs(capture.GlobalCaptureConfig.GetResultChannel())

	//todo: this could be a better place to handle intrrupts, logrotate and even statsd
	select {}
}
