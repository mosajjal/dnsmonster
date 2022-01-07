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
	printStatsTicker := time.NewTicker(stdConfig.General.PrintStatsDelay)
	isOutputJson := stdConfig.StdoutOutputFormat == "json"
	for {
		select {
		case data := <-stdConfig.ResultChannel:
			for _, dnsQuery := range data.DNS.Question {

				if util.CheckIfWeSkip(stdConfig.StdoutOutputType, dnsQuery.Name) {
					stdoutstats.Skipped++
					continue
				}
				stdoutstats.SentToOutput++
				if isOutputJson {
					fmt.Printf("%s\n", data.String())
				} else {
					fmt.Printf("%s\n", data.CsvRow())
				}
			}

		case <-printStatsTicker.C:
			log.Infof("output: %+v", stdoutstats)
		}
	}
}

func StdoutOutput(stdConfig types.StdoutConfig) {
	isOutputJson := stdConfig.StdoutOutputFormat == "json"
	if !isOutputJson {
		types.PrintCsvHeader()
	}
	for i := 0; i < 8; i++ {
		go stdoutOutputWorker(stdConfig)
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
	printStatsTicker := time.NewTicker(fConfig.General.PrintStatsDelay)

	isOutputJson := fConfig.FileOutputFormat == "json"
	if !isOutputJson {
		types.PrintCsvHeader()
	}

	for {
		select {
		case data := <-fConfig.ResultChannel:
			for _, dnsQuery := range data.DNS.Question {

				if util.CheckIfWeSkip(fConfig.FileOutputType, dnsQuery.Name) {
					fileoutstats.Skipped++
					continue
				}
				fileoutstats.SentToOutput++
				if isOutputJson {
					_, err := fileObject.WriteString(fmt.Sprintf("%s\n", data.String()))
					util.ErrorHandler(err)
				} else {
					_, err := fileObject.WriteString(fmt.Sprintf("%s\n", data.CsvRow()))
					util.ErrorHandler(err)
				}
			}

		case <-printStatsTicker.C:
			log.Infof("output: %+v", fileoutstats)
		}
	}
}
