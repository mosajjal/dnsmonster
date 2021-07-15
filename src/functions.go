package main

import (
	"bufio"
	"net/http"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

// checkSkipDomainList returns true if the domain exists in the domainList
func checkSkipDomainList(domainName string, domainList [][]string) bool {
	for _, item := range domainList {
		if len(item) == 2 {
			if item[1] == "suffix" {
				if strings.HasSuffix(domainName, item[0]) {
					return true
				}
			} else if item[1] == "fqdn" {
				if domainName == item[0] {
					return true
				}
			} else if item[1] == "prefix" {
				if strings.HasPrefix(domainName, item[0]) {
					return true
				}
			}
		}
	}
	return false
}

// checkSkipDomainHash returns true if the domain exists in the inputHashTable
func checkSkipDomainHash(domainName string, inputHashTable map[string]bool) bool {
	return inputHashTable[domainName]
}

//0: none, 1: all, 2: apply skipdomains logic, 3: apply allowdomains logic, 4: apply both skip and allow domains logic.
func checkIfWeSkip(outputType uint, query string) bool {
	switch outputType {
	case 0:
		return true //always skip
	case 1:
		return false // never skip
	case 2:
		if skipDomainMapBool {
			if checkSkipDomainHash(query, skipDomainMap) {
				return true
			}
		} else if checkSkipDomainList(query, skipDomainList) {
			return true
		}
		return false
	case 3:
		if allowDomainMapBool {
			if checkSkipDomainHash(query, allowDomainMap) {
				return false
			}
		} else if checkSkipDomainList(query, allowDomainList) {
			return false
		}
		return true
	// 4 means apply two logics, so we apply the two logics and && them together
	case 4:
		if !checkIfWeSkip(2, query) {
			return checkIfWeSkip(3, query)
		}
		return true
	}
	return true
}

func loadDomainsToList(Filename string) [][]string {
	log.Info("Loading the domain from file/url to a list")
	var lines [][]string
	var scanner *bufio.Scanner
	if strings.HasPrefix(Filename, "http://") || strings.HasPrefix(Filename, "https://") {
		log.Info("domain list is a URL, trying to fetch")
		client := http.Client{
			CheckRedirect: func(r *http.Request, via []*http.Request) error {
				r.URL.Opaque = r.URL.Path
				return nil
			},
		}
		resp, err := client.Get(Filename)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("(re)fetching URL: ", Filename)
		defer resp.Body.Close()
		scanner = bufio.NewScanner(resp.Body)

	} else {
		file, err := os.Open(Filename)
		errorHandler(err)
		log.Info("(re)loading File: ", Filename)
		defer file.Close()
		scanner = bufio.NewScanner(file)
	}

	for scanner.Scan() {
		lowerCaseLine := strings.ToLower(scanner.Text())
		lines = append(lines, strings.Split(lowerCaseLine, ","))
	}
	log.Infof("%s loaded with %d lines", Filename, len(lines))
	return lines
}

func errorHandler(err error) {
	if err != nil {
		log.Fatal("fatal Error: ", err)
	}
}

func loadDomainsToMap(Filename string) map[string]bool {
	log.Info("Loading the domain from file/url to a hashmap")
	lines := make(map[string]bool)
	var scanner *bufio.Scanner
	if strings.HasPrefix(Filename, "http://") || strings.HasPrefix(Filename, "https://") {
		log.Info("domain list is a URL, trying to fetch")
		client := http.Client{
			CheckRedirect: func(r *http.Request, via []*http.Request) error {
				r.URL.Opaque = r.URL.Path
				return nil
			},
		}
		resp, err := client.Get(Filename)
		if err != nil {
			log.Fatal(err)
		}
		log.Info("(re)fetching URL: ", Filename)
		defer resp.Body.Close()
		scanner = bufio.NewScanner(resp.Body)

	} else {
		file, err := os.Open(Filename)
		errorHandler(err)
		log.Info("(re)loading File: ", Filename)
		defer file.Close()
		scanner = bufio.NewScanner(file)
	}

	for scanner.Scan() {
		lowerCaseLine := strings.ToLower(scanner.Text())
		fqdn := strings.Split(lowerCaseLine, ",")[0]
		lines[fqdn] = true
	}
	log.Infof("%s loaded with %d lines", Filename, len(lines))
	return lines
}
