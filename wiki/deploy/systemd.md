# systemd

Run GoPOSIX as a persistent daemon on a Linux host managed by systemd.

## Unit File

Save as `/etc/systemd/system/goposix.service`:

```ini
[Unit]
Description=GoPOSIX JSON-RPC Daemon
Documentation=https://github.com/ramayac/goposix
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/goposix daemon -s /var/run/goposix.sock -w 4
ExecReload=/bin/kill -HUP $MAINPID
Restart=always
RestartSec=2

# Security hardening
User=nobody
Group=nogroup
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/var/run
PrivateTmp=yes
PrivateDevices=yes
ProtectKernelTunables=yes
ProtectKernelModules=yes
ProtectControlGroups=yes
RestrictAddressFamilies=AF_UNIX
SystemCallFilter=@default
SystemCallArchitectures=native
MemoryMax=128M

[Install]
WantedBy=multi-user.target
```

## Activate

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now goposix
sudo systemctl status goposix
```

## Socket Activation

For on-demand daemon spawning, create a companion socket unit:

`/etc/systemd/system/goposix.socket`:

```ini
[Unit]
Description=GoPOSIX Socket

[Socket]
ListenStream=/var/run/goposix.sock
SocketMode=0660
SocketUser=nobody
SocketGroup=nogroup

[Install]
WantedBy=sockets.target
```

And update the service to use `GoPOSIX.socket`:

```ini
[Service]
Type=simple
# Accept the socket from systemd (via LISTEN_FDS)
ExecStart=/usr/local/bin/goposix daemon -s /var/run/goposix.sock
```

> When using systemd socket activation, remove the `-s` argument and let GoPOSIX inherit the fd. Check the release notes for sd_listen support.

Then enable both:

```bash
sudo systemctl enable --now goposix.socket
```

## Healthcheck

```bash
echo '{"jsonrpc":"2.0","method":"goposix.ping","id":1}' | nc -U /var/run/goposix.sock
# → {"jsonrpc":"2.0","result":{"pong":true,"version":"..."},"id":1}
```

## Logs

```bash
journalctl -u goposix -f
```

## Troubleshooting

**Socket already in use**: Remove stale socket file:

```bash
sudo rm /var/run/goposix.sock
sudo systemctl restart goposix
```

**Permission denied**: The socket is mode `0660` owned by `nobody:nogroup`. Ensure client processes are in the `nogroup` group:

```bash
sudo usermod -aG nogroup myapp
```
