---
title: "Elasticsearch/OpenSearch"
linkTitle: "Elasticsearch/OpenSearch"
weight: 4
---

Elasticsearch is a full-text search engine and it's used widely across a lot of security tools. `dnsmonster` supports Elastic 7.x out of the box. The support for 6.x and 8.x has not been tested.

There is also a fork of Elasticsearch called Opendistro, later renamed to Opensearch. Both are compatible with 7.10.x Elastic, so it should also be supported too.

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