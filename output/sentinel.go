package output

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
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

type SignatureElements struct {
	Date          string // in rfc1123date format ('%a, %d %b %Y %H:%M:%S GMT')
	ContentLength uint
	Method        string
	ContentType   string
	Resource      string
}

func BuildSignature(s SignatureElements, seConfig types.SentinelConfig) string {
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
	if err := tmpl.Execute(&buf, s); err != nil {
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

func sendBatch(batch string, count int, seConfig types.SentinelConfig) {
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
	signature := BuildSignature(s, seConfig)
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
func SentinelOutput(seConfig types.SentinelConfig) {
	log.Infof("starting SentinelOutput")
	sentinelSkipped := metrics.GetOrRegisterCounter("sentinelSkipped", metrics.DefaultRegistry)

	batch := "["
	cnt := 0
	//todo: solve the batch delay issue
	for data := range seConfig.ResultChannel {
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
				sendBatch(batch, cnt, seConfig)
				//reset counters
				batch = "["
				cnt = 0
			}
		}
	}
}
