package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

var stdoutstats = outputStats{"Stdout", 0, 0}
var fileoutstats = outputStats{"File", 0, 0}

func stdoutOutput(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	printStatsTicker := time.Tick(*printStatsDelay)

	for {
		select {
		case data := <-resultChannel:
			for _, dnsQuery := range data.DNS.Question {

				if checkIfWeSkip(*stdoutOutputType, dnsQuery.Name) {
					stdoutstats.Skipped++
					continue
				}
				stdoutstats.SentToOutput++

				fullQuery, _ := json.Marshal(data)
				fmt.Printf("%s\n", fullQuery)
			}
		case <-exiting:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", stdoutstats)
		}
	}
}

func fileOutput(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	var fileObject *os.File
	if *fileOutputType > 0 {
		var err error
		fileObject, err = os.OpenFile(*fileOutputPath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		errorHandler(err)
		defer fileObject.Close()
	}
	printStatsTicker := time.Tick(*printStatsDelay)

	for {
		select {
		case data := <-resultChannel:
			for _, dnsQuery := range data.DNS.Question {

				if checkIfWeSkip(*fileOutputType, dnsQuery.Name) {
					fileoutstats.Skipped++
					continue
				}
				fileoutstats.SentToOutput++

				fullQuery, _ := json.Marshal(data)
				_, err := fileObject.WriteString(fmt.Sprintf("%s\n", fullQuery))
				errorHandler(err)
			}
		case <-exiting:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", fileoutstats)
		}
	}
}
