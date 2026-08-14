---
title: Inputs and filters
description: Every capture-side flag in dnsmonster — live interface, pcap file and dnstap socket, plus the packet handling parameters around them.
sidebar:
  label: Overview
  order: 1
---

To get raw data into the `dnsmonster` pipeline you must specify an input stream. There are three
supported input methods:

- a live interface
- a pcap file
- a `dnstap` socket

Configuration for inputs and packet processing lives in the `capture` section of the configuration.

## Input selection

| Flag | Purpose |
| --- | --- |
| `--devName` | Enables live capture on the device. One interface per `dnsmonster` instance. |
| `--pcapFile` | Enables offline pcap mode. Use `-` to read from stdin. |
| `--dnstapSocket` | Enables dnstap mode. Accepts a socket path, e.g. `unix:///tmp/dnstap.sock` or `tcp://127.0.0.1:8080`. |

## Capture parameters

| Flag | Default | Purpose |
| --- | --- | --- |
| `--port` | `53` | Port used to filter packets. Works independently of the BPF filter. |
| `--sampleRatio` | `1:1` | Packet sampling ratio at capture time. All packets passing the BPF are processed by default. |
| `--dedup` | off | Enables the experimental de-duplication engine. |
| `--dedupCleanupInterval` | `60s` | Cleans up the packet hash table used by `--dedup`. |
| `--dnstapPermission` | `755` | Permission on the dnstap socket. Only applies to `unix://`. |
| `--filter` | — | BPF filter applied to the packet stream. |
| `--useAfpacket` | off | Switches on the `afpacket` sniff method on live interfaces. |
| `--noEtherframe` | off | Set when incoming packets (from a pcap file) do not contain the Ethernet frame. |
| `--noPromiscuous` | off | Prevents `dnsmonster` from putting `devName` into promiscuous mode. |

## Worker and channel sizing

| Flag | Default | Purpose |
| --- | --- | --- |
| `--packetHandlerCount` | `2` | Workers handling received packets. |
| `--packetChannelSize` | `1000` | Size of the packet handler channel. |
| `--tcpHandlerCount` | `1` | Routines handling TCP DNS packets. |
| `--tcpAssemblyChannelSize` | — | Goroutine channel size for the TCP assembler, which de-fragments incoming TCP packets without slowing down normal UDP packets. |
| `--tcpResultChannelSize` | `10000` | Size of the TCP result channel. |
| `--defraggerChannelSize` | `10000` | Size of the channel carrying raw packets to be de-fragmented. |
| `--defraggerChannelReturnSize` | `10000` | Size of the channel where de-fragmented packets are sent to the output queue. |
| `--afpacketBuffersizeMb` | `64` | Afpacket buffer size in MB. |

These flags are used in a variety of combinations. See [filters and masks](/inputs/filters-and-masks/)
and [input options](/inputs/input-options/) for detail, and [performance](/configuration/performance/)
for how to tune them under load.
