package main

import (
	"encoding/json"
	"fmt"
	"log"
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
					if checkSkipDomain(dnsQuery.Name, skipDomainList) {
						myStats.skippedDomains++
						continue
					}
				}

				// check allowdomains
				if allowDomainsBool {
					if !checkSkipDomain(dnsQuery.Name, allowDomainList) {
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
			skipDomainList = loadDomains(*skipDomainsFile)
		case <-allowDomainsFileTickerChan:
			allowDomainList = loadDomains(*allowDomainsFile)
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
		if err != nil {
			log.Println(err)
		}
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
					if checkSkipDomain(dnsQuery.Name, skipDomainList) {
						myStats.skippedDomains++
						continue
					}
				}

				// check allowdomains
				if allowDomainsBool {
					if !checkSkipDomain(dnsQuery.Name, allowDomainList) {
						myStats.skippedDomains++
						continue
					}
				}

				fullQuery, _ := json.Marshal(data)
				_, err := fileObject.WriteString(fmt.Sprintf("%s\n", fullQuery))
				if err != nil {
					log.Println("error in writing to the log file: ", err)
				}
			}
		case <-exiting:
			return
		case <-skipDomainsFileTickerChan:
			skipDomainList = loadDomains(*skipDomainsFile)
		case <-allowDomainsFileTickerChan:
			allowDomainList = loadDomains(*allowDomainsFile)
		}
	}
}
