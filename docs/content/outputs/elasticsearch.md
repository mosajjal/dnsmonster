+++
title = "Elasticsearch and OpenSearch"
linkTitle = "Elasticsearch / OpenSearch"
description = "Send dnsmonster records to Elasticsearch 7.x, OpenSearch or Opendistro."
weight = 4
+++

Elasticsearch is a full-text search engine used widely across security tooling. `dnsmonster`
supports Elastic 7.x out of the box; 6.x and 8.x have not been tested.

There is also a fork of Elasticsearch called Opendistro, later renamed OpenSearch. Both are
compatible with Elastic 7.10.x, so both should work.

## Configuration parameters

```ini
[elastic_output]
; What should be written to elastic. options:
;	0: Disable Output
;	1: Enable Output without any filters
;	2: Enable Output and apply skipdomains logic
;	3: Enable Output and apply allowdomains logic
;	4: Enable Output and apply both skip and allow domains logic
ElasticOutputType = 0

; elastic endpoint address, example: http://127.0.0.1:9200. Used if elasticOutputType is not none
ElasticOutputEndpoint =

; elastic index
ElasticOutputIndex = default

; Send data to Elastic in batch sizes
ElasticBatchSize = 1000

; Interval between sending results to Elastic if Batch size is not filled
ElasticBatchDelay = 1s
```
