package capture

import (
	"fmt"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
)

// These functions implement the interface gopacket.DecodingLayer to detect
// if a packet is either IPv4 or IPv6.

func (i *detectIP) LayerType() gopacket.LayerType {
	return layerTypeDetectIP
}

func (i *detectIP) DecodeFromBytes(data []byte, df gopacket.DecodeFeedback) error {
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

func (i *detectIP) CanDecode() gopacket.LayerClass {
	return layerTypeDetectIP
}

func (i *detectIP) NextLayerType() gopacket.LayerType {
	return i.family.LayerType()
}
