package util

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
)

var releaseVersion string = "DEVELOPMENT"
var GlobalParser = flags.NewNamedParser("dnsmonster", flags.PassDoubleDash|flags.PrintErrors)

var SkipDomainMapBool = false
var AllowDomainMapBool = false

// skipDomainList represents the list of skipped domains
var SkipDomainList [][]string
var AllowDomainList [][]string

var SkipDomainMap = make(map[string]bool)
var AllowDomainMap = make(map[string]bool)

// Ratio numbers used for input sampling
var RatioA int
var RatioB int

var CaptureFlags struct {
	DevName              string `long:"devName"              env:"DNSMONSTER_DEVNAME"              default:""                                                                                                  description:"Device used to capture"`
	PcapFile             string `long:"pcapFile"             env:"DNSMONSTER_PCAPFILE"             default:""                                                                                                  description:"Pcap filename to run"`
	DnstapSocket         string `long:"dnstapSocket"         env:"DNSMONSTER_DNSTAPSOCKET"         default:""                                                                                                  description:"dnstrap socket path. Example: unix:///tmp/dnstap.sock, tcp://127.0.0.1:8080"`
	Port                 uint   `long:"port"                 env:"DNSMONSTER_PORT"                 default:"53"                                                                                                description:"Port selected to filter packets"`
	SampleRatio          string `long:"sampleRatio"          env:"DNSMONSTER_SAMPLERATIO"          default:"1:1"                                                                                               description:"Capture Sampling by a:b. eg sampleRatio of 1:100 will process 1 percent of the incoming packets"`
	DnstapPermission     string `long:"dnstapPermission"     env:"DNSMONSTER_DNSTAPPERMISSION"     default:"755"                                                                                               description:"Set the dnstap socket permission, only applicable when unix:// is used"`
	PacketHandlerCount   uint   `long:"packetHandlerCount"   env:"DNSMONSTER_PACKETHANDLERCOUNT"   default:"2"                                                                                                 description:"Number of routines used to handle received packets"`
	PacketChannelSize    uint   `long:"packetChannelSize"    env:"DNSMONSTER_PACKETCHANNELSIZE"    default:"1000"                                                                                              description:"Size of the packet handler channel"`
	AfpacketBuffersizeMb uint   `long:"afpacketBuffersizeMb" env:"DNSMONSTER_AFPACKETBUFFERSIZEMB" default:"64"                                                                                                description:"Afpacket Buffersize in MB"`
	Filter               string `long:"filter"               env:"DNSMONSTER_FILTER"               default:"((ip and (ip[9] == 6 or ip[9] == 17)) or (ip6 and (ip6[6] == 17 or ip6[6] == 6 or ip6[6] == 44)))" description:"BPF filter applied to the packet stream. If port is selected, the packets will not be defragged."`
	UseAfpacket          bool   `long:"useAfpacket"          env:"DNSMONSTER_USEAFPACKET"          description:"Use AFPacket for live captures. Supported on Linux 3.0+ only"`
	NoEthernetframe      bool   `long:"noEtherframe"         env:"DNSMONSTER_NOETHERFRAME"         description:"The PCAP capture does not contain ethernet frames"`
}

