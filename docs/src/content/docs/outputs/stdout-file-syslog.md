---
title: Stdout, file and syslog
description: Plain outputs for SIEM agents — stdout, a log file, or a syslog endpoint, with JSON, OCSF, CSV and Go template formats.
sidebar:
  label: Stdout, file, syslog
  order: 12
---

Stdout, syslog and file are supported out of the box. They are useful when a SIEM agent reads files
as they arrive.

`dnsmonster` does not handle log rotation or disk capacity while writing to a file — use something
like `logrotate` for cleanup. Rotation signalling (`SIGHUP`) has not been tested with `dnsmonster`.

The JSON schema used for these outputs can be configured to be compatible with the Open Cybersecurity
Schema Framework (OCSF).

Syslog output is currently only supported on Linux.

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

; Output format for file. options:json, json-ocsf, csv, csv_no_header, gotemplate. note that the csv splits the datetime format into multiple fields
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
