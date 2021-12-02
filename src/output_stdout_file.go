package main

import (
	"fmt"
	"os"
	"time"

	"github.com/mosajjal/dnsmonster/types"
	log "github.com/sirupsen/logrus"
)

var stdoutstats = outputStats{"Stdout", 0, 0}
var fileoutstats = outputStats{"File", 0, 0}

func stdoutOutputWorker(stdConfig stdoutConfig) {
	printStatsTicker := time.Tick(stdConfig.general.printStatsDelay)

	for {
		select {
		case data := <-stdConfig.resultChannel:
			for _, dnsQuery := range data.DNS.Question {

				if checkIfWeSkip(stdConfig.stdoutOutputType, dnsQuery.Name) {
					stdoutstats.Skipped++
					continue
				}
				stdoutstats.SentToOutput++

				fmt.Printf("%s\n", data.String())
			}
		case <-types.GlobalExitChannel:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", stdoutstats)
		}
	}
}

func stdoutOutput(stdConfig stdoutConfig) {
	for i := 0; i < 8; i++ {
		go stdoutOutputWorker(stdConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
	}
}

func fileOutput(fConfig fileConfig) {
	var fileObject *os.File
	if fConfig.fileOutputType > 0 {
		var err error
		fileObject, err = os.OpenFile(fConfig.fileOutputPath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		errorHandler(err)
		defer fileObject.Close()
	}
	printStatsTicker := time.Tick(fConfig.general.printStatsDelay)

	for {
		select {
		case data := <-fConfig.resultChannel:
			for _, dnsQuery := range data.DNS.Question {

				if checkIfWeSkip(fConfig.fileOutputType, dnsQuery.Name) {
					fileoutstats.Skipped++
					continue
				}
				fileoutstats.SentToOutput++

				_, err := fileObject.WriteString(fmt.Sprintf("%s\n", data.String()))
				errorHandler(err)
			}
		case <-types.GlobalExitChannel:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", fileoutstats)
		}
	}
}
