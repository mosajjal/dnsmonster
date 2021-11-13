package main

import (
	"fmt"
	"os"
	"time"

	flags "github.com/jessevdk/go-flags"
)

var captureOptions struct {
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

var generalOptions struct {
	Config                      flags.Filename `long:"config"                      env:"DNSMONSTER_CONFIG"                      default:""                            no-ini:"true"               description:"path to config file"`
	GcTime                      time.Duration  `long:"gcTime"                      env:"DNSMONSTER_GCTIME"                      default:"10s"                                                     description:"Garbage Collection interval for tcp assembly and ip defragmentation"`
	CaptureStatsDelay           time.Duration  `long:"captureStatsDelay"           env:"DNSMONSTER_CAPTURESTATSDELAY"           default:"1s"                                                      description:"Duration to calculate interface stats"`
	PrintStatsDelay             time.Duration  `long:"printStatsDelay"             env:"DNSMONSTER_PRINTSTATSDELAY"             default:"10s"                                                     description:"Duration to print capture and database stats"`
	MaskSize4                   int            `long:"maskSize4"                   env:"DNSMONSTER_MASKSIZE4"                   default:"32"                                                      description:"Mask IPv4s by bits. 32 means all the bits of IP is saved in DB"`
	MaskSize6                   int            `long:"maskSize6"                   env:"DNSMONSTER_MASKSIZE6"                   default:"128"                                                     description:"Mask IPv6s by bits. 32 means all the bits of IP is saved in DB"`
	ServerName                  string         `long:"serverName"                  env:"DNSMONSTER_SERVERNAME"                  default:"default"                                                 description:"Name of the server used to index the metrics."`
	TcpAssemblyChannelSize      uint           `long:"tcpAssemblyChannelSize"      env:"DNSMONSTER_TCPASSEMBLYCHANNELSIZE"      default:"1000"                                                    description:"Size of the tcp assembler"`
	TcpResultChannelSize        uint           `long:"tcpResultChannelSize"        env:"DNSMONSTER_TCPRESULTCHANNELSIZE"        default:"1000"                                                    description:"Size of the tcp result channel"`
	TcpHandlerCount             uint           `long:"tcpHandlerCount"             env:"DNSMONSTER_TCPHANDLERCOUNT"             default:"1"                                                       description:"Number of routines used to handle tcp assembly"`
	ResultChannelSize           uint           `long:"resultChannelSize"           env:"DNSMONSTER_RESULTCHANNELSIZE"           default:"100000"                                                  description:"Size of the result processor channel size"`
	LogLevel                    uint           `long:"logLevel"                    env:"DNSMONSTER_LOGLEVEL"                    default:"3"                                                       description:"Set debug Log level, 0:PANIC, 1:ERROR, 2:WARN, 3:INFO, 4:DEBUG" choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	DefraggerChannelSize        uint           `long:"defraggerChannelSize"        env:"DNSMONSTER_DEFRAGGERCHANNELSIZE"        default:"500"                                                     description:"Size of the channel to send packets to be defragged"`
	DefraggerChannelReturnSize  uint           `long:"defraggerChannelReturnSize"  env:"DNSMONSTER_DEFRAGGERCHANNELRETURNSIZE"  default:"500"                                                     description:"Size of the channel where the defragged packets are returned"`
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

var outputOptions struct {
	ClickhouseAddress       string         `long:"clickhouseAddress"       env:"DNSMONSTER_CLICKHOUSEADDRESS"       default:"localhost:9000"                       description:"Address of the clickhouse database to save the results"`
	ClickhouseDelay         time.Duration  `long:"clickhouseDelay"         env:"DNSMONSTER_CLICKHOUSEDELAY"         default:"1s"                                   description:"Interval between sending results to ClickHouse"`
	ClickhouseDebug         bool           `long:"clickhouseDebug"         env:"DNSMONSTER_CLICKHOUSEDEBUG"         description:"Debug Clickhouse connection"`
	ClickhouseSaveFullQuery bool           `long:"clickhouseSaveFullQuery" env:"DNSMONSTER_CLICKHOUSESAVEFULLQUERY" description:"Save full packet query and response in JSON format."`
	ClickhouseOutputType    uint           `long:"clickhouseOutputType"    env:"DNSMONSTER_CLICKHOUSEOUTPUTTYPE"    default:"0"                                    description:"What should be written to clickhouse. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"    choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	ClickhouseBatchSize     uint           `long:"clickhouseBatchSize"     env:"DNSMONSTER_CLICKHOUSEBATCHSIZE"     default:"100000"                               description:"Minimun capacity of the cache array used to send data to clickhouse. Set close to the queries per second received to prevent allocations"`
	FileOutputType          uint           `long:"fileOutputType"          env:"DNSMONSTER_FILEOUTPUTTYPE"          default:"0"                                    description:"What should be written to file. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"          choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	FileOutputPath          flags.Filename `long:"fileOutputPath"          env:"DNSMONSTER_FILEOUTPUTPATH"          default:""                                     description:"Path to output file. Used if fileOutputType is not none"`
	StdoutOutputType        uint           `long:"stdoutOutputType"        env:"DNSMONSTER_STDOUTOUTPUTTYPE"        default:"0"                                    description:"What should be written to stdout. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"        choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	SyslogOutputType        uint           `long:"syslogOutputType"        env:"DNSMONSTER_SYSLOGOUTPUTTYPE"        default:"0"                                    description:"What should be written to Syslog server. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic" choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	SyslogOutputEndpoint    string         `long:"syslogOutputEndpoint"    env:"DNSMONSTER_SYSLOGOUTPUTENDPOINT"    default:""                                     description:"Syslog endpoint address, example: udp://127.0.0.1:514, tcp://127.0.0.1:514. Used if syslogOutputType is not none"`
	KafkaOutputType         uint           `long:"kafkaOutputType"         env:"DNSMONSTER_KAFKAOUTPUTTYPE"         default:"0"                                    description:"What should be written to kafka. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"         choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	KafkaOutputBroker       string         `long:"kafkaOutputBroker"       env:"DNSMONSTER_KAFKAOUTPUTBROKER"       default:""                                     description:"kafka broker address, example: 127.0.0.1:9092. Used if kafkaOutputType is not none"`
	KafkaOutputTopic        string         `long:"kafkaOutputTopic"        env:"DNSMONSTER_KAFKAOUTPUTTOPIC"        default:"dnsmonster"                           description:"Kafka topic for logging"`
	KafkaBatchSize          uint           `long:"kafkaBatchSize"          env:"DNSMONSTER_KAFKABATCHSIZE"          default:"1000"                                 description:"Minimun capacity of the cache array used to send data to Kafka"`
	KafkaBatchDelay         time.Duration  `long:"kafkaBatchDelay"         env:"DNSMONSTER_KAFKABATCHDELAY"         default:"1s"                                   description:"Interval between sending results to Kafka if Batch size is not filled"`
	ElasticOutputType       uint           `long:"elasticOutputType"       env:"DNSMONSTER_ELASTICOUTPUTTYPE"       default:"0"                                    description:"What should be written to elastic. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"       choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	ElasticOutputEndpoint   string         `long:"elasticOutputEndpoint"   env:"DNSMONSTER_ELASTICOUTPUTENDPOINT"   default:""                                     description:"elastic endpoint address, example: http://127.0.0.1:9200. Used if elasticOutputType is not none"`
	ElasticOutputIndex      string         `long:"elasticOutputIndex"      env:"DNSMONSTER_ELASTICOUTPUTINDEX"      default:"default"                              description:"elastic index"`
	ElasticBatchSize        uint           `long:"elasticBatchSize"        env:"DNSMONSTER_ELASTICBATCHSIZE"        default:"1000"                                 description:"Send data to Elastic in batch sizes"`
	ElasticBatchDelay       time.Duration  `long:"elasticBatchDelay"       env:"DNSMONSTER_ELASTICBATCHDELAY"       default:"1s"                                   description:"Interval between sending results to Elastic if Batch size is not filled"`
	SplunkOutputType        uint           `long:"splunkOutputType"        env:"DNSMONSTER_SPLUNKOUTPUTTYPE"        default:"0"                                    description:"What should be written to HEC. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"           choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	SplunkOutputEndpoints   []string       `long:"splunkOutputEndpoints"   env:"DNSMONSTER_SPLUNKOUTPUTENDPOINTS"   default:""                                     description:"splunk endpoint address, example: http://127.0.0.1:8088. Used if splunkOutputType is not none"`
	SplunkOutputToken       string         `long:"splunkOutputToken"       env:"DNSMONSTER_SPLUNKOUTPUTTOKEN"       default:"00000000-0000-0000-0000-000000000000" description:"Splunk HEC Token"`
	SplunkOutputIndex       string         `long:"splunkOutputIndex"       env:"DNSMONSTER_SPLUNKOUTPUTINDEX"       default:"temp"                                 description:"Splunk Output Index"`
	SplunkOutputSource      string         `long:"splunkOutputSource"      env:"DNSMONSTER_SPLUNKOUTPUTSOURCE"      default:"dnsmonster"                           description:"Splunk Output Source"`
	SplunkOutputSourceType  string         `long:"splunkOutputSourceType"  env:"DNSMONSTER_SPLUNKOUTPUTSOURCETYPE"  default:"json"                                 description:"Splunk Output Sourcetype"`
	SplunkBatchSize         uint           `long:"splunkBatchSize"         env:"DNSMONSTER_SPLUNKBATCHSIZE"         default:"1000"                                 description:"Send data to HEC in batch sizes"`
	SplunkBatchDelay        time.Duration  `long:"splunkBatchDelay"        env:"DNSMONSTER_SPLUNKBATCHDELAY"        default:"1s"                                   description:"Interval between sending results to HEC if Batch size is not filled"`
}

var helpOptions struct {
	Help           bool `long:"help"  short:"h" no-ini:"true"      description:"Print this help to stdout"`
	ManPage        bool `long:"manPage"         no-ini:"true"      description:"Print Manpage for dnsmonster to stdout"`
	BashCompletion bool `long:"bashCompletion"  no-ini:"true"      description:"Print bash completion script to stdout"`
	FishCompletion bool `long:"fishCompletion"  no-ini:"true"      description:"Print fish completion script to stdout"`
	// SystemdService bool           `long:"systemdService"  no-ini:"true"      description:"Print a sample systemd service to stdout"`
	WriteConfig flags.Filename `long:"writeConfig"     no-ini:"true"      description:"generate a config file based on current inputs (flags, input config file and environment variables) and write to provided path" default:""`
}

func flagsProcess() {

	var parser = flags.NewNamedParser("dnsmonster", flags.PassDoubleDash|flags.PrintErrors)
	iniParser := flags.NewIniParser(parser)
	parser.AddGroup("general", "General Options", &generalOptions)
	parser.AddGroup("help", "Help Options", &helpOptions)
	parser.AddGroup("capture", "Options specific to capture side", &captureOptions)
	parser.AddGroup("output", "Options specific to output side", &outputOptions)
	parser.Parse()

	// process help options first
	if helpOptions.Help {
		parser.WriteHelp(os.Stdout)
		os.Exit(0)
	}
	if helpOptions.ManPage {
		parser.WriteManPage(os.Stdout)
		os.Exit(0)
	}
	if helpOptions.BashCompletion {
		fmt.Print(BASH_COMPLETION_TEMPLATE)
		os.Exit(0)
	}
	if helpOptions.FishCompletion {
		for _, g := range parser.Groups() {
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
	if generalOptions.Config != "" {
		err := iniParser.ParseFile(string(generalOptions.Config))
		if err != nil {
			errorHandler(err)
		}
		//  re-parse the argument from command line to give them priority
		parser.Parse()
	}

}
