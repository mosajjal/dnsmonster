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
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/mosajjal/dnsmonster/internal/util"
	"github.com/parquet-go/parquet-go"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type parquetConfig struct {
	ParquetOutputType uint           `long:"parquetoutputtype"              ini-name:"parquetoutputtype"              env:"DNSMONSTER_PARQUETOUTPUTTYPE"              default:"0"                                                       description:"What should be written to parquet file. options:\n;\t0: Disable Output\n;\t1: Enable Output without any filters\n;\t2: Enable Output and apply skipdomains logic\n;\t3: Enable Output and apply allowdomains logic\n;\t4: Enable Output and apply both skip and allow domains logic"          choice:"0" choice:"1" choice:"2" choice:"3" choice:"4"`
	ParquetOutputPath flags.Filename `long:"parquetoutputpath"              ini-name:"parquetoutputpath"              env:"DNSMONSTER_PARQUETOUTPUTPATH"              default:""                                                        description:"Path to output folder. Used if parquetoutputtype is not none"`
	// ParquetOutputRotateCron  string         `long:"parquetoutputrotatecron"        ini-name:"parquetoutputrotatecron"        env:"DNSMONSTER_PARQUETOUTPUTROTATECRON"        default:"0 0 * * *"                                               description:"Interval to rotate the parquet file in cron format"`
	// ParquetOutputRotateCount uint           `long:"parquetoutputrotatecount"       ini-name:"parquetoutputrotatecount"       env:"DNSMONSTER_PARQUETOUTPUTROTATECOUNT"       default:"4"                                                       description:"Number of parquet files to keep. 0 to disable rotation"`
	ParquetFlushBatchSize  uint `long:"parquetflushbatchsize"          ini-name:"parquetflushbatchsize"          env:"DNSMONSTER_PARQUETFLUSHBATCHSIZE"          default:"10000"                                                   description:"Number of records to write to parquet file before flushing"`
	ParquetWorkerCount     uint `long:"parquetworkercount"             ini-name:"parquetworkercount"             env:"DNSMONSTER_PARQUETWORKERCOUNT"             default:"4"                                                       description:"Number of workers to write to parquet file"`
	ParquetWriteBufferSize uint `long:"parquetwritebuffersize"             ini-name:"parquetwritebuffersize"             env:"DNSMONSTER_PARQUETWRITEBUFFERSIZE"             default:"256000"                                                       description:"Size of the write buffer in bytes"`
	outputChannel          chan util.DNSResult
	closeChannel           chan bool
	writer                 io.WriteCloser
	parquetWriter          *parquet.GenericWriter[parquetRow]
	parquetWriterLock      *sync.RWMutex
	parquetSentToOutput    metrics.Counter
	parquetSkipped         metrics.Counter
}

type parquetRow struct {
	Timestamp    time.Time `parquet:"timestamp,snappy"`
	IPVersion    uint32    `parquet:"ip_version,snappy,dict"`
	SrcIP        net.IP    `parquet:"src_ip,snappy"`
	DstIP        net.IP    `parquet:"dst_ip,snappy"`
	Protocol     string    `parquet:"protocol,snappy,dict"`
	QR           uint32    `parquet:"qr,snappy,dict"`
	Opcode       uint32    `parquet:"opcode,snappy,dict"`
	Qclass       uint32    `parquet:"qclass,snappy,dict"`
	Qtype        uint32    `parquet:"qtype,snappy,dict"`
	EDNS         uint32    `parquet:"edns,snappy,dict"`
	DoBit        uint32    `parquet:"do_bit,snappy,dict"`
	Rcode        uint32    `parquet:"rcode,snappy"`
	QueryName    string    `parquet:"query_name,brotli,dict"`
	PacketLength uint32    `parquet:"packet_length,snappy"`
	Identity     string    `parquet:"identity,snappy,optional"`
	Version      string    `parquet:"version,snappy,optional"`
}

func init() {
	c := parquetConfig{}
	if _, err := util.GlobalParser.AddGroup("parquet_output", "Parquet Output", &c); err != nil {
		log.Fatalf("error adding output Module")
	}
	c.outputChannel = make(chan util.DNSResult, util.GeneralFlags.ResultChannelSize)
	util.GlobalDispatchList = append(util.GlobalDispatchList, &c)

}

