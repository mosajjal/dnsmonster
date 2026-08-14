---
title: Apache Kafka
description: Ship dnsmonster records to Kafka with compression, TLS and multiple brokers — the recommended output for enterprise deployments.
sidebar:
  order: 3
---

Possibly the most versatile output supported by `dnsmonster`. Kafka lets you connect to an endless
list of downstream sinks, and it is the recommended output for enterprise designs because it is
fault tolerant and can absorb outages at the sink.

The Kafka output supports compression, TLS and multiple brokers. To use multiple brokers, specify
the broker parameter more than once.

## Configuration parameters

```ini
[kafka_output]
; What should be written to kafka. options:
;	0: Disable Output
;	1: Enable Output without any filters
;	2: Enable Output and apply skipdomains logic
;	3: Enable Output and apply allowdomains logic
;	4: Enable Output and apply both skip and allow domains logic
KafkaOutputType = 0

; kafka broker address(es), example: 127.0.0.1:9092. Used if kafkaOutputType is not none
KafkaOutputBroker =

; Kafka topic for logging
KafkaOutputTopic = dnsmonster

; Minimum capacity of the cache array used to send data to Kafka
KafkaBatchSize = 1000

; Kafka connection timeout in seconds
KafkaTimeout = 3

; Interval between sending results to Kafka if Batch size is not filled
KafkaBatchDelay = 1s

; Compress Kafka connection
KafkaCompress = false

; Compression Type[gzip, snappy, lz4, zstd] default is snappy
KafkaCompressiontype = snappy

; Use TLS for kafka connection
KafkaSecure = false

; Path of CA certificate that signs Kafka broker certificate
KafkaCACertificatePath =

; Path of TLS certificate to present to broker
KafkaTLSCertificatePath =

; Path of TLS certificate key
KafkaTLSKeyPath =
```
