# Architecture

`turnable-manager-panel` is intentionally a wrapper, not a fork of Turnable.

The manager owns:

- admin UI;
- persistent panel state;
- route/user CRUD;
- rendering Turnable `config.json`;
- rendering Turnable `store.json`;
- calling `turnable config generate` for client exports;
- optional systemd start/stop/restart/status.

Turnable owns:

- TURN/SFU tunnel establishment;
- relay/P2P protocol;
- user authentication by UUID;
- route forwarding;
- traffic transport.

Xray owns:

- RU to NL encrypted outbound;
- VLESS/Reality/XHTTP/gRPC or any other configured protocol;
- final exit routing.

## Data flow

```text
Client app
  -> Turnable client config
  -> allowed carrier / TURN or SFU
  -> RU VPS turnable server
  -> route destination, for example 127.0.0.1:10808
  -> RU VPS Xray client
  -> NL Xray inbound
  -> Internet
```

## Why route to local Xray

Turnable is a tunnel. It does not replace a full proxy/VPN stack. For the RU/NL design, the RU node should not be the internet exit. It should receive carrier-friendly traffic and immediately forward it to a local Xray client that sends traffic to the NL server.

## Files

The panel keeps its own state in:

```text
/var/lib/turnable-manager-panel/state.json
```

It writes Turnable runtime config to:

```text
/etc/turnable/config.json
/etc/turnable/store.json
```

This separation is deliberate. If Turnable changes its format later, migration can be done in the panel generator without losing admin metadata.
