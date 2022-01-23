// dnsmonster implements a packet sniffer for DNS traffic. It can accept traffic from a pcap file or a live interface,
// and can be used to index and store hundreds of thousands of queries per second. It aims to be scalable and easy to use, and help
// security teams to understand the details about an enterprise's DNS traffic. It does not aim to breach
// the privacy of the end-users, with the ability to mask source IP, making the data potentially untraceable.
// the project has been developed as a monolith, but now it has been split into multiple modules.
// the modules will be expanded and improved over time, and will have a robust API to be used by other projects.
package main

import (
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/mosajjal/dnsmonster/capture"
	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	"github.com/pkg/profile"
	"github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

var resultChannel = make(chan types.DNSResult, util.GeneralFlags.ResultChannelSize)

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
			util.ErrorHandler(err)
			runtime.GC() // get up-to-date statistics

			err = pprof.Lookup("heap").WriteTo(f, 0)
			util.ErrorHandler(err)
			f.Close()
		}()
	}

	// Setup our output channels
	setupOutputs()

	// todo: this needs to be its own file with configurable output formats and endpoints (stdout, file, syslog, prometheus, etc)
	go metrics.Log(metrics.DefaultRegistry, util.GeneralFlags.PrintStatsDelay, log.StandardLogger())

	// Start listening if we're using pcap or afpacket
	if util.CaptureFlags.DnstapSocket == "" {
		capturer := capture.NewDNSCapturer(capture.CaptureOptions{
			DevName:                      util.CaptureFlags.DevName,
			UseAfpacket:                  util.CaptureFlags.UseAfpacket,
			PcapFile:                     util.CaptureFlags.PcapFile,
			Filter:                       util.CaptureFlags.Filter,
			Port:                         uint16(util.CaptureFlags.Port),
			GcTime:                       util.GeneralFlags.GcTime,
			ResultChannel:                resultChannel,
			PacketHandlerCount:           util.CaptureFlags.PacketHandlerCount,
			PacketChannelSize:            util.CaptureFlags.PacketChannelSize,
			TCPHandlerCount:              util.GeneralFlags.TcpHandlerCount,
			TCPAssemblyChannelSize:       util.GeneralFlags.TcpAssemblyChannelSize,
			TCPResultChannelSize:         util.GeneralFlags.TcpResultChannelSize,
			IPDefraggerChannelSize:       util.GeneralFlags.DefraggerChannelSize,
			IPDefraggerReturnChannelSize: util.GeneralFlags.DefraggerChannelReturnSize,
			NoEthernetframe:              util.CaptureFlags.NoEthernetframe,
		})

		capturer.Start()
		// Wait for the output to finish
		log.Info("Exiting..")

	} else { // dnstap si totally different, hence only the result channel is being pushed to it
		capture.StartDNSTap(resultChannel)
	}
}