var GeneralFlags struct {
	Config                      flags.Filename `long:"config"                      env:"DNSMONSTER_CONFIG"                      default:""                            no-ini:"true"               description:"path to config file"`
	GcTime                      time.Duration  `long:"gcTime"                      env:"DNSMONSTER_GCTIME"                      default:"10s"                                                     description:"Garbage Collection interval for tcp assembly and ip defragmentation"`
	CaptureStatsDelay           time.Duration  `long:"captureStatsDelay"           env:"DNSMONSTER_CAPTURESTATSDELAY"           default:"1s"                                                      description:"Duration to calculate interface stats"`
	PrintStatsDelay             time.Duration  `long:"printStatsDelay"             env:"DNSMONSTER_PRINTSTATSDELAY"             default:"10s"                                                     description:"Duration to print capture and database stats"`
	MaskSize4                   int            `long:"maskSize4"                   env:"DNSMONSTER_MASKSIZE4"                   default:"32"                                                      description:"Mask IPv4s by bits. 32 means all the bits of IP is saved in DB"`
	MaskSize6                   int            `long:"maskSize6"                   env:"DNSMONSTER_MASKSIZE6"                   default:"128"                                                     description:"Mask IPv6s by bits. 32 means all the bits of IP is saved in DB"`
	ServerName                  string         `long:"serverName"                  env:"DNSMONSTER_SERVERNAME"                  default:"default"                                                 description:"Name of the server used to index the metrics."`
	TcpAssemblyChannelSize      uint           `long:"tcpAssemblyChannelSize"      env:"DNSMONSTER_TCPASSEMBLYCHANNELSIZE"      default:"10000"                                                   description:"Size of the tcp assembler"`
	TcpResultChannelSize        uint           `long:"tcpResultChannelSize"        env:"DNSMONSTER_TCPRESULTCHANNELSIZE"        default:"10000"                                                   description:"Size of the tcp result channel"`
	TcpHandlerCount             uint           `long:"tcpHandlerCount"             env:"DNSMONSTER_TCPHANDLERCOUNT"             default:"1"                                                       description:"Number of routines used to handle tcp assembly"`
	ResultChannelSize           uint           `long:"resultChannelSize"           env:"DNSMONSTER_RESULTCHANNELSIZE"           default:"100000"                                                  description:"Size of the result processor channel size"`
	LogLevel                    uint           `long:"logLevel"                    env:"DNSMONSTER_LOGLEVEL"                    default:"3"                                                       description:"Set debug Log level, 0:PANIC, 1:ERROR, 2:WARN, 3:INFO, 4:DEBUG" choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	DefraggerChannelSize        uint           `long:"defraggerChannelSize"        env:"DNSMONSTER_DEFRAGGERCHANNELSIZE"        default:"10000"                                                   description:"Size of the channel to send packets to be defragged"`
	DefraggerChannelReturnSize  uint           `long:"defraggerChannelReturnSize"  env:"DNSMONSTER_DEFRAGGERCHANNELRETURNSIZE"  default:"10000"                                                   description:"Size of the channel where the defragged packets are returned"`
	Cpuprofile                  string         `long:"cpuprofile"                  env:"DNSMONSTER_CPUPROFILE"                  default:""                                                        description:"write cpu profile to file"`
	Memprofile                  string         `long:"memprofile"                  env:"DNSMONSTER_MEMPROFILE"                  default:""                                                        description:"write memory profile to file"`
	Gomaxprocs                  int            `long:"gomaxprocs"                  env:"DNSMONSTER_GOMAXPROCS"                  default:"-1"                                                      description:"GOMAXPROCS variable"`
	PacketLimit                 int            `long:"packetLimit"                 env:"DNSMONSTER_PACKETLIMIT"                 default:"0"                                                       description:"Limit of packets logged to clickhouse every iteration. Default 0 (disabled)"`
	SkipDomainsFile             string         `long:"skipDomainsFile"             env:"DNSMONSTER_SKIPDOMAINSFILE"             default:""                                                        description:"Skip outputing domains matching items in the CSV file path. Can accept a URL (http:// or https://) or path"`
	SkipDomainsRefreshInterval  time.Duration  `long:"skipDomainsRefreshInterval"  env:"DNSMONSTER_SKIPDOMAINSREFRESHINTERVAL"  default:"60s"                                                     description:"Hot-Reload skipDomainsFile interval"`
	SkipDomainsFileType         string         `long:"skipDomainsFileType"         env:"DNSMONSTER_SKIPDOMAINSFILETYPE"         default:"csv"                                                     description:"skipDomainsFile type. Options: csv and hashtable. Hashtable is ONLY fqdn, csv can support fqdn, prefix and suffix logic but it's much slower"`
	AllowDomainsFile            string         `long:"allowDomainsFile"            env:"DNSMONSTER_ALLOWDOMAINSFILE"            default:""                                                        description:"Allow Domains logic input file. Can accept a URL (http:// or https://) or path"`
	AllowDomainsRefreshInterval time.Duration  `long:"allowDomainsRefreshInterval" env:"DNSMONSTER_ALLOWDOMAINSREFRESHINTERVAL" default:"60s"                                                     description:"Hot-Reload allowDomainsFile file interval"`
	AllowDomainsFileType        string         `long:"allowDomainsFileType"        env:"DNSMONSTER_ALLOWDOMAINSFILETYPE"        default:"csv"                                                     description:"allowDomainsFile type. Options: csv and hashtable. Hashtable is ONLY fqdn, csv can support fqdn, prefix and suffix logic but it's much slower"`
	SkipTLSVerification         bool           `long:"skipTLSVerification"         env:"DNSMONSTER_SKIPTLSVERIFICATION"         description:"Skip TLS verification when making HTTPS connections"`
	Version                     bool           `long:"version"                     env:"DNSMONSTER_VERSION"                     description:"show version and quit."  no-ini:"true" `
}

var OutputFlags struct {
	FileOutputType     uint           `long:"fileOutputType"              env:"DNSMONSTER_FILEOUTPUTTYPE"              default:"0"                                                       description:"What should be written to file. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"          choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	FileOutputPath     flags.Filename `long:"fileOutputPath"              env:"DNSMONSTER_FILEOUTPUTPATH"              default:""                                                        description:"Path to output file. Used if fileOutputType is not none"`
	FileOutputFormat   string         `long:"fileOutputFormat"            env:"DNSMONSTER_FILEOUTPUTFORMAT"            default:"json"                                                    description:"Output format for file. options:json,csv. note that the csv splits the datetime format into multiple fields"                                                                                                                                                                          choice:"json" choice:"csv"`
	StdoutOutputType   uint           `long:"stdoutOutputType"            env:"DNSMONSTER_STDOUTOUTPUTTYPE"            default:"0"                                                       description:"What should be written to stdout. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"        choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	StdoutOutputFormat string         `long:"stdoutOutputFormat"          env:"DNSMONSTER_STDOUTOUTPUTFORMAT"          default:"json"                                                    description:"Output format for stdout. options:json,csv. note that the csv splits the datetime format into multiple fields"                                                                                                                                                                        choice:"json" choice:"csv"`
}

var helpOptions struct {
	Help           bool `long:"help"  short:"h" no-ini:"true"      description:"Print this help to stdout"`
	ManPage        bool `long:"manPage"         no-ini:"true"      description:"Print Manpage for dnsmonster to stdout"`
	BashCompletion bool `long:"bashCompletion"  no-ini:"true"      description:"Print bash completion script to stdout"`
	FishCompletion bool `long:"fishCompletion"  no-ini:"true"      description:"Print fish completion script to stdout"`
	// SystemdService bool           `long:"systemdService"  no-ini:"true"      description:"Print a sample systemd service to stdout"`
	WriteConfig flags.Filename `long:"writeConfig"     no-ini:"true"      description:"generate a config file based on current inputs (flags, input config file and environment variables) and write to provided path" default:""`
}

func ProcessFlags() {
	//todo: flags are camel-case but ini is not. this needs to be consistent

	iniParser := flags.NewIniParser(GlobalParser)
	GlobalParser.AddGroup("general", "General Options", &GeneralFlags)
	GlobalParser.AddGroup("help", "Help Options", &helpOptions)
	GlobalParser.AddGroup("capture", "Options specific to capture side", &CaptureFlags)
	GlobalParser.AddGroup("output", "Options specific to output side", &OutputFlags)
	GlobalParser.Parse()

	// process help options first
	if helpOptions.Help {
		GlobalParser.WriteHelp(os.Stdout)
		os.Exit(0)
	}
	if helpOptions.ManPage {
		GlobalParser.WriteManPage(os.Stdout)
		os.Exit(0)
	}
	if helpOptions.BashCompletion {
		fmt.Print(BASH_COMPLETION_TEMPLATE)
		os.Exit(0)
	}
	if helpOptions.FishCompletion {
		for _, g := range GlobalParser.Groups() {
			for _, arg := range g.Options() {
				fmt.Printf("complete -f -c dnsmonster -o -%s -d %#v\n", arg.LongName, arg.Description)
			}
		}
		os.Exit(0)
	}
	if helpOptions.WriteConfig != "" {
		iniParser.WriteFile(string(helpOptions.WriteConfig), flags.IniIncludeDefaults|flags.IniIncludeComments)
		os.Exit(0)
	}

	// check for config file option and parse it
	if GeneralFlags.Config != "" {
		err := iniParser.ParseFile(string(GeneralFlags.Config))
		if err != nil {
			ErrorHandler(err)
		}
		//  re-parse the argument from command line to give them priority
		GlobalParser.Parse()
	}

	// default logging to warning
	var lvl log.Level = log.WarnLevel
	switch GeneralFlags.LogLevel {
	case 0:
		lvl = log.PanicLevel
	case 1:
		lvl = log.ErrorLevel
	case 2:
		lvl = log.WarnLevel
	case 3:
		lvl = log.InfoLevel
	case 4:
		lvl = log.DebugLevel
	}
	log.SetLevel(lvl)

	if GeneralFlags.Version {
		log.Fatalln("dnsmonster version:", releaseVersion)
	}

	//TODO: log format needs to be a configurable parameter
	// log.SetFormatter(&log.JSONFormatter{})

	if GeneralFlags.SkipDomainsFile != "" {
		log.Info("skipDomainsFile is provided")
		// todo: check to see if the file provided exists
		// commented because this now can be either filepath or URL
		// if _, err := os.Stat(generalOptions.SkipDomainsFile); err != nil {
		// 	log.Fatal("error in finding SkipDomains file. You must provide a path to an existing filename")
		// }
		if GeneralFlags.SkipDomainsFileType != "csv" && GeneralFlags.SkipDomainsFileType != "hashtable" {
			log.Fatal("skipDomainsFileType must be either csv or hashtable")
		}
		if GeneralFlags.SkipDomainsFileType == "hashtable" {
			SkipDomainMapBool = true
		}
	}

	if GeneralFlags.AllowDomainsFile != "" {
		log.Info("allowDomainsFile is provided")
		// todo: check to see if the file provided exists
		// commented because this now can be either filepath or URL
		// if _, err := os.Stat(generalOptions.AllowDomainsFile); err != nil {
		// 	log.Fatal("error in finding allowDomainsFile. You must provide a path to an existing filename")
		// }
		if GeneralFlags.AllowDomainsFileType != "csv" && GeneralFlags.AllowDomainsFileType != "hashtable" {
			log.Fatal("allowDomainsFileType must be either csv or hashtable")
		}
		if GeneralFlags.AllowDomainsFileType == "hashtable" {
			AllowDomainMapBool = true
		}
	}

	// todo: check to see if there's at least one output is enabled. possibly can add all the types and see if it's a positive number

	if OutputFlags.StdoutOutputType >= 5 {
		log.Fatal("stdoutOutputType must be one of 0, 1, 2, 3 or 4")
	}
	if OutputFlags.FileOutputType >= 5 {
		log.Fatal("fileOutputType must be one of 0, 1, 2, 3 or 4")
	} else if OutputFlags.FileOutputType > 0 {
		if OutputFlags.FileOutputPath == "" {
			log.Fatal("fileOutputType is set but fileOutputPath is not provided. Exiting")
		}
	}
	if CaptureFlags.Port > 65535 {
		log.Fatal("--port must be between 1 and 65535")
	}
	if GeneralFlags.MaskSize4 > 32 || GeneralFlags.MaskSize4 < 0 {
		log.Fatal("--maskSize4 must be between 0 and 32")
	}
	if GeneralFlags.MaskSize6 > 128 || GeneralFlags.MaskSize4 < 0 {
		log.Fatal("--maskSize6 must be between 0 and 128")
	}
	if CaptureFlags.DevName == "" && CaptureFlags.PcapFile == "" && CaptureFlags.DnstapSocket == "" {
		log.Fatal("one of --devName, --pcapFile or --dnstapSocket is required")
	}

	if CaptureFlags.DevName != "" {
		if CaptureFlags.PcapFile != "" || CaptureFlags.DnstapSocket != "" {
			log.Fatal("You must set only --devName, --pcapFile or --dnstapSocket")
		}
	} else {
		if CaptureFlags.PcapFile != "" && CaptureFlags.DnstapSocket != "" {
			log.Fatal("You must set only --devName, --pcapFile or --dnstapSocket")
		}
	}

	if CaptureFlags.DnstapSocket != "" {
		if !strings.HasPrefix(CaptureFlags.DnstapSocket, "unix://") && !strings.HasPrefix(CaptureFlags.DnstapSocket, "tcp://") {
			log.Fatal("You must provide a unix:// or tcp:// socket for dnstap")
		}
	}

	if GeneralFlags.PacketLimit < 0 {
		log.Fatal("--packetLimit must be equal or greather than 0")
	}

	ratioNumbers := strings.Split(CaptureFlags.SampleRatio, ":")
	if len(ratioNumbers) != 2 {
		log.Fatal("wrong --sampleRatio syntax")
	}
	var errA error
	var errB error
	RatioA, errA = strconv.Atoi(ratioNumbers[0])
	RatioB, errB = strconv.Atoi(ratioNumbers[1])
	if errA != nil || errB != nil || RatioA > RatioB {
		log.Fatal("wrong --sampleRatio syntax")
	}

	// load the skipDomainFile if exists
	if GeneralFlags.SkipDomainsFile != "" {
		if SkipDomainMapBool {
			SkipDomainMap = LoadDomainsToMap(GeneralFlags.SkipDomainsFile)
		} else {
			SkipDomainList = LoadDomainsToList(GeneralFlags.SkipDomainsFile)
		}
	}
	if GeneralFlags.AllowDomainsFile != "" {
		if AllowDomainMapBool {
			AllowDomainMap = LoadDomainsToMap(GeneralFlags.AllowDomainsFile)
		} else {
			AllowDomainList = LoadDomainsToList(GeneralFlags.AllowDomainsFile)
		}
	}

}
