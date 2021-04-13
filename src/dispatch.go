package main

import (
	"sync"
	"time"
)

func dispatchOutput(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	// Set up various tickers for different tasks
	skipDomainsFileTicker := time.NewTicker(*skipDomainsRefreshInterval)
	skipDomainsFileTickerChan := skipDomainsFileTicker.C
	if *skipDomainsFile == "" {
		skipDomainsFileTicker.Stop()
	}

	allowDomainsFileTicker := time.NewTicker(*allowDomainsRefreshInterval)
	allowDomainsFileTickerChan := allowDomainsFileTicker.C
	if *allowDomainsFile == "" {
		allowDomainsFileTicker.Stop()
	}

	for {
		select {
		case data := <-resultChannel:
			if *stdoutOutputType > 0 {
				stdoutResultChannel <- data
			}
			if *fileOutputType > 0 {
				fileResultChannel <- data
			}
			if *clickhouseOutputType > 0 {
				clickhouseResultChannel <- data
			}
			if *kafkaOutputType > 0 {
				kafkaResultChannel <- data
			}
			if *elasticOutputType > 0 {
				elasticResultChannel <- data
			}
			if *splunkOutputType > 0 {
				splunkResultChannel <- data
			}
		case <-exiting:
			return
		case <-skipDomainsFileTickerChan:
			if skipDomainMapBool {
				skipDomainMap = loadDomainsToMap(*skipDomainsFile)
			} else {
				skipDomainList = loadDomainsToList(*skipDomainsFile)
			}
		case <-allowDomainsFileTickerChan:
			if allowDomainMapBool {
				allowDomainMap = loadDomainsToMap(*allowDomainsFile)
			} else {
				allowDomainList = loadDomainsToList(*allowDomainsFile)
			}
		}
	}
}
