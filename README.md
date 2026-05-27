# turnable-manager-panel

`turnable-manager-panel` is a lightweight web admin panel for managing a Turnable server on a VPS.

It is designed for the architecture discussed for a RU entry node and an NL exit node:

```text
Phone / client
  -> Turnable through allowed TURN/SFU carrier
  -> RU VPS: turnable server
  -> local route, for example 127.0.0.1:10808
  -> Xray client on RU VPS
  -> NL VLESS / Reality / XHTTP
  -> Internet
```

The panel does not modify Turnable core. It generates the files Turnable already understands:

- `/etc/turnable/config.json`
- `/etc/turnable/store.json`

It also provides a web UI for users, routes, settings, config export, file apply and optional systemd control.

## Current status

MVP, but usable:

- admin login;
- first-run password bootstrap;
- JSON state storage;
- CRUD for routes;
- CRUD for users;
- generation of Turnable `config.json` and `store.json`;
- client config export through `turnable config generate` when the Turnable binary exists;
- manual fallback `turnable://...` config URL generation;
- systemd actions for `turnable` and `xray` services;
- install script;
- nginx example;
- no database dependency and no CGO requirement.

## Install on Ubuntu

Clone the repository on the RU server and run:

```bash
sudo bash scripts/install.sh
```

The installer will:

- build `turnable-manager-panel`;
- clone and build `TheAirBlow/Turnable` if `/usr/local/bin/turnable` does not exist;
- create `/etc/turnable-manager-panel/config.json`;
- create `/etc/systemd/system/turnable-manager-panel.service`;
- create `/etc/systemd/system/turnable.service`;
- start the panel on `127.0.0.1:8899`;
- save first admin credentials to `/root/turnable-manager-admin.txt`.

Open through SSH tunnel first:

```bash
ssh -L 8899:127.0.0.1:8899 root@RU_SERVER_IP
```

Then open:

```text
http://127.0.0.1:8899/admin
```

Do not expose the panel publicly without HTTPS, firewall and a strong password.

## Manual build

```bash
go test ./...
go build -o turnable-manager-panel ./cmd/turnable-manager-panel
TURNABLE_MANAGER_ADMIN_PASSWORD='strong-password' ./turnable-manager-panel -config deploy/config.example.json
```

## First setup in panel

1. Open **Settings**.
2. Set `platform_id`, usually `vk.com` for current Turnable.
3. Set `call_id`.
4. Generate Turnable key pair on the RU server:

```bash
/usr/local/bin/turnable config keygen
```

5. Paste `priv_key` and `pub_key` into Settings.
6. Set RU server public IP and relay UDP port.
7. Add route to local Xray or another local service.
8. Add user and allow route.
9. Press **Apply config/store**.
10. Restart Turnable from Dashboard or run:

```bash
sudo systemctl restart turnable
```

## Route to local Xray

Example route for the RU -> NL bridge:

```text
ID: xray-tcp
Name: Local Xray client TCP
Address: 127.0.0.1
Port: 10808
Socket: tcp
Transport: kcp
Encryption: handshake
```

On the same RU server, Xray should listen locally and send outbound traffic to your NL VLESS server.

## Generated Turnable files

`config.json`:

```json
{
  "platform_id": "vk.com",
  "call_id": "...",
  "priv_key": "...",
  "pub_key": "...",
  "relay": {
    "enabled": true,
    "proto": "dtls",
    "cloak": "none",
    "public_ip": "RU_SERVER_IP",
    "port": 56000
  },
  "p2p": {
    "enabled": false,
    "username": "",
    "cloak": "none"
  },
  "provider": {
    "type": "json",
    "path": "store.json"
  }
}
```

`store.json`:

```json
{
  "routes": [
    {
      "id": "xray-tcp",
      "address": "127.0.0.1",
      "port": 10808,
      "socket": "tcp",
      "transport": "kcp",
      "encryption": "handshake",
      "name": "Local Xray client TCP"
    }
  ],
  "users": [
    {
      "uuid": "...",
      "allowed_routes": ["xray-tcp"],
      "username": "client01",
      "type": "relay",
      "peers": 4
    }
  ]
}
```

## Security notes

The panel controls files and can restart services. Keep it on localhost and put nginx with HTTPS/auth/firewall in front of it.

The user UUID is an authentication secret for Turnable clients. Do not publish exported client configs.

`p2p` is exposed in the UI because Turnable has this mode in its model, but the practical default should be `relay` until the upstream project stabilizes P2P.

## Repository structure

```text
cmd/turnable-manager-panel/    entrypoint
internal/app/                  backend, templates, storage, Turnable generator
scripts/install.sh             one-command install for Ubuntu
scripts/build.sh               local build helper
deploy/                        systemd/nginx/config examples
docs/                          architecture and operational notes
```
