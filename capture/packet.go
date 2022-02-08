package capture

import (
	"math/rand"
	"time"

	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	mkdns "github.com/miekg/dns"
	"github.com/mosajjal/dnsmonster/types"
	"github.com/mosajjal/dnsmonster/util"
	log "github.com/sirupsen/logrus"
)

func (config CaptureConfig) processTransport(foundLayerTypes *[]gopacket.LayerType, udp *layers.UDP, tcp *layers.TCP, flow gopacket.Flow, timestamp time.Time, IPVersion uint8, SrcIP, DstIP net.IP) {
	for _, layerType := range *foundLayerTypes {
		switch layerType {
		case layers.LayerTypeUDP:
			if uint16(udp.DstPort) == uint16(config.Port) || uint16(udp.SrcPort) == uint16(config.Port) {
				msg := mkdns.Msg{}
				err := msg.Unpack(udp.Payload)
				// Process if no error or truncated, as it will have most of the information it have available
				if err == nil {
					MaskSize := util.GeneralFlags.MaskSize4
					BitSize := 8 * net.IPv4len
					if IPVersion == 6 {
						MaskSize = util.GeneralFlags.MaskSize6
						BitSize = 8 * net.IPv6len
					}
					config.resultChannel <- types.DNSResult{Timestamp: timestamp,
						DNS: msg, IPVersion: IPVersion, SrcIP: SrcIP.Mask(net.CIDRMask(MaskSize, BitSize)),
						DstIP: DstIP.Mask(net.CIDRMask(MaskSize, BitSize)), Protocol: "udp", PacketLength: uint16(len(udp.Payload)),
					}
				}
			}
		case layers.LayerTypeTCP:
			if uint16(tcp.SrcPort) == uint16(config.Port) || uint16(tcp.DstPort) == uint16(config.Port) {
				config.tcpAssembly <- tcpPacket{
					IPVersion,
					*tcp,
					timestamp,
					flow,
				}
			}
		}
	}

}

func (config CaptureConfig) inputHandlerWorker(p chan *rawPacketBytes) {

	var detectIP DetectIP
	var ethLayer layers.Ethernet
	var vlan layers.Dot1Q
	var ip4 layers.IPv4
	var ip6 layers.IPv6
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
	if config.NoEthernetframe {
		decodeLayers[0] = &detectIP
		startLayer = LayerTypeDetectIP
	}

	parser := gopacket.NewDecodingLayerParser(startLayer, decodeLayers...)
	foundLayerTypes := []gopacket.LayerType{}
	for packet := range p {
		timestamp := packet.info.Timestamp
		if timestamp.IsZero() {
			timestamp = time.Now()
		}
		parser.DecodeLayers(packet.bytes, &foundLayerTypes)
		// for _, layer := range foundLayerTypes {
		// 	log.Warnf("found %#+v layer", layer.String()) //todo:remove
		// }
		// first parse the ip layer, so we can find fragmented packets
		for _, layerType := range foundLayerTypes {
			switch layerType {
			case layers.LayerTypeIPv4:
				// Check for fragmentation
				if ip4.Flags&layers.IPv4DontFragment == 0 && (ip4.Flags&layers.IPv4MoreFragments != 0 || ip4.FragOffset != 0) {
					// Packet is fragmented, send it to the defragger
					config.ip4Defrgger <- ipv4ToDefrag{
						ip4,
						timestamp,
					}
				} else {
					// log.Infof("packet %v coming to %p\n", timestamp, &encoder)

					config.processTransport(&foundLayerTypes, &udp, &tcp, ip4.NetworkFlow(), timestamp, 4, ip4.SrcIP, ip4.DstIP)
				}
			case layers.LayerTypeIPv6:
				// Store the packet metadata
				if ip6.NextHeader == layers.IPProtocolIPv6Fragment {
					// TODO: Move the parsing to DecodingLayer when gopacket support it. Currently we have to fully reconstruct the packet from eth layer which is super slow
					reconstructedPacket := gopacket.NewPacket(packet.bytes, layers.LayerTypeEthernet, gopacket.Default)
					if frag := reconstructedPacket.Layer(layers.LayerTypeIPv6Fragment).(*layers.IPv6Fragment); frag != nil {
						config.ip6Defrgger <- ipv6FragmentInfo{
							ip6,
							*frag,
							timestamp,
						}
					}
				} else {
					config.processTransport(&foundLayerTypes, &udp, &tcp, ip6.NetworkFlow(), timestamp, 6, ip6.SrcIP, ip6.DstIP)
				}
			}
		}

	}

}

func (config CaptureConfig) StartPacketDecoder() {

	rand.Seed(20)
	var ip4 layers.IPv4

	var udp layers.UDP
	var tcp layers.TCP

	parserOnlyUDP := gopacket.NewDecodingLayerParser(
		layers.LayerTypeUDP,
		&udp,
	)
	parserOnlyTCP := gopacket.NewDecodingLayerParser(
		layers.LayerTypeTCP,
		&tcp,
	)
	foundLayerTypes := []gopacket.LayerType{}

	// workerHandlerChannel := make(chan *rawPacketBytes, config.PacketChannelSize)
	for i := 0; i < int(config.PacketHandlerCount); i++ {
		log.Infof("Creating handler #%d", i)
		go config.inputHandlerWorker(config.processingChannel)
	}

	for {
		select {
		case data := <-config.tcpReturnChannel:
			msg := mkdns.Msg{}
			if err := msg.Unpack(data.data); err == nil {
				MaskSize := util.GeneralFlags.MaskSize4
				BitSize := 8 * net.IPv4len
				if data.IPVersion == 6 {
					MaskSize = util.GeneralFlags.MaskSize6
					BitSize = 8 * net.IPv6len
				}
				config.resultChannel <- types.DNSResult{Timestamp: data.timestamp,
					DNS: msg, IPVersion: data.IPVersion, SrcIP: data.SrcIP.Mask(net.CIDRMask(MaskSize, BitSize)),
					DstIP: data.DstIP.Mask(net.CIDRMask(MaskSize, BitSize)), Protocol: "tcp", PacketLength: uint16(len(data.data)),
				}
			}
		case packet := <-config.ip4DefrggerReturn:
			// Packet was defragged, parse the remaining data
			if packet.ip.Protocol == layers.IPProtocolUDP {
				parserOnlyUDP.DecodeLayers(packet.ip.Payload, &foundLayerTypes)
			} else if ip4.Protocol == layers.IPProtocolTCP {
				parserOnlyTCP.DecodeLayers(packet.ip.Payload, &foundLayerTypes)
			} else {
				// Protocol not supported
				break
			}
			config.processTransport(&foundLayerTypes, &udp, &tcp, ip4.NetworkFlow(), packet.timestamp, 4, packet.ip.SrcIP, packet.ip.DstIP)
		case packet := <-config.ip6DefrggerReturn:
			// Packet was defragged, parse the remaining data
			if packet.ip.NextHeader == layers.IPProtocolUDP {
				parserOnlyUDP.DecodeLayers(packet.ip.Payload, &foundLayerTypes)
			} else if packet.ip.NextHeader == layers.IPProtocolTCP {
				parserOnlyTCP.DecodeLayers(packet.ip.Payload, &foundLayerTypes)
			} else {
				// Protocol not supported
				break
			}
			config.processTransport(&foundLayerTypes, &udp, &tcp, packet.ip.NetworkFlow(), packet.timestamp, 6, packet.ip.SrcIP, packet.ip.DstIP)
			// case packet := <-config.processingChannel:
			// 	workerHandlerChannel <- packet
		}
	}
}
