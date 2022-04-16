---
title: "InfluxDB"
linkTitle: "InfluxDB"
weight: 4
---

InfluxDB is a time series database used to store logs and metrics with high ingestion rate. 


## Configuration options
```ini
[influx_output]
; What should be written to influx. options:
;	0: Disable Output
;	1: Enable Output without any filters
;	2: Enable Output and apply skipdomains logic
;	3: Enable Output and apply allowdomains logic
;	4: Enable Output and apply both skip and allow domains logic
InfluxOutputType = 0

; influx Server address, example: http://localhost:8086. Used if influxOutputType is not none
InfluxOutputServer =

; Influx Server Auth Token
InfluxOutputToken = dnsmonster

; Influx Server Bucket
InfluxOutputBucket = dnsmonster

; Influx Server Org
InfluxOutputOrg = dnsmonster

; Minimum capacity of the cache array used to send data to Influx
InfluxOutputWorkers = 8

; Minimum capacity of the cache array used to send data to Influx
InfluxBatchSize = 1000
```