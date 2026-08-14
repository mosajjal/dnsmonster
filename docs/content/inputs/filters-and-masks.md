+++
title = "Filters and masks"
description = "Every stage where dnsmonster can drop, sample, mask or de-duplicate packets — from kernel BPF down to output-level domain lists."
weight = 3
+++

There are several ways to manipulate incoming packets at various steps of the `dnsmonster` pipeline.
They operate at different levels of the stack and have very different performance implications — the
earlier a packet is dropped, the cheaper it is.

| Filter | Applied at |
| --- | --- |
| [BPF](#bpf) | kernel level |
| [Sample ratio](#sample-ratio) | capture level |
| [De-duplication](#de-duplication) | capture level |
| [Port](#port) | early process level |
| [IP masks](#ip-masks) | process level |
| [Allow and skip domain lists](#allow-and-skip-domain-lists) | output level |

## BPF

*Applied at kernel level.*

BPF is by far the most performant way to filter incoming packets. It is only supported on live
capture (`--devName`). It uses `tcpdump`'s
[pcap-filter](https://www.tcpdump.org/manpages/pcap-filter.7.html) language.

## Sample ratio

*Applied at capture level.*

`--sampleRatio` reduces the number of packets pushed into the pipeline purely by count. The default
is `1:1`, meaning one packet is processed for every packet that arrives. Lower it if your hardware
cannot process everything, or if the output is not keeping up. See the
[performance guide](/configuration/performance/#sampling-and-bpf-based-traffic-splitting) for detail.

## De-duplication

*Applied at capture level.*

The experimental `--dedup` feature provides rudimentary packet de-duplication. It runs a
non-cryptographic hash ([FNV-1](https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function))
over raw packets and builds a hash table as they arrive. The hash is computed *before* stripping the
`802.1q`, `vxlan` and `ethernet` layers, so de-duplication happens purely on packet bytes.

`--dedupCleanupInterval` sets the cleanup time for that hash table. Around cleanup time you may see
a few duplicate packets, since the table is not time-bound on its own — it is flushed completely at
each interval.

Applied after sample ratio, per packet.

## Port

*Applied at early process level.*

`--port` filters on the port of each packet. Since the vast majority of DNS is served on port 53,
this parameter should have no effect by default. It is not applied to fragmented packets.

## IP masks

*Applied at process level.*

While processing packets, source and destination IPv4 and IPv6 addresses can be masked by a
specified number of bits (`--maskSize4` and `--maskSize6`). Because this happens after
de-duplication, the output can contain entries that look duplicated purely because their IP prefixes
now match.

## Allow and skip domain lists

*Applied at output level.*

These two filters give you an allowlist and a skip list for domain output.

`--skipDomainsFile` avoids writing noisy, repetitive data to your output. The skip domain list is a
CSV file (or a URL serving one) with two columns: a string representing part or all of an FQDN, and
the matching logic for it. Three logics are supported: `prefix`, `suffix` and `fqdn`. `prefix` and
`suffix` skip domains starting or ending with the string. Because matching runs against DNS
questions, your string will usually need a trailing `.` — see `skipdomains.csv.sample` in the
repository. A full `fqdn` match is the way to drop a single highly noisy name.

`--allowDomainsFile` is the exact opposite: output is limited to entries in that list.

Both files have an automatic refresh interval and re-fetch their FQDNs on
`--skipDomainsRefreshInterval` and `--allowDomainsRefreshInterval`.

For each output type you can choose which of these tables apply. See the
[outputs section](/outputs/) for the output type modes.
