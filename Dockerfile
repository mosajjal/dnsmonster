FROM alpine:edge
LABEL maintainer "Ali Mosajjal <hi@n0p.me>"

ENV REPO="github.com/mosajjal/dnsmonster"

RUN apk add --no-cache libcap-static libpcap-dev linux-headers git go file --repository http://dl-cdn.alpinelinux.org/alpine/edge/testing/

RUN git clone https://${REPO}.git /opt/dnsmonster --depth 1 \
    && cd /opt/dnsmonster \
    && git fetch --tags \ 
    && export LATEST_TAG=`git describe --tags --always` \
    && go build --ldflags "-L /usr/lib/libcap.a -linkmode external -X ${REPO}/util.releaseVersion=${LATEST_TAG} -extldflags \"-static\""

FROM scratch
COPY --from=0 /opt/dnsmonster/dnsmonster /dnsmonster
ENTRYPOINT ["/dnsmonster"] 
