// dnsmonster implements a packet sniffer for DNS traffic. It can accept traffic from a pcap file or a live interface,
// and can be used to index and store thousands of queries per second. It aims to be scalable and easy to use, and help
// security teams to understand the details about an enterprise's DNS traffic. It does not aim to breach
// the privacy of the end-users, with the ability to mask source IP from 1 to 32 bits, making the data potentially untraceable.

package main

import (
	"bufio"
	"log"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/namsral/flag"
)

var fs = flag.NewFlagSetWithEnvPrefix(os.Args[0], "DNSMONSTER", 0)
var devName = fs.String("devName", "", "Device used to capture")
var pcapFile = fs.String("pcapFile", "", "Pcap filename to run")
var dnstapSocket = fs.String("dnstapSocket", "", "dnstrap socket path. Example: unix:///tmp/dnstap.sock, tcp://127.0.0.1:8080")

// Filter is not using "(port 53)", as it will filter out fragmented udp packets, instead, we filter by the ip protocol
// and check again in the application.
var config = fs.String(flag.DefaultConfigFlagname, "", "path to config file")
var filter = fs.String("filter", "((ip and (ip[9] == 6 or ip[9] == 17)) or (ip6 and (ip6[6] == 17 or ip6[6] == 6 or ip6[6] == 44)))", "BPF filter applied to the packet stream. If port is selected, the packets will not be defragged.")
var port = fs.Uint("port", 53, "Port selected to filter packets")
var gcTime = fs.Duration("gcTime", 10*time.Second, "Garbage Collection interval for tcp assembly and ip defragmentation")
var clickhouseAddress = fs.String("clickhouseAddress", "localhost:9000", "Address of the clickhouse database to save the results")
var clickhouseDelay = fs.Duration("clickhouseDelay", 1*time.Second, "Interval between sending results to ClickHouse")
var clickhouseDebug = fs.Bool("clickhouseDebug", false, "Debug Clickhouse connection")
var clickhouseDryRun = fs.Bool("clickhouseDryRun", false, "process the packets but don't write them to clickhouse. This option will still try to connect to db. For testing only")
var captureStatsDelay = fs.Duration("captureStatsDelay", time.Second, "Duration to calculate interface stats")
var printStatsDelay = fs.Duration("printStatsDelay", time.Second*10, "Duration to print capture and database stats")
var maskSize = fs.Int("maskSize", 32, "Mask source IPs by bits. 32 means all the bits of IP is saved in DB")
var serverName = fs.String("serverName", "default", "Name of the server used to index the metrics.")
var batchSize = fs.Uint("batchSize", 100000, "Minimun capacity of the cache array used to send data to clickhouse. Set close to the queries per second received to prevent allocations")
var sampleRatio = fs.String("sampleRatio", "1:1", "Capture Sampling by a:b. eg sampleRatio of 1:100 will process 1 percent of the incoming packets")
var saveFullQuery = fs.Bool("saveFullQuery", false, "Save full packet query and response in JSON format")
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
var gomaxprocs = fs.Int("gomaxprocs", -1, "GOMAXPROCS variable")
var loggerFilename = fs.Bool("loggerFilename", false, "Show the file name and number of the logged string")
var packetLimit = fs.Int("packetLimit", 0, "Limit of packets logged to clickhouse every iteration. Default 0 (disabled)")
var skipDomainsFile = fs.String("skipDomainsFile", "", "Skip saving the domains in the text file, matching the skipDomainsBehavior")
var skipDomainsRefreshInterval = fs.Duration("skipDomainsRefreshInterval", 60*time.Second, "Hot-Reload SkipDomains file interval")
var dnstapPermission = fs.String("dnstapPermission", "755", "Set the dnstap socket permission, only applicable when unix:// is used")

// Ratio numbers
var ratioA int
var ratioB int

// SkipDomainList represents the list of skipped domains
var SkipDomainList [][]string

// SkipDomainsBool is a boolean to see if we're actually doing skipDomainsFile or not
var SkipDomainsBool bool

func checkFlags() {
	err := fs.Parse(os.Args[1:])
	if err != nil {
		log.Fatal("Errors in parsing args")
	}

	SkipDomainsBool = *skipDomainsFile != ""
	if SkipDomainsBool {
		// check to see if the file provided exists
		if _, err := os.Stat(*skipDomainsFile); err != nil {
			log.Fatal("error in finding SkipDomains file. You must provide a path to an existing filename")
		}
	}

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
	if *devName == "" && *pcapFile == "" && *dnstapSocket == "" {
		log.Fatal("one of -devName, -pcapFile or -dnstapSocket is required")
	}

	if *devName != "" {
		if *pcapFile != "" || *dnstapSocket != "" {
			log.Fatal("You must set only -devName, -pcapFile or -dnstapSocket")
		}
	} else {
		if *pcapFile != "" && *dnstapSocket != "" {
			log.Fatal("You must set only -devName, -pcapFile or -dnstapSocket")
		}
	}

	if *dnstapSocket != "" {
		if !strings.HasPrefix(*dnstapSocket, "unix://") && !strings.HasPrefix(*dnstapSocket, "tcp://") {
			log.Fatal("You must provide a unix:// or tcp:// socket for dnstap")
		}
	}

	if *packetLimit < 0 {
		log.Fatal("-packetLimit must be equal or greather than 0")
	}

	ratioNumbers := strings.Split(*sampleRatio, ":")
	if len(ratioNumbers) != 2 {
		log.Fatal("wrong -sampleRatio syntax")
	}
	var errA error
	var errB error
	ratioA, errA = strconv.Atoi(ratioNumbers[0])
	ratioB, errB = strconv.Atoi(ratioNumbers[1])
	if errA != nil || errB != nil || ratioA > ratioB {
		log.Fatal("wrong -sampleRatio syntax")
	}
}

func loadSkipDomains() [][]string {
	file, err := os.Open(*skipDomainsFile)
	if err != nil {
		log.Fatal("error in opening skipDomainsFile: ", err)
	}
	log.Println("(re)loading skipDomainsFile: ", *skipDomainsFile)
	defer file.Close()

	var lines [][]string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, strings.Split(scanner.Text(), ","))
	}
	return lines
}

func main() {
	checkFlags()
	runtime.GOMAXPROCS(*gomaxprocs)
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

	// load the skipDomainFile if exists
	if SkipDomainsBool {
		SkipDomainList = loadSkipDomains()
	}

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
		log.Println("Exiting")
		wg.Wait()
	} else {
		startDNSTap(resultChannel)
	}
}
