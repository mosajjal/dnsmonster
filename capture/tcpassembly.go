package capture

import (
	"encoding/binary"
	"io"
	"net"
	"time"

	"context"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/tcpassembly"
	"github.com/gopacket/gopacket/tcpassembly/tcpreader"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (ds *dnsStream) processStream(ctx context.Context) error {
	var data []byte
	tmp := make([]byte, 4096)

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		for {
			count, err := ds.reader.Read(tmp)

			if err == io.EOF {
				return err
			} else if err != nil {
				log.Info("Error when reading DNS buf", err)
			} else if count > 0 {
				data = append(data, tmp[0:count]...)
				for curLength := len(data); curLength >= 2; curLength = len(data) {
					expected := int(binary.BigEndian.Uint16(data[:2])) + 2
					if curLength >= expected {
						result := data[2:expected]

						// Send the data to be processed
						ds.tcpReturnChannel <- tcpData{
							IPVersion: ds.IPVersion,
							data:      result,
							SrcIP:     net.IP(ds.Net.Src().Raw()),
							DstIP:     net.IP(ds.Net.Dst().Raw()),
							timestamp: ds.timestamp,
						}
						// Save the remaining data for future queries
						data = data[expected:]
					} else {
						break
					}
				}
			}
			return nil
		}
	})
	<-gCtx.Done()
	log.Debug("ending processstream goroutine") //todo:remove
	return nil
}

func (stream *dnsStreamFactory) New(net, transport gopacket.Flow) tcpassembly.Stream {
	dstream := &dnsStream{
		Net:              net,
		reader:           tcpreader.NewReaderStream(),
		tcpReturnChannel: stream.tcpReturnChannel,
		IPVersion:        stream.IPVersion,
		timestamp:        stream.currentTimestamp, // This variable is updated before the assemble call
	}

	// We must read all the data from the reader or we will have the data standing in memory
	//todo: re-defining context here and it's not passed on
	g, gCtx := errgroup.WithContext(context.Background())
	g.Go(func() error { return dstream.processStream(gCtx) })

	return &dstream.reader
}

func tcpAssembler(ctx context.Context, tcpchannel chan tcpPacket, tcpReturnChannel chan tcpData, gcTime time.Duration) error {
	// TCP reassembly init
	streamFactoryV4 := &dnsStreamFactory{
		tcpReturnChannel: tcpReturnChannel,
		IPVersion:        4,
	}
	streamPoolV4 := tcpassembly.NewStreamPool(streamFactoryV4)
	assemblerV4 := tcpassembly.NewAssembler(streamPoolV4)

	streamFactoryV6 := &dnsStreamFactory{
		tcpReturnChannel: tcpReturnChannel,
		IPVersion:        6,
	}
	streamPoolV6 := tcpassembly.NewStreamPool(streamFactoryV6)
	assemblerV6 := tcpassembly.NewAssembler(streamPoolV6)
	ticker := time.NewTicker(gcTime)
	for {
		select {
		case packet := <-tcpchannel:
			{
				switch packet.IPVersion {
				case 4:
					streamFactoryV4.currentTimestamp = packet.timestamp
					assemblerV4.AssembleWithTimestamp(packet.flow, &packet.tcp, time.Now())
				case 6:
					streamFactoryV6.currentTimestamp = packet.timestamp
					assemblerV6.AssembleWithTimestamp(packet.flow, &packet.tcp, time.Now())
				}
			}
		case <-ticker.C:
			{
				// Flush connections that haven't seen activity in the past GcTime.
				assemblerV4.FlushOlderThan(time.Now().Add(gcTime * -1))
				assemblerV6.FlushOlderThan(time.Now().Add(gcTime * -1))
			}
		case <-ctx.Done():
			log.Debug("exitting out of TCP assembly goroutine") //todo:remove
			return nil
		}
	}
}
