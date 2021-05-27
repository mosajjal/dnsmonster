FROM alpine:edge
LABEL maintainer "Ali Mosajjal <hi@n0p.me>"

RUN apk add --no-cache libcap-static libpcap-dev linux-headers git go file --repository http://dl-cdn.alpinelinux.org/alpine/edge/testing/

RUN git clone https://github.com/mosajjal/dnsmonster.git /opt/dnsmonster --depth 1 \
    && cd /opt/dnsmonster/src \
    && export LATEST_TAG=`git describe --tags` \
    && go build --ldflags "-L /usr/lib/libcap.a -linkmode external -X main.releaseVersion=$LATEST_TAG -extldflags \"-static\""

FROM scratch
COPY --from=0 /opt/dnsmonster/src/dnsmonster /dnsmonster
ENTRYPOINT ["/dnsmonster"] 
