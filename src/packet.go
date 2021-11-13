package main

import (
	"time"

	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	mkdns "github.com/miekg/dns"
	"github.com/mosajjal/dnsmonster/types"
)

func (encoder *packetEncoder) processTransport(foundLayerTypes *[]gopacket.LayerType, udp *layers.UDP, tcp *layers.TCP, flow gopacket.Flow, timestamp time.Time, IPVersion uint8, SrcIP, DstIP net.IP) {
	for _, layerType := range *foundLayerTypes {
		switch layerType {
		case layers.LayerTypeUDP:
			if uint16(udp.DstPort) == encoder.port || uint16(udp.SrcPort) == encoder.port {
				msg := mkdns.Msg{}
				err := msg.Unpack(udp.Payload)
				// Process if no error or truncated, as it will have most of the information it have available
				if err == nil {
					MaskSize := generalOptions.MaskSize4
					BitSize := 8 * net.IPv4len
					if IPVersion == 6 {
						MaskSize = generalOptions.MaskSize6
						BitSize = 8 * net.IPv6len
					}
					encoder.resultChannel <- types.DNSResult{timestamp, msg, IPVersion, SrcIP.Mask(net.CIDRMask(MaskSize, BitSize)), DstIP.Mask(net.CIDRMask(MaskSize, BitSize)), "udp", uint16(len(udp.Payload))}
				}
			}
		case layers.LayerTypeTCP:
			if uint16(tcp.SrcPort) == encoder.port || uint16(tcp.DstPort) == encoder.port {
				encoder.tcpAssembly[flow.FastHash()%uint64(len(encoder.tcpAssembly))] <- tcpPacket{
					IPVersion,
					*tcp,
					timestamp,
					flow,
				}
			}
		}
	}

}

func (encoder *packetEncoder) run() {
	var detectIP DetectIP
	var ethLayer layers.Ethernet
	var ip4 layers.IPv4
	var ip6 layers.IPv6
	var vlan layers.Dot1Q
	var udp layers.UDP
	var tcp layers.TCP

	startLayer := layers.LayerTypeEthernet
	decodeLayers := []gopacket.DecodingLayer{
		&ethLayer,
		&vlan,
		&ip4,
		&ip6,
		&udp,
		&tcp,
	}
	// Use the IP Family detector when no ethernet frame is present.
	if encoder.NoEthernetframe {
		decodeLayers[0] = &detectIP
		startLayer = LayerTypeDetectIP
	}

	parser := gopacket.NewDecodingLayerParser(startLayer, decodeLayers...)
	parserOnlyUDP := gopacket.NewDecodingLayerParser(
		layers.LayerTypeUDP,
		&udp,
	)
	parserOnlyTCP := gopacket.NewDecodingLayerParser(
		layers.LayerTypeTCP,
		&tcp,
	)
	foundLayerTypes := []gopacket.LayerType{}
	for {
		select {
		case data := <-encoder.tcpReturnChannel:
			msg := mkdns.Msg{}
			if err := msg.Unpack(data.data); err == nil {
				MaskSize := generalOptions.MaskSize4
				BitSize := 8 * net.IPv4len
				if data.IPVersion == 6 {
					MaskSize = generalOptions.MaskSize6
					BitSize = 8 * net.IPv6len
				}
				encoder.resultChannel <- types.DNSResult{data.timestamp, msg, data.IPVersion, data.SrcIP.Mask(net.CIDRMask(MaskSize, BitSize)), data.DstIP.Mask(net.CIDRMask(MaskSize, BitSize)), "tcp", uint16(len(data.data))}
			}
		case packet := <-encoder.ip4DefrggerReturn:
			// Packet was defragged, parse the remaining data
			if packet.ip.Protocol == layers.IPProtocolUDP {
				parserOnlyUDP.DecodeLayers(packet.ip.Payload, &foundLayerTypes)
			} else if ip4.Protocol == layers.IPProtocolTCP {
				parserOnlyTCP.DecodeLayers(packet.ip.Payload, &foundLayerTypes)
			} else {
				// Protocol not supported
				break
			}
			encoder.processTransport(&foundLayerTypes, &udp, &tcp, ip4.NetworkFlow(), packet.timestamp, 4, packet.ip.SrcIP, packet.ip.DstIP)
		case packet := <-encoder.ip6DefrggerReturn:
			// Packet was defragged, parse the remaining data
			if packet.ip.NextHeader == layers.IPProtocolUDP {
				parserOnlyUDP.DecodeLayers(packet.ip.Payload, &foundLayerTypes)
			} else if packet.ip.NextHeader == layers.IPProtocolTCP {
				parserOnlyTCP.DecodeLayers(packet.ip.Payload, &foundLayerTypes)
			} else {
				// Protocol not supported
				break
			}
			encoder.processTransport(&foundLayerTypes, &udp, &tcp, packet.ip.NetworkFlow(), packet.timestamp, 6, packet.ip.SrcIP, packet.ip.DstIP)
		case packet := <-encoder.input:
			{
				timestamp := packet.Metadata().Timestamp
				if timestamp.IsZero() {
					timestamp = time.Now()
				}
				_ = parser.DecodeLayers(packet.Data(), &foundLayerTypes)
				// first parse the ip layer, so we can find fragmented packets
				for _, layerType := range foundLayerTypes {
					switch layerType {
					case layers.LayerTypeIPv4:
						// Check for fragmentation
						if ip4.Flags&layers.IPv4DontFragment == 0 && (ip4.Flags&layers.IPv4MoreFragments != 0 || ip4.FragOffset != 0) {
							// Packet is fragmented, send it to the defragger
							encoder.ip4Defrgger <- ipv4ToDefrag{
								ip4,
								timestamp,
							}
							break
						}
						// log.Infof("packet %v coming to %p\n", timestamp, &encoder)
						encoder.processTransport(&foundLayerTypes, &udp, &tcp, ip4.NetworkFlow(), timestamp, 4, ip4.SrcIP, ip4.DstIP)
						break
					case layers.LayerTypeIPv6:
						// Store the packet metadata
						if ip6.NextHeader == layers.IPProtocolIPv6Fragment {
							// TODO: Move the parsing to DecodingLayer when gopacket support it
							if frag := packet.Layer(layers.LayerTypeIPv6Fragment).(*layers.IPv6Fragment); frag != nil {
								encoder.ip6Defrgger <- ipv6FragmentInfo{
									ip6,
									*frag,
									timestamp,
								}
							}
						} else {
							encoder.processTransport(&foundLayerTypes, &udp, &tcp, ip6.NetworkFlow(), timestamp, 6, ip6.SrcIP, ip6.DstIP)
						}
					}
				}
				break
			}
		case <-encoder.done:
			break
		}
	}
}
