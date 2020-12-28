package main

import (
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"sync"
	"time"

	"github.com/namsral/flag"
)

var fs = flag.NewFlagSetWithEnvPrefix(os.Args[0], "DNSMONSTER", 0)
var devName = fs.String("devName", "", "Device used to capture")
var pcapFile = fs.String("pcapFile", "", "Pcap filename to run")

// Filter is not using "(port 53)", as it will filter out fragmented udp packets, instead, we filter by the ip protocol
// and check again in the application.
var config = fs.String(flag.DefaultConfigFlagname, "", "path to config file")
var filter = fs.String("filter", "((ip and (ip[9] == 6 or ip[9] == 17)) or (ip6 and (ip6[6] == 17 or ip6[6] == 6 or ip6[6] == 44)))", "BPF filter applied to the packet stream. If port is selected, the packets will not be defragged.")
var port = fs.Uint("port", 53, "Port selected to filter packets")
var gcTime = fs.Uint("gcTime", 10, "Time in seconds to garbage collect the tcp assembly and ip defragmentation")
var clickhouseAddress = fs.String("clickhouseAddress", "localhost:9000", "Address of the clickhouse database to save the results")
var clickhouseDelay = fs.Uint("clickhouseDelay", 1, "Number of seconds to batch the packets")
var captureStatsDelay = fs.Duration("captureStatsDelay", time.Second, "Number of seconds to calculate interface stats")
var printStatsDelay = fs.Duration("printStatsDelay", time.Second*10, "Number of seconds to print capture and database stats")
var maskSize = fs.Int("maskSize", 32, "Mask source IPs by bits. 32 means all the bits of IP is saved in DB")
var serverName = fs.String("serverName", "default", "Name of the server used to index the metrics.")
var batchSize = fs.Uint("batchSize", 100000, "Minimun capacity of the cache array used to send data to clickhouse. Set close to the queries per second received to prevent allocations")
var packetHandlerCount = fs.Uint("packetHandlers", 1, "Number of routines used to handle received packets")
var tcpHandlerCount = fs.Uint("tcpHandlers", 1, "Number of routines used to handle tcp assembly")
var useAfpacket = fs.Bool("useAfpacket", false, "Use AFPacket for live captures")
var afPacketBuffersizeMb = fs.Uint("AfpacketBuffersizeMb", 64, "Afpacket Buffersize in MB")
var packetChannelSize = fs.Uint("packetHandlerChannelSize", 100000, "Size of the packet handler channel")
var tcpAssemblyChannelSize = fs.Uint("tcpAssemblyChannelSize", 1000, "Size of the tcp assembler")
var tcpResultChannelSize = fs.Uint("tcpResultChannelSize", 1000, "Size of the tcp result channel")
var resultChannelSize = fs.Uint("resultChannelSize", 100000, "Size of the result processor channel size")
var defraggerChannelSize = fs.Uint("defraggerChannelSize", 500, "Size of the channel to send packets to be defragged")
var defraggerChannelReturnSize = fs.Uint("defraggerChannelReturnSize", 500, "Size of the channel where the defragged packets are returned")
var cpuprofile = fs.String("cpuprofile", "", "write cpu profile to file")
var memprofile = fs.String("memprofile", "", "write memory profile to file")
var loggerFilename = fs.Bool("loggerFilename", false, "Show the file name and number of the logged string")
var packetLimit = fs.Int("packetLimit", 0, "Limit of packets logged to clickhouse every iteration. Default 0 (disabled)")

func checkFlags() {

	fs.Parse(os.Args[1:])
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
		log.Println("Writing CPU profile")
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

			if err := pprof.Lookup("heap").WriteTo(f, 0); err != nil {
				log.Fatal("could not write memory profile: ", err)
			}
			f.Close()
		}()
	}

	// Start listening
	capturer := NewDNSCapturer(CaptureOptions{
		*devName,
		*useAfpacket,
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
