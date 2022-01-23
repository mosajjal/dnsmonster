package output

import (
	"bytes"
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

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type SentinelConfig struct {
	SentinelOutputType       uint          `long:"sentinelOutputType"          env:"DNSMONSTER_SENTINELOUTPUTTYPE"          default:"0"                                                       description:"What should be written to Microsoft Sentinel. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic" choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	SentinelOutputSharedKey  string        `long:"sentinelOutputSharedKey"     env:"DNSMONSTER_SENTINELOUTPUTSHAREDKEY"     default:""                                                        description:"Sentinel Shared Key, either the primary or secondary, can be found in Agents Management page under Log Analytics workspace"`
	SentinelOutputCustomerId string        `long:"sentinelOutputCustomerId"    env:"DNSMONSTER_SENTINELOUTPUTCUSTOMERID"    default:""                                                        description:"Sentinel Customer Id. can be found in Agents Management page under Log Analytics workspace"`
	SentinelOutputLogType    string        `long:"sentinelOutputLogType"       env:"DNSMONSTER_SENTINELOUTPUTLOGTYPE"       default:"dnsmonster"                                              description:"Sentinel Output LogType"`
	SentinelOutputProxy      string        `long:"sentinelOutputProxy"         env:"DNSMONSTER_SENTINELOUTPUTPROXY"         default:""                                                        description:"Sentinel Output Proxy in URI format"`
	SentinelBatchSize        uint          `long:"sentinelBatchSize"           env:"DNSMONSTER_SENTINELBATCHSIZE"           default:"100"                                                     description:"Sentinel Batch Size"`
	SentinelBatchDelay       time.Duration `long:"sentinelBatchDelay"          env:"DNSMONSTER_SENTINELBATCHDELAY"          default:"1s"                                                      description:"Interval between sending results to Sentinel if Batch size is not filled"`
	outputChannel            chan types.DNSResult
	closeChannel             chan bool
}

func (seConfig SentinelConfig) initializeFlags() error {
	// this line will run at import time, before parsing the flags, hence showing up in --help as well as actually working
	_, err := util.GlobalParser.AddGroup("sentinel_output", "Microsoft Sentinel Output", &seConfig)

	seConfig.outputChannel = make(chan types.DNSResult, util.GeneralFlags.ResultChannelSize)

	types.GlobalDispatchList = append(types.GlobalDispatchList, &seConfig)
	return err
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (seConfig SentinelConfig) Initialize() error {
	if seConfig.SentinelOutputType > 0 && seConfig.SentinelOutputType < 5 {
		log.Info("Creating Sentinel Output Channel")
		go seConfig.Output()
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (seConfig SentinelConfig) Close() {
	//todo: implement this
	<-seConfig.closeChannel
}

func (seConfig SentinelConfig) OutputChannel() chan types.DNSResult {
	return seConfig.outputChannel
}

// don't think this needs to be a struct type, might be better to define it as a variable
type SignatureElements struct {
	Date          string // in rfc1123date format ('%a, %d %b %Y %H:%M:%S GMT')
	ContentLength uint
	Method        string
	ContentType   string
	Resource      string
}

func (seConfig SentinelConfig) BuildSignature(sigelements SignatureElements) string {
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
	signature := fmt.Sprintf("SharedKey %s:%s", seConfig.SentinelOutputCustomerId, base64.StdEncoding.EncodeToString(h.Sum(nil)))
	return signature
}

func (seConfig SentinelConfig) sendBatch(batch string, count int) {
	sentinelSentToOutput := metrics.GetOrRegisterCounter("sentinelSentToOutput", metrics.DefaultRegistry)
	sentinelFailed := metrics.GetOrRegisterCounter("sentinelFailed", metrics.DefaultRegistry)
	// send batch to Microsoft Sentinel
	// build signature
	location, _ := time.LoadLocation("GMT")
	s := SignatureElements{
		Date:          time.Now().In(location).Format(time.RFC1123),
		Method:        "POST",
		ContentLength: uint(len(batch)),
		ContentType:   "application/json",
		Resource:      "/api/logs",
	}
	signature := seConfig.BuildSignature(s)
	// build request
	uri := "https://" + seConfig.SentinelOutputCustomerId + ".ods.opinsights.azure.com" + s.Resource + "?api-version=2016-04-01"
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
func (seConfig SentinelConfig) Output() {

	log.Infof("starting SentinelOutput")
	sentinelSkipped := metrics.GetOrRegisterCounter("sentinelSkipped", metrics.DefaultRegistry)

	batch := "["
	cnt := 0
	//todo: solve the batch delay issue
	for data := range seConfig.outputChannel {
		for _, dnsQuery := range data.DNS.Question {

			if util.CheckIfWeSkip(seConfig.SentinelOutputType, dnsQuery.Name) {
				sentinelSkipped.Inc(1)
				continue
			}

			cnt++
			batch += data.String()
			batch += ","
			if cnt == int(seConfig.SentinelBatchSize) {
				// remove the last ,
				batch = strings.TrimSuffix(batch, ",")
				batch += "]"
				seConfig.sendBatch(batch, cnt)
				//reset counters
				batch = "["
				cnt = 0
			}
		}
	}
}

// actually run this as a goroutine
var _ = SentinelConfig{}.initializeFlags()
