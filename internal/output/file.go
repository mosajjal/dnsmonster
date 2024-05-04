/* {{{ Copyright (C) 2022 Ali Mosajjal
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>. }}} */

package output

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/arthurkiller/rollingwriter"
	"github.com/jessevdk/go-flags"
	"github.com/mosajjal/dnsmonster/internal/util"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type fileConfig struct {
	FileOutputType        uint           `long:"fileoutputtype"              ini-name:"fileoutputtype"              env:"DNSMONSTER_FILEOUTPUTTYPE"              default:"0"                                                       description:"What should be written to file. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"          choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	FileOutputPath        flags.Filename `long:"fileoutputpath"              ini-name:"fileoutputpath"              env:"DNSMONSTER_FILEOUTPUTPATH"              default:""                                                        description:"Path to output folder. Used if fileoutputType is not none"`
	FileOutputRotateCron  string         `long:"fileoutputrotatecron"        ini-name:"fileoutputrotatecron"        env:"DNSMONSTER_FILEOUTPUTROTATECRON"        default:"0 0 * * *"                                               description:"Interval to rotate the file in cron format"`
	FileOutputRotateCount uint           `long:"fileoutputrotatecount"       ini-name:"fileoutputrotatecount"       env:"DNSMONSTER_FILEOUTPUTROTATECOUNT"       default:"4"                                                       description:"Number of files to keep. 0 to disable rotation"`
	FileOutputFormat      string         `long:"fileoutputformat"            ini-name:"fileoutputformat"            env:"DNSMONSTER_FILEOUTPUTFORMAT"            default:"json"                                                    description:"Output format for file. options:json, csv, csv_no_header, gotemplate. note that the csv splits the datetime format into multiple fields"                                                                                                                                               choice:"json" choice:"csv" choice:"csv_no_header" choice:"gotemplate"`
	FileOutputGoTemplate  string         `long:"fileoutputgotemplate"        ini-name:"fileoutputgotemplate"        env:"DNSMONSTER_FILEOUTPUTGOTEMPLATE"        default:"{{.}}"                                                   description:"Go Template to format the output as needed"`
	outputChannel         chan util.DNSResult
	closeChannel          chan bool
	outputMarshaller      util.OutputMarshaller
	writer                rollingwriter.RollingWriter
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

		rollerConfig := &rollingwriter.Config{
			LogPath:                string(config.FileOutputPath),
			TimeTagFormat:          time.RFC3339,
			FileName:               "dnsmonster",
			MaxRemain:              int(config.FileOutputRotateCount),
			RollingPolicy:          rollingwriter.TimeRolling,
			RollingTimePattern:     fmt.Sprintf("0 %s", config.FileOutputRotateCron), // remove the second option from the cron to make it compatible with unix style
			RollingVolumeSize:      "0",
			WriterMode:             "lock",
			BufferWriterThershould: 64,
			Compress:               true,
		}

		config.writer, err = rollingwriter.NewWriterFromConfig(rollerConfig)
		// config.writer, err = os.OpenFile(string(config.FileOutputPath), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Fatal(err)
		}

		go config.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}

	// print the header
	if header != "" {
		_, err = config.writer.Write([]byte(fmt.Sprintf("%s\n", header)))
	}
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
				_, err := config.writer.Write(config.outputMarshaller.Marshal(data))
				if err != nil {
					log.Fatal(err)
				}
				_, _ = config.writer.Write([]byte("\n"))

			}

		case <-ctx.Done():
			config.writer.Close()
			log.Debug("exiting out of file output") //todo:remove
			return
		}
	}
}

// vim: foldmethod=marker
