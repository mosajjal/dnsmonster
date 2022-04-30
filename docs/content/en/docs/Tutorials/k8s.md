---
title: "Kubernetes"
linkTitle: "Kubernetes"
weight: 6
date: 2022-04-28
description: >
  use dnsmonster to monitor Kubernetes DNS traffic
---

In this guide, I'll go through the steps to inject a custom configuration into Kubernetes' `coredns` DNS server to provide a `dnstap` logger, and set up a `dnsmonster` pod to receive the logs, process them and send them to intended outputs. 

## dnsmonster deployment

In order to make `dnsmonster` see the dnstap connection coming from `coredns` pod, we need to create the `dnsmonster` Service inside the same namespace (`kube-system` or equivalent)

{{< alert color="warning" title="Warning" >}}Avoid setting your services and pod names "dnsmonster". Reason is, Kubernetes injects a few environment variables to your `dnsmonster` instance with `DNSMONSTER_` prefix, and the `dnsmonster` binary will interpret those as an input command line. {{< /alert >}}


```bash
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    k8s-app: dnsmonster-dnstap
  name: dnsmonster-dnstap
  namespace: kube-system
spec:
  # change the replica count to how many you might need to comfortably ingest the data
  replicas: 1
  selector:
    matchLabels:
      k8s-app: dnsmonster-dnstap
  template:
    metadata:
      labels:
        k8s-app: dnsmonster-dnstap
    spec:
      containers:
      - name: dnsm-dnstap
        image: ghcr.io/mosajjal/dnsmonster:v0.9.3
        args: 
          - "--dnstapSocket=tcp://0.0.0.0:7878"
          - "--stdoutOutputType=1"
        imagePullPolicy: IfNotPresent
        ports:
          - containerPort: 7878
---
apiVersion: v1
# https://kubernetes.io/docs/concepts/services-networking/connect-applications-service/#creating-a-service
# as per above documentation, each service will have a unique IP address that won't change for the lifespan of the service
kind: Service
metadata:
  name: dnsmonster-dnstap
  namespace: kube-system
spec:
  selector:
    k8s-app: dnsmonster-dnstap
  ports:
  - name: dnsmonster-dnstap
    protocol: TCP
    port: 7878
    targetPort: 7878
EOF
```

now we can get the static IP assigned to the service to use it in coredns custom ConfigMap. Note that since CoreDNS itself is providing DNS, it does not support FQDN as a dnstap endpoint. 

```bash
SVCIP=$(kubectl get service dnsmonster-dnstap --output go-template --template='{{.spec.clusterIP}}')
```

## locate and edit the `coredns` config

Let's try and see if we can see and manipulate configuration inside coredns pods. Using below command, we can get a list of running coredns containers

`kubectl get pod --output yaml --all-namespaces | grep coredns`

In above command, you should be able to see many objects associated with coredns, most notably, `coredns-custom`. `coredns-custom` ConfigMap allows us to customize coredns configuration file and enable builtin plugins for it. Many cloud providers have built `coredns-custom` ConfigMap into the offering. Take a look at [AKS](https://docs.microsoft.com/en-us/azure/aks/coredns-custom), [Oracle Cloud](https://docs.oracle.com/en-us/iaas/Content/ContEng/Tasks/contengconfiguringdnsserver.htm) and [DigitalOcean](https://docs.digitalocean.com/products/kubernetes/how-to/customize-coredns/) docs for more details. 

in Amazon's EKS, there's no `coredns-custom`. So the configuration needs to be edited on the main configuration file instead. On top of that, EKS will keep overriding your configuration with the "default" value through a DNS add-on. That add-on needs to be disabled for any customization in coredns configuration as well. Take a look at [This issue](https://github.com/aws/containers-roadmap/issues/1159) for more information. 

Below command has been tested on DigitalOcean managed Kubernetes

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: coredns-custom
  namespace: kube-system
data:
  log.override: |
    dnstap tcp://$SVCIP:7878 full
EOF
```

After running the above command, you will see the logs inside your dnsmonster pod. As commented in the yaml definitions, customizing the configuration parameters should be fairly straightforward. 

