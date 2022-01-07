package types

import (
	"encoding/binary"
	"fmt"
)

func (d *DNSResult) CsvRow() string {
	//	timestamp, Server, IPVersion, SrcIP, DstIP, Protocol, QR, OpCode, Class, Type, ResponseCode, Question, Size, Edns0Present, DoBit,FullQuery, ID
	timestamp := fmt.Sprintf("%d,%d,%d,%d,%d,%d,%d", d.Timestamp.Year(), d.Timestamp.Month(), d.Timestamp.Day(), d.Timestamp.Hour(), d.Timestamp.Minute(), d.Timestamp.Second(), d.Timestamp.Nanosecond())
	server := fmt.Sprintf("%s", "dnsmonster") //todo: change this to flag parameter
	ipVersion := fmt.Sprintf("%d", d.IPVersion)

	var SrcIP, DstIP uint64

	if d.IPVersion == 4 {
		SrcIP = uint64(binary.BigEndian.Uint32(d.SrcIP))
		DstIP = uint64(binary.BigEndian.Uint32(d.DstIP))
	} else {
		SrcIP = binary.BigEndian.Uint64(d.SrcIP[8:]) //limitation of clickhouse-go doesn't let us go more than 64 bits for ipv6 at the moment
		DstIP = binary.BigEndian.Uint64(d.DstIP[8:])
	}

	srcIP := fmt.Sprintf("%d", SrcIP)
	dstIP := fmt.Sprintf("%d", DstIP)
	protocol := fmt.Sprintf("%s", d.Protocol) // todo: for ML, it's better to use an integer for this
	QR := uint8(0)
	if d.DNS.Response {
		QR = 1
	}
	qr := fmt.Sprintf("%d", QR)
	opCode := fmt.Sprintf("%d", d.DNS.Opcode)
	class := fmt.Sprintf("%d", d.DNS.Question[0].Qclass) //todo: multiple questions needs to be dealt with
	type_ := fmt.Sprintf("%d", d.DNS.Question[0].Qtype)
	responseCode := fmt.Sprintf("%d", d.DNS.Rcode)
	question := fmt.Sprintf("%s", d.DNS.Question[0].Name)
	size := fmt.Sprintf("%d", d.PacketLength)
	edns, dobit := uint8(0), uint8(0)
	if edns0 := d.DNS.IsEdns0(); edns0 != nil {
		edns = 1
		if edns0.Do() {
			dobit = 1
		}
	}
	edns0Present := fmt.Sprintf("%d", edns)
	doBit := fmt.Sprintf("%d", dobit)
	id := fmt.Sprintf("%d", d.DNS.Id)
	return fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s,%s", timestamp, server, ipVersion, srcIP, dstIP, protocol, qr, opCode, class, type_, responseCode, question, size, edns0Present, doBit, id)
}

func PrintCsvHeader() {
	fmt.Println("year,month,day,hour,minute,second,ns,server,ipVersion,srcIP,dstIP,protocol,qr,opCode,class,type,responseCode,question,size,edns0Present,doBit,id") // print headers for csv
}
