+++
title = "Post-installation"
description = "Set up systemd services and shell completions so dnsmonster runs on system startup."
weight = 2
+++

After installing `dnsmonster`, a few extra steps make it run automatically on system startup. These
are not part of the installation process by default.

## Systemd service

On a modern distribution — Debian, Ubuntu, Fedora, Arch, RHEL — you are almost certainly running
`systemd`. To run `dnsmonster` as a service, create `/etc/systemd/system/dnsmonster.service` and
define your unit there. The service name is arbitrary.

```sh
cat > /etc/systemd/system/dnsmonster.service << EOF
[Unit]
Description=Dnsmonster Service
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
Restart=always
RestartSec=3
ExecStart=/sbin/dnsmonster --config /etc/dnsmonster.ini

[Install]
WantedBy=multi-user.target

EOF
```

The unit above reads `/etc/dnsmonster.ini` as its configuration file. See the
[configuration section](/configuration/) for how that file is generated.

To start the service and enable it at boot:

```sh
sudo systemctl enable --now dnsmonster.service
```

### One instance per interface

You can also build a templated unit that takes the interface name dynamically and runs one
`dnsmonster` instance per interface:

```sh
cat > /etc/systemd/system/dnsmonster@.service << EOF
[Unit]
Description=Dnsmonster Service
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
Restart=always
RestartSec=3
ExecStart=/sbin/dnsmonster --devName=%i --config /etc/dnsmonster.ini

[Install]
WantedBy=multi-user.target

EOF
```

To run it for the loopback interface (`lo`):

```sh
sudo systemctl enable --now dnsmonster@lo.service
```

This only works if the configuration file does not also specify a `dnstap` socket or a local `pcap`
file as input — a run takes exactly one input.
