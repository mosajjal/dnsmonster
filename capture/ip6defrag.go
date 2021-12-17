// Package ip6defrag implements a IPv6 defragmenter
package capture

import (
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/gopacket/layers"
)

// Quick and Easy to use debug code to trace
// how defrag works.
var debug debugging = false // or flip to true
type debugging bool

func (d debugging) Printf(format string, args ...interface{}) {
	if d {
		log.Infof(format, args...)
	}
}

// Constants determining how to handle fragments.
const (
	IPv6MaximumSize            = 65535
	IPv6MaximumFragmentOffset  = 8191
	IPv6MaximumFragmentListLen = 8191
)

// DefragIPv6 takes in an IPv6 packet with a fragment payload.
//
// It do not modify the IPv6 layer in place, 'in' and 'inFragment'
// remains untouched. It returns a ready-to be used IPv6 layer.
//
// If we don't have all fragments, it will return nil and store
// whatever internal information it needs to eventually defrag the packet.
//
// If the IPv6 layer is the last fragment needed to reconstruct
// the packet, a new IPv6 layer will be returned, and will be set to
// the entire defragmented packet,
//
// It use a map of all the running flows
//
// Usage example:
//
// func HandlePacket(in *layers.IPv6, inFragment *layers.IPv6Fragment) err {
//     defragger := ip6defrag.NewIPv6Defragmenter()
//     in, err := defragger.DefragIPv6(in, inFragment)
//     if err != nil {
//         return err
//     } else if in == nil {
//         return nil  // packet fragment, we don't have whole packet yet.
//     }
//     // At this point, we know that 'in' is defragmented.
//     //It may be the same 'in' passed to
//	   // HandlePacket, or it may not, but we don't really care :)
//	   ... do stuff to 'in' ...
//}
//
func (d *IPv6Defragmenter) DefragIPv6(in *layers.IPv6, inFragment *layers.IPv6Fragment) (*layers.IPv6, error) {
	return d.DefragIPv6WithTimestamp(in, inFragment, time.Now())
}

// DefragIPv6WithTimestamp provides functionality of DefragIPv6 with
// an additional timestamp parameter which is used for discarding
// old fragments instead of time.Now()
//
// This is useful when operating on pcap files instead of live captured data
//
func (d *IPv6Defragmenter) DefragIPv6WithTimestamp(in *layers.IPv6, inFragment *layers.IPv6Fragment, t time.Time) (*layers.IPv6, error) {
	// perform security checks
	st, err := d.securityChecks(inFragment)
	if err != nil || !st {
		debug.Printf("defrag: alert security check")
		return nil, err
	}

	// ok, got a fragment
	debug.Printf("defrag: got a new fragment in.Id=%d in.FragOffset=%d\n",
		inFragment.Identification, inFragment.FragmentOffset*8)

	// have we already seen a flow between src/dst with that Id?
	ipf := newIPv6(in, inFragment)
	var fl *fragmentList
	var exist bool
	d.Lock()
	fl, exist = d.ipFlows[ipf]
	if !exist {
		debug.Printf("defrag: unknown flow, creating a new one\n")
		fl = new(fragmentList)
		d.ipFlows[ipf] = fl
	}
	d.Unlock()
	// insert, and if final build it
	out, err2 := fl.insert(in, inFragment, t)

	// at last, if we hit the maximum frag list len
	// without any defrag success, we just drop everything and
	// raise an error
	if out == nil && fl.List.Len()+1 > IPv6MaximumFragmentListLen {
		d.flush(ipf)
		return nil, fmt.Errorf("defrag: Fragment List hits its maximum"+
			"size(%d), without success. Flushing the list",
			IPv6MaximumFragmentListLen)
	}

	// if we got a packet, it's a new one, and he is defragmented
	if out != nil {
		// when defrag is done for a flow between two ip
		// clean the list
		d.flush(ipf)
		return out, nil
	}
	return nil, err2
}

// DiscardOlderThan forgets all packets without any activity since
// time t. It returns the number of FragmentList aka number of
// fragment packets it has discarded.
func (d *IPv6Defragmenter) DiscardOlderThan(t time.Time) int {
	var nb int
	d.Lock()
	for k, v := range d.ipFlows {
		if v.LastSeen.Before(t) {
			nb = nb + 1
			delete(d.ipFlows, k)
		}
	}
	d.Unlock()
	return nb
}

// flush the fragment list for a particular flow
func (d *IPv6Defragmenter) flush(ipf ipv6) {
	d.Lock()
	fl := new(fragmentList)
	d.ipFlows[ipf] = fl
	d.Unlock()
}

