package main

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

func dispatchOutput(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	// Set up various tickers for different tasks
	skipDomainsFileTicker := time.NewTicker(generalOptions.SkipDomainsRefreshInterval)
	skipDomainsFileTickerChan := skipDomainsFileTicker.C
	if generalOptions.SkipDomainsFile == "" {
		skipDomainsFileTicker.Stop()
	}

	allowDomainsFileTicker := time.NewTicker(generalOptions.AllowDomainsRefreshInterval)
	allowDomainsFileTickerChan := allowDomainsFileTicker.C
	if generalOptions.AllowDomainsFile == "" {
		log.Infof("skipping allowDomains refresh since it's empty")
		allowDomainsFileTicker.Stop()
	} else {
		log.Infof("allowDomains refresh interval is %s", generalOptions.AllowDomainsRefreshInterval)
	}

	for {
		select {
		case data := <-resultChannel:
			if outputOptions.StdoutOutputType > 0 {
				stdoutResultChannel <- data
			}
			if outputOptions.FileOutputType > 0 {
				fileResultChannel <- data
			}
			if outputOptions.SyslogOutputType > 0 {
				syslogResultChannel <- data
			}
			if outputOptions.ClickhouseOutputType > 0 {
				clickhouseResultChannel <- data
			}
			if outputOptions.KafkaOutputType > 0 {
				kafkaResultChannel <- data
			}
			if outputOptions.ElasticOutputType > 0 {
				elasticResultChannel <- data
			}
			if outputOptions.SplunkOutputType > 0 {
				splunkResultChannel <- data
			}
		case <-exiting:
			return
		case <-skipDomainsFileTickerChan:
			log.Infof("reached skipDomains tick")
			if skipDomainMapBool {
				skipDomainMap = loadDomainsToMap(generalOptions.SkipDomainsFile)
			} else {
				skipDomainList = loadDomainsToList(generalOptions.SkipDomainsFile)
			}
		case <-allowDomainsFileTickerChan:
			log.Infof("reached allowDomains tick")
			if allowDomainMapBool {
				allowDomainMap = loadDomainsToMap(generalOptions.AllowDomainsFile)
			} else {
				allowDomainList = loadDomainsToList(generalOptions.AllowDomainsFile)
			}
		}
	}
}
