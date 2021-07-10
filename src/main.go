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

var clickhouseResultChannel = make(chan DNSResult, *resultChannelSize)
var kafkaResultChannel = make(chan DNSResult, *resultChannelSize)
var elasticResultChannel = make(chan DNSResult, *resultChannelSize)
var splunkResultChannel = make(chan DNSResult, *resultChannelSize)
var stdoutResultChannel = make(chan DNSResult, *resultChannelSize)
var fileResultChannel = make(chan DNSResult, *resultChannelSize)
var syslogResultChannel = make(chan DNSResult, *resultChannelSize)
var resultChannel = make(chan DNSResult, *resultChannelSize)

func main() {
	checkFlags()
	runtime.GOMAXPROCS(*gomaxprocs)
	if *cpuprofile != "" {
		log.Warn("Writing CPU profile")
		f, err := os.Create(*cpuprofile)
		errorHandler(err)
		err = pprof.StartCPUProfile(f)
		errorHandler(err)
		defer pprof.StopCPUProfile()
	}

	// Setup output routine
	exiting := make(chan bool)

	// load the skipDomainFile if exists
	if *skipDomainsFile != "" {
		if skipDomainMapBool {
			skipDomainMap = loadDomainsToMap(*skipDomainsFile)
		} else {
			skipDomainList = loadDomainsToList(*skipDomainsFile)
		}
	}
	if *allowDomainsFile != "" {
		if allowDomainMapBool {
			allowDomainMap = loadDomainsToMap(*allowDomainsFile)
		} else {
			allowDomainList = loadDomainsToList(*allowDomainsFile)
		}
	}

	var wg sync.WaitGroup

	// Setup our output channels
	setupOutputs(wg, exiting)

	// Setup the memory profile if reuqested
	if *memprofile != "" {
		go func() {
			time.Sleep(120 * time.Second)
			log.Warn("Writing memory profile")
			f, err := os.Create(*memprofile)
			errorHandler(err)
			runtime.GC() // get up-to-date statistics

			err = pprof.Lookup("heap").WriteTo(f, 0)
			errorHandler(err)
			f.Close()
		}()
	}

	// Start listening if we're using pcap or afpacket
	if *dnstapSocket == "" {
		capturer := newDNSCapturer(CaptureOptions{
			*devName,
			*useAfpacket,
			*pcapFile,
			*filter,
			uint16(*port),
			*gcTime,
			resultChannel,
			*packetHandlerCount,
			*packetChannelSize,
			*tcpHandlerCount,
			*tcpAssemblyChannelSize,
			*tcpResultChannelSize,
			*defraggerChannelSize,
			*defraggerChannelReturnSize,
			exiting,
		})
		capturer.start()
		// Wait for the output to finish
		log.Info("Exiting")
		wg.Wait()
	} else {
		startDNSTap(resultChannel)
	}
}