// initialize function should not block. otherwise the dispatcher will get stuck
func (config *parquetConfig) Initialize(ctx context.Context) error {
	if config.ParquetOutputType > 0 && config.ParquetOutputType < 5 {
		log.Info("Creating Parquet Output Channel")

		config.parquetSentToOutput = metrics.GetOrRegisterCounter("parquetSentToOutput", metrics.DefaultRegistry)
		config.parquetSkipped = metrics.GetOrRegisterCounter("parquetSkipped", metrics.DefaultRegistry)
		config.parquetWriterLock = &sync.RWMutex{}
		// TODO: pending github.com/arthurkiller/rollingwriter/issues/52

		// rollerConfig := &rollingwriter.Config{
		// 	LogPath:                string(config.ParquetOutputPath),
		// 	TimeTagFormat:          time.RFC3339,
		// 	FileName:               "dnsmonster",
		// 	MaxRemain:              int(config.ParquetOutputRotateCount),
		// 	RollingPolicy:          rollingwriter.TimeRolling,
		// 	RollingTimePattern:     fmt.Sprintf("0 %s", config.ParquetOutputRotateCron), // remove the second option from the cron to make it compatible with unix style
		// 	RollingVolumeSize:      "0",
		// 	WriterMode:             "lock",
		// 	BufferWriterThershould: 64,
		// 	Compress:               true,
		// }
		// var err error
		// config.writer, err = rollingwriter.NewWriterFromConfig(rollerConfig)
		// if err != nil {
		// 	log.Fatal(err)
		// 	return err
		// }

		var err error
		config.writer, err = os.OpenFile(string(config.ParquetOutputPath), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatal(err)
			return err
		}

		config.parquetWriter = parquet.NewGenericWriter[parquetRow](config.writer,
			parquet.BloomFilters(
				parquet.SplitBlockFilter(10, "query_name"),
			),
			parquet.WriteBufferSize(int(config.ParquetWriteBufferSize)), // 256KB
			parquet.CreatedBy("dnsmonster", "version", "build"),         //TODO: bring real values here
		)

		go config.Output(ctx)
	} else {
		// we will catch this error in the dispatch loop and remove any output from the registry if they don't have the correct output type
		return errors.New("no output")
	}
	return nil
}

func (config parquetConfig) Close() {
	// todo: implement this
	<-config.closeChannel
}

func (config parquetConfig) OutputChannel() chan util.DNSResult {
	return config.outputChannel
}

func (config parquetConfig) Output(ctx context.Context) {
	for i := uint(0); i < config.ParquetWorkerCount; i++ {
		go config.OutputWorker(ctx)
	}
	<-ctx.Done()
	config.parquetWriterLock.Lock()
	if err := config.parquetWriter.Close(); err != nil {
		log.Error(err)
	}
	if err := config.writer.Close(); err != nil {
		log.Error(err)
	}
	config.parquetWriterLock.Unlock()
}

func (config *parquetConfig) OutputWorker(ctx context.Context) {

	cnt := uint(0)
	dataArr := []parquetRow{}
	// todo: output channel will duplicate output when we have malformed DNS packets with multiple questions
	for {
		select {
		case data := <-config.outputChannel:
			cnt++

			// we have only one DNS question per DNS packet. if we have more than one question, we will skip the packet
			if len(data.DNS.Question) > 1 || len(data.DNS.Question) == 0 {
				config.parquetSkipped.Inc(1)
				continue
			}
			q0 := data.DNS.Question[0]
			if util.CheckIfWeSkip(config.ParquetOutputType, q0.Name) {
				config.parquetSkipped.Inc(1)
				continue
			}

			QR := uint8(0)
			if data.DNS.Response {
				QR = 1
			}
			edns, doBit := uint8(0), uint8(0)
			if edns0 := data.DNS.IsEdns0(); edns0 != nil {
				edns = 1
				if edns0.Do() {
					doBit = 1
				}
			}

			dataArr = append(dataArr, parquetRow{
				Timestamp:    data.Timestamp,
				IPVersion:    uint32(data.IPVersion),
				SrcIP:        data.SrcIP,
				DstIP:        data.DstIP,
				Protocol:     data.Protocol,
				QR:           uint32(QR),
				Opcode:       uint32(data.DNS.Opcode),
				Qclass:       uint32(q0.Qclass),
				Qtype:        uint32(q0.Qtype),
				EDNS:         uint32(edns),
				DoBit:        uint32(doBit),
				Rcode:        uint32(data.DNS.Rcode),
				QueryName:    q0.Name,
				PacketLength: uint32(data.PacketLength),
				Identity:     data.Identity,
				Version:      data.Version,
			})
			if cnt%config.ParquetFlushBatchSize == 0 {
				config.parquetWriterLock.Lock()
				if n, err := config.parquetWriter.Write(dataArr); err != nil {
					config.parquetSkipped.Inc(int64(n))
					log.Warn(err)
				}
				if err := config.parquetWriter.Flush(); err != nil {
					log.Error(err)
				} else {
					config.parquetSentToOutput.Inc(int64(len(dataArr)))
				}
				config.parquetWriterLock.Unlock()
				dataArr = []parquetRow{}
			}

		case <-ctx.Done():
			log.Debug("exitting out of parquet output") //todo:remove
			config.parquetWriterLock.Lock()
			if n, err := config.parquetWriter.Write(dataArr); err != nil {
				config.parquetSkipped.Inc(int64(n))
				log.Warn(err)
			}
			if err := config.parquetWriter.Flush(); err != nil {
				log.Error(err)
			} else {
				config.parquetSentToOutput.Inc(int64(len(dataArr)))
			}
			config.parquetWriterLock.Unlock()
			return
		}
	}
}

// vim: foldmethod=marker
