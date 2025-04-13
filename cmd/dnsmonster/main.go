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
	"github.com/mosajjal/dnsmonster/internal/config"
	"github.com/mosajjal/dnsmonster/internal/output"
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

func normalizeCmdArgs(args []string) {
	re := regexp.MustCompile(`(?m)--(\w+)`)
	for i := range args {
		args[i] = re.ReplaceAllStringFunc(args[i], func(m string) string {
			return strings.ToLower(m)
		})
	}
}

func main() {
	normalizeCmdArgs(os.Args)

	var ctx context.Context
	ctx, util.GlobalCancel = context.WithCancel(context.Background())
	g, gCtx := errgroup.WithContext(ctx)

	// process and handle flags
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	cfgJson, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal config: %v", err)
	}
	log.Infof("Loaded config:\n%s", string(cfgJson))

	// Integrate cfg into the rest of the application: register outputs from config
	// Elastic output
	if cfg.Outputs.Elastic.OutputType > 0 {
		elasticOutput := output.NewElasticConfig().
			WithOutputType(uint(cfg.Outputs.Elastic.OutputType)).
			WithAddress(cfg.Outputs.Elastic.Address).
			WithOutputIndex(cfg.Outputs.Elastic.OutputIndex).
			WithBatchSize(cfg.Outputs.Elastic.BatchSize).
			WithBatchDelay(cfg.Outputs.Elastic.BatchDelay).
			WithChannelSize(int(util.GeneralFlags.ResultChannelSize))
		if err != nil {
			log.Fatalf("Failed to configure elastic output: %v", err)
		}
		util.GlobalDispatchList = append(util.GlobalDispatchList, elasticOutput)
	}

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
	capture.GlobalCaptureConfig.CheckFlagsAndStart(gCtx)
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
