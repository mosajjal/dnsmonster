---
title: "post-installation"
linkTitle: "post-installation"
weight: 1
description: >
  Set up services and shell completions for dnsmonster 
---

## Post-install

After you install dnsmonster, you might need to take a few extra steps to build services so `dnsmonster` runs automatically on system startup. These steps are not included in the installation process by default.

### Systemd service

If you're using a modern and popular distro like Debian, Ubuntu, Fedora, Arch, RHEL, you're probably using `systemd` as your init system. In order to add `dnsmonster` as a service, created a file in `/etc/systemd/system/` named `dnsmonster.service`, and define your systemd unit there. The name `dnsmonster` as a service name is totally optional.

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

The above systemd service looks at `/etc/dnsmonster.ini` as a configuration file. Checkout the configuration section to see how that configuration file is generated. 

to start the service and ebable it at boot time, run the following

```sh
sudo systemctl enable --now dnsmonster.service
```

You can also build a systemd service that takes the interface name dynamically and runs the `dnsmonster` instance per interface. To do so, create a service unit like this:

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
ExecStart=/sbin/dnsmonste --devName=%i --config /etc/dnsmonster.ini

[Install]
WantedBy=multi-user.target

EOF
```

The above unit creates a dynamic systemd service that can be enabled for multiple Interfaces. For example, to run the service for the loopback interface in linux (`lo`), run the following:

```sh
sudo systemctl enable --now dnsmonster@lo.service
```

Note that the above example only works if you're not specifying a `dnstap` or a local `pcap` file as an input inside the configuration file.  

### init.d service

### bash and fish completion
