package main

import (
	"time"

	_ "github.com/mosajjal/dnsmonster/output" //this will automatically set up all the outputs
	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"
)

func removeIndex(s []types.GenericOutput, index int) []types.GenericOutput {
	return append(s[:index], s[index+1:]...)
}

func setupOutputs(resultChannel chan types.DNSResult) {
	log.Info("Creating the dispatch Channel")
	// go through all the registered outputs, and see if they are configured to push data, otherwise, remove them from the dispatch list
	for i := 0; i < len(types.GlobalDispatchList); i++ {
		err := types.GlobalDispatchList[i].Initialize()
		if err != nil {
			// the output does not exist, time to remove the item from our globaldispatcher
			types.GlobalDispatchList = removeIndex(types.GlobalDispatchList, i)
			// since we just removed the last item, we should go back one index to keep it consistent
			i--
		}

	}

	skipDomainsFileTicker := time.NewTicker(util.GeneralFlags.SkipDomainsRefreshInterval)
	skipDomainsFileTickerChan := skipDomainsFileTicker.C
	if util.GeneralFlags.SkipDomainsFile == "" {
		log.Infof("skipping skipDomains refresh since it's not provided")
		skipDomainsFileTicker.Stop()
	} else {
		log.Infof("skipDomains refresh interval is %s", util.GeneralFlags.SkipDomainsRefreshInterval)
	}

	allowDomainsFileTicker := time.NewTicker(util.GeneralFlags.AllowDomainsRefreshInterval)
	allowDomainsFileTickerChan := allowDomainsFileTicker.C
	if util.GeneralFlags.AllowDomainsFile == "" {
		log.Infof("skipping allowDomains refresh since it's not provided")
		allowDomainsFileTicker.Stop()
	} else {
		log.Infof("allowDomains refresh interval is %s", util.GeneralFlags.AllowDomainsRefreshInterval)
	}

	for {
		select {
		case data := <-resultChannel:

			for _, o := range types.GlobalDispatchList {
				// todo: this blocks on type0 outputs. This is still blocking for some reason
				o.OutputChannel() <- data
			}

		case <-skipDomainsFileTickerChan:
			if util.SkipDomainMapBool {
				util.SkipDomainMap = util.LoadDomainsToMap(util.GeneralFlags.SkipDomainsFile)
			} else {
				util.SkipDomainList = util.LoadDomainsToList(util.GeneralFlags.SkipDomainsFile)
			}
		case <-allowDomainsFileTickerChan:
			if util.AllowDomainMapBool {
				util.AllowDomainMap = util.LoadDomainsToMap(util.GeneralFlags.AllowDomainsFile)
			} else {
				util.AllowDomainList = util.LoadDomainsToList(util.GeneralFlags.AllowDomainsFile)
			}
		}
	}
}
