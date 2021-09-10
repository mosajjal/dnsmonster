// dnsmonster implements a packet sniffer for DNS traffic. It can accept traffic from a pcap file or a live interface,
// and can be used to index and store thousands of queries per second. It aims to be scalable and easy to use, and help
// security teams to understand the details about an enterprise's DNS traffic. It does not aim to breach
// the privacy of the end-users, with the ability to mask source IP from 1 to 32 bits, making the data potentially untraceable.

package main

import (
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var releaseVersion string = "DEVELOPMENT"

// Ratio numbers used for input sampling
var ratioA int
var ratioB int

// skipDomainList represents the list of skipped domains
var skipDomainList [][]string
var allowDomainList [][]string

var skipDomainMap = make(map[string]bool)
var allowDomainMap = make(map[string]bool)

var skipDomainMapBool = false
var allowDomainMapBool = false

var clickhouseResultChannel = make(chan DNSResult, generalOptions.ResultChannelSize)
var kafkaResultChannel = make(chan DNSResult, generalOptions.ResultChannelSize)
var elasticResultChannel = make(chan DNSResult, generalOptions.ResultChannelSize)
var splunkResultChannel = make(chan DNSResult, generalOptions.ResultChannelSize)
var stdoutResultChannel = make(chan DNSResult, generalOptions.ResultChannelSize)
var fileResultChannel = make(chan DNSResult, generalOptions.ResultChannelSize)
var syslogResultChannel = make(chan DNSResult, generalOptions.ResultChannelSize)
var resultChannel = make(chan DNSResult, generalOptions.ResultChannelSize)

func main() {
	flagsProcess()
	checkFlags()
	runtime.GOMAXPROCS(generalOptions.Gomaxprocs)
	if generalOptions.Cpuprofile != "" {
		log.Warn("Writing CPU profile")
		f, err := os.Create(generalOptions.Cpuprofile)
		errorHandler(err)
		err = pprof.StartCPUProfile(f)
		errorHandler(err)
		defer pprof.StopCPUProfile()
	}

	// load the skipDomainFile if exists
	if generalOptions.SkipDomainsFile != "" {
		if skipDomainMapBool {
			skipDomainMap = loadDomainsToMap(generalOptions.SkipDomainsFile)
		} else {
			skipDomainList = loadDomainsToList(generalOptions.SkipDomainsFile)
		}
	}
	if generalOptions.AllowDomainsFile != "" {
		if allowDomainMapBool {
			allowDomainMap = loadDomainsToMap(generalOptions.AllowDomainsFile)
		} else {
			allowDomainList = loadDomainsToList(generalOptions.AllowDomainsFile)
		}
	}

	// Setup output routine
	exiting := make(chan bool)
	var wg sync.WaitGroup

	// Setup our output channels
	setupOutputs(wg, exiting)

	// Setup the memory profile if reuqested
	if generalOptions.Memprofile != "" {
		go func() {
			time.Sleep(120 * time.Second)
			log.Warn("Writing memory profile")
			f, err := os.Create(generalOptions.Memprofile)
			errorHandler(err)
			runtime.GC() // get up-to-date statistics

			err = pprof.Lookup("heap").WriteTo(f, 0)
			errorHandler(err)
			f.Close()
		}()
	}

	// Start listening if we're using pcap or afpacket

	if captureOptions.DnstapSocket == "" {
		capturer := newDNSCapturer(CaptureOptions{
			captureOptions.DevName,
			captureOptions.UseAfpacket,
			captureOptions.PcapFile,
			captureOptions.Filter,
			uint16(captureOptions.Port),
			generalOptions.GcTime,
			resultChannel,
			captureOptions.PacketHandlerCount,
			captureOptions.PacketChannelSize,
			generalOptions.TcpHandlerCount,
			generalOptions.TcpAssemblyChannelSize,
			generalOptions.TcpResultChannelSize,
			generalOptions.DefraggerChannelSize,
			generalOptions.DefraggerChannelReturnSize,
			exiting,
			captureOptions.NoEthernetframe,
		})

		// if *dnstapSocket == "" {
		// 	capturer := newDNSCapturer(CaptureOptions{
		// 		*devName,
		// 		*useAfpacket,
		// 		*pcapFile,
		// 		*filter,
		// 		uint16(*port),
		// 		*gcTime,
		// 		resultChannel,
		// 		*packetHandlerCount,
		// 		*packetChannelSize,
		// 		*tcpHandlerCount,
		// 		*tcpAssemblyChannelSize,
		// 		*tcpResultChannelSize,
		// 		*defraggerChannelSize,
		// 		*defraggerChannelReturnSize,
		// 		exiting,
		// 	})
		capturer.start()
		// Wait for the output to finish
		log.Info("Exiting")
		wg.Wait()
	} else {
		startDNSTap(resultChannel)
	}
}
