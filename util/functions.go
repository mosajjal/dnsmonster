package util

import (
	"bufio"
	"net/http"
	"os"
	"strings"

	"github.com/golang-collections/collections/tst"
	log "github.com/sirupsen/logrus"
)

const (
	OUTPUT_NONE  = 0
	OUTPUT_ALL   = 1
	OUTPUT_SKIP  = 2
	OUTPUT_ALLOW = 3
	OUTPUT_BOTH  = 4

	MATCH_PREFIX = 1
	MATCH_SUFFIX = 2
	MATCH_FQDN   = 3
)

//0: none, 1: all, 2: apply skipdomains logic, 3: apply allowdomains logic, 4: apply both skip and allow domains logic.
func CheckIfWeSkip(outputType uint, fqdn string) bool {
	fqdnLower := strings.ToLower(fqdn) //todo:check performance for this function
	switch outputType {
	case OUTPUT_NONE:
		return true //always skip
	case OUTPUT_ALL:
		return false // never skip
	case OUTPUT_SKIP:
		// check for fqdn match
		if GeneralFlags.skipTypeHt[fqdnLower] == MATCH_FQDN {
			return true
		}
		// check for prefix match
		if longestPrefix := GeneralFlags.skipPrefixTst.GetLongestPrefix(fqdnLower); longestPrefix != nil {
			// check if the longest prefix is present in the type hashtable as a prefix
			if GeneralFlags.skipTypeHt[longestPrefix.(string)] == MATCH_PREFIX {
				return true
			}
		}
		// check for suffix match. Note that suffix is just prefix reversed
		if longestSuffix := GeneralFlags.skipSuffixTst.GetLongestPrefix(Reverse(fqdnLower)); longestSuffix != nil {
			// check if the longest suffix is present in the type hashtable as a suffix
			if GeneralFlags.skipTypeHt[longestSuffix.(string)] == MATCH_SUFFIX {
				return true
			}
		}

		return false
	case OUTPUT_ALLOW:
		// check for fqdn match
		if GeneralFlags.allowTypeHt[fqdnLower] == MATCH_FQDN {
			return false
		}
		// check for prefix match
		if longestPrefix := GeneralFlags.allowPrefixTst.GetLongestPrefix(fqdnLower); longestPrefix != nil {
			// check if the longest prefix is present in the type hashtable as a prefix
			if GeneralFlags.allowTypeHt[longestPrefix.(string)] == MATCH_PREFIX {
				return false
			}
		}
		// check for suffix match. Note that suffix is just prefix reversed
		if longestSuffix := GeneralFlags.allowSuffixTst.GetLongestPrefix(Reverse(fqdnLower)); longestSuffix != nil {
			// check if the longest suffix is present in the type hashtable as a suffix
			if GeneralFlags.allowTypeHt[longestSuffix.(string)] == MATCH_SUFFIX {
				return false
			}
		}
		return true
	// 4 means apply two logics, so we apply the two logics and && them together
	case OUTPUT_BOTH:
		if !CheckIfWeSkip(OUTPUT_SKIP, fqdn) {
			return CheckIfWeSkip(OUTPUT_ALLOW, fqdn)
		}
		return true
	}
	return true
}
func Reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

// Loads a domains Csv file/URL. returns 3 parameters:
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
			entryTypeHt[fqdn[0]] = MATCH_PREFIX
			prefixTst.Insert(fqdn[0], fqdn[0])
		case "suffix":
			entryTypeHt[fqdn[0]] = MATCH_SUFFIX
			// suffix match is much faster if we rever(se the strings and match for prefix
			suffixTst.Insert(Reverse(fqdn[0]), fqdn[0])
		case "fqdn":
			entryTypeHt[fqdn[0]] = MATCH_FQDN
		default:
			log.Warnf("%s is not a valid line, assuming fqdn", lowerCaseLine)
			entryTypeHt[fqdn[0]] = MATCH_FQDN
		}
	}
	log.Infof("%s loaded with %d prefix, %d suffix and %d fqdn", Filename, prefixTst.Len(), suffixTst.Len(), len(entryTypeHt)-prefixTst.Len()-suffixTst.Len())
	return prefixTst, suffixTst, entryTypeHt
}
