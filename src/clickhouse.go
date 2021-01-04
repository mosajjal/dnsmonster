package main

import (
	"encoding/binary"
	"fmt"

	"log"
	"net"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go"
	data "github.com/ClickHouse/clickhouse-go/lib/data"
	"github.com/rogpeppe/fastuuid"
)

var uuidGen = fastuuid.MustNewGenerator()

func connectClickhouseRetry(exiting chan bool, clickhouseHost string) clickhouse.Clickhouse {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		c, err := connectClickhouse(exiting, clickhouseHost)
		if err == nil {
			return c
		}

		// Error getting connection, wait the timer or check if we are exiting
		select {
		case <-exiting:
			// When exiting, return immediately
			return nil
		case <-tick.C:
			continue
		}
	}
}

func connectClickhouse(exiting chan bool, clickhouseHost string) (clickhouse.Clickhouse, error) {
	connection, err := clickhouse.OpenDirect(fmt.Sprintf("tcp://%v?debug=%v", clickhouseHost, *clickhouseDebug))
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return connection, err
}

func min(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func output(resultChannel chan DNSResult, exiting chan bool, wg *sync.WaitGroup, clickhouseHost string, batchSize uint, batchDelay time.Duration, limit int, server string) {
	wg.Add(1)
	defer wg.Done()
	serverByte := []byte(server)

	connect := connectClickhouseRetry(exiting, clickhouseHost)
	batch := make([]DNSResult, 0, batchSize)

	ticker := time.Tick(batchDelay)
	for {
		select {
		case data := <-resultChannel:
			if limit == 0 || len(batch) < limit {
				batch = append(batch, data)
			}
		case <-ticker:
			if err := SendData(connect, batch, serverByte); err != nil {
				log.Println(err)
				connect = connectClickhouseRetry(exiting, clickhouseHost)
			} else {
				batch = make([]DNSResult, 0, batchSize)
			}
		case <-exiting:
			return
		}
	}
}

func SendData(connect clickhouse.Clickhouse, batch []DNSResult, server []byte) error {
	if len(batch) == 0 {
		return nil
	}

	// Return if the connection is null, we are exiting
	if connect == nil {
		return nil
	}
	// log.Println("Sending ", len(batch))
	myStats.sentToDB += len(batch)
	_, err := connect.Begin()
	if err != nil {
		return err
	}

	_, err = connect.Prepare("INSERT INTO DNS_LOG (DnsDate, timestamp, Server, IPVersion, IPPrefix, Protocol, QR, OpCode, Class, Type, ResponseCode, Question, Size, Edns0Present, DoBit, ID) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		return err
	}

	block, err := connect.Block()
	if err != nil {
		return err
	}

	blocks := []*data.Block{block}

	count := len(blocks)
	var wg sync.WaitGroup
	wg.Add(len(blocks))
	for i := range blocks {
		b := blocks[i]
		start := i * (len(batch)) / count
		end := min((i+1)*(len(batch))/count, len(batch))

		go func() {
			defer wg.Done()
			b.Reserve()
			for k := start; k < end; k++ {
				for _, dnsQuery := range batch[k].DNS.Question {
					// getting variables ready
					ip := batch[k].DstIP
					if batch[k].IPVersion == 4 {
						ip = ip.Mask(net.CIDRMask(*maskSize, 32))
					}
					QR := uint8(0)
					if batch[k].DNS.Response {
						QR = 1
					}
					edns, doBit := uint8(0), uint8(0)
					if edns0 := batch[k].DNS.IsEdns0(); edns0 != nil {
						edns = 1
						if edns0.Do() {
							doBit = 1
						}
					}

					b.NumRows++
					//writing the vars into a SQL statement
					b.WriteDate(0, batch[k].Timestamp)
					b.WriteDateTime(1, batch[k].Timestamp)
					b.WriteBytes(2, server)
					b.WriteUInt8(3, batch[k].IPVersion)
					b.WriteUInt32(4, binary.BigEndian.Uint32(ip[:4]))
					b.WriteFixedString(5, []byte(batch[k].Protocol))
					b.WriteUInt8(6, QR)
					b.WriteUInt8(7, uint8(batch[k].DNS.Opcode))
					b.WriteUInt16(8, uint16(dnsQuery.Qclass))
					b.WriteUInt16(9, uint16(dnsQuery.Qtype))
					b.WriteUInt8(10, uint8(batch[k].DNS.Rcode))
					b.WriteString(11, string(dnsQuery.Name))
					b.WriteUInt16(12, batch[k].PacketLength)
					b.WriteUInt8(13, edns)
					b.WriteUInt8(14, doBit)
					// b.WriteFixedString(15, uuid.NewV4().Bytes())
					myUUID := uuidGen.Next()
					b.WriteFixedString(15, myUUID[:16])
					// b.WriteArray(15, uuidGen.Next())
				}
			}
			if err := connect.WriteBlock(b); err != nil {
				return
			}
		}()
	}

	wg.Wait()
	if err := connect.Commit(); err != nil {
		return err
	}

	return nil
}
