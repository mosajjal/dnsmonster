package main

import (
	"encoding/hex"
	"flag"
	"log"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/afpacket"
	"github.com/google/gopacket/layers"
)

type afpacketHandle struct {
	TPacket *afpacket.TPacket
}

func (h *afpacketHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.TPacket.ReadPacketData()
}

func (h *afpacketHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	return h.TPacket.ZeroCopyReadPacketData()
}
func (h *afpacketHandle) LinkType() layers.LinkType {
	return layers.LinkTypeEthernet
}

func sendpacket(handle *afpacket.TPacket, packet []byte) {
	for {
		handle.WritePacketData(packet)
	}
}

func main() {

	// SrcIp := flag.String("SrcIp", "1.1.1.1", "Source IP, Defaults to 1.1.1.1")
	// DstIp := flag.String("DstIp", "2.2.2.2", "Destination IP, Defaults to 2.2.2.2")
	Interface := flag.String("Interface", "lo", "Interface to use, defaults to lo")
	Workers := flag.Uint("Workers", 4, "Number of Woekers, defaults to 4")
	flag.Parse()

	// ip := &layers.IPv4{
	// 	SrcIP:    net.IP{1, 2, 3, 4},
	// 	DstIP:    net.IP{5, 6, 7, 8},
	// 	Protocol: 17,
	// }
	// buf := gopacket.NewSerializeBuffer()
	// opts := gopacket.SerializeOptions{} // See SerializeOptions for more details.
	// err := ip.SerializeTo(buf, opts)

	var err error
	var handle *afpacket.TPacket

	handle, err = afpacket.NewTPacket(
		afpacket.OptInterface(*Interface),
		afpacket.TPacketVersion3)
	if err != nil {
		log.Fatalf("Error opening afpacket interface: %s", err)
	}
	defer handle.Close()
	packeta, _ := hex.DecodeString("e0cc7a8246cdb00cd14502b3080045000036c20040004011e0e2c0a80b82c0a80b01b1bb0035002298079a1b010000010000000000000462696e6703636f6d0000010001")
	packetb, _ := hex.DecodeString("b00cd14502b3e0cc7a8246cd0800450000560b6840004011975bc0a80b01c0a80b820035b1bb0042ae519a1b818000010002000000000462696e6703636f6d0000010001c00c000100010000087f0004cc4fc5c8c00c000100010000087f00040d6b15c8")
	var wg sync.WaitGroup
	for i := uint(1); i <= *Workers; i++ {
		wg.Add(1)
		go sendpacket(handle, packeta)
		go sendpacket(handle, packetb)
	}
	wg.Wait()
	log.Println("Waiting for packets using AFpacket")
}
