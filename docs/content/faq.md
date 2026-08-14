+++
title = "FAQ"
description = "Why dnsmonster exists, how fast it is, which output to pick, and what to do when it drops packets."
weight = 99
+++

## Do I need passive DNS capture?

DNS is one of the most prevalent indicators of compromise in all attacks. The vast majority of
external communication from malware or a backdoor — around 92% according to Cisco — involves DNS
somewhere in the chain. A few good reads on why DNS security monitoring matters:

- [The security holes that only DNS can plug — Infoblox](https://blogs.infoblox.com/security/the-security-holes-that-only-dns-can-plug/)
- [Passive DNS — Cisco](https://docs.umbrella.com/investigate/docs/passive-dns)
- [WatchTower blog](https://www.watchtowerhq.co/what-is-dns-monitoring-why-important/)
- [PagerDuty blog](https://www.pagerduty.com/resources/learn/dns-monitoring)

## Why dnsmonster specifically?

`dnsmonster` is one of the few products supporting a wide range of inputs — pcap file, dnstap, live
interface on Windows and \*nix, afpacket — and a variety of outputs, with minimum configuration and
maximum performance. It can send data natively to your database of choice or a Kafka topic, and has
built-in metrics reporting. The full feature set is in [getting started](/).

It also uses every CPU core available and has built-in buffers to cope with sudden traffic spikes.

## Why the name?

When `dnsmonster` was first tested on a giant DNS pcap file — 220+ million queries and responses —
it outperformed other products in the same category. Describing it to a friend, the phrase was that
it "devoured those packets like the cookie monster". That is where the monster came from.

## What OS should I run it on?

`dnsmonster` will always offer first-class support for the modern Linux kernel (4.x), so a modern
Linux distribution is recommended. It also compiles for Windows, \*BSD and macOS, but many of the
performance tweaks do not work as well there. On non-Unix systems, for example, `dnsmonster` stops
manipulating JSON objects with [sonic](https://github.com/bytedance/sonic).

It builds successfully for `arm7`, `aarch64` and `amd64`. No performance benchmark has been done to
determine which architecture works best.

## Why is dnsmonster not working for me?

There could be several reasons. The best way to start troubleshooting is to have a Go compiler handy
so you can build from source, then try the following:

- Build the main branch and run it with `stdoutOutput` to see whether there is any output at all.
- Try running with and without afpacket support, and with various buffer sizes.
- Use a different capture method to confirm the packets are visible at all — `tcpdump` and
  `netsniff-ng` are good for this.
- Pipe packets from `tcpdump` into `dnsmonster` and see whether that changes anything:
  `sudo tcpdump -nni eth0 -w - | dnsmonster --pcapFile=- --stdoutOutputType=1`
- Check the `port` variable if your DNS packets are on a port other than 53. That parameter is
  separate from BPF — and while you are there, make sure your BPF is not too restrictive.

If none of that works, open an issue with the details. If you plan to attach a `pcap` file, be sure
to [anonymise it](https://isc.sans.edu/forums/diary/Truncating+Payloads+and+Anonymizing+PCAP+files/23990/)
first.

## How do I upgrade between versions?

Before 1.x.x, breaking changes between releases are expected. Read the release notes between your
current version and the target one by one, to see whether you need to upgrade in increments.

After 1.x.x the plan is to maintain backwards compatibility within major versions — every 1.x.x
installation will work as part of an upgrade. That will not necessarily hold for ClickHouse tables:
ClickHouse moves fast, so the schema may need to change regardless of `dnsmonster`'s major release.

The JSON output fields, which are the basis for most `dnsmonster` outputs, are bound to Miek
Gieben's [dns library](https://github.com/miekg/dns). That library has been stable and used the same
data structure for years. The plan is to keep the JSON schema stable within each major release so
SIEM parsers such as ASIM and CIM keep working. `dnsmonster` also supports `go-template` output,
similar to `kubectl`, which makes it easy to standardise the output to your own needs.

## How fast is it?

`dnsmonster` has [demonstrated](https://n0p.me/2020/02/2020-02-05-dnsmonster/) 200,000 packets per
second on a beefy server, with ClickHouse running on the same machine over SSD storage. Performance
for both packet ingestion and the output pipeline has improved since then, to the point where you
can ingest the same rate on a commodity laptop. For the majority of use cases, `dnsmonster` will not
be the bottleneck in your data collection.

If you have a heavy workload you have tested with `dnsmonster`, feedback is welcome and the numbers
are worth sharing with the community.

## Which output should I use?

It depends. Sticking with the toolset you already have is usually right. Most organisations have
built a `syslog` or `kafka` pipeline to get data into their ingestion point, and both are fully
supported. To test the product and its output quickly, `file` and `stdout` are the easiest — keep
disk IO in mind for `file` if you are writing a lot of data.

If you are building something new from scratch, look at ClickHouse. `dnsmonster` was originally
built with ClickHouse in mind, and it remains one of the better tools for ingesting DNS logs. See how
Cloudflare uses ClickHouse to monitor 1.1.1.1
[here](https://blog.cloudflare.com/how-cloudflare-analyzes-1m-dns-queries-per-second/).

## Why am I dropping packets?

There are many possible reasons. Several of them, with fixes, are covered in the
[performance section](/configuration/performance/).

## Is there a Slack or Discord?

Not yet. The repository's
[discussions](https://github.com/mosajjal/dnsmonster/discussions) exist for this purpose. If that
proves less than ideal, a dedicated Discord/Slack/Telegram channel is on the table — let us know.

## How do I contribute?

Contribution splits into a few categories:

- **Security and bug disclosure** — see `SECURITY.md` in the main repository for how to report
  vulnerabilities responsibly.
- **Bugfixes** — open an issue before submitting a PR. That way other contributors know what is
  being worked on, and there is less duplicate effort. Sometimes a bugfix is specific to one
  deployment and there is a mitigation that does not require a code change.
- **New features and output modules** — raise an issue first, so the work can be assigned and
  scheduled for the next major release, and requirements can be agreed in discussion.

There are also many `//todo` comments in the code — feel free to take a stab at those.

Last but not least, this documentation needs your help too. Every page has an edit link in the
footer that takes you straight to the source.
