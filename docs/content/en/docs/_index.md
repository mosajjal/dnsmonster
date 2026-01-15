---
title: "Documentation"
linkTitle: "Documentation"
weight: 20
menu:
  main:
    weight: 20
---

welcome to `dnsmonster` documentation!

The documentation is broken down into different sections, Getting Started focuses on installation and post installation work like compiling `dnsmonster` from source, setting up services, shell complations and more. Configuration gets into details of how to configure `dnsmonster`, and how to identify and solve potential performance bottlenecks. The majority of your configuration is done inside the Input and Output sections.

You'll learn where can you put filter on incoming traffic, sample inputs, mask IP addresses before even passing the packets on processor. After process, you'll be able to exclude certain FQDNs from being sent to output, or include certain domains to be logged.

---

> **We're exploring a managed SaaS solution for dnsmonster!** Help shape the future of passive DNS monitoring by sharing your feedback and requirements: [Take our quick survey](https://tally.so/r/2EAxBe)

---

All above will generate a ton of useful metrics for your DNS infrastructure. `dnsmonster` has a builtin metrics system that can integrate to your favourite metrics aggregator like `prometheus` or `statsd`. 


