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
// vim: foldmethod=marker
