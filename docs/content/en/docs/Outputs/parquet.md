---
title: "Parquet"
linkTitle: "Apache Parquet"
weight: 4
---

Parquet output module is designed to send `dnsmonster` logs to parquet files.


## Configuration Parameters
```ini
[parquet_output]
; What should be written to parquet file. options:
;	0: Disable Output
;	1: Enable Output without any filters
;	2: Enable Output and apply skipdomains logic
;	3: Enable Output and apply allowdomains logic
;	4: Enable Output and apply both skip and allow domains logic
parquetoutputtype = 0

; Path to output folder. Used if parquetoutputtype is not none
parquetoutputpath =

; Number of records to write to parquet file before flushing
parquetflushbatchsize = 10000

; Number of workers to write to parquet file
parquetworkercount = 4

; Size of the write buffer in bytes
parquetwritebuffersize = 256000
```