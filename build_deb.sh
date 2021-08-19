#!/bin/sh

# get version from latest git tag
export VER=$(git describe --tags --abbrev=0|sed -e 's/^v//')

# create a directory to build the debian package in
DIR="/tmp/dnsmonster_$VER-1_amd64"

# remove possible previous build temp files
rm -rf $DIR

mkdir -p $DIR/sbin $DIR/usr/share/man/man7 $DIR/DEBIAN $DIR/etc/bash_completion.d/ $DIR/usr/share/fish/vendor_completions.d


docker build -t dnsmonster-build:temp --no-cache --pull -f Dockerfile-release .
ID=$(docker create dnsmonster-build:temp)

docker cp $ID:/tmp/dnsmonster-linux-amd64.bin $DIR/sbin/dnsmonster

# remove temp container and image
docker rm -f $ID
docker rmi  -f dnsmonster-build:temp

# generate manfile and completion files into appropiate locations
$DIR/sbin/dnsmonster --manPage > $DIR/usr/share/man/man7/dnsmonster.7
$DIR/sbin/dnsmonster --bashCompletion > $DIR/etc/bash_completion.d/dnsmonster.bash
$DIR/sbin/dnsmonster --fishCompletion > $DIR/usr/share/fish/vendor_completions.d/dnsmonster.fish


cat << EOF > $DIR/DEBIAN/control
Package: dnsmonster
Version: $VER
Architecture: amd64
Maintainer: Ali Mosajjal <hi@n0p.me>
Description: Passive DNS monitoring framework.
EOF

dpkg-deb --build --root-owner-group $DIR