---
title: Kubernetes
description: Inject a dnstap logger into coredns and receive the logs in a dnsmonster pod inside the same namespace.
sidebar:
  order: 3
---

This guide injects a custom configuration into Kubernetes' `coredns` DNS server to provide a
`dnstap` logger, then sets up a `dnsmonster` pod to receive, process and forward those logs.

## dnsmonster deployment

For `dnsmonster` to see the dnstap connection coming from the `coredns` pod, create the `dnsmonster`
Service inside the same namespace (`kube-system` or equivalent).

:::caution[Do not name it "dnsmonster"]
Avoid naming your service and pod `dnsmonster`. Kubernetes injects environment variables with a
`DNSMONSTER_` prefix into the pod, and the `dnsmonster` binary interprets those as command line
input.
:::

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

Now grab the static IP assigned to the service, for use in the coredns ConfigMap. CoreDNS provides
DNS itself, so it does not support an FQDN as a dnstap endpoint — it needs the IP.

```bash
SVCIP=$(kubectl get service dnsmonster-dnstap --output go-template --template='{{.spec.clusterIP}}')
```

## Locate and edit the coredns config

First check whether you can see and manipulate configuration inside the coredns pods:

```bash
kubectl get pod --output yaml --all-namespaces | grep coredns
```

You should see many objects associated with coredns, most notably `coredns-custom`. That ConfigMap
lets you customise the coredns configuration file and enable built-in plugins. Many cloud providers
build `coredns-custom` into their offering — see the
[AKS](https://docs.microsoft.com/en-us/azure/aks/coredns-custom),
[Oracle Cloud](https://docs.oracle.com/en-us/iaas/Content/ContEng/Tasks/contengconfiguringdnsserver.htm)
and [DigitalOcean](https://docs.digitalocean.com/products/kubernetes/how-to/customize-coredns/) docs.

Amazon EKS has no `coredns-custom`, so the configuration has to be edited in the main configuration
file instead. On top of that, EKS keeps overriding your configuration with the default value through
a DNS add-on, which must be disabled before any coredns customisation sticks. See
[this issue](https://github.com/aws/containers-roadmap/issues/1159) for more.

The command below has been tested on DigitalOcean managed Kubernetes:

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

After that, logs appear inside your `dnsmonster` pod. Customising the configuration parameters from
here is straightforward — see [configuration](/configuration/) and [outputs](/outputs/) for where to
send the data next.
