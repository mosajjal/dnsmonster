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

package capture

import (
	"context"
	"net"
	"time"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
	mkdns "github.com/miekg/dns"
	"github.com/mosajjal/dnsmonster/internal/util"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

func (config captureConfig) processTransport(foundLayerTypes *[]gopacket.LayerType, udp *layers.UDP, tcp *layers.TCP, flow gopacket.Flow, timestamp time.Time, IPVersion uint8, SrcIP, DstIP net.IP) {
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
					config.resultChannel <- util.DNSResult{
						Timestamp: timestamp,
						Server: util.GeneralFlags.ServerName,
						DNS:       msg, IPVersion: IPVersion, SrcIP: SrcIP.Mask(net.CIDRMask(MaskSize, BitSize)),
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

func (config captureConfig) inputHandlerWorker(ctx context.Context, p chan *rawPacketBytes) error {
	var detectIP detectIP
	var ethLayer layers.Ethernet
	var vlan layers.Dot1Q
	var vxlan layers.VXLAN
	var ip4 layers.IPv4
	var ip6 layers.IPv6
	var udp layers.UDP
	var tcp layers.TCP

	startLayer := layers.LayerTypeEthernet
	decodeLayers := []gopacket.DecodingLayer{
		&ethLayer,
		&vlan,
		&vxlan,
		&ip4,
		&ip6,
		&udp,
		&tcp,
	}
	// Use the IP Family detector when no ethernet frame is present.
	if config.NoEthernetframe {
		decodeLayers[0] = &detectIP
		startLayer = layerTypeDetectIP
	}

	parser := gopacket.NewDecodingLayerParser(startLayer, decodeLayers...)
	foundLayerTypes := []gopacket.LayerType{}
	for {
		select {
		case packet := <-p:
			timestamp := packet.info.Timestamp
			if timestamp.IsZero() {
				timestamp = time.Now()
			}
			if err := parser.DecodeLayers(packet.bytes, &foundLayerTypes); err != nil {
			}
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
		case <-ctx.Done():
			log.Debug("exiting out of inputhandler goroutine") //todo:remove
			return nil
		}

	}
}

func (config captureConfig) StartPacketDecoder(ctx context.Context) error {
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
		g, gCtx := errgroup.WithContext(ctx)
		g.Go(func() error { return config.inputHandlerWorker(gCtx, config.processingChannel) })
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
				config.resultChannel <- util.DNSResult{
					Timestamp: data.timestamp,
					DNS:       msg, IPVersion: data.IPVersion, SrcIP: data.SrcIP.Mask(net.CIDRMask(MaskSize, BitSize)),
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
		case <-ctx.Done():
			log.Debug("exiting out of packet decoder goroutine") //todo:remove
			return nil
		}
	}
}

// vim: foldmethod=marker
