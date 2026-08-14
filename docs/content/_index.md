+++
title = "Getting started"
description = "dnsmonster is a passive DNS monitoring framework that captures DNS traffic from a live interface, a pcap file or a dnstap socket and ships it to 15+ backends at 200k+ queries per second."
weight = 1
+++

# Passive DNS monitoring

`dnsmonster` sniffs DNS traffic off a live interface, a pcap file or a dnstap socket, filters it,
masks it, and ships it to the backend you already run — 200,000+ queries per second on commodity
hardware, with no agents on your resolvers.

{{< stats >}}
200k+ | queries per second, single instance
15+ | output backends
3 | input types: interface, pcap, dnstap
1 | statically linked binary
{{< /stats >}}

[Install dnsmonster](/getting-started/installation/) · [Browse outputs](/outputs/) ·
[GitHub](https://github.com/mosajjal/dnsmonster)

## What dnsmonster is

`dnsmonster` is a passive DNS monitoring framework written in Go. It accepts traffic from a `pcap`
file, a network interface (802.1q, Ethernet, IP packet, VXLAN) or a `dnstap` socket, and indexes
hundreds of thousands of DNS queries per second. It aims to be scalable, simple and easy to use, and
to help security and operations teams gain visibility over DNS.

`dnsmonster` does not follow DNS conversations. It indexes DNS packets as soon as they arrive. It
also does not aim to breach the privacy of end users: Layer 3 IPs (IPv4 and IPv6) can be masked, so
teams can run trend analysis on aggregated data without being able to trace a query back to an
individual. There is a longer write-up on the design in the
[introduction blog post](https://blog.n0p.me/dnsmonster/).

{{< callout type="caution" title="Pre-1.0" >}}
Code before version 1.x is considered beta quality and is subject to breaking changes. Check the
release notes between tags for the list of breaking scenarios and how to mitigate potential data
loss.
{{< /callout >}}

## Install and run

The fastest way to see output is the container image. Raw packet capture needs elevated
capabilities, so the daemon must be granted `NET_RAW` and `NET_ADMIN`.

```sh
sudo docker run --rm -it --net=host \
  --cap-add NET_RAW --cap-add NET_ADMIN \
  --name dnsmonster ghcr.io/mosajjal/dnsmonster:latest \
  --devName lo --stdoutOutputType=1
```

Or read from a capture file:

```sh
dnsmonster --pcapFile=capture.pcap --stdoutOutputType=1
```

Or build it:

```sh
git clone https://github.com/mosajjal/dnsmonster --depth 1 /tmp/dnsmonster
cd /tmp/dnsmonster
go get
go build -o dnsmonster ./cmd/dnsmonster
```

Every run needs **one input** and **at least one output**. The example above uses the `lo` interface
as input and stdout as output. See [installation](/getting-started/installation/) for prebuilt
binaries, `deb`/`rpm` packages and static builds.

## Main features

- **Zero-copy capture** — Linux `afpacket` with zero-copy packet capture, plus BPF filter support
  for kernel-level filtering.
- **IP masking** — mask IPv4 and IPv6 source and destination addresses at process time to keep
  aggregate analysis privacy-preserving.
- **Skip and allow lists** — drop noisy FQDNs or log only the domains you care about, by prefix,
  suffix or exact match, with hot reload from a file or URL.
- **Pre-process sampling** — set a sampling ratio at capture time so an over-subscribed pipeline
  degrades predictably instead of dropping at random.
- **Modular outputs** — fan out to as many backends as you like, each with its own filtering logic
  and its own worker pool.
- **Full protocol coverage** — DNS over TCP, fragmented DNS over UDP and TCP, IPv6, and `dnstap`
  over a Unix socket or TCP.
- **Built-in metrics** — ship instance metrics to Prometheus or statsd, or print them to stderr.
- **Single binary** — ships as one statically linked binary. Configure it with CLI flags,
  environment variables or an INI file.
- **ClickHouse-native** — automatic retention through ClickHouse TTL, LZ4 compression, sampleable
  tables and a bundled Grafana dashboard.
- **SIEM integration** — first-class output modules for Splunk HEC and Microsoft Sentinel, plus
  OCSF-compatible JSON.

## Where to go next

- [Installation](/getting-started/installation/) — binaries, packages, containers and source builds.
- [Post-installation](/getting-started/post-installation/) — systemd units and shell completion.
- [Configuration](/configuration/) — flags, environment variables and the INI file, in precedence order.
- [Inputs and filters](/inputs/) — interface, pcap, pcap-over-IP and dnstap, plus every filter stage.
- [Outputs](/outputs/) — the full backend list and each module's parameters.
- [FAQ](/faq/) — why dnsmonster exists, how fast it is, and what to do when it drops packets.

## Get involved

`dnsmonster` is open source and contributions are welcome. Open an issue or a pull request on
[GitHub](https://github.com/mosajjal/dnsmonster), or start a thread in
[Discussions](https://github.com/mosajjal/dnsmonster/discussions) for roadmap conversations and
setup showcases.

{{< callout type="tip" title="Shaping a managed offering" >}}
We're exploring a managed SaaS version of dnsmonster. If you have opinions about what that should
look like, [take the survey](https://tally.so/r/2EAxBe).
{{< /callout >}}
