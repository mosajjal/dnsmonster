package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

func stdoutOutput(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup) {
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
			for _, dnsQuery := range data.DNS.Question {

				// check skiplist
				if skipDomainsBool {
					if skipDomainMapBool {
						if checkSkipDomainHash(dnsQuery.Name, skipDomainMap) {
							myStats.skippedDomains++
							continue
						}
					} else if checkSkipDomainList(dnsQuery.Name, skipDomainList) {
						myStats.skippedDomains++
						continue
					}
				}

				// check allowdomains
				if allowDomainsBool {
					if allowDomainMapBool {
						if !checkSkipDomainHash(dnsQuery.Name, allowDomainMap) {
							myStats.skippedDomains++
							continue
						}
					} else if !checkSkipDomainList(dnsQuery.Name, allowDomainList) {
						myStats.skippedDomains++
						continue
					}
				}

				fullQuery, _ := json.Marshal(data)
				fmt.Printf("%s\n", fullQuery)
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

func fileOutput(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	var fileObject *os.File
	if fileOutputBool {
		var err error
		fileObject, err = os.OpenFile(*fileOutputPath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		errorHandler(err)
		defer fileObject.Close()
	}

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
			for _, dnsQuery := range data.DNS.Question {

				// check skiplist
				if skipDomainsBool {
					if skipDomainMapBool {
						if checkSkipDomainHash(dnsQuery.Name, skipDomainMap) {
							myStats.skippedDomains++
							continue
						}
					} else if checkSkipDomainList(dnsQuery.Name, skipDomainList) {
						myStats.skippedDomains++
						continue
					}
				}

				// check allowdomains
				if allowDomainsBool {
					if allowDomainMapBool {
						if !checkSkipDomainHash(dnsQuery.Name, allowDomainMap) {
							myStats.skippedDomains++
							continue
						}
					} else if !checkSkipDomainList(dnsQuery.Name, allowDomainList) {
						myStats.skippedDomains++
						continue
					}
				}

				fullQuery, _ := json.Marshal(data)
				_, err := fileObject.WriteString(fmt.Sprintf("%s\n", fullQuery))
				errorHandler(err)
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
