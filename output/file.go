package output

import (
	"errors"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type FileConfig struct {
	FileOutputType   uint           `long:"fileOutputType"              env:"DNSMONSTER_FILEOUTPUTTYPE"              default:"0"                                                       description:"What should be written to file. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"          choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	FileOutputPath   flags.Filename `long:"fileOutputPath"              env:"DNSMONSTER_FILEOUTPUTPATH"              default:""                                                        description:"Path to output file. Used if fileOutputType is not none"`
	FileOutputFormat string         `long:"fileOutputFormat"            env:"DNSMONSTER_FILEOUTPUTFORMAT"            default:"json"                                                    description:"Output format for file. options:json,csv. note that the csv splits the datetime format into multiple fields"                                                                                                                                                                          choice:"json" choice:"csv"`
	outputChannel    chan util.DNSResult
	closeChannel     chan bool
}

func (config FileConfig) initializeFlags() error {
	// this line will run at import time, before parsing the flags, hence showing up in --help as well as actually working
	_, err := util.GlobalParser.AddGroup("file_output", "File Output", &config)

	config.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)

	util.GlobalDispatchList = append(util.GlobalDispatchList, &config)
	return err
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (config FileConfig) Initialize() error {
	if config.FileOutputType > 0 && config.FileOutputType < 5 {
		log.Info("Creating File Output Channel")
		go config.Output()
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (config FileConfig) Close() {
	//todo: implement this
	<-config.closeChannel
}

func (config FileConfig) OutputChannel() chan util.DNSResult {
	return config.outputChannel
}

func (fConfig FileConfig) Output() {
	fileSentToOutput := metrics.GetOrRegisterCounter("fileSentToOutput", metrics.DefaultRegistry)
	fileSkipped := metrics.GetOrRegisterCounter("fileSkipped", metrics.DefaultRegistry)
	var fileObject *os.File
	if fConfig.FileOutputType > 0 {
		var err error
		fileObject, err = os.OpenFile(string(fConfig.FileOutputPath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer fileObject.Close()
	}

	isOutputJson := fConfig.FileOutputFormat == "json"
	if !isOutputJson {
		_, err := fileObject.WriteString(util.GetCsvHeaderRow() + "\n")
		if err != nil {
			log.Fatal(err)
		}
	}

	for data := range fConfig.outputChannel {
		for _, dnsQuery := range data.DNS.Question {

			if util.CheckIfWeSkip(fConfig.FileOutputType, dnsQuery.Name) {
				fileSkipped.Inc(1)
				continue
			}
			fileSentToOutput.Inc(1)
			if isOutputJson {
				_, err := fileObject.WriteString(data.GetJson() + "\n")
				if err != nil {
					log.Fatal(err)
				}
			} else {
				_, err := fileObject.WriteString(data.GetCsvRow() + "\n")
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
}

// This will allow an instance to be spawned at import time
var _ = FileConfig{}.initializeFlags()
