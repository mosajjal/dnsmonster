---
title: "Zinc Search"
linkTitle: "Zinc Search"
weight: 4
---

Zinc Search output module is designed to send `dnsmonster` logs to [zincsearch](https://github.com/zincsearch/zincsearch).


## Configuration Parameters
```ini

[zinc_output]
; What should be written to zinc. options:
;	0: Disable Output
;	1: Enable Output without any filters
;	2: Enable Output and apply skipdomains logic
;	3: Enable Output and apply allowdomains logic
;	4: Enable Output and apply both skip and allow domains logic
zincoutputtype = 0

; index used to save data in Zinc
zincoutputindex = dnsmonster

; zinc endpoint address, example: http://127.0.0.1:9200/api/default/_bulk. Used if zincOutputType is not none
zincoutputendpoint =

; zinc username, example: admin@admin.com. Used if zincOutputType is not none
zincoutputusername =

; zinc password, example: password. Used if zincOutputType is not none
zincoutputpassword =

; Send data to Zinc in batch sizes
zincbatchsize = 1000

; Interval between sending results to Zinc if Batch size is not filled
zincbatchdelay = 1s

; Zing request timeout
zinctimeout = 10s
```