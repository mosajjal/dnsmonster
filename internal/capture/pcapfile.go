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
	"bufio"
	"io"
	"os"

	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/pcapgo"
	log "github.com/sirupsen/logrus"
)

type pcapFileHandle struct {
	reader   *pcapgo.Reader
	file     io.Reader
	pktsRead uint
}

func initializeOfflineCapture(fileName string, filter string) genericPacketHandler {
	var f *os.File
	if fileName == "-" {
		f = os.Stdin
	} else {
		var err error
		f, err = os.Open(fileName)
		if err != nil {
			return nil
		}
	}

	bufF := bufio.NewReader(f)
	magic, err := bufF.Peek(4)
	if err != nil {
		return nil
	}

	if magic[0] == 0x0a && magic[1] == 0x0d && magic[2] == 0x0d && magic[3] == 0x0a {
		log.Infof("using pcapng file: %s", fileName)
		return initializeOfflinePcapNg(bufF, filter)
	} else {
		log.Infof("using pcap file: %s", fileName)
		return initializeOfflinePcap(bufF, filter)
	}

}

func initializeOfflinePcap(f io.Reader, filter string) *pcapFileHandle {
	handle, err := pcapgo.NewReader(f)
	// Set Filter
	log.Warnf("BPF Filter is not supported in offline mode.")
	if err != nil {
		log.Fatal(err)
	}
	return &pcapFileHandle{handle, f, 0}
}

func (h *pcapFileHandle) ReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	data, ci, err = h.reader.ReadPacketData()
	if err == nil {
		h.pktsRead++
	}
	return
}

func (h *pcapFileHandle) ZeroCopyReadPacketData() (data []byte, ci gopacket.CaptureInfo, err error) {
	data, ci, err = h.reader.ZeroCopyReadPacketData()
	if err == nil {
		h.pktsRead++
	}
	return
}

func (h *pcapFileHandle) Close() {
	// h.file.Close()
}

func (h *pcapFileHandle) Stat() (uint, uint, error) {
	// `pcapgo.Reader` doesn't have a Stats() method, so we track packets
	// captured by ourselves. There should be no loss for a PCAP file since
	// it's controlled by I/O and not network
	return h.pktsRead, 0, nil
}
// vim: foldmethod=marker
