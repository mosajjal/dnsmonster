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
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mosajjal/dnsmonster/internal/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

// SentinelConfig is the configuration and runtime struct for Sentinel output.
type SentinelConfig struct {
	OutputType       uint
	SharedKey        string
	CustomerID       string
	LogType          string
	Proxy            string
	BatchSize        uint
	BatchDelay       time.Duration
	outputChannel    chan util.DNSResult
	outputMarshaller util.OutputMarshaller
	closeChannel     chan bool
}

// NewSentinelConfig creates a new SentinelConfig with default values.
func NewSentinelConfig() *SentinelConfig {
	return &SentinelConfig{}
}

// WithOutputType sets the OutputType and returns the config for chaining.
func (c *SentinelConfig) WithOutputType(t uint) *SentinelConfig {
	c.OutputType = t
	return c
}
func (c *SentinelConfig) WithSharedKey(k string) *SentinelConfig {
	c.SharedKey = k
	return c
}
func (c *SentinelConfig) WithCustomerID(id string) *SentinelConfig {
	c.CustomerID = id
	return c
}
func (c *SentinelConfig) WithLogType(lt string) *SentinelConfig {
	c.LogType = lt
	return c
}
func (c *SentinelConfig) WithProxy(p string) *SentinelConfig {
	c.Proxy = p
	return c
}
func (c *SentinelConfig) WithBatchSize(bs uint) *SentinelConfig {
	c.BatchSize = bs
	return c
}
func (c *SentinelConfig) WithBatchDelay(d time.Duration) *SentinelConfig {
	c.BatchDelay = d
	return c
}
func (c *SentinelConfig) WithChannelSize(channelSize int) *SentinelConfig {
	c.outputChannel = make(chan util.DNSResult, channelSize)
	c.closeChannel = make(chan bool)
	return c
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (seConfig *SentinelConfig) Initialize(ctx context.Context) error {
	var err error
	seConfig.outputMarshaller, _, err = util.OutputFormatToMarshaller("json", "")
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

	if seConfig.OutputType > 0 && seConfig.OutputType < 5 {
		log.Info("Creating Sentinel Output Channel")
		go seConfig.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (seConfig *SentinelConfig) Close() {
	// todo: implement this
	<-seConfig.closeChannel
}

func (seConfig *SentinelConfig) OutputChannel() chan util.DNSResult {
	return seConfig.outputChannel
}

// todo: don't think this needs to be a struct type, might be better to define it as a variable
type signatureElements struct {
	Date          string // in rfc1123date format ('%a, %d %b %Y %H:%M:%S GMT')
	ContentLength uint
	Method        string
	ContentType   string
	Resource      string
}

func (seConfig *SentinelConfig) BuildSignature(sigelements signatureElements) string {
	// build HMAC signature
	tmpl, err := template.New("sign").Parse(`{{.Method}}
{{.ContentLength}}
{{.ContentType}}
x-ms-date:{{.Date}}
{{.Resource}}`)
	if err != nil {
		log.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, sigelements); err != nil {
		log.Fatal(err)
	}
	sharedKeyBytes, err := base64.StdEncoding.DecodeString(seConfig.SharedKey)
	if err != nil {
		panic(err)
	}
	h := hmac.New(sha256.New, []byte(sharedKeyBytes))
	h.Write(buf.Bytes())
	signature := fmt.Sprintf("SharedKey %s:%s", seConfig.CustomerID, base64.StdEncoding.EncodeToString(h.Sum(nil)))
	return signature
}

func (seConfig *SentinelConfig) sendBatch(batch string, count int) {
	sentinelSentToOutput := metrics.GetOrRegisterCounter("sentinelSentToOutput", metrics.DefaultRegistry)
	sentinelFailed := metrics.GetOrRegisterCounter("sentinelFailed", metrics.DefaultRegistry)
	// send batch to Microsoft Sentinel
	// build signature
	location, _ := time.LoadLocation("GMT")
	s := signatureElements{
		Date:          time.Now().In(location).Format(time.RFC1123),
		Method:        "POST",
		ContentLength: uint(len(batch)),
		ContentType:   "application/json",
		Resource:      "/api/logs",
	}
	signature := seConfig.BuildSignature(s)
	// build request
	uri := "https://" + seConfig.CustomerID + ".ods.opinsights.azure.com" + s.Resource + "?api-version=2016-04-01"
	headers := map[string]string{
		"x-ms-date":     s.Date,
		"content-type":  s.ContentType,
		"Authorization": signature,
		"Log-Type":      seConfig.LogType,
	}
	// send request
	req, err := http.NewRequest("POST", uri, bytes.NewBuffer([]byte(batch)))
	var res *http.Response
	if err != nil {
		panic(err)
	}
	for k, v := range headers {
		req.Header[k] = []string{v}
	}
	if seConfig.Proxy != "" {
		proxyURL, err := url.Parse(seConfig.Proxy)
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
		sentinelSentToOutput.Inc(int64(count))
	} else {
		log.Errorf("batch not sent, with code %d", res.StatusCode)
		sentinelFailed.Inc(int64(count))
	}
}

func (seConfig *SentinelConfig) Output(ctx context.Context) {
	log.Infof("starting SentinelOutput")
	sentinelSkipped := metrics.GetOrRegisterCounter("sentinelSkipped", metrics.DefaultRegistry)

	batch := "["
	cnt := uint(0)

	ticker := time.NewTicker(time.Second * 5)
	div := 0
	if seConfig.BatchDelay > 0 {
		seConfig.BatchSize = 1
		div = -1
		ticker = time.NewTicker(seConfig.BatchDelay)
	} else {
		ticker.Stop()
	}
	for {
		select {
		case data := <-seConfig.outputChannel:
			for _, dnsQuery := range data.DNS.Question {

				if util.CheckIfWeSkip(seConfig.OutputType, dnsQuery.Name) {
					sentinelSkipped.Inc(1)
					continue
				}

				cnt++
				batch += string(seConfig.outputMarshaller.Marshal(data))
				batch += ","
				if int(cnt%seConfig.BatchSize) == div {
					// remove the last ,
					batch = strings.TrimSuffix(batch, ",")
					batch += "]"
					seConfig.sendBatch(batch, int(cnt))
					// reset counters
					batch = "["
					cnt = 0
				}
			}
		case <-ticker.C:
			batch = strings.TrimSuffix(batch, ",")
			batch += "]"
			seConfig.sendBatch(batch, int(cnt))
			// reset counters
			batch = "["
			cnt = 0
		}
	}
}

// This will allow an instance to be spawned at import time
// var _ = sentinelConfig{}.initializeFlags()
// vim: foldmethod=marker
