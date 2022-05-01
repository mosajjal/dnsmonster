package util

import (
	"encoding/binary"
	"fmt"
	"reflect"
)

type CsvOutput struct{}

type CsvRow struct {
	Year         int
	Month        int
	Day          int
	Hour         int
	Minute       int
	Second       int
	Ns           int
	Server       string
	IpVersion    uint8
	SrcIP        uint64
	DstIP        uint64
	Protocol     int
	Qr           int
	OpCode       int
	Class        uint16
	Type         uint16
	ResponseCode int
	Question     string
	Size         uint16
	Edns0Present int
	DoBit        int
	Id           uint16
}

// currently there's not a better way to do this unless you sacrifice performance by 10x
func formatCsvRow(csvrow CsvRow) string {
	return fmt.Sprintf("%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v,%v",
		csvrow.Year,
		csvrow.Month,
		csvrow.Day,
		csvrow.Hour,
		csvrow.Minute,
		csvrow.Second,
		csvrow.Ns,
		csvrow.Server,
		csvrow.IpVersion,
		csvrow.SrcIP,
		csvrow.DstIP,
		csvrow.Protocol,
		csvrow.Qr,
		csvrow.OpCode,
		csvrow.Class,
		csvrow.Type,
		csvrow.ResponseCode,
		csvrow.Question,
		csvrow.Size,
		csvrow.Edns0Present,
		csvrow.DoBit,
		csvrow.Id,
	)
}

func (c CsvOutput) Marshal(d DNSResult) string {
	// the integer version of the IP is much more useful in Machine learning than the string
	var SrcIP, DstIP uint64
	if d.IPVersion == 4 {
		SrcIP = uint64(binary.BigEndian.Uint32(d.SrcIP))
		DstIP = uint64(binary.BigEndian.Uint32(d.DstIP))
	} else {
		SrcIP = binary.BigEndian.Uint64(d.SrcIP[:8]) // limitation of clickhouse-go doesn't let us go more than 64 bits for ipv6 at the moment
		DstIP = binary.BigEndian.Uint64(d.DstIP[:8])
	}

	// calculating the protocol number in integer
	protocolNumber := 0
	if d.Protocol == "udp" {
		protocolNumber = 17
	} else {
		protocolNumber = 6
	}

	// QR should be one if the packet has a response section
	QR := 0
	if d.DNS.Response {
		QR = 1
	}

	// calculate edns and dobit
	edns, dobit := 0, 0
	if edns0 := d.DNS.IsEdns0(); edns0 != nil {
		edns = 1
		if edns0.Do() {
			dobit = 1
		}
	}
	s := CsvRow{
		Year:         d.Timestamp.Year(),
		Month:        int(d.Timestamp.Month()),
		Day:          d.Timestamp.Day(),
		Hour:         d.Timestamp.Hour(),
		Minute:       d.Timestamp.Minute(),
		Second:       d.Timestamp.Second(),
		Ns:           d.Timestamp.Nanosecond(),
		Server:       GeneralFlags.ServerName,
		IpVersion:    d.IPVersion,
		SrcIP:        SrcIP,
		DstIP:        DstIP,
		Protocol:     protocolNumber,
		Qr:           QR,
		OpCode:       d.DNS.Opcode,
		Class:        d.DNS.Question[0].Qclass,
		Type:         d.DNS.Question[0].Qtype,
		ResponseCode: d.DNS.Rcode,
		Question:     d.DNS.Question[0].Name,
		Size:         d.PacketLength,
		Edns0Present: edns,
		DoBit:        dobit,
		Id:           d.DNS.Id,
	}
	return formatCsvRow(s)
}

// return headers for above csv
func (c CsvOutput) Init() (string, error) {
	v := reflect.ValueOf(CsvRow{})
	typeOfV := v.Type()
	csvHeader := ""
	for i := 0; i < v.NumField(); i++ {
		// Get the field, returns https://golang.org/pkg/reflect/#StructField
		csvHeader += typeOfV.Field(i).Name + "," // todo: do we need to lowercase the headers
	}
	// remove trailing comma
	return csvHeader[:len(csvHeader)-1], nil
}
