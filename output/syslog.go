package output

import (
	"errors"
	"net/url"
	"time"

	syslog "github.com/hashicorp/go-syslog"

	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type SyslogConfig struct {
	SyslogOutputType     uint   `long:"syslogOutputType"            env:"DNSMONSTER_SYSLOGOUTPUTTYPE"            default:"0"                                                       description:"What should be written to Syslog server. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic" choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	SyslogOutputEndpoint string `long:"syslogOutputEndpoint"        env:"DNSMONSTER_SYSLOGOUTPUTENDPOINT"        default:"udp://127.0.0.1:514"                                     description:"Syslog endpoint address, example: udp://127.0.0.1:514, tcp://127.0.0.1:514. Used if syslogOutputType is not none"`
	outputChannel        chan util.DNSResult
	closeChannel         chan bool
	outputMarshaller     util.OutputMarshaller
}

func (config SyslogConfig) initializeFlags() error {
	// this line will run at import time, before parsing the flags, hence showing up in --help as well as actually working
	_, err := util.GlobalParser.AddGroup("syslog_output", "Syslog Output", &config)

	config.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)

	util.GlobalDispatchList = append(util.GlobalDispatchList, &config)
	return err
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (config SyslogConfig) Initialize() error {
	var err error
	config.outputMarshaller, _, err = util.OutputFormatToMarshaller("json", "")
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}
	if config.SyslogOutputType > 0 && config.SyslogOutputType < 5 {
		log.Info("Creating Syslog Output Channel")
		go config.Output()
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (config SyslogConfig) Close() {
	// todo: implement this
	<-config.closeChannel
}

func (config SyslogConfig) OutputChannel() chan util.DNSResult {
	return config.outputChannel
}

func (sysConfig SyslogConfig) connectSyslogRetry() syslog.Syslogger {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if sysConfig.SyslogOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := sysConfig.connectSyslog()
		if err == nil {
			return conn
		} else {
			log.Info(err)
		}

		// Error getting connection, wait the timer or check if we are exiting
		<-tick.C
		continue

	}
}

func (sysConfig SyslogConfig) connectSyslog() (syslog.Syslogger, error) {
	u, _ := url.Parse(sysConfig.SyslogOutputEndpoint)
	log.Infof("Connecting to syslog server %v with protocol %v", u.Host, u.Scheme)
	sysLog, err := syslog.DialLogger(u.Scheme, u.Host, syslog.LOG_WARNING, "USER", util.GeneralFlags.ServerName) // todo: maybe facility as a parameter?
	if err != nil {
		return nil, err
	}
	return sysLog, err
}

func (sysConfig SyslogConfig) Output() {
	writer := sysConfig.connectSyslogRetry()
	syslogSentToOutput := metrics.GetOrRegisterCounter("syslogSentToOutput", metrics.DefaultRegistry)
	syslogSkipped := metrics.GetOrRegisterCounter("syslogSkipped", metrics.DefaultRegistry)

	for data := range sysConfig.outputChannel {
		for _, dnsQuery := range data.DNS.Question {

			if util.CheckIfWeSkip(sysConfig.SyslogOutputType, dnsQuery.Name) {
				syslogSkipped.Inc(1)
				continue
			}
			syslogSentToOutput.Inc(1)

			err := writer.WriteLevel(syslog.LOG_ALERT, []byte(sysConfig.outputMarshaller.Marshal(data)))
			// don't exit on connection failure, try to connect again if need be
			if err != nil {
				log.Info(err)
			}
			// we should skip to the next data since we've already saved all the questions. Multi-Question DNS queries are not common
			continue
		}
	}
}

// This will allow an instance to be spawned at import time
var _ = SyslogConfig{}.initializeFlags()
