+++
title = "Microsoft Sentinel"
description = "Send dnsmonster logs to Microsoft Sentinel or any Log Analytics workspace."
weight = 6
+++

The Microsoft Sentinel output module sends `dnsmonster` logs to Sentinel. It also supports sending
logs to any Log Analytics workspace, whether or not that workspace is connected to Sentinel.

See Microsoft's official documentation for how the Customer ID and Shared Key are obtained.

## Configuration parameters

```ini
[sentinel_output]
; What should be written to Microsoft Sentinel. options:
;	0: Disable Output
;	1: Enable Output without any filters
;	2: Enable Output and apply skipdomains logic
;	3: Enable Output and apply allowdomains logic
;	4: Enable Output and apply both skip and allow domains logic
SentinelOutputType = 0

; Sentinel Shared Key, either the primary or secondary, can be found in Agents Management page under Log Analytics workspace
SentinelOutputSharedKey =

; Sentinel Customer Id. can be found in Agents Management page under Log Analytics workspace
SentinelOutputCustomerId =

; Sentinel Output LogType
SentinelOutputLogType = dnsmonster

; Sentinel Output Proxy in URI format
SentinelOutputProxy =

; Sentinel Batch Size
SentinelBatchSize = 100

; Interval between sending results to Sentinel if Batch size is not filled
SentinelBatchDelay = 1s
```
