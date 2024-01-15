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
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/mosajjal/dnsmonster/internal/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type zincConfig struct {
	ZincOutputType     uint          `long:"zincoutputtype"           ini-name:"zincoutputtype"           env:"DNSMONSTER_ZINCOUTPUTTYPE"           default:"0"                   description:"What should be written to zinc. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"       choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	ZincOutputIndex    string        `long:"zincoutputindex"          ini-name:"zincoutputindex"          env:"DNSMONSTER_ZINCOUTPUTINDEX"          default:"dnsmonster"          description:"index used to save data in Zinc"`
	ZincOutputEndpoint string        `long:"zincoutputendpoint"       ini-name:"zincoutputendpoint"       env:"DNSMONSTER_ZINCOUTPUTENDPOINT"       default:""                    description:"zinc endpoint address, example: http://127.0.0.1:9200/api/default/_bulk. Used if zincOutputType is not none"`
	ZincOutputUsername string        `long:"zincoutputusername"       ini-name:"zincoutputusername"       env:"DNSMONSTER_ZINCOUTPUTUSERNAME"       default:""                    description:"zinc username, example: admin@admin.com. Used if zincOutputType is not none"`
	ZincOutputpassword string        `long:"zincoutputpassword"       ini-name:"zincoutputpassword"       env:"DNSMONSTER_ZINCOUTPUTPASSWORD"       default:""                    description:"zinc password, example: password. Used if zincOutputType is not none"`
	ZincBatchSize      uint          `long:"zincbatchsize"            ini-name:"zincbatchsize"            env:"DNSMONSTER_ZINCBATCHSIZE"            default:"1000"                description:"Send data to Zinc in batch sizes"`
	ZincBatchDelay     time.Duration `long:"zincbatchdelay"           ini-name:"zincbatchdelay"           env:"DNSMONSTER_ZINCBATCHDELAY"           default:"1s"                  description:"Interval between sending results to Zinc if Batch size is not filled"`
	ZincTimeout        time.Duration `long:"zinctimeout"              ini-name:"zinctimeout"              env:"DNSMONSTER_ZINCTIMEOUT"              default:"10s"                 description:"Zing request timeout"`
	outputChannel      chan util.DNSResult
	outputMarshaller   util.OutputMarshaller
	closeChannel       chan bool
}

func init() {
	c := zincConfig{}
	if _, err := util.GlobalParser.AddGroup("zinc_output", "Zinc Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (zConfig *zincConfig) Initialize(ctx context.Context) error {

	var err error
	zConfig.outputMarshaller, _, err = util.OutputFormatToMarshaller("json", "")
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

	if zConfig.ZincOutputType > 0 && zConfig.ZincOutputType < 5 {
		log.Info("Creating Zinc Output Channel")
		go zConfig.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (zConfig zincConfig) Close() {
	// todo: implement this
	<-zConfig.closeChannel
}

func (zConfig zincConfig) OutputChannel() chan util.DNSResult {
	return zConfig.outputChannel
}

func (zConfig zincConfig) connectzinc(ctx context.Context) *http.Client {

	// TODO: TLS support
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: zConfig.ZincTimeout,
	}

	return client

}

func (zConfig zincConfig) Output(ctx context.Context) {

	client := zConfig.connectzinc(ctx)
	batch := make([]byte, 0, zConfig.ZincBatchSize)
	c := 0

	ticker := time.NewTicker(zConfig.ZincBatchDelay)

	itemPrefix := fmt.Sprintf(`{ "index" : { "_index" : "%s" } }`, zConfig.ZincOutputIndex)

	for {
		select {
		case data := <-zConfig.outputChannel:
			c++
			if util.GeneralFlags.PacketLimit == 0 || len(batch) < util.GeneralFlags.PacketLimit {
				batch = append(batch, []byte(itemPrefix)...)
				batch = append(batch, '\n')
				batch = append(batch, zConfig.outputMarshaller.Marshal(data)...)
				// add a newline to the end of the batch
				batch = append(batch, '\n')
			}
			if c >= int(zConfig.ZincBatchSize) {
				// Batch size is reached first, flush and stop the timer
				if err := zConfig.zincSendData(ctx, client, batch); err != nil {
					log.Info(err)
				}
				client = zConfig.connectzinc(ctx)
				batch = make([]byte, 0, zConfig.ZincBatchSize)
				ticker.Stop()
				// } else if !ticker.Stop() {
				// 	// Timer is not already stopped, so reset it
				// 	ticker.Reset(zConfig.ZincBatchDelay)
			}

		case <-ticker.C:

			if c > 0 {
				if err := zConfig.zincSendData(ctx, client, batch); err != nil {
					log.Info(err)
				}
				client = zConfig.connectzinc(ctx)
				batch = make([]byte, 0, zConfig.ZincBatchSize)
				c = 0
			}

		}
	}
}

func (zConfig *zincConfig) zincSendData(ctx context.Context, client *http.Client, batch []byte) error {
	sentCount := metrics.GetOrRegisterCounter("zincSent", metrics.DefaultRegistry)
	failedCount := metrics.GetOrRegisterCounter("zincFailed", metrics.DefaultRegistry)
	// convery batch to io.Reader
	data := bytes.NewReader(batch)

	req, err := http.NewRequest("POST", zConfig.ZincOutputEndpoint, data)
	if err != nil {
		return err
	}
	req.SetBasicAuth(zConfig.ZincOutputUsername, zConfig.ZincOutputpassword)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "dnsmonster")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	// we don't care about the response, just make sure we get a 200
	_, err = io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		records := bytes.Count(batch, []byte{'\n'}) / 2
		failedCount.Inc(int64(records))
		return fmt.Errorf("zinc returned status code %d", resp.StatusCode)
	}
	// count newlines in batch to get number of records. /2 because we have 2 newlines per record
	records := bytes.Count(batch, []byte{'\n'}) / 2
	sentCount.Inc(int64(records))
	return nil
}

// vim: foldmethod=marker
