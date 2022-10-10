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
	"regexp"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/mosajjal/dnsmonster/capture"
	"github.com/mosajjal/dnsmonster/util"
	"github.com/pkg/profile"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func handleInterrupt(ctx context.Context) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	if runtime.GOOS == "linux" {
		signal.Notify(c, syscall.SIGPIPE)
	}
	go func() {
		<-c
		log.Infof("SIGINT Received. Stopping capture...")
		go util.GlobalCancel()
		go ctx.Done()
		<-time.After(2 * time.Second)
		log.Fatal("emergency exit")
		os.Exit(1)
	}()
}

func main() {

	for i := range os.Args {

		var re = regexp.MustCompile(`(?m)--(\w+)`)
		os.Args[i] = (re.ReplaceAllStringFunc(os.Args[i], func(m string) string {
			return strings.ToLower(m)
		}))

	}

	var ctx context.Context
	ctx, util.GlobalCancel = context.WithCancel(context.Background())
	g, _ := errgroup.WithContext(ctx)
	// process and handle flags
	util.ProcessFlags(ctx)

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
	handleInterrupt(ctx)

	// set up capture
	g.Go(func() error { capture.GlobalCaptureConfig.CheckFlagsAndStart(ctx); return nil })

	// Set up output dispatch
	g.Go(func() error { return setupOutputs(ctx, capture.GlobalCaptureConfig.GetResultChannel()) })

	// block until capture and output finish their loop, in order to exit cleanly
	g.Wait()
	// <-time.After(2 * time.Second)
	// print metrics for one last time before exiting the program
	metricsJSON, _ := json.Marshal(metrics.DefaultRegistry.GetAll())
	os.Stderr.WriteString(fmt.Sprintf("metrics: %s\n", metricsJSON))
}
