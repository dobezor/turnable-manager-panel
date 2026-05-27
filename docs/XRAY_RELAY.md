# RU Turnable -> NL Xray relay

This document describes the intended bridge.

## RU server

Run:

- `turnable-manager-panel` on `127.0.0.1:8899`;
- `turnable` with `/etc/turnable/config.json` and `/etc/turnable/store.json`;
- `xray` with a local inbound.

Example local Xray inbound idea:

```json
{
  "inbounds": [
    {
      "tag": "local-in",
      "listen": "127.0.0.1",
      "port": 10808,
      "protocol": "socks",
      "settings": { "udp": true }
    }
  ],
  "outbounds": [
    {
      "tag": "nl-vless",
      "protocol": "vless",
      "settings": {
        "vnext": [
          {
            "address": "NL_DOMAIN_OR_IP",
            "port": 443,
            "users": [
              { "id": "UUID", "encryption": "none" }
            ]
          }
        ]
      },
      "streamSettings": {
        "network": "xhttp",
        "security": "reality"
      }
    }
  ]
}
```

The exact Xray config depends on your NL inbound. Keep the Turnable route destination local.

## Turnable route in panel

```text
Address: 127.0.0.1
Port: 10808
Socket: tcp
Transport: kcp
Encryption: handshake
```

For UDP routes, use `socket=udp` and `transport=none`, but test this separately. TCP with KCP is the safer first path.
