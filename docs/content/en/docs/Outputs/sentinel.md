---
title: "Microsoft Sentinel"
linkTitle: "Microsoft Sentinel"
weight: 4
---

Microsoft Sentinel output module is designed to send `dnsmonster` logs to Sentinel. In addition to that, this module supports sending the logs to any Log Analytics workspace no matter if they are connected to Sentinel or not.

Please take a look at Microsoft's official documentation to see how Customer ID and Shared key are obtained. 


## Configuration Parameters
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