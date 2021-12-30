package output

import (
	"fmt"
	"os"
	"time"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"
)

var stdoutstats = types.OutputStats{Name: "Stdout", SentToOutput: 0, Skipped: 0}
var fileoutstats = types.OutputStats{Name: "File", SentToOutput: 0, Skipped: 0}

func stdoutOutputWorker(stdConfig types.StdoutConfig) {
	defer types.GlobalWaitingGroup.Done()
	printStatsTicker := time.NewTicker(stdConfig.General.PrintStatsDelay)

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
		case <-printStatsTicker.C:
			log.Infof("output: %+v", stdoutstats)
		}
	}
}

func StdoutOutput(stdConfig types.StdoutConfig) {
	defer types.GlobalWaitingGroup.Done()
	for i := 0; i < 8; i++ {
		types.GlobalWaitingGroup.Add(1)
		go stdoutOutputWorker(stdConfig)
	}
}

func FileOutput(fConfig types.FileConfig) {
	defer types.GlobalWaitingGroup.Done()
	var fileObject *os.File
	if fConfig.FileOutputType > 0 {
		var err error
		fileObject, err = os.OpenFile(fConfig.FileOutputPath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		util.ErrorHandler(err)
		defer fileObject.Close()
	}
	printStatsTicker := time.NewTicker(fConfig.General.PrintStatsDelay)

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
		case <-printStatsTicker.C:
			log.Infof("output: %+v", fileoutstats)
		}
	}
}
