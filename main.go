// dnsmonster implements a packet sniffer for DNS traffic. It can accept traffic from a pcap file or a live interface,
// and can be used to index and store hundreds of thousands of queries per second. It aims to be scalable and easy to use, and help
// security teams to understand the details about an enterprise's DNS traffic. It does not aim to breach
// the privacy of the end-users, with the ability to mask source IP, making the data potentially untraceable.
// the project has been developed as a monolith, but now it has been split into multiple modules.
// the modules will be expanded and improved over time, and will have a robust API to be used by other projects.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
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
	if runtime.GOOS == "linux" {
		signal.Notify(c, syscall.SIGPIPE)
	}
	go func() {
		for range c {
			for {
				log.Infof("SIGINT Received. Stopping capture...")
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				go func() {
					for i := 0; i < runtime.NumGoroutine(); i++ {
						*util.GeneralFlags.GetExit() <- true
					}
					for range ctx.Done() {
						fmt.Println("Canceled by timeout")
						return
					}
				}()
				<-time.After(2 * time.Second)
				log.Fatal("emergency exit")
				return
			}
		}
	}()
}

func main() {

	// convert all argv to lowercase
	for i := range os.Args {
		os.Args[i] = strings.ToLower(os.Args[i])
	}

	// process and handle flags
	util.ProcessFlags()

	// debug and profile options
	runtime.GOMAXPROCS(util.GeneralFlags.Gomaxprocs)
	if util.GeneralFlags.Cpuprofile != "" {
		defer profile.Start(profile.CPUProfile).Stop()
	}
	// Setup the memory profile if requested
	if util.GeneralFlags.Memprofile != "" {
		defer profile.Start(profile.MemProfile).Stop()
	}

	// Setup SIGINT handling
	handleInterrupt()

	// set up capture
	capture.GlobalCaptureConfig.CheckFlagsAndStart()

	// Set up output dispatch
	setupOutputs(capture.GlobalCaptureConfig.GetResultChannel())

	// block until capture and output finish their loop, in order to exit cleanly
	util.GeneralFlags.GetWg().Wait()
	<-time.After(2 * time.Second)
	// print metrics for one last time before exiting the program
	metricsJson, _ := json.Marshal(metrics.DefaultRegistry.GetAll())
	os.Stderr.WriteString(fmt.Sprintf("metrics: %s", metricsJson))
}
