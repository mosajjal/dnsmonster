package main

import (
	"github.com/google/gopacket/ip4defrag"
	"github.com/google/gopacket/layers"

	// "github.com/mosajjal/dnsmonster/ip6defrag"
	"time"
)

type ipv4ToDefrag struct {
	ip        layers.IPv4
	timestamp time.Time
}

type ipv4Defragged struct {
	ip        layers.IPv4
	timestamp time.Time
}

type ipv6FragmentInfo struct {
	ip         layers.IPv6
	ipFragment layers.IPv6Fragment
	timestamp  time.Time
}

type ipv6Defragged struct {
	ip        layers.IPv6
	timestamp time.Time
}

func ipv4Defragger(ipInput <-chan ipv4ToDefrag, ipOut chan ipv4Defragged, gcTime time.Duration, done chan bool) {
	ipv4Defragger := ip4defrag.NewIPv4Defragmenter()
	ticker := time.NewTicker(1 * gcTime)
	for {
		select {
		case packet := <-ipInput:
			result, err := ipv4Defragger.DefragIPv4(&packet.ip)
			if err == nil && result != nil {
				ipOut <- ipv4Defragged{
					*result,
					packet.timestamp,
				}
			}
		case <-ticker.C:
			ipv4Defragger.DiscardOlderThan(time.Now().Add(gcTime * -1))
		case <-done:
			ticker.Stop()
			return
		}
	}
}

func ipv6Defragger(ipInput <-chan ipv6FragmentInfo, ipOut chan ipv6Defragged, gcTime time.Duration, done chan bool) {
	ipv4Defragger := NewIPv6Defragmenter()
	ticker := time.NewTicker(1 * gcTime)
	for {
		select {
		case packet := <-ipInput:
			result, err := ipv4Defragger.DefragIPv6(&packet.ip, &packet.ipFragment)
			if err == nil && result != nil {
				ipOut <- ipv6Defragged{
					*result,
					packet.timestamp,
				}
			}
		case <-ticker.C:
			ipv4Defragger.DiscardOlderThan(time.Now().Add(gcTime * -1))
		case <-done:
			ticker.Stop()
			return
		}
	}
}
