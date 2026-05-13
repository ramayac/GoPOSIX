# systemd

Run KoreGo as a persistent daemon on a Linux host managed by systemd.

## Unit File

Save as `/etc/systemd/system/korego.service`:

```ini
[Unit]
Description=KoreGo JSON-RPC Daemon
Documentation=https://github.com/ramayac/korego
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/korego daemon -s /var/run/korego.sock -w 4
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
sudo systemctl enable --now korego
sudo systemctl status korego
```

## Socket Activation

For on-demand daemon spawning, create a companion socket unit:

`/etc/systemd/system/korego.socket`:

```ini
[Unit]
Description=KoreGo Socket

[Socket]
ListenStream=/var/run/korego.sock
SocketMode=0660
SocketUser=nobody
SocketGroup=nogroup

[Install]
WantedBy=sockets.target
```

And update the service to use `KoreGo.socket`:

```ini
[Service]
Type=simple
# Accept the socket from systemd (via LISTEN_FDS)
ExecStart=/usr/local/bin/korego daemon -s /var/run/korego.sock
```

> When using systemd socket activation, remove the `-s` argument and let KoreGo inherit the fd. Check the release notes for sd_listen support.

Then enable both:

```bash
sudo systemctl enable --now korego.socket
```

## Healthcheck

```bash
echo '{"jsonrpc":"2.0","method":"korego.ping","id":1}' | nc -U /var/run/korego.sock
# → {"jsonrpc":"2.0","result":{"pong":true,"version":"..."},"id":1}
```

## Logs

```bash
journalctl -u korego -f
```

## Troubleshooting

**Socket already in use**: Remove stale socket file:

```bash
sudo rm /var/run/korego.sock
sudo systemctl restart korego
```

**Permission denied**: The socket is mode `0660` owned by `nobody:nogroup`. Ensure client processes are in the `nogroup` group:

```bash
sudo usermod -aG nogroup myapp
```
