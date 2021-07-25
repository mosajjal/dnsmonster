package main

import (
	"sync"
	"time"
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
		allowDomainsFileTicker.Stop()
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
			if skipDomainMapBool {
				skipDomainMap = loadDomainsToMap(generalOptions.SkipDomainsFile)
			} else {
				skipDomainList = loadDomainsToList(generalOptions.SkipDomainsFile)
			}
		case <-allowDomainsFileTickerChan:
			if allowDomainMapBool {
				allowDomainMap = loadDomainsToMap(generalOptions.AllowDomainsFile)
			} else {
				allowDomainList = loadDomainsToList(generalOptions.AllowDomainsFile)
			}
		}
	}
}
