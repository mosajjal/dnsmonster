---
title: "Stdout, syslog or Log File"
linkTitle: "Stdout, syslog, or Log File"
weight: 4
---

Stdout, syslog and file are supported outputs for `dnsmonster` out of the box. They are useful specially if you have a SIEM agent reading the files as they come in. Note that `dnsmonster` does not provide support for log rotation and the capacity of the hard drive while writing into a file. You can use a tool like `logrotate` to perform cleanups on the log files. The signalling on log rotation (SIGHUP) has not been tested with `dnsmonster`. 

Currently, Syslog output is only supported on Linux.

## Configuration parameters

```ini
[file_output]
; What should be written to file. options:
;	0: Disable Output
;	1: Enable Output without any filters
;	2: Enable Output and apply skipdomains logic
;	3: Enable Output and apply allowdomains logic
;	4: Enable Output and apply both skip and allow domains logic
FileOutputType = 0

; Path to output file. Used if fileOutputType is not none
FileOutputPath =

; Output format for file. options:json,csv, csv_no_header, gotemplate. note that the csv splits the datetime format into multiple fields
FileOutputFormat = json

; Go Template to format the output as needed
FileOutputGoTemplate = {{.}}

[stdout_output]
; What should be written to stdout. options:
;	0: Disable Output
;	1: Enable Output without any filters
;	2: Enable Output and apply skipdomains logic
;	3: Enable Output and apply allowdomains logic
;	4: Enable Output and apply both skip and allow domains logic
StdoutOutputType = 0

; Output format for stdout. options:json,csv, csv_no_header, gotemplate. note that the csv splits the datetime format into multiple fields
StdoutOutputFormat = json

; Go Template to format the output as needed
StdoutOutputGoTemplate = {{.}}

; Number of workers
StdoutOutputWorkerCount = 8

[syslog_output]
; What should be written to Syslog server. options:
;	0: Disable Output
;	1: Enable Output without any filters
;	2: Enable Output and apply skipdomains logic
;	3: Enable Output and apply allowdomains logic
;	4: Enable Output and apply both skip and allow domains logic
SyslogOutputType = 0

; Syslog endpoint address, example: udp://127.0.0.1:514, tcp://127.0.0.1:514. Used if syslogOutputType is not none
SyslogOutputEndpoint = udp://127.0.0.1:514
```
