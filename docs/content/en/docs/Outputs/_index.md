---
title: "Outputs"
linkTitle: "Outputs"
weight: 4
description: >
  Set up output(s) and gather metrics
---

`dnsmonster` follows a pipeline architecture for each individual packet. After the Capture and filter, each processed packet arrives at the output dispatcher. The dispatcher sends a copy of the output to each individual output module that have been configured to produce output. For instance, if you specify `stdoutOutputType=1` and `--fileOutputType=1 --fileOutputPath=/dev/stdout`, you'll see each processed output twice in your stdout. One coming from the stdout output type, and the other from the file output type which happens to have the same address (`/dev/stdout`).  

In general, each output has its own configuration section. You can see the sections with "_output" suffix when running `dnsmonster --help` from the command line. The most important parameter for each output is their "Type". Each output has 5 different types:

- Type 0:
- Type 1: An output module configured as Type 1 will ignore "SkipDomains" and "AllowDomains" and will generate output for all the incoming processed packets. Note that the output types does *not* nullify input filters since it is applied after capture and early packet filters. Take a look at [Filters and Masks](/docs/inputs/filters_masks) to see the order of the filters applied.  
- Type 2: An output module configured as Type 2 will ignore "AllowDomains" and only applies the "SkipDmains" logic to the incoming processed packets.
- Type 3: An output module configured as Type 3 will ignore "SkipDmains" and only applies the "AllowDomains" logic to the incoming processed packets.
- Type 4: An output module configured as Type 4 will apply both "SkipDmains" and "AllowDomains" logic to the incoming processed packets.

Other than `Type`, each output module may require additional configuration parameters. For more information, refer to each module's documentation.

