/* {{{ Copyright (C) 2022 Ali Mosajjal
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. }}} */

package output

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mosajjal/dnsmonster/internal/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type victoriaConfig struct {
	VictoriaOutputEndpoint string        `long:"victoriaoutputendpoint"      ini-name:"victoriaoutputendpoint"      env:"DNSMONSTER_VICTORIAOUTPUTENDPOINT"      default:""       description:"Victoria Output Endpoint. example: http://localhost:9428/insert/jsonline?_msg_field=rcode_id&_time_field=time"`
	VictoriaOutputType     uint          `long:"victoriaoutputtype"          ini-name:"victoriaoutputtype"          env:"DNSMONSTER_VICTORIAOUTPUTTYPE"          default:"0"      description:"What should be written to Microsoft Victoria. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic" choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	VictoriaOutputProxy    string        `long:"victoriaoutputproxy"         ini-name:"victoriaoutputproxy"         env:"DNSMONSTER_VICTORIAOUTPUTPROXY"         default:""       description:"Victoria Output Proxy in URI format"`
	VictoriaOutputWorkers  uint          `long:"victoriaoutputworkers"       ini-name:"victoriaoutputworkers"       env:"DNSMONSTER_VICTORIAOUTPUTWORKERS"       default:"8"      description:"Number of workers"`
	VictoriaBatchSize      uint          `long:"victoriabatchsize"           ini-name:"victoriabatchsize"           env:"DNSMONSTER_VICTORIABATCHSIZE"           default:"100"    description:"Victoria Batch Size"`
	VictoriaBatchDelay     time.Duration `long:"victoriabatchdelay"          ini-name:"victoriabatchdelay"          env:"DNSMONSTER_VICTORIABATCHDELAY"          default:"0s"     description:"Interval between sending results to Victoria if Batch size is not filled. Any value larger than zero takes precedence over Batch Size"`
	outputChannel          chan util.DNSResult
	outputMarshaller       util.OutputMarshaller
	closeChannel           chan bool
}

func init() {
	c := victoriaConfig{}
	if _, err := util.GlobalParser.AddGroup("victoria_output", "Victoria Logs Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (viConfig victoriaConfig) Initialize(ctx context.Context) error {
	var err error
	viConfig.outputMarshaller, _, err = util.OutputFormatToMarshaller("json-ocsf", "")
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

	if viConfig.VictoriaOutputType > 0 && viConfig.VictoriaOutputType < 5 {
		log.Info("Creating Victoria Output Channel")
		go viConfig.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (viConfig victoriaConfig) Close() {
	// todo: implement this
	<-viConfig.closeChannel
}

func (viConfig victoriaConfig) OutputChannel() chan util.DNSResult {
	return viConfig.outputChannel
}

func (viConfig victoriaConfig) sendBatch(batch string, count int) {
	victoriaSentToOutput := metrics.GetOrRegisterCounter("victoriaSentToOutput", metrics.DefaultRegistry)
	victoriaFailed := metrics.GetOrRegisterCounter("victoriaFailed", metrics.DefaultRegistry)

	// build request
	headers := map[string]string{
		"content-type": "application/json",
	}
	// send request
	req, err := http.NewRequest("POST", viConfig.VictoriaOutputEndpoint, bytes.NewBuffer([]byte(batch)))
	var res *http.Response
	if err != nil {
		panic(err)
	}
	for k, v := range headers {
		req.Header[k] = []string{v}
	}
	if viConfig.VictoriaOutputProxy != "" {
		proxyURL, err := url.Parse(viConfig.VictoriaOutputProxy)
		if err != nil {
			panic(err)
		}
		client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
		res, err = client.Do(req)
		if err != nil {
			panic(err)
		}
	} else {
		res, err = http.DefaultClient.Do(req)
		if err != nil {
			panic(err)
		}
	}
	if res.StatusCode >= 200 && res.StatusCode < 300 {
		log.Infof("batch sent, with code %d", res.StatusCode)
		victoriaSentToOutput.Inc(int64(count))
	} else {
		log.Errorf("batch not sent, with code %d", res.StatusCode)
		victoriaFailed.Inc(int64(count))
	}
}

func (viConfig victoriaConfig) victoriaOutputWorker(_ context.Context) {
	log.Infof("starting VictoriaOutput")
	victoriaSkipped := metrics.GetOrRegisterCounter("victoriaSkipped", metrics.DefaultRegistry)

	batch := "["
	cnt := uint(0)

	ticker := time.NewTicker(time.Second * 5)
	div := 0
	if viConfig.VictoriaBatchDelay > 0 {
		viConfig.VictoriaBatchSize = 1
		div = -1
		ticker = time.NewTicker(viConfig.VictoriaBatchDelay)
	} else {
		ticker.Stop()
	}
	for {
		select {
		case data := <-viConfig.outputChannel:
			for _, dnsQuery := range data.DNS.Question {

				if util.CheckIfWeSkip(viConfig.VictoriaOutputType, dnsQuery.Name) {
					victoriaSkipped.Inc(1)
					continue
				}

				cnt++
				batch += string(viConfig.outputMarshaller.Marshal(data))
				batch += "\n"
				if int(cnt%viConfig.VictoriaBatchSize) == div {
					// remove the last ,
					batch = strings.TrimSuffix(batch, "\n")
					viConfig.sendBatch(batch, int(cnt))
					// reset counters
					batch = ""
					cnt = 0
				}
			}
		case <-ticker.C:
			batch = strings.TrimSuffix(batch, "\n")
			viConfig.sendBatch(batch, int(cnt))
			// reset counters
			batch = ""
			cnt = 0
		}
	}
}

func (viConfig victoriaConfig) Output(ctx context.Context) {
	for i := 0; i < int(viConfig.VictoriaOutputWorkers); i++ { // todo: make this configurable
		go viConfig.victoriaOutputWorker(ctx)
	}
}

// This will allow an instance to be spawned at import time
// var _ = victoriaConfig{}.initializeFlags()
// vim: foldmethod=marker
