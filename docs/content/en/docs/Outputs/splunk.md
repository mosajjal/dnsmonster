---
title: "Splunk HEC"
linkTitle: "Splunk HEC"
weight: 4
---

Splunk HTTP Event Collector is a widely used component of Splunk to ingest raw and JSON data. `dnsmonster` uses the JSON output to push the logs into a Splunk index. various configurations are also supported. You can also use multiple HEC endpoints to have load balancing and fault tolerance across multiple index heads. Note that the token and other settings are shared between multiple endpoints.

## Configuration Parameters

```ini
[splunk_output]
; What should be written to HEC. options:
;	0: Disable Output
;	1: Enable Output without any filters
;	2: Enable Output and apply skipdomains logic
;	3: Enable Output and apply allowdomains logic
;	4: Enable Output and apply both skip and allow domains logic
SplunkOutputType = 0

; splunk endpoint address, example: http://127.0.0.1:8088. Used if splunkOutputType is not none, can be specified multiple times for load balanace and HA
SplunkOutputEndpoint =

; Splunk HEC Token
SplunkOutputToken = 00000000-0000-0000-0000-000000000000

; Splunk Output Index
SplunkOutputIndex = temp

; Splunk Output Proxy in URI format
SplunkOutputProxy =

; Splunk Output Source
SplunkOutputSource = dnsmonster

; Splunk Output Sourcetype
SplunkOutputSourceType = json

; Send data to HEC in batch sizes
SplunkBatchSize = 1000

; Interval between sending results to HEC if Batch size is not filled
SplunkBatchDelay = 1s
```