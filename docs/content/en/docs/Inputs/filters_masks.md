---
title: "Filters and masks"
linkTitle: "Filters and masks"
weight: 4
---

There are a few ways to manipulate incoming packets in various steps of `dnsmonster` pipeline. They operate in different levels of stack and have different performance implications.

## BPF
{{< alert >}}Applied at kernel level{{< /alert >}} 

BPF is by far the most performant way to filter incoming packets. It's only supported on live capture (`--devName`). It uses the `tcpdump`'s [pcap-filter](https://www.tcpdump.org/manpages/pcap-filter.7.html) language to filter out the packets. There are plans to potentially move away from this method and accept base64-encoded `bpf` bytecode in the future. 

## Sample Ratio
{{< alert >}}Applied at capture level{{< /alert >}} 

Sample ratio (`--sampleRatio`) is an easy way to reduce the number of packets being pushed to the pipeline purely by numbers. the default value is 1:1 meaning for each 1 incoming packet, 1 gets pushed to the pipeline. you can change that if you have a huge number of packets or your output is not catching up with the input. Checkout [performance guide](../../configuration/performance#sampling-and-bpf-based-split-of-traffic) for more detail. 


## De-duplication

{{< alert >}}Applied at capture level{{< /alert >}} 

The experimental de-duplication (`--dedup`) feature is implemented to provide a rudimentary packet de-duplication capability. The functionality of de-duplication is very simple. It uses a non-cryptography hashing function ([FNV-1](https://en.wikipedia.org/wiki/Fowler%E2%80%93Noll%E2%80%93Vo_hash_function)) on the raw packets and generates a hash table of incoming packets as the come in. Note that the hashing function happens before stripping `802.1q`, `vxlan`, `ethernet` layers so the de-duplication happens purely on the packet bytes. 

There's also the option `--dedupCleanupInterval` to specify cleanup time for the hash table. around the time of cleanup, there could be a few duplicate packets since the hash table is not time-bound on its own. It gets flushed completely at the interval. 

Applied after Sample Ratio for each packet.

## Port
{{< alert >}}Applied at early process level{{< /alert >}} 

There's an additional filter specifying the port (`--port`) of each packet. since the vast majority of the DNS packets are served out of port 53, this parameter shouldn't have any effect by default. note that this filter will not be applied to fragmented packets.

## IP Masks
{{< alert >}}Applied at process level{{< /alert >}} 

While processing the packets, the source and destination IPv4 and IPv6 packets can be masked by a specified number of bytes (`--maskSize4` and `--maskSize6` options). Since this step happens after de-duplication, there could be seemingly duplicate entries in the output purely because of the fact that IP prefixes appear the same.   

## Allow and Skip Domain list
{{< alert >}}Applied at output level{{< /alert >}} 

These two filters specify an allowlist and a skip list for the domain outputs. `--skipDomainsFile` is used to avoid writing noisy, repetitive data to your Output. The skip domain list is a csv-formatted file (or a URL containing the file), with only two columns: a string representing part or all of a FQDN, and a logic for that particular string. `dnsmonster` supports three logics for each entry: `prefix`, `suffix` and `fqdn`. `prefix` and `suffix` means that only the domains starting/ending with the mentioned string will be skipped from being sent to output. Note that since the process is being done on DNS questions, your string will most likely have a trailing `.` that needs to be included in your skip list row as well (take a look at [skipdomains.csv.sample](skipdomains.csv.sample) for a better view). You can also have a full FQDN match to avoid writing highly noisy FQDNs into your database.


`--allowDomainsFile` provides the exact opposite of skip domain logic, meaning your output will be limited to the entries inside this list. 

both `--skipDomainsFile` and `--allowDomainsFile` have an automatic refresh interval and re-fetch the FQDNs using `--skipDomainsRefreshInterval`and `--allowDomainsRefreshInterval` options.

For each output type, you can specify which of these tables are used. Check the output section for more detail regarding the output modes. 