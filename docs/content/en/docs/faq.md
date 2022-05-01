---
title: "FAQ"
linkTitle: "FAQ"
weight: 20
menu:
  main:
    weight: 20
---

# Why should I use dnsmonster
I've broken this question into two. Why do you need to monitor your DNS, and why is dnsmonster a good choice to do so.

## Do I need passive DNS capture capability
DNS is one of, if not the most prevalent indicators of compromise in all attacks. The vast majority of external communication of a malware or a backdoor (~92% according to Cisco) have some sort of DNS connectivity in their chain. Here are some great articles around the importance of DNS Security monitoring

- [The Security Holes That Only DNS Can Plug - InfoBlox](https://blogs.infoblox.com/security/the-security-holes-that-only-dns-can-plug/)
- [Passive DNS - Cisco](https://docs.umbrella.com/investigate/docs/passive-dns)
- [WatchTower Blog](https://www.watchtowerhq.co/what-is-dns-monitoring-why-important/)
- [PagerDuty Blog](https://www.pagerduty.com/resources/learn/dns-monitoring)

## Why dnsmonster specifically?

`dnsmonster` is one of the few products supporting a wide range of inputs (pcap file, dnstap, live interface on Windows and *Nix, afpacket) and a variety of outputs with minimum configuration and maximum performance. It can natively send data to your favorite Database service or a Kafka topic, and has a builtin capability of sending its metrics to a metrics endpoint. Check out the full feature set of monster in the "Getting Started" section. 

In addition, `dnsmonster` also offers a fantastic performance by utilizing all CPU cores available on the machine, and has builtin buffers to cope with sudden traffic spikes.

## Why did you name it dnsmonster

When I first tested `dnsmonster` on a giant DNS pcap file (220+ Million DNS Queries and responses) and saw it outperform other products in the same category, I described it to one of my mates that it "devoured those packets like the cookie monster" and that's how the monster within dnsmonster was born

## What OS should I use to run DNSmonster

`dnsmonster` will always offer first-class support for the modern Linux kernel (4.x) so it is recommended that you use dnsmonster on a modern Linux distribution. It can also be compiled for Windows, *BSD and Mac OS, but many of the performance tweaks will not work as well as they do for Linux. 

For example, when `dnsmonster` is build on non-Unix systems, it stops manipulating the JSON objects with [sonic](https://github.com/bytedance/sonic).

As for architecture, `dnsmonster` builds successfully against `arm7`, `aarch64` and `amd64`, but the performance benchmark has not been done to determine which architecture works best with it. 

## Why is dnsmonster is not working for me

There could be several reasons behind why `dnsmonster` is not working. The best way to start troubleshooting dnsmonster is to have a Go compiler handy so you can build dnsmonster from the source and try the following:

- Try building the master branch and run it with stdoutOutput to see if there is any output
- Try running `dnsmonster` with or without afpacket support and various buffer sizes
- Use a different packet capture method than `dnsmonster` to see if the packets are visible (`tcpdump` and `netsniff-ng` are good examples)
- Try piping the packets from `tcpdump` to `dnsmonster` with stdoutOutput and see if that makes any difference. like so:
  - `sudo tcpdump -nni eth0 -w - | dnsmonster --pcapFile=- --stdoutOutputType=1`
- Pay attention to the `port` variable if your DNS packets are being sent on a different port than 53. That parameter is different than BPF. Speaking of which, make sure your BPF is not too restrictive.

If none of the above works, feel free to open an issue with the details of your problem. If you are planning to attach a `pcap` file as part of your issue, make sure to [anonymize it](https://isc.sans.edu/forums/diary/Truncating+Payloads+and+Anonymizing+PCAP+files/23990/)

## How do I upgrade between the version

Before the product hits 1.x.x, breaking changes between each release is expected. Read the release note between your current version and desired version one by one to see if you need to upgrade in increments or not. 

After 1.x.x, the plan is to maintain backwards compatibility in major versions (eg every 1.x.x installation will work as part of an upgrade). However, that will not necessarily be the case for ClickHouse tables. Since ClickHouse is a fast moving product, there might be a need to change the schema of the tables regardless of `dnsmonster`'s major release. 

The JSON output fields, which is the basis for the majority of `dnsmonster` outputs, is bound to Miekg's [dns library](https://github.com/miekg/dns). The library seems to be fairly stable and have used the same data structure for years. For `dnsmonster`, the plan is to maintain the JSON schema the same for each major release so SIEM parsers such as ASIM and CIM can maintain functionality. `dnsmonster` also supports `go-template` output similar to `kubectl` and makes it easy to customize and standardize your output to cater for your needs.

## How fast is dnsmonster

`dnsmonster` have [demonstrated](https://n0p.me/2020/02/2020-02-05-dnsmonster/) 200,000 packets per second ingestion on a beefy server with ClickHouse being run on the same machine with SSD storage backend. Since then, the performance of `dnsmonster` for both packet ingestion and output pipeline have been improved, to the point that you can ingest the same number of packets per second on a commodity laptop. I would say for the majority of use cases, `dnsmonster` will not be the bottleneck of your data collection. 

If you have a heavy workload that you have tested with `dnsmonster`, I would be happy to receive your feedback and share the numbers with the community

## Which output do I use

Depends. I would recommend sticking with the current toolset you have. Majority of organizations have built a `syslog` or `kafka` pipeline to get the data into the ingestion point, and both are fully supported by `dnsmonster`. If you want to test the product and its output, you can use `file` and `stdout` quite easily. Keep in mind that for `file`, you should consider your disk IO if you're writing a ton of data into disk.

If you're keen to build a new solution from scratch, I would recommend looking at ClickHouse. `dnsmonster`was originally built with ClickHouse in mind, and ClickHouse remains one of the better tools to ingest DNS logs. Take a look at how CloudFlare is leveraging ClickHouse to monitor 1.1.1.1 [here](https://blog.cloudflare.com/how-cloudflare-analyzes-1m-dns-queries-per-second/)

## Why am I dropping packets

There could be many reasons behind packet loss. I went through some of them with possible solutions in the [performance](/docs/configuration/performance) section.

## Is there a Slack or Discord I can join

Not yet. At the moment, the repo's [discussions](https://github.com/mosajjal/dnsmonster/discussions) is created for this purpose. If that proves to be less than ideal, I'm open to have a Discord/Slack/Telegram channel dedicated to `dnsmonster`. Let me know!

## How to contribute

I have broken contribution into different sections

- For security and bug disclosure, please visit `SECURITY.md` in the main repository to get more info on how to report vulnerabilities responsibly
- For bugfixes, please open an issue first, before submitting a PR. That way the other contributors know what is being worked on and there will be less duplicate work on bugfixes. Also, sometimes the bugfixes are more related to a particular client and there could be other mitigation other than changing the code
- For new features and Output modules, please raise an issue first. That way the work can be assigned and timeline-d for the next major release and we can get together in the discussions to set the requirements

There are also many `//todo` comments in the code, feel free to take a stab at those.

Last but not least, this very documentation needs your help! On the right hand side of each page, you can see some helper links to raise an issue with a page, or propose an edit or even create a new page. 