// securityChecks performs the needed security checks
func (d *IPv6Defragmenter) securityChecks(ip *layers.IPv6Fragment) (bool, error) {
	// don't allow too big fragment offset
	if ip.FragmentOffset > IPv6MaximumFragmentOffset {
		return false, fmt.Errorf("defrag: fragment offset too big "+
			"(handcrafted? %d > %d)", ip.FragmentOffset, IPv6MaximumFragmentOffset)
	}
	fragOffset := uint32(ip.FragmentOffset * 8)
	// don't allow fragment that would oversize an IP packet
	if fragOffset+uint32(len(ip.Payload)) > IPv6MaximumSize {
		return false, fmt.Errorf("defrag: fragment will overrun "+
			"(handcrafted? %d > %d)", fragOffset+uint32(len(ip.Payload)), IPv6MaximumFragmentOffset)
	}
	return true, nil
}

// insert insert an IPv6 fragment/packet into the Fragment List
// It use the following strategy : we are inserting fragment based
// on their offset, latest first. This is sometimes called BSD-Right.
// See: http://www.sans.org/reading-room/whitepapers/detection/ip-fragment-reassembly-scapy-33969
func (f *fragmentList) insert(in *layers.IPv6, fragment *layers.IPv6Fragment, t time.Time) (*layers.IPv6, error) {
	// TODO: should keep a copy of *in in the list
	// or not (ie the packet source is reliable) ? -> depends on Lazy / last packet
	fragOffset := fragment.FragmentOffset * 8
	if fragOffset >= f.Highest {
		f.List.PushBack(fragment)
	} else {
		for e := f.List.Front(); e != nil; e = e.Next() {
			frag, _ := e.Value.(*layers.IPv6Fragment)
			if fragment.FragmentOffset == frag.FragmentOffset {
				// TODO: what if we receive a fragment
				// that begins with duplicate data but
				// *also* has new data? For example:
				//
				// AAAA
				//     BB
				//     BBCC
				//         DDDD
				//
				// In this situation we completely
				// ignore CC and the complete packet can
				// never be reassembled.
				debug.Printf("defrag: ignoring frag %d as we already have it (duplicate?)\n",
					fragOffset)
				return nil, nil
			}
			if fragment.FragmentOffset < frag.FragmentOffset {
				debug.Printf("defrag: inserting frag %d before existing frag %d\n",
					fragOffset, frag.FragmentOffset*8)
				f.List.InsertBefore(fragment, e)
				break
			}
		}
	}

	f.LastSeen = t

	fragLength := uint16(len(fragment.Payload))
	// After inserting the Fragment, we update the counters
	if f.Highest < fragOffset+fragLength {
		f.Highest = fragOffset + fragLength
	}
	f.Current = f.Current + fragLength

	debug.Printf("defrag: insert ListLen: %d Highest:%d Current:%d\n",
		f.List.Len(),
		f.Highest, f.Current)

	// Final Fragment ?
	if !fragment.MoreFragments {
		f.FinalReceived = true
	}
	// Ready to try defrag ?
	if f.FinalReceived && f.Highest == f.Current {
		return f.build(in, fragment)
	}
	return nil, nil
}

// Build builds the final datagram, creating a new ip.
// It puts priority to packet in the early position of the list.
// See Insert for more details.
func (f *fragmentList) build(in *layers.IPv6, fragment *layers.IPv6Fragment) (*layers.IPv6, error) {
	var final []byte
	var currentOffset uint16

	// NOTE: Overlapping IPv5 Fragments MUST be dropped
	for e := f.List.Front(); e != nil; e = e.Next() {
		frag, _ := e.Value.(*layers.IPv6Fragment)
		if frag.FragmentOffset*8 == currentOffset {
			debug.Printf("defrag: building - adding %d\n", frag.FragmentOffset*8)
			final = append(final, frag.Payload...)
			currentOffset = currentOffset + uint16(len(frag.Payload))
		} else {
			// Houston - we have an hole !
			debug.Printf("defrag: hole found while building, " +
				"stopping the defrag process\n")
			return nil, fmt.Errorf("defrag: building - hole found")
		}
		debug.Printf("defrag: building - next is %d\n", currentOffset)
	}

	// TODO recompute IP Checksum
	out := &layers.IPv6{
		Version:      in.Version,
		TrafficClass: in.TrafficClass,
		FlowLabel:    in.FlowLabel,
		Length:       f.Highest,
		NextHeader:   fragment.NextHeader,
		HopLimit:     in.HopLimit,
		SrcIP:        in.SrcIP,
		DstIP:        in.DstIP,
		HopByHop:     in.HopByHop,
	}
	out.Payload = final

	return out, nil
}

// newIPv6 returns a new initialized IPv6 Flow
func newIPv6(ip *layers.IPv6, frag *layers.IPv6Fragment) ipv6 {
	return ipv6{
		ip4: ip.NetworkFlow(),
		id:  frag.Identification,
	}
}

// NewIPv6Defragmenter returns a new IPv6Defragmenter
// with an initialized map.
func NewIPv6Defragmenter() *IPv6Defragmenter {
	return &IPv6Defragmenter{
		ipFlows: make(map[ipv6]*fragmentList),
	}
}
