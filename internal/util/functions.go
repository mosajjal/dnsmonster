/* {{{ Copyright (C) 2022 Ali Mosajjal
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. }}} */

package util

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/golang-collections/collections/tst"
	log "github.com/sirupsen/logrus"
)

const (
	outputNone  = 0
	outputAll   = 1
	outputSkip  = 2
	outputAllow = 3
	outputBoth  = 4

	matchPrefix = 1
	matchSuffix = 2
	matchFQDN   = 3
)

// CheckIfWeSkip checks a fqdn against an output type and make a decision if
// the fqdn is meant to be sent to output or not.
func CheckIfWeSkip(outputType uint, fqdn string) bool {
	fqdnLower := strings.ToLower(fqdn) //todo:check performance for this function
	switch outputType {
	case outputNone:
		return true // always skip
	case outputAll:
		return false // never skip
	case outputSkip:
		// check for fqdn match
		if GeneralFlags.skipTypeHt[fqdnLower] == matchFQDN {
			return true
		}
		// check for prefix match
		if longestPrefix := GeneralFlags.skipPrefixTst.GetLongestPrefix(fqdnLower); longestPrefix != nil {
			// check if the longest prefix is present in the type hashtable as a prefix
			if GeneralFlags.skipTypeHt[longestPrefix.(string)] == matchPrefix {
				return true
			}
		}
		// check for suffix match. Note that suffix is just prefix reversed
		if longestSuffix := GeneralFlags.skipSuffixTst.GetLongestPrefix(reverse(fqdnLower)); longestSuffix != nil {
			// check if the longest suffix is present in the type hashtable as a suffix
			if GeneralFlags.skipTypeHt[longestSuffix.(string)] == matchSuffix {
				return true
			}
		}

		return false
	case outputAllow:
		// check for fqdn match
		if GeneralFlags.allowTypeHt[fqdnLower] == matchFQDN {
			return false
		}
		// check for prefix match
		if longestPrefix := GeneralFlags.allowPrefixTst.GetLongestPrefix(fqdnLower); longestPrefix != nil {
			// check if the longest prefix is present in the type hashtable as a prefix
			if GeneralFlags.allowTypeHt[longestPrefix.(string)] == matchPrefix {
				return false
			}
		}
		// check for suffix match. Note that suffix is just prefix reversed
		if longestSuffix := GeneralFlags.allowSuffixTst.GetLongestPrefix(reverse(fqdnLower)); longestSuffix != nil {
			// check if the longest suffix is present in the type hashtable as a suffix
			if GeneralFlags.allowTypeHt[longestSuffix.(string)] == matchSuffix {
				return false
			}
		}
		return true
	// 4 means apply two logics, so we apply the two logics and && them together
	case outputBoth:
		if !CheckIfWeSkip(outputSkip, fqdn) {
			return CheckIfWeSkip(outputAllow, fqdn)
		}
		return true
	}
	return true
}

func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// LoadDomainsCsv loads a domains Csv file/URL. returns 3 parameters:
// 1. a TST for all the prefixes (type 1)
// 2. a TST for all the suffixes (type 2)
// 3. a hashtable for all the full match fqdn (type 3)
func LoadDomainsCsv(Filename string) (*tst.TernarySearchTree, *tst.TernarySearchTree, map[string]uint8) {
	log.Info("Loading the domain from file/url")
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
		if err != nil {
			log.Fatal(err)
		}
		log.Info("(re)loading File: ", Filename)
		defer file.Close()
		scanner = bufio.NewScanner(file)
	}

	prefixTst := tst.New()
	suffixTst := tst.New()
	entryTypeHt := make(map[string]uint8)

	for scanner.Scan() {
		lowerCaseLine := strings.ToLower(scanner.Text())
		// split the line by comma to understand the logic
		fqdn := strings.Split(lowerCaseLine, ",")
		if len(fqdn) != 2 {
			log.Warnf("%s is not a valid line, assuming fqdn", lowerCaseLine)
			fqdn = []string{lowerCaseLine, "fqdn"}
		}
		// add the fqdn to the hashtable with its type
		switch entryType := fqdn[1]; entryType {
		case "prefix":
			entryTypeHt[fqdn[0]] = matchPrefix
			prefixTst.Insert(fqdn[0], fqdn[0])
		case "suffix":
			entryTypeHt[fqdn[0]] = matchSuffix
			// suffix match is much faster if we reverse the strings and match for prefix
			suffixTst.Insert(reverse(fqdn[0]), fqdn[0])
		case "fqdn":
			entryTypeHt[fqdn[0]] = matchFQDN
		default:
			log.Warnf("%s is not a valid line, assuming fqdn", lowerCaseLine)
			entryTypeHt[fqdn[0]] = matchFQDN
		}
	}
	log.Infof("%s loaded with %d prefix, %d suffix and %d fqdn", Filename, prefixTst.Len(), suffixTst.Len(), len(entryTypeHt)-prefixTst.Len()-suffixTst.Len())
	return prefixTst, suffixTst, entryTypeHt
}

// OutputFormatToMarshaller gets the outputFormat string and a template used in gotemplate
func OutputFormatToMarshaller(outputFormat string, t string) (OutputMarshaller, string, error) {
	switch outputFormat {
	case "json":
		return jsonOutput{}, "", nil
	case "csv":
		csvOut := csvOutput{}
		header, _ := csvOut.Init()
		return csvOut, header, nil
	case "csv_no_header":
		return csvOutput{}, "", nil
	case "gotemplate":
		goOut := goTemplateOutput{RawTemplate: t}
		_, err := goOut.Init()
		return &goOut, "", err
	case "gob":
		gobOut := gobOutput{}
		_, err := gobOut.Init()
		return &gobOut, "", err
	}
	return nil, "", fmt.Errorf("%s is not a valid output format", outputFormat)
}
// vim: foldmethod=marker
