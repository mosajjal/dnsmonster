// Package config provides the hierarchical configuration structs for dnsmonster,
// supporting TOML config files and environment variable overrides.
package config

import "time"

// Config is the root configuration structure.
type Config struct {
	Input   InputConfig   `mapstructure:"input"`
	Process ProcessConfig `mapstructure:"process"`
	Outputs OutputsConfig `mapstructure:"outputs"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	General GeneralConfig `mapstructure:"general"`
}

// InputConfig holds all input/capture-related settings.
type InputConfig struct {
	DevName                string        `mapstructure:"devname"`
	PcapFile               string        `mapstructure:"pcapfile"`
	DnstapSocket           string        `mapstructure:"dnstapsocket"`
	Port                   int           `mapstructure:"port"`
	SampleRatio            string        `mapstructure:"sampleratio"`
	DedupCleanupInterval   time.Duration `mapstructure:"dedupcleanupinterval"`
	DnstapPermission       int           `mapstructure:"dnstappermission"`
	PacketHandlerCount     int           `mapstructure:"packethandlercount"`
	TCPAssemblyChannelSize int           `mapstructure:"tcpassemblychannelsize"`
	TCPResultChannelSize   int           `mapstructure:"tcpresultchannelsize"`
	TCPHandlerCount        int           `mapstructure:"tcphandlercount"`
	DefraggerChannelSize   int           `mapstructure:"defraggerchannelsize"`
	DefraggerReturnSize    int           `mapstructure:"defraggerchannelreturnsize"`
	PacketChannelSize      int           `mapstructure:"packetchannelsize"`
	AFPacketBufferSizeMB   int           `mapstructure:"afpacketbuffersizemb"`
	Filter                 string        `mapstructure:"filter"`
	UseAFPacket            bool          `mapstructure:"useafpacket"`
	NoEtherFrame           bool          `mapstructure:"noetherframe"`
	Dedup                  bool          `mapstructure:"dedup"`
	NoPromiscuous          bool          `mapstructure:"nopromiscuous"`
}

// ProcessConfig holds processing and filtering logic.
type ProcessConfig struct {
	GCTime                      time.Duration `mapstructure:"gctime"`
	CaptureStatsDelay           time.Duration `mapstructure:"capturestatsdelay"`
	MaskSize4                   int           `mapstructure:"masksize4"`
	MaskSize6                   int           `mapstructure:"masksize6"`
	ResultChannelSize           int           `mapstructure:"resultchannelsize"`
	PacketLimit                 int           `mapstructure:"packetlimit"`
	SkipDomainsFile             string        `mapstructure:"skipdomainsfile"`
	SkipDomainsRefreshInterval  time.Duration `mapstructure:"skipdomainsrefreshinterval"`
	AllowDomainsFile            string        `mapstructure:"allowdomainsfile"`
	AllowDomainsRefreshInterval time.Duration `mapstructure:"allowdomainsrefreshinterval"`
	SkipTLSVerification         bool          `mapstructure:"skiptlsverification"`
}

// ClickhouseOutputConfig holds Clickhouse output-related settings.
type ClickhouseOutputConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	Address       []string      `mapstructure:"address"`
	Username      string        `mapstructure:"username"`
	Password      string        `mapstructure:"password"`
	Database      string        `mapstructure:"database"`
	BatchSize     uint          `mapstructure:"batch_size"`
	BatchDelay    time.Duration `mapstructure:"batch_delay"`
	Compress      int           `mapstructure:"compress"`
	Debug         bool          `mapstructure:"debug"`
	Secure        bool          `mapstructure:"secure"`
	SaveFullQuery bool          `mapstructure:"save_full_query"`
	Workers       int           `mapstructure:"workers"`
	FilterMode    string        `mapstructure:"filter_mode"`
}

// ElasticOutputConfig holds Elasticsearch output-related settings.
type ElasticOutputConfig struct {
	Enabled     bool          `mapstructure:"enabled"`
	Address     []string      `mapstructure:"address"`
	Username    string        `mapstructure:"username"`
	Password    string        `mapstructure:"password"`
	IndexPrefix string        `mapstructure:"index_prefix"`
	BatchSize   uint          `mapstructure:"batch_size"`
	BatchDelay  time.Duration `mapstructure:"batch_delay"`
	Debug       bool          `mapstructure:"debug"`
	Secure      bool          `mapstructure:"secure"`
	Workers     int           `mapstructure:"workers"`
	FilterMode  string        `mapstructure:"filter_mode"`
	OutputType  int           `mapstructure:"output_type"`
	OutputIndex string        `mapstructure:"output_index"`
}

// OutputsConfig holds all output-related settings.
type OutputsConfig struct {
	Clickhouse ClickhouseOutputConfig `mapstructure:"clickhouse"`
	Elastic    ElasticOutputConfig    `mapstructure:"elastic"`
	// ...other outputs...
}

// MetricsConfig holds metrics/exporter settings.
type MetricsConfig struct {
	EndpointType       string        `mapstructure:"endpointtype"`
	StatsdAgent        string        `mapstructure:"statsdagent"`
	PrometheusEndpoint string        `mapstructure:"metricprometheusendpoint"`
	StderrFormat       string        `mapstructure:"metricstderrformat"`
	FlushInterval      time.Duration `mapstructure:"metricflushinterval"`
}

// GeneralConfig holds general application settings.
type GeneralConfig struct {
	ServerName string `mapstructure:"servername"`
	LogFormat  string `mapstructure:"logformat"`
	LogLevel   int    `mapstructure:"loglevel"`
	CPUProfile string `mapstructure:"cpuprofile"`
	MemProfile string `mapstructure:"memprofile"`
	GoMaxProcs int    `mapstructure:"gomaxprocs"`
}
