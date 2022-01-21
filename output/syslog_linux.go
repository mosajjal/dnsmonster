package output

import (
	"net/url"
	"time"

	syslog "log/syslog"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

func connectSyslogRetry(sysConfig types.SyslogConfig) *syslog.Writer {
	tick := time.NewTicker(5 * time.Second)
	// don't retry connection if we're doing dry run
	if sysConfig.SyslogOutputType == 0 {
		tick.Stop()
	}
	defer tick.Stop()
	for {
		conn, err := connectSyslog(sysConfig)
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

func connectSyslog(sysConfig types.SyslogConfig) (*syslog.Writer, error) {
	u, _ := url.Parse(sysConfig.SyslogOutputEndpoint)
	log.Infof("Connecting to syslog server %v with protocol %v", u.Host, u.Scheme)
	sysLog, err := syslog.Dial(u.Scheme, u.Host, syslog.LOG_WARNING|syslog.LOG_DAEMON, sysConfig.General.ServerName)
	if err != nil {
		return nil, err
	}
	return sysLog, err
}

func SyslogOutput(sysConfig types.SyslogConfig) {
	writer := connectSyslogRetry(sysConfig)
	syslogSentToOutput := metrics.GetOrRegisterCounter("syslogSentToOutput", metrics.DefaultRegistry)
	syslogSkipped := metrics.GetOrRegisterCounter("syslogSkipped", metrics.DefaultRegistry)

	for data := range sysConfig.ResultChannel {
		for _, dnsQuery := range data.DNS.Question {

			if util.CheckIfWeSkip(sysConfig.SyslogOutputType, dnsQuery.Name) {
				syslogSkipped.Inc(1)
				continue
			}
			syslogSentToOutput.Inc(1)

			err := writer.Alert(data.String())
			// don't exit on connection failure, try to connect again if need be
			if err != nil {
				log.Info(err)
			}
			// we should skip to the next data since we've already saved all the questions. Multi-Question DNS queries are not common
			continue
		}
	}
}
