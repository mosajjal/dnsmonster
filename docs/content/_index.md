+++
title = "Getting started"
weight = 1

[cascade]
  type = "docs"
+++

DNSMonster is a Passive DNS monitoring framework written in Golang. It can accept traffic from a
`pcap` file, a network interface (802.1q, Ethernet, IP Packet, VXLAN) or a dnstap socket, and can be
used to index and store hundreds of thousands of DNS queries per second. It aims to be scalable,
simple and easy to use, and to help security and operation teams to gain visibility over DNS.

`dnsmonster` does not look to follow DNS conversations, rather it aims to index DNS packets as soon
as they come in. It also does not aim to breach the privacy of the end-users, with the ability to
mask Layer 3 IPs (IPv4 and IPv6), enabling teams to perform trend analysis on aggregated data
without being able to trace back the queries to an individual.
[Blogpost](https://blog.n0p.me/dnsmonster/)

{{< callout type="warning" >}}
The code before version 1.x is considered beta quality and is subject to breaking changes. Please
check the release notes for each tag to see the list of breaking scenarios between each release, and
how to mitigate potential data loss.
{{< /callout >}}

## Main features

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
- Ability to be shipped as a single, statically linked binary
- Ability to be configured using environment variables, command line options or configuration file
- Ability to sample outputs using ClickHouse's SAMPLE capability
- Ability to send metrics using `prometheus` and `statstd`
- High compression ratio thanks to ClickHouse's built-in LZ4 storage
- Supports DNS Over TCP, Fragmented DNS (udp/tcp) and IPv6
- Supports [dnstrap](https://github.com/dnstap/golang-dnstap) over Unix socket or TCP
- built-in SIEM integration with Splunk and Microsoft Sentinel

## Install and run

The container image is the quickest way to see output. Raw packet capture needs elevated
capabilities, so the daemon must be granted `NET_RAW` and `NET_ADMIN`.

```sh
sudo docker run --rm -it --net=host \
  --cap-add NET_RAW --cap-add NET_ADMIN \
  --name dnsmonster ghcr.io/mosajjal/dnsmonster:latest \
  --devName lo --stdoutOutputType=1
```

To read from a capture file instead:

```sh
dnsmonster --pcapFile=capture.pcap --stdoutOutputType=1
```

One input and at least one output must be defined. See [installation](/getting-started/installation/)
for prebuilt binaries, `deb`/`rpm` packages and source builds.

## Where to go next

- [Installation](/getting-started/installation/)
- [Post-installation](/getting-started/post-installation/)
- [Configuration](/configuration/)
- [Inputs and filters](/inputs/)
- [Outputs](/outputs/)
- [FAQ](/faq/)

## Contributions welcome

Open an Issue or a [Pull Request](https://github.com/mosajjal/dnsmonster/pulls) on
[GitHub](https://github.com/mosajjal/dnsmonster). New users are always welcome. For announcements,
roadmap discussion and setup showcases,
[Discussions](https://github.com/mosajjal/dnsmonster/discussions) is the best place to start.
