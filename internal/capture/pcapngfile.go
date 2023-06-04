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
	"io"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/pcapgo"
	log "github.com/sirupsen/logrus"
)

type pcapngFileHandle struct {
	reader   *pcapgo.NgReader
	file     io.Reader
	pktsRead uint
}

func initializeOfflinePcapNg(f io.Reader, filter string) *pcapngFileHandle {

	handle, err := pcapgo.NewNgReader(f, pcapgo.DefaultNgReaderOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Set Filter
	log.Warnf("BPF Filter is not supported in offline mode.")
	if err != nil {
		log.Fatal(err)
	}

	return &pcapngFileHandle{handle, f, 0}
}

func (h *pcapngFileHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	data, ci, err = h.reader.ReadPacketData()
	if err == nil {
		h.pktsRead++
	}
	return
}

func (h *pcapngFileHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	data, ci, err = h.reader.ZeroCopyReadPacketData()
	if err == nil {
		h.pktsRead++
	}
	return
}

func (h *pcapngFileHandle) Close() {
	// h.file.Close()
}

func (h *pcapngFileHandle) Stat() (uint, uint, error) {
	// `pcapnggo.Reader` doesn't have a Stats() method, so we track packets
	// captured by ourselves. There should be no loss for a PCAP file since
	// it's controlled by I/O and not network
	return h.pktsRead, 0, nil
}
// vim: foldmethod=marker
