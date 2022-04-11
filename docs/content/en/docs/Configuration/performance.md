---
title: "Performance"
linkTitle: "Performance"
weight: 3
description: >
  Performance considerations when configuring dnsmonster
---

## Use afpacket

If you're using dnsmonster as a sniffer, and you're not keeping up with the number of packets that are coming in, consider switching on afpacket by using the flag `--useAfpacket`. Afpacket tends to drastically improve packet ingestion rate of dnsmonster. If you still having packet drop issues, increase `--afpacketBuffersizeMb` to a higher value. the buffer size will take up more memory on startup, and will increase the startup time depending how much have you assigned to it.

In some tests, values above 4096MB tend to have negative impact on the overall performance of the daemon. If you're using 4096MB of buffer size and still seeing performance issues, There's a good chance the issue isn't on the capture size, and more on the process and output side. 

## Proper Output and packet handlers

Simply put, if you have an output that can accept 1000 inserts per second, but you have an incoming packet rate of 10,000 packets per second, you're going to see a lot of packet drops. The packet drop will get worse and worse as time goes by as well. When selecting an output, consider the capacity of your technology and what you expect to be ingested.

If you are seeing a considerable amount of packet loss which gets worse as time goes on, consider testing `--stdoutOutputType=1` and remove your current output, and redirect the output to `/dev/null`. You can also tweak the number of workers converting the data to JSON to further experiment with it. Take the following example

```sh
dnsmonster --devName=lo --packetHandlerCount=16 --stdoutOutputType=1 --useAfpacket | pv --rate --line-mode > /dev/null
```

In above command, you can see the exact output line per second while maintaining a view on metrics and packet loss to see if your packet loss is still present. by default, `--stdoutOutputWorkerCount` is set to 8. If you have a strong enough CPU, you can increase that amount to see what's the max rate you can achieve. On a small server, you shouldn't have a problem ingesting 500k packet per second.

Note that the `--packetHandlerCount` is also set to 16 to make sure enough workers are ingesting packets coming in. That's also an important parameter to tweak to achieve the optimum performance. The default, `2`, might be too low for you if you have hundreds of thousands of packets per second on an interface.

## Sampling and BPF-based split of traffic

Sometimes, the packets are simply too much to process. `dnsmonster` offers a few options to deal with this problem. `--sampleRatio` simply ignores packets by the defined ratio. default is `1:1`, meaning for each incoming packet, one gets processed, aka 100%. you can tweak this number if your hardware isn't capable of processing all the packets, or `dnsmonster` has simply reached its limit. 

For example, putting `2:7` as your sample ratio means for each 7 packets that come in, only the first two get processed. 

If after testing all options you've reached the conclusion that `dnsmonster` can not handle more than what you need it to do, please raise an issue about it, but also you can run multiple instances of `dnsmonster` looking at the same traffic like so:

```sh
dnsmonster --devName=lo --stdoutOutputType=1 --filter="src portrange 1024-32000"
dnsmonster --devName=lo --stdoutOutputType=1 --filter="src portrange 32001-65535"
```

The above processes will split the traffic between them based on the port range. Note that only high ports are included since majority of the clients use ports above 1024 to conduct a DNS query. you can change the filter based on any BPF that makes sense for your environment.

## Profile CPU and Memory

To take a look at what exactly is using your CPU and RAM, take a look at the Golang profiler tools available through `--memprofile` and `--cpuprofile` flags. to use them, issue the following

```sh
# profile CPU
dnsmonster --devName=lo --stdoutOutputType=1 --cpuprofile=1

# you'll see something like this in the beginning of your logs
# 2022/04/11 19:13:51 profile: cpu profiling enabled, /tmp/profile452510705/cpu.pprof

# profile RAM
dnsmonster --devName=lo --stdoutOutputType=1 --memprofile=1

# you'll see something like this in the beginning of your logs
# 2022/04/11 19:15:00 profile: memory profiling enabled (rate 4096), /tmp/profile1290716652/mem.pprof
```

After `dnsmonster` exits gracefully, you can use Go's perf tools to open the generated `pprof` file in a browser and dig deep into functions that are being bottleneck in the code. After installing `pprof`, use it like below

```sh
~/go/bin/pprof -http 127.0.0.1:8882 /tmp/profile2392236212/mem.pprof
```

A browser session will automatically open with the performance metrics for your execution. 

