/* {{{ Copyright (C) 2022 Ali Mosajjal
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. }}} */

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

	"github.com/mosajjal/dnsmonster/internal/capture"
	"github.com/mosajjal/dnsmonster/internal/util"
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
		<-time.After(4 * time.Second)
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
	var c chan util.DNSResult
	for {
		c = capture.GlobalCaptureConfig.GetResultChannel()
		if c == nil {
			time.Sleep(10 * time.Millisecond)
			continue
		} else {
			break
		}
	}

	g.Go(func() error { return setupOutputs(ctx, &c) })

	// block until capture and output finish their loop, in order to exit cleanly
	g.Wait()
	<-time.After(1 * time.Second)
	// print metrics for one last time before exiting the program
	metricsJSON, _ := json.Marshal(metrics.DefaultRegistry.GetAll())
	os.Stderr.WriteString(fmt.Sprintf("metrics: %s\n", metricsJSON))
}

// vim: foldmethod=marker
