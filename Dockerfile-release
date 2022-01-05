FROM alpine:edge
LABEL maintainer "Ali Mosajjal <hi@n0p.me>"

SHELL ["/bin/ash", "-c"]

RUN apk add --no-cache libcap-static libpcap-dev linux-headers git go file dpkg rpm --repository http://dl-cdn.alpinelinux.org/alpine/edge/testing/

ENV REPO="github.com/mosajjal/dnsmonster"
RUN git clone https://${REPO}.git /opt/dnsmonster --depth 1 \
    && cd /opt/dnsmonster \
    && git fetch --tags \ 
    && export LATEST_TAG=`git describe --tags --always` \
    && go build --ldflags "-L /usr/lib/libcap.a -linkmode external -X ${REPO}/util.releaseVersion=${LATEST_TAG} -extldflags \"-static\"" -o /tmp/dnsmonster-linux-amd64.bin

ENV CGO_ENABLED=1
ENV GOOS=windows
ENV GOARCH=amd64
RUN sh -c 'cd /opt/dnsmonster && go build -o /tmp/dnsmonster-windows-amd64.exe'

WORKDIR /opt/dnsmonster

# build Deb file
RUN cd /opt/dnsmonster \
    && export LATEST_TAG=`git describe --tags` \
    && export DIR="/tmp/dnsmonster_${LATEST_TAG:1}-1_amd64" \
    && mkdir -p $DIR/sbin $DIR/etc/dnsmonster $DIR/usr/share/man/man7 $DIR/DEBIAN $DIR/etc/bash_completion.d $DIR/usr/share/fish/vendor_completions.d \
    && cp /tmp/dnsmonster-linux-amd64.bin $DIR/sbin/dnsmonster \
    && $DIR/sbin/dnsmonster --manPage > $DIR/usr/share/man/man7/dnsmonster.7 \
    && $DIR/sbin/dnsmonster --bashCompletion > $DIR/etc/bash_completion.d/dnsmonster.bash \
    && $DIR/sbin/dnsmonster --fishCompletion > $DIR/usr/share/fish/vendor_completions.d/dnsmonster.fish \
    && $DIR/sbin/dnsmonster --writeConfig $DIR/etc/dnsmonster/dnsmonster.ini \
    && echo -e  "Package: dnsmonster\nVersion: ${LATEST_TAG:1}\nArchitecture: amd64\nMaintainer: Ali Mosajjal <hi@n0p.me>\nDescription: Passive DNS monitoring framework." > $DIR/DEBIAN/control \
    && dpkg-deb --build --root-owner-group $DIR \
    && mv /tmp/dnsmonster_*_amd64.deb /tmp/dnsmonster-latest.deb

# build rpm file and move it to /tmp/dnsmonster-latest.rpm
RUN export LATEST_TAG=`git describe --abbrev=0 --tags` && echo -e '\
Name:       dnsmonster \n\
Version:    '${LATEST_TAG:1}' \n\
Release:    1 \n\
Summary:    Passive DNS monitoring framework. \n\
License:    GPLv2 \n\
\n\
%description \n\
This is my first RPM package, which does nothing. \n\
\n\
%prep \n\
# we have no source, so nothing here \n\
\n\
%define _rpmdir /tmp \n\
\n\
%build \n\
export LATEST_TAG=`git describe --tags` \n\
mkdir -p %{buildroot}/sbin %{buildroot}/etc/dnsmonster %{buildroot}/usr/share/man/man7 %{buildroot}/DEBIAN %{buildroot}/etc/bash_completion.d %{buildroot}/usr/share/fish/vendor_completions.d \n\
cp /tmp/dnsmonster-linux-amd64.bin %{buildroot}/sbin/dnsmonster \n\
%{buildroot}/sbin/dnsmonster --manPage > %{buildroot}/usr/share/man/man7/dnsmonster.7 \n\
%{buildroot}/sbin/dnsmonster --bashCompletion > %{buildroot}/etc/bash_completion.d/dnsmonster.bash \n\
%{buildroot}/sbin/dnsmonster --fishCompletion > %{buildroot}/usr/share/fish/vendor_completions.d/dnsmonster.fish \n\
%{buildroot}/sbin/dnsmonster --writeConfig %{buildroot}/etc/dnsmonster/dnsmonster.ini \n\
\n\
%install \n\
cp /tmp/dnsmonster-linux-amd64.bin %{buildroot}/sbin/dnsmonster  \n\
\n\
%files \n\
/sbin/dnsmonster \n\
/usr/share/man/man7/dnsmonster.7 \n\
/etc/bash_completion.d/dnsmonster.bash \n\
/usr/share/fish/vendor_completions.d/dnsmonster.fish \n\
/etc/dnsmonster/dnsmonster.ini \n\
\n\
%changelog \n\
# let's skip this for now
    '> /tmp/dnsmonster-specfile
RUN  cat /tmp/dnsmonster-specfile && rpmbuild -ba --build-in-place  /tmp/dnsmonster-specfile && mv /tmp/*/dnsmonster-*.x86_64.rpm /tmp/dnsmonster-latest.rpm

# build Arch Package
# RUN export LATEST_TAG=`git describe --abbrev=0 --tags` && echo -e '\
# # Maintainer "Ali Mosajjal <hi@n0p.me>" \n\
# pkgname=NAME \n\
# pkgver=VERSION \n\
# pkgrel=1 \n\
# pkgdesc="" \n\
# arch=(\'x86_64\') \n\
# url="" \n\
# license=(\'GPL\') \n\
# groups=() \n\
# depends=() \n\
# makedepends=() \n\
# optdepends=() \n\
# provides=() \n\
# conflicts=() \n\
# replaces=() \n\
# backup=() \n\
# options=() \n\
# install= \n\
# changelog= \n\
# source=($pkgname-$pkgver.tar.gz) \n\
# noextract=() \n\
# md5sums=() #generate with \'makepkg -g\' \n\
#  \n\
# build() { \n\
#   cd "$srcdir/$pkgname-$pkgver" \n\
#  \n\
#   ./configure --prefix=/usr \n\
#   make \n\
# } \n\
#  \n\
# package() { \n\
#   cd "$srcdir/$pkgname-$pkgver" \n\
#  \n\
#   make DESTDIR="$pkgdir/" install \n\
# } ' > 