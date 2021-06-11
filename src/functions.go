package main

import "strings"

type outputStats struct {
	Name         string
	SentToOutput int
	Skipped      int
}

// captureStats is capturing statistics about our current live captures. At this point it's not accurate for PCAP files.
type captureStats struct {
	PacketsGot        int
	PacketsLost       int
	PacketLossPercent float32
}

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
	case 3:
		if allowDomainMapBool {
			if checkSkipDomainHash(query, allowDomainMap) {
				return false
			}
		} else if checkSkipDomainList(query, allowDomainList) {
			return false
		}
	// 4 means apply two logics, so we apply the two logics and && them together
	case 4:
		if !checkIfWeSkip(2, query) {
			return checkIfWeSkip(3, query)
		}
		return true
	}
	return true
}
