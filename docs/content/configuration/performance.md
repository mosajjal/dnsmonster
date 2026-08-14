+++
title = "Performance"
description = "Tuning dnsmonster — afpacket, worker counts, sampling, BPF-based traffic splitting, and CPU/memory profiling."
weight = 2
+++

## Use afpacket

If you are using `dnsmonster` as a sniffer and not keeping up with incoming packets, switch on
afpacket with `--useAfpacket`. Afpacket tends to drastically improve ingestion rate. If you still see
packet drops, raise `--afpacketBuffersizeMb`. A larger buffer takes more memory at startup and
increases startup time proportionally.

In testing, values above 4096 MB tend to hurt overall daemon performance. If you are at 4096 MB and
still seeing problems, the bottleneck is probably not capture — look at the process and output side
instead.

## Match output capacity to packet rate

If your output accepts 1,000 inserts per second and packets arrive at 10,000 per second, you will
drop packets, and the drop rate will get worse over time. When picking an output, consider the
capacity of the technology behind it against what you expect to ingest.

To isolate whether the output is the problem, test with `--stdoutOutputType=1`, drop your real
output, and redirect to `/dev/null`:

```sh
dnsmonster --devName=lo --packetHandlerCount=16 --stdoutOutputType=1 --useAfpacket | pv --rate --line-mode > /dev/null
```

That gives you output lines per second while keeping metrics and packet loss visible. By default
`--stdoutOutputWorkerCount` is 8; on a strong CPU you can raise it to find your ceiling. On a small
server you should have no trouble ingesting 500k packets per second.

`--packetHandlerCount` is set to 16 above to make sure enough workers are consuming incoming
packets. It is an important parameter to tune — the default of `2` is likely too low if you have
hundreds of thousands of packets per second on an interface.

## Sampling and BPF-based traffic splitting

Sometimes there are simply too many packets. `--sampleRatio` ignores packets by ratio. The default
is `1:1`, meaning every incoming packet is processed. Setting `2:7` means that for every 7 packets
that arrive, only the first two get processed.

If you conclude that a single `dnsmonster` cannot handle your load, please raise an issue — and in
the meantime, run multiple instances against the same traffic, split by BPF:

```sh
dnsmonster --devName=lo --stdoutOutputType=1 --filter="src portrange 1024-32000"
dnsmonster --devName=lo --stdoutOutputType=1 --filter="src portrange 32001-65535"
```

These two processes split traffic by port range. Only high ports are included, since the majority of
clients use ports above 1024 for DNS queries. Adapt the filter to whatever BPF makes sense in your
environment.

## Profile CPU and memory

To see exactly what is using CPU and RAM, use the Go profiler hooks behind the `--cpuprofile` and
`--memprofile` flags:

```sh
# profile CPU
dnsmonster --devName=lo --stdoutOutputType=1 --cpuprofile=1

# you'll see something like this at the beginning of your logs
# 2022/04/11 19:13:51 profile: cpu profiling enabled, /tmp/profile452510705/cpu.pprof

# profile RAM
dnsmonster --devName=lo --stdoutOutputType=1 --memprofile=1

# you'll see something like this at the beginning of your logs
# 2022/04/11 19:15:00 profile: memory profiling enabled (rate 4096), /tmp/profile1290716652/mem.pprof
```

After `dnsmonster` exits gracefully, open the generated `pprof` file in a browser with Go's perf
tooling and dig into the functions that are bottlenecking:

```sh
~/go/bin/pprof -http 127.0.0.1:8882 /tmp/profile2392236212/mem.pprof
```

A browser session opens automatically with the performance metrics for that run.
