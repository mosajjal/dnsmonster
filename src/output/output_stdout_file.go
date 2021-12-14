package output

import (
	"fmt"
	"os"
	"time"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"
)

var stdoutstats = types.OutputStats{"Stdout", 0, 0}
var fileoutstats = types.OutputStats{"File", 0, 0}

func stdoutOutputWorker(stdConfig types.StdoutConfig) {
	printStatsTicker := time.Tick(stdConfig.General.PrintStatsDelay)

	for {
		select {
		case data := <-stdConfig.ResultChannel:
			for _, dnsQuery := range data.DNS.Question {

				if util.CheckIfWeSkip(stdConfig.StdoutOutputType, dnsQuery.Name) {
					stdoutstats.Skipped++
					continue
				}
				stdoutstats.SentToOutput++

				fmt.Printf("%s\n", data.String())
			}
		case <-types.GlobalExitChannel:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", stdoutstats)
		}
	}
}

func StdoutOutput(stdConfig types.StdoutConfig) {
	for i := 0; i < 8; i++ {
		go stdoutOutputWorker(stdConfig)
		types.GlobalWaitingGroup.Add(1)
		defer types.GlobalWaitingGroup.Done()
	}
}

func FileOutput(fConfig types.FileConfig) {
	var fileObject *os.File
	if fConfig.FileOutputType > 0 {
		var err error
		fileObject, err = os.OpenFile(fConfig.FileOutputPath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		util.ErrorHandler(err)
		defer fileObject.Close()
	}
	printStatsTicker := time.Tick(fConfig.General.PrintStatsDelay)

	for {
		select {
		case data := <-fConfig.ResultChannel:
			for _, dnsQuery := range data.DNS.Question {

				if util.CheckIfWeSkip(fConfig.FileOutputType, dnsQuery.Name) {
					fileoutstats.Skipped++
					continue
				}
				fileoutstats.SentToOutput++

				_, err := fileObject.WriteString(fmt.Sprintf("%s\n", data.String()))
				util.ErrorHandler(err)
			}
		case <-types.GlobalExitChannel:
			return
		case <-printStatsTicker:
			log.Infof("output: %+v", fileoutstats)
		}
	}
}
