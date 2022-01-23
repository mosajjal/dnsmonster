package main

import (
	"time"

	_ "github.com/mosajjal/dnsmonster/output" //this will automatically set up all the outputs
	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"
)

func setupOutputs() {

	log.Info("Creating the dispatch Channel")

	go dispatchOutput(resultChannel)

}

func RemoveIndex(s []types.GenericOutput, index int) []types.GenericOutput {
	return append(s[:index], s[index+1:]...)
}

func dispatchOutput(resultChannel chan types.DNSResult) {

	// the new simplified output method
	for i := 0; i < len(types.GlobalDispatchList); i++ {
		err := types.GlobalDispatchList[i].Initialize()
		if err != nil {
			// the output does not exist, time to remove the item from our globaldispatcher
			types.GlobalDispatchList = RemoveIndex(types.GlobalDispatchList, i)
			// since we just removed the last item, we should go back one index to keep it consistent
			i--
		}

	}

	// Set up various tickers for different tasks
	skipDomainsFileTicker := time.NewTicker(util.GeneralFlags.SkipDomainsRefreshInterval)
	skipDomainsFileTickerChan := skipDomainsFileTicker.C
	if util.GeneralFlags.SkipDomainsFile == "" {
		skipDomainsFileTicker.Stop()
	}

	allowDomainsFileTicker := time.NewTicker(util.GeneralFlags.AllowDomainsRefreshInterval)
	allowDomainsFileTickerChan := allowDomainsFileTicker.C
	if util.GeneralFlags.AllowDomainsFile == "" {
		log.Infof("skipping allowDomains refresh since it's empty")
		allowDomainsFileTicker.Stop()
	} else {
		log.Infof("allowDomains refresh interval is %s", util.GeneralFlags.AllowDomainsRefreshInterval)
	}

	for {
		select {
		case data := <-resultChannel:

			// new simplified output method. only works with Sentinel
			for _, o := range types.GlobalDispatchList {
				// todo: this blocks on type0 outputs. This is still blocking for some reason
				o.OutputChannel() <- data
			}

		case <-skipDomainsFileTickerChan:
			log.Infof("reached skipDomains tick")
			if util.SkipDomainMapBool {
				util.SkipDomainMap = util.LoadDomainsToMap(util.GeneralFlags.SkipDomainsFile)
			} else {
				util.SkipDomainList = util.LoadDomainsToList(util.GeneralFlags.SkipDomainsFile)
			}
		case <-allowDomainsFileTickerChan:
			log.Infof("reached allowDomains tick")
			if util.AllowDomainMapBool {
				util.AllowDomainMap = util.LoadDomainsToMap(util.GeneralFlags.AllowDomainsFile)
			} else {
				util.AllowDomainList = util.LoadDomainsToList(util.GeneralFlags.AllowDomainsFile)
			}
		}
	}
}
