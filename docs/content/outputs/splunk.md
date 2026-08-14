+++
title = "Splunk HEC"
description = "Push dnsmonster JSON into a Splunk index through the HTTP Event Collector, with multiple endpoints for load balancing."
weight = 5
+++

Splunk's HTTP Event Collector is a widely used component for ingesting raw and JSON data.
`dnsmonster` uses the JSON output to push logs into a Splunk index.

You can specify multiple HEC endpoints for load balancing and fault tolerance across index heads.
The token and the other settings are shared between all endpoints.

## Configuration parameters

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
