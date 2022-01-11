package types

import (
	"net"
	"time"

	mkdns "github.com/miekg/dns"
	"github.com/mosajjal/Go-Splunk-HTTP/splunk/v2"
)

// DNSResult is the middleware that connects the packet encoder to Clickhouse.
// For DNStap, this is probably going to be replaced with something else.
type DNSResult struct {
	Timestamp    time.Time
	DNS          mkdns.Msg
	IPVersion    uint8
	SrcIP        net.IP
	DstIP        net.IP
	Protocol     string
	PacketLength uint16
}

type GeneralConfig struct {
	MaskSize4           int
	MaskSize6           int
	PacketLimit         int
	ServerName          string
	PrintStatsDelay     time.Duration
	SkipTlsVerification bool
}

type ClickHouseConfig struct {
	ResultChannel               chan DNSResult
	ClickhouseAddress           string
	ClickhouseBatchSize         uint
	ClickhouseOutputType        uint
	ClickhouseSaveFullQuery     bool
	ClickhouseDebug             bool
	ClickhouseDelay             time.Duration
	ClickhouseWorkers           uint
	ClickhouseWorkerChannelSize uint
	General                     GeneralConfig
}

type ElasticConfig struct {
	ResultChannel         chan DNSResult
	ElasticOutputEndpoint string
	ElasticOutputIndex    string
	ElasticOutputType     uint
	ElasticBatchSize      uint
	ElasticBatchDelay     time.Duration
	General               GeneralConfig
}

type KafkaConfig struct {
	ResultChannel     chan DNSResult
	KafkaOutputBroker string
	KafkaOutputTopic  string
	KafkaOutputType   uint
	KafkaBatchSize    uint
	KafkaBatchDelay   time.Duration
	General           GeneralConfig
}

type SplunkConfig struct {
	ResultChannel          chan DNSResult
	SplunkOutputEndpoints  []string
	SplunkOutputToken      string
	SplunkOutputType       uint
	SplunkOutputIndex      string
	SplunkOutputSource     string
	SplunkOutputSourceType string
	SplunkBatchSize        uint
	SplunkBatchDelay       time.Duration
	General                GeneralConfig
}

type SplunkConnection struct {
	Client    *splunk.Client
	Unhealthy uint
	Err       error
}

type SyslogConfig struct {
	ResultChannel        chan DNSResult
	SyslogOutputEndpoint string
	SyslogOutputType     uint
	General              GeneralConfig
}

type FileConfig struct {
	ResultChannel    chan DNSResult
	FileOutputPath   string
	FileOutputType   uint
	FileOutputFormat string
	General          GeneralConfig
}

type StdoutConfig struct {
	ResultChannel      chan DNSResult
	StdoutOutputType   uint
	StdoutOutputFormat string
	General            GeneralConfig
}
