package output

import (
	"context"
	"errors"
	"fmt"

	"github.com/mosajjal/dnsmonster/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type stdoutConfig struct {
	StdoutOutputType        uint   `long:"stdoutoutputtype"            ini-name:"stdoutoutputtype"            env:"DNSMONSTER_STDOUTOUTPUTTYPE"            default:"0"                                                       description:"What should be written to stdout. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"        choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	StdoutOutputFormat      string `long:"stdoutoutputformat"          ini-name:"stdoutoutputformat"          env:"DNSMONSTER_STDOUTOUTPUTFORMAT"          default:"json"                                                    description:"Output format for stdout. options:json,csv, csv_no_header, gotemplate. note that the csv splits the datetime format into multiple fields"                                                                                                                                             choice:"json" choice:"csv" choice:"csv_no_header" choice:"gotemplate"`
	StdoutOutputGoTemplate  string `long:"stdoutoutputgotemplate"      ini-name:"stdoutoutputgotemplate"      env:"DNSMONSTER_STDOUTOUTPUTGOTEMPLATE"      default:"{{.}}"                                                   description:"Go Template to format the output as needed"`
	StdoutOutputWorkerCount uint   `long:"stdoutoutputworkercount"     ini-name:"stdoutoutputworkercount"     env:"DNSMONSTER_STDOUTOUTPUTWORKERCOUNT"     default:"8"                                                       description:"Number of workers"`
	outputChannel           chan util.DNSResult
	closeChannel            chan bool
	outputMarshaller        util.OutputMarshaller
}

func init() {
	c := stdoutConfig{}
	if _, err := util.GlobalParser.AddGroup("stdout_output", "Stdout Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)
}

// initialize function should not block. otherwise the dispatcher will get stuck
func (stdConfig stdoutConfig) Initialize(ctx context.Context) error {
	var err error
	var header string
	stdConfig.outputMarshaller, header, err = util.OutputFormatToMarshaller(stdConfig.StdoutOutputFormat, stdConfig.StdoutOutputGoTemplate)
	if err != nil {
		log.Warnf("Could not initialize output marshaller, removing output: %s", err)
		return err
	}

	// print header to stdout
	fmt.Println(header)

	if stdConfig.StdoutOutputType > 0 && stdConfig.StdoutOutputType < 5 {
		log.Info("Creating Stdout Output Channel")
		go stdConfig.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return err
}

func (stdConfig stdoutConfig) Close() {
	// todo: implement this
	<-stdConfig.closeChannel
}

func (stdConfig stdoutConfig) OutputChannel() chan util.DNSResult {
	return stdConfig.outputChannel
}

func (stdConfig stdoutConfig) stdoutOutputWorker() {
	stdoutSentToOutput := metrics.GetOrRegisterCounter("stdoutSentToOutput", metrics.DefaultRegistry)
	stdoutSkipped := metrics.GetOrRegisterCounter("stdoutSkipped", metrics.DefaultRegistry)
	for data := range stdConfig.outputChannel {
		for _, dnsQuery := range data.DNS.Question {

			if util.CheckIfWeSkip(stdConfig.StdoutOutputType, dnsQuery.Name) {
				stdoutSkipped.Inc(1)
				continue
			}
			stdoutSentToOutput.Inc(1)
			fmt.Print(stdConfig.outputMarshaller.Marshal(data) + "\n")
		}
	}
}

func (stdConfig stdoutConfig) Output(ctx context.Context) {
	for i := 0; i < int(stdConfig.StdoutOutputWorkerCount); i++ { // todo: make this configurable
		go stdConfig.stdoutOutputWorker()
	}
}

// This will allow an instance to be spawned at import time
// var _ = stdoutConfig{}.initializeFlags()
