---
title: "Outputs"
linkTitle: "Outputs"
weight: 4
description: >
  Set up output(s) and gather metrics
---

`dnsmonster` follows a pipeline architecture for each individual packet. After the Capture and filter, each processed packet arrives at the output dispatcher. The dispatcher sends a copy of the output to each individual output module that have been configured to produce output. For instance, if you specify `stdoutOutputType=1` and `--fileOutputType=1 --fileOutputPath=/dev/stdout`, you'll see each processed output twice in your stdout. One coming from the stdout output type, and the other from the file output type which happens to have the same address (`/dev/stdout`).  

In general, each output has its own configuration section. You can see the sections with "_output" suffix when running `dnsmonster --help` from the command line. The most important parameter for each output is their "Type". Each output has 5 different types:

- Type 0:
- Type 1: An output module configured as Type 1 will ignore "SkipDomains" and "AllowDomains" and will generate output for all the incoming processed packets. Note that the output types does *not* nullify input filters since it is applied after capture and early packet filters. Take a look at [Filters and Masks](/docs/inputs/filters_masks) to see the order of the filters applied.  
- Type 2: An output module configured as Type 2 will ignore "AllowDomains" and only applies the "SkipDmains" logic to the incoming processed packets.
- Type 3: An output module configured as Type 3 will ignore "SkipDmains" and only applies the "AllowDomains" logic to the incoming processed packets.
- Type 4: An output module configured as Type 4 will apply both "SkipDmains" and "AllowDomains" logic to the incoming processed packets.

Other than `Type`, each output module may require additional configuration parameters. For more information, refer to each module's documentation.

## Output Formats

`dnsmonster` supports multiple output formats:

- `json`: the standard JSON output. The output looks like below sample
```json
{"Timestamp":"2020-08-08T00:19:42.567768Z","DNS":{"Id":54443,"Response":true,"Opcode":0,"Authoritative":false,"Truncated":false,"RecursionDesired":true,"RecursionAvailable":true,"Zero":false,"AuthenticatedData":false,"CheckingDisabled":false,"Rcode":0,"Question":[{"Name":"imap.gmail.com.","Qtype":1,"Qclass":1}],"Answer":[{"Hdr":{"Name":"imap.gmail.com.","Rrtype":1,"Class":1,"Ttl":242,"Rdlength":4},"A":"172.217.194.108"},{"Hdr":{"Name":"imap.gmail.com.","Rrtype":1,"Class":1,"Ttl":242,"Rdlength":4},"A":"172.217.194.109"}],"Ns":null,"Extra":null},"IPVersion":4,"SrcIP":"1.1.1.1","DstIP":"2.2.2.2","Protocol":"udp","PacketLength":64}
```
- `csv`: the CSV output. The fields and headers are non-customizable at the moment. to get a custom output, please look at `gotemplate`.
```csv
Year,Month,Day,Hour,Minute,Second,Ns,Server,IpVersion,SrcIP,DstIP,Protocol,Qr,OpCode,Class,Type,ResponseCode,Question,Size,Edns0Present,DoBit,Id
2020,8,8,0,19,42,567768000,default,4,2050551041,2050598324,17,1,0,1,1,0,imap.gmail.com.,64,0,0,54443
```
- `csv_no_headers`: Looks exactly like the CSV but with no header print at the beginning
- `gotemplate`: Customizable template to come up with your own formatting. let's look at a few examples with the same packet we've looked at using JSON and CSV

```sh
$ dnsmonster --pcapFile input.pcap --stdoutOutputType=1 --stdoutOutputFormat=gotemplate --stdoutOutputGoTemplate="timestamp=\"{{.Timestamp}}\" id={{.DNS.Id}} question={{(index .DNS.Question 0).Name}}"
timestamp="2020-08-08 00:19:42.567735 +0000 UTC" id=54443 question=imap.gmail.com.
```

Take a look at the [official docs](https://pkg.go.dev/text/template) for more info regarding text/template and your various options.