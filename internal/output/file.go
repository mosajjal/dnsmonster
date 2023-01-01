package output

import (
	"context"
	"errors"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/mosajjal/dnsmonster/internal/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type fileConfig struct {
	FileOutputType       uint           `long:"fileoutputtype"              ini-name:"fileoutputtype"              env:"DNSMONSTER_FILEOUTPUTTYPE"              default:"0"                                                       description:"What should be written to file. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"          choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	FileOutputPath       flags.Filename `long:"fileoutputpath"              ini-name:"fileoutputpath"              env:"DNSMONSTER_FILEOUTPUTPATH"              default:""                                                        description:"Path to output file. Used if fileOutputType is not none"`
	FileOutputFormat     string         `long:"fileoutputformat"            ini-name:"fileoutputformat"            env:"DNSMONSTER_FILEOUTPUTFORMAT"            default:"json"                                                    description:"Output format for file. options:json,csv, csv_no_header, gotemplate. note that the csv splits the datetime format into multiple fields"                                                                                                                                               choice:"json" choice:"csv" choice:"csv_no_header" choice:"gotemplate"`
	FileOutputGoTemplate string         `long:"fileoutputgotemplate"        ini-name:"fileoutputgotemplate"        env:"DNSMONSTER_FILEOUTPUTGOTEMPLATE"        default:"{{.}}"                                                   description:"Go Template to format the output as needed"`
	outputChannel        chan util.DNSResult
	closeChannel         chan bool
	outputMarshaller     util.OutputMarshaller
	fileObject           *os.File
}

func init() {
	c := fileConfig{}
	if _, err := util.GlobalParser.AddGroup("file_output", "File Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)

}

// initialize function should not block. otherwise the dispatcher will get stuck
func (config fileConfig) Initialize(ctx context.Context) error {
	var err error
	var header string
	config.outputMarshaller, header, err = util.OutputFormatToMarshaller(config.FileOutputFormat, config.FileOutputGoTemplate)
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

	if config.FileOutputType > 0 && config.FileOutputType < 5 {
		log.Info("Creating File Output Channel")

		config.fileObject, err = os.OpenFile(string(config.FileOutputPath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatal(err)
		}

		go config.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}

	// print the header
	_, err = config.fileObject.WriteString(header + "\n")

	return err
}

func (config fileConfig) Close() {
	// todo: implement this
	<-config.closeChannel
}

func (config fileConfig) OutputChannel() chan util.DNSResult {
	return config.outputChannel
}

func (config fileConfig) Output(ctx context.Context) {
	fileSentToOutput := metrics.GetOrRegisterCounter("fileSentToOutput", metrics.DefaultRegistry)
	fileSkipped := metrics.GetOrRegisterCounter("fileSkipped", metrics.DefaultRegistry)

	// todo: output channel will duplicate output when we have malformed DNS packets with multiple questions
	for {
		select {
		case data := <-config.outputChannel:
			for _, dnsQuery := range data.DNS.Question {

				if util.CheckIfWeSkip(config.FileOutputType, dnsQuery.Name) {
					fileSkipped.Inc(1)
					continue
				}
				fileSentToOutput.Inc(1)
				_, err := config.fileObject.WriteString(config.outputMarshaller.Marshal(data) + "\n")
				if err != nil {
					log.Fatal(err)
				}

			}

		case <-ctx.Done():
			config.fileObject.Close()
			log.Debug("exitting out of file output") //todo:remove
		}
	}
}

// This will allow an instance to be spawned at import time
// var _ = fileConfig{}.initializeFlags()
