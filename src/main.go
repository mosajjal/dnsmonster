package main

import (
	"flag"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"
)

var devName = flag.String("devName", "", "Device used to capture")
var pcapFile = flag.String("pcapFile", "", "Pcap filename to run")

// Filter is not using "(port 53)", as it will filter out fragmented udp packets, instead, we filter by the ip protocol
// and check again in the application.
var filter = flag.String("filter", "((ip and (ip[9] == 6 or ip[9] == 17)) or (ip6 and (ip6[6] == 17 or ip6[6] == 6 or ip6[6] == 44)))", "BPF filter applied to the packet stream. If port is selected, the packets will not be defragged.")
var port = flag.Uint("port", 53, "Port selected to filter packets")
var gcTime = flag.Uint("gcTime", 10, "Time in seconds to garbage collect the tcp assembly and ip defragmentation")
var clickhouseAddress = flag.String("clickhouseAddress", "localhost:9000", "Address of the clickhouse database to save the results")
var clickhouseDelay = flag.Uint("clickhouseDelay", 1, "Number of seconds to batch the packets")
var maskSize = flag.Int("maskSize", 32, "Mask source IPs by bits. 32 means all the bits of IP is saved in DB")
var serverName = flag.String("serverName", "default", "Name of the server used to index the metrics.")
var batchSize = flag.Uint("batchSize", 100000, "Minimun capacity of the cache array used to send data to clickhouse. Set close to the queries per second received to prevent allocations")
var packetHandlerCount = flag.Uint("packetHandlers", 1, "Number of routines used to handle received packets")
var tcpHandlerCount = flag.Uint("tcpHandlers", 1, "Number of routines used to handle tcp assembly")
var packetChannelSize = flag.Uint("packetHandlerChannelSize", 100000, "Size of the packet handler channel")
var tcpAssemblyChannelSize = flag.Uint("tcpAssemblyChannelSize", 1000, "Size of the tcp assembler")
var tcpResultChannelSize = flag.Uint("tcpResultChannelSize", 1000, "Size of the tcp result channel")
var resultChannelSize = flag.Uint("resultChannelSize", 100000, "Size of the result processor channel size")
var defraggerChannelSize = flag.Uint("defraggerChannelSize", 500, "Size of the channel to send packets to be defragged")
var defraggerChannelReturnSize = flag.Uint("defraggerChannelReturnSize", 500, "Size of the channel where the defragged packets are returned")
var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
var memprofile = flag.String("memprofile", "", "write memory profile to file")
var loggerFilename = flag.Bool("loggerFilename", false, "Show the file name and number of the logged string")
var packetLimit = flag.Int("packetLimit", 0, "Limit of packets logged to clickhouse every iteration. Default 0 (disabled)")

func checkFlags() {
	flag.Parse()
	if *loggerFilename {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	} else {
		log.SetFlags(log.LstdFlags)
	}
	if *port > 65535 {
		log.Fatal("-port must be between 1 and 65535")
	}
	if *maskSize > 32 || *maskSize < 0 {
		log.Fatal("-maskSize must be between 0 and 32")
	}
	if *devName == "" && *pcapFile == "" {
		log.Fatal("-devName or -pcapFile is required")
	}

	if *devName != "" && *pcapFile != "" {
		log.Fatal("You must set only -devName or -pcapFile, and not both")
	}

	if *packetLimit < 0 {
		log.Fatal("-packetLimit must be equal or greather than 0")
	}
}

func main() {
	checkFlags()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	resultChannel := make(chan DNSResult, *resultChannelSize)

	// Setup output routine
	exiting := make(chan bool)
	var wg sync.WaitGroup
	go output(resultChannel, exiting, &wg, *clickhouseAddress, *batchSize, *clickhouseDelay, *packetLimit, *serverName)

	if *memprofile != "" {
		go func() {
			time.Sleep(120 * time.Second)
			log.Println("Writing memory profile")
			f, err := os.Create(*memprofile)
			if err != nil {
				log.Fatal("could not create memory profile: ", err)
			}
			runtime.GC() // get up-to-date statistics
			if err := pprof.WriteHeapProfile(f); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
			f.Close()
		}()
	}

	// Start listening
	capturer := NewDNSCapturer(CaptureOptions{
		*devName,
		*pcapFile,
		*filter,
		uint16(*port),
		time.Duration(*gcTime) * time.Second,
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
	capturer.Start()

	// Wait for the output to finish
	log.Println("Exiting")
	wg.Wait()
}
