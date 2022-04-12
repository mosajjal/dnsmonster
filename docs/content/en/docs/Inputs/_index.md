---
title: "Inputs and Filters"
linkTitle: "Inputs and Filters"
weight: 4
description: >
  Set up an input to receive data
---

To get the raw data into `dnsmonster` pipeline, you must specify an input stream. Currently there are three supported Input methods:

- Live interface
- Pcap file
- `dnstap` socket

The configuration for inputs and packet processing is contained within the `capture` section of the configuration:

- `--devName`: Enables live capture mode on the device. Only one interface per `dnsmonster` instance is supported. 

- `--pcapFile`: Enables offline pcap mode. You can specify "-" as pcap file to read from stdin

- `--dnstapSocket`: Enables dnstap mode. Accepts a socket path. Example: unix:///tmp/dnstap.sock, tcp://127.0.0.1:8080.

- `--port`: Port selected to filter packets (default: 53). Works independently from BPF filter

- `--sampleRatio`: Specifies packet sampling ratio at capture time. default is 1:1 meaning all packets passing the bpf will get processed.

- `--dedupCleanupInterval`: In case --dedup is enabled, cleans up packet hash table used for it (default: 60s) 

- `--dnstapPermission`: Set the dnstap socket permission, only applicable when unix:// is used (default: 755) 

- `--packetHandlerCount`: Number of workers used to handle received packets (default: 2)

- `--tcpAssemblyChannelSize`: Specifies the goroutine channel size for the TCP assembler. TCP assembler is used to de-fragment incoming fragmented TCP packets in a way that won't slow down the process of "normal" UDP packets.

- `--tcpResultChannelSize`: Size of the tcp result channel (default: 10000) 

- `--tcpHandlerCount`: Number of routines used to handle TCP DNS packets (default: 1) 

- `--defraggerChannelSize`: Size of the channel to send raw packets to be de-fragmented (default: 10000) 

- `--defraggerChannelReturnSize`: Size of the channel where the de-fragmented packets are sent to the output queue (default: 10000) 

- `--packetChannelSize`: Size of the packet handler channel (default: 1000) 

- `--afpacketBuffersizeMb`: Afpacket buffer size in MB (default: 64) 

- `--filter`: BPF filter applied to the packet stream.

- `--useAfpacket`: Use this boolean flag to switch on `afpacket` sniff method on live interfaces

- `--noEtherframe`: Use this boolean flag if the incoming packets (pcap file) do not contain the Ethernet frame 

- `--dedup`: Boolean flag to enable experimental de-duplication engine

- `--noPromiscuous`: Boolean flag to prevent `dnsmonster` to automatically put the `devName` in promiscuous mode  


Above flags are used in variety of ways. Check the [Filters and Masks](./filters_masks) and [inputs](./inputs) for more detailed info.