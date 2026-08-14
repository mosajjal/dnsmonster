+++
title = "VictoriaLogs"
description = "Send dnsmonster logs to a VictoriaLogs JSON line insert endpoint."
weight = 9
+++

The VictoriaLogs output module sends `dnsmonster` logs to
[VictoriaLogs](https://docs.victoriametrics.com/victorialogs/index.html).

## Configuration parameters

```ini
[victoria_output]
; Victoria Output Endpoint. example: http://localhost:9428/insert/jsonline?_msg_field=rcode_id&_time_field=time
victoriaoutputendpoint =

; What should be written to Victoria. options:
;	0: Disable Output
;	1: Enable Output without any filters
;	2: Enable Output and apply skipdomains logic
;	3: Enable Output and apply allowdomains logic
;	4: Enable Output and apply both skip and allow domains logic
victoriaoutputtype = 0

; Victoria Output Proxy in URI format
victoriaoutputproxy =

; Number of workers
victoriaoutputworkers = 8

; Victoria Batch Size
victoriabatchsize = 100

; Interval between sending results to Victoria if Batch size is not filled. Any value larger than zero takes precedence over Batch Size
victoriabatchdelay = 0s
```
