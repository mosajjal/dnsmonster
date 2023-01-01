package output

import (
	"context"
	"errors"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/mosajjal/dnsmonster/internal/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type influxConfig struct {
	InfluxOutputType    uint   `long:"influxoutputtype"             ini-name:"influxoutputtype"             env:"DNSMONSTER_INFLUXOUTPUTTYPE"             default:"0"                                                       description:"What should be written to influx. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"         choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	InfluxOutputServer  string `long:"influxoutputserver"           ini-name:"influxoutputserver"           env:"DNSMONSTER_INFLUXOUTPUTSERVER"           default:""                                                        description:"influx Server address, example: http://localhost:8086. Used if influxOutputType is not none"`
	InfluxOutputToken   string `long:"influxoutputtoken"            ini-name:"influxoutputtoken"            env:"DNSMONSTER_INFLUXOUTPUTTOKEN"            default:"dnsmonster"                                              description:"Influx Server Auth Token"`
	InfluxOutputBucket  string `long:"influxoutputbucket"           ini-name:"influxoutputbucket"           env:"DNSMONSTER_INFLUXOUTPUTBUCKET"           default:"dnsmonster"                                              description:"Influx Server Bucket"`
	InfluxOutputOrg     string `long:"influxoutputorg"              ini-name:"influxoutputorg"              env:"DNSMONSTER_INFLUXOUTPUTORG"              default:"dnsmonster"                                              description:"Influx Server Org"`
	InfluxOutputWorkers uint   `long:"influxoutputworkers"          ini-name:"influxoutputworkers"          env:"DNSMONSTER_INFLUXOUTPUTWORKERS"          default:"8"                                                       description:"Minimum capacity of the cache array used to send data to Influx"`
	InfluxBatchSize     uint   `long:"influxbatchsize"              ini-name:"influxbatchsize"              env:"DNSMONSTER_INFLUXBATCHSIZE"              default:"1000"                                                    description:"Minimum capacity of the cache array used to send data to Influx"`
	outputChannel       chan util.DNSResult
	closeChannel        chan bool
}

func init() {
	c := influxConfig{}
	if _, err := util.GlobalParser.AddGroup("influx_output", "Influx Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)
}

// Initialize function should not block. otherwise the dispatcher will get stuck
func (c influxConfig) Initialize(ctx context.Context) error {
	if c.InfluxOutputType > 0 && c.InfluxOutputType < 5 {
		log.Info("Creating Influx Output Channel")
		go c.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (c influxConfig) Close() {
	// todo: implement this
	<-c.closeChannel
}

func (c influxConfig) OutputChannel() chan util.DNSResult {
	return c.outputChannel
}

func (c influxConfig) connectInfluxRetry() influxdb2.Client {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if c.InfluxOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn := c.connectInflux()
		if conn != nil {
			return conn
		}

		// Error getting connection, wait the timer or check if we are exiting
		<-tick.C
		continue

	}
}

func (c influxConfig) connectInflux() influxdb2.Client {
	client := influxdb2.NewClientWithOptions(c.InfluxOutputServer, c.InfluxOutputToken, influxdb2.DefaultOptions().SetBatchSize(c.InfluxBatchSize))
	return client
}

func (c influxConfig) Output(ctx context.Context) {
	for i := 0; i < int(c.InfluxOutputWorkers); i++ {
		go c.InfluxWorker()
	}
}

func (c influxConfig) InfluxWorker() {
	influxSentToOutput := metrics.GetOrRegisterCounter("influxSentToOutput", metrics.DefaultRegistry)
	influxSkipped := metrics.GetOrRegisterCounter("stdoutSkipped", metrics.DefaultRegistry)
	client := c.connectInfluxRetry()
	writeAPI := client.WriteAPI(c.InfluxOutputOrg, c.InfluxOutputBucket)

	for data := range c.outputChannel {
		for _, dnsQuery := range data.DNS.Question {
			if util.CheckIfWeSkip(c.InfluxOutputType, dnsQuery.Name) {
				influxSkipped.Inc(1)
				continue
			}
			influxSentToOutput.Inc(1)

			edns, dobit := uint8(0), uint8(0)
			if edns0 := data.DNS.IsEdns0(); edns0 != nil {
				edns = 1
				if edns0.Do() {
					dobit = 1
				}
			}
			row := map[string]interface{}{
				"ipversion":    data.IPVersion,
				"SrcIP":        data.SrcIP,
				"DstIP":        data.DstIP,
				"protocol":     data.Protocol,
				"qr":           data.DNS.Response,
				"opCode":       data.DNS.Opcode,
				"class":        dnsQuery.Qclass,
				"qtype":        dnsQuery.Qtype,
				"responseCode": data.DNS.Rcode,
				"question":     dnsQuery.Name,
				"size":         data.PacketLength,
				"edns":         edns,
				"dobit":        dobit,
				"id":           data.DNS.Id,
			}

			p := influxdb2.NewPoint("system", map[string]string{
				"hostname": util.GeneralFlags.ServerName,
			}, row, data.Timestamp)
			writeAPI.WritePoint(p)
		}
	}

	// Force all unwritten data to be sent
	writeAPI.Flush()
	// Ensures background processes finishes
	client.Close()
}

// This will allow an instance to be spawned at import time
// var _ = influxConfig{}.initializeFlags()
