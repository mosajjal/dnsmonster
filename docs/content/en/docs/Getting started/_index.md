---
title: "Getting Started"
linkTitle: "Getting Started"
weight: 1
description: >
  Getting Started with dnsmonster
---

Passive DNS monitoring framework built on Golang. 
`dnsmonster` implements a packet sniffer for DNS traffic. It Ability to accept traffic from a `pcap` file, a live interface or a `dnstap` socket, 
and Ability to be used to index and store hundreds of thousands of DNS queries per second as it has shown to be capable of indexing 200k+ DNS queries per second on a commodity computer. It aims to be scalable, simple and easy to use, and help
security teams to understand the details about an enterprise's DNS traffic. `dnsmonster` does not look to follow DNS conversations, rather it aims to index DNS packets as soon as they come in. It also does not aim to breach
the privacy of the end-users, with the ability to mask Layer 3 IPs (IPv4 and IPv6), enabling teams to perform trend analysis on aggregated data without being able to trace back the queries to an individual. [Blogpost](https://blog.n0p.me/dnsmonster/)


{{% alert title="Warning" color="warning" %}}
The code before version 1.x is considered beta quality and is subject to breaking changes. Please check the release notes for each tag to see the list of breaking scenarios between each release, and how to mitigate potential data loss.
{{% /alert %}}

# Main features

- Ability to use Linux's `afpacket` and zero-copy packet capture.
- Supports BPF
- Ability to mask IP address to enhance privacy
- Ability to have a pre-processing sampling ratio
- Ability to have a list of "skip" `fqdn`s to avoid writing some domains/suffix/prefix to storage
- Ability to have a list of "allow" domains, used to log access to certain domains
- Hot-reload of skip and allow domain files/urls
- Modular output with configurable logic per output stream.
- Automatic data retention policy using ClickHouse's TTL attribute
- Built-in Grafana dashboard for ClickHouse output.
- Ability to be shipped as a single, statically-linked binary
- Ability to be configured using environment variables, command line options or configuration file
- Ability to sample outputs using ClickHouse's SAMPLE capability
- Ability to send metrics using `prometheus` and `statstd`
- High compression ratio thanks to ClickHouse's built-in LZ4 storage
- Supports DNS Over TCP, Fragmented DNS (udp/tcp) and IPv6
- Supports [dnstrap](https://github.com/dnstap/golang-dnstap) over Unix socket or TCP
- built-in SIEM integration with Splunk and Microsoft Sentinel