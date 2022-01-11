package output

import (
	"fmt"
	"os"

	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
)

func stdoutOutputWorker(stdConfig types.StdoutConfig) {
	stdoutSentToOutput := metrics.GetOrRegisterCounter("stdoutSentToOutput", metrics.DefaultRegistry)
	stdoutSkipped := metrics.GetOrRegisterCounter("stdoutSkipped", metrics.DefaultRegistry)
	isOutputJson := stdConfig.StdoutOutputFormat == "json"
	for data := range stdConfig.ResultChannel {
		for _, dnsQuery := range data.DNS.Question {

			if util.CheckIfWeSkip(stdConfig.StdoutOutputType, dnsQuery.Name) {
				stdoutSkipped.Inc(1)
				continue
			}
			stdoutSentToOutput.Inc(1)
			if isOutputJson {
				fmt.Printf("%s\n", data.String())
			} else {
				fmt.Printf("%s\n", data.CsvRow())
			}
		}
	}

}

func StdoutOutput(stdConfig types.StdoutConfig) {
	if stdConfig.StdoutOutputFormat == "csv" {
		types.PrintCsvHeader()
	}
	for i := 0; i < 8; i++ { //todo: make this configurable
		go stdoutOutputWorker(stdConfig)
	}
}

func FileOutput(fConfig types.FileConfig) {
	fileSentToOutput := metrics.GetOrRegisterCounter("fileSentToOutput", metrics.DefaultRegistry)
	fileSkipped := metrics.GetOrRegisterCounter("fileSkipped", metrics.DefaultRegistry)
	var fileObject *os.File
	if fConfig.FileOutputType > 0 {
		var err error
		fileObject, err = os.OpenFile(fConfig.FileOutputPath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		util.ErrorHandler(err)
		defer fileObject.Close()
	}

	isOutputJson := fConfig.FileOutputFormat == "json"
	if !isOutputJson {
		types.PrintCsvHeader()
	}

	for data := range fConfig.ResultChannel {
		for _, dnsQuery := range data.DNS.Question {

			if util.CheckIfWeSkip(fConfig.FileOutputType, dnsQuery.Name) {
				fileSkipped.Inc(1)
				continue
			}
			fileSentToOutput.Inc(1)
			if isOutputJson {
				_, err := fileObject.WriteString(fmt.Sprintf("%s\n", data.String()))
				util.ErrorHandler(err)
			} else {
				_, err := fileObject.WriteString(fmt.Sprintf("%s\n", data.CsvRow()))
				util.ErrorHandler(err)
			}
		}
	}
}
