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

	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type sentinelConfig struct {
	SentinelOutputType       uint          `long:"sentineloutputtype"          ini-name:"sentineloutputtype"          env:"DNSMONSTER_SENTINELOUTPUTTYPE"          default:"0"                                                       description:"What should be written to Microsoft Sentinel. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic" choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	SentinelOutputSharedKey  string        `long:"sentineloutputsharedkey"     ini-name:"sentineloutputsharedkey"     env:"DNSMONSTER_SENTINELOUTPUTSHAREDKEY"     default:""                                                        description:"Sentinel Shared Key, either the primary or secondary, can be found in Agents Management page under Log Analytics workspace"`
	SentinelOutputCustomerID string        `long:"sentineloutputcustomerid"    ini-name:"sentineloutputcustomerid"    env:"DNSMONSTER_SENTINELOUTPUTCUSTOMERID"    default:""                                                        description:"Sentinel Customer Id. can be found in Agents Management page under Log Analytics workspace"`
	SentinelOutputLogType    string        `long:"sentineloutputlogtype"       ini-name:"sentineloutputlogtype"       env:"DNSMONSTER_SENTINELOUTPUTLOGTYPE"       default:"dnsmonster"                                              description:"Sentinel Output LogType"`
	SentinelOutputProxy      string        `long:"sentineloutputproxy"         ini-name:"sentineloutputproxy"         env:"DNSMONSTER_SENTINELOUTPUTPROXY"         default:""                                                        description:"Sentinel Output Proxy in URI format"`
	SentinelBatchSize        uint          `long:"sentinelbatchsize"           ini-name:"sentinelbatchsize"           env:"DNSMONSTER_SENTINELBATCHSIZE"           default:"100"                                                     description:"Sentinel Batch Size"`
	SentinelBatchDelay       time.Duration `long:"sentinelbatchdelay"          ini-name:"sentinelbatchdelay"          env:"DNSMONSTER_SENTINELBATCHDELAY"          default:"0s"                                                      description:"Interval between sending results to Sentinel if Batch size is not filled. Any value larger than zero takes precedence over Batch Size"`
	outputChannel            chan util.DNSResult
	outputMarshaller         util.OutputMarshaller
	closeChannel             chan bool
}

func init() {
	c := sentinelConfig{}
	if _, err := util.GlobalParser.AddGroup("sentinel_output", "Microsoft Sentinel Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (seConfig sentinelConfig) Initialize(ctx context.Context) error {
	var err error
	seConfig.outputMarshaller, _, err = util.OutputFormatToMarshaller("json", "")
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

	if seConfig.SentinelOutputType > 0 && seConfig.SentinelOutputType < 5 {
		log.Info("Creating Sentinel Output Channel")
		go seConfig.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (seConfig sentinelConfig) Close() {
	// todo: implement this
	<-seConfig.closeChannel
}

func (seConfig sentinelConfig) OutputChannel() chan util.DNSResult {
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

func (seConfig sentinelConfig) BuildSignature(sigelements signatureElements) string {
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
	sharedKeyBytes, err := base64.StdEncoding.DecodeString(seConfig.SentinelOutputSharedKey)
	if err != nil {
		panic(err)
	}
	h := hmac.New(sha256.New, []byte(sharedKeyBytes))
	h.Write(buf.Bytes())
	signature := fmt.Sprintf("SharedKey %s:%s", seConfig.SentinelOutputCustomerID, base64.StdEncoding.EncodeToString(h.Sum(nil)))
	return signature
}

func (seConfig sentinelConfig) sendBatch(batch string, count int) {
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
	uri := "https://" + seConfig.SentinelOutputCustomerID + ".ods.opinsights.azure.com" + s.Resource + "?api-version=2016-04-01"
	headers := map[string]string{
		"x-ms-date":     s.Date,
		"content-type":  s.ContentType,
		"Authorization": signature,
		"Log-Type":      seConfig.SentinelOutputLogType,
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
	if seConfig.SentinelOutputProxy != "" {
		proxyURL, err := url.Parse(seConfig.SentinelOutputProxy)
		if err != nil {
			panic(err)
		}
		client := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}}
		res, err = client.Do(req)
		if err != nil {
			panic(err)
		}
	} else {
		// command, _ := http2curl.GetCurlCommand(req)
		// fmt.Println(command)
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

func (seConfig sentinelConfig) Output(ctx context.Context) {
	log.Infof("starting SentinelOutput")
	sentinelSkipped := metrics.GetOrRegisterCounter("sentinelSkipped", metrics.DefaultRegistry)

	batch := "["
	cnt := uint(0)

	ticker := time.NewTicker(time.Second * 5)
	div := 0
	if seConfig.SentinelBatchDelay > 0 {
		seConfig.SentinelBatchSize = 1
		div = -1
		ticker = time.NewTicker(seConfig.SentinelBatchDelay)
	} else {
		ticker.Stop()
	}
	for {
		select {
		case data := <-seConfig.outputChannel:
			for _, dnsQuery := range data.DNS.Question {

				if util.CheckIfWeSkip(seConfig.SentinelOutputType, dnsQuery.Name) {
					sentinelSkipped.Inc(1)
					continue
				}

				cnt++
				batch += seConfig.outputMarshaller.Marshal(data)
				batch += ","
				if int(cnt%seConfig.SentinelBatchSize) == div {
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
