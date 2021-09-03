package main

import (
	"fmt"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// These functions implement the interface gopacket.DecodingLayer to detect
// if a packet is either IPv4 or IPv6.

func (i *DetectIP) LayerType() gopacket.LayerType {
	return LayerTypeDetectIP
}

func (i *DetectIP) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
	family := int(data[0] >> 4)
	switch family {
	case 4:
		i.family = layers.EthernetTypeIPv4
	case 6:
		i.family = layers.EthernetTypeIPv6
	default:
		return fmt.Errorf("unknown IP family %d", family)
	}
	i.Payload = data
	return nil
}

func (i *DetectIP) CanDecode() gopacket.LayerClass {
	return LayerTypeDetectIP
}

func (i *DetectIP) NextLayerType() gopacket.LayerType {
	return i.family.LayerType()
}
