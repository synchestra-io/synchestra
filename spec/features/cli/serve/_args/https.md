# --https

Starts an HTTPS listener on the Synchestra server.

| Detail | Value |
|---|---|
| Type | String (optional value: `host:port`) |
| Required | At least one of `--http`, `--https`, or `--mcp` is required |
| Default | `localhost:8443` when flag is present without a value |

## Supported by

| Command |
|---|
| [`serve`](../README.md) |

## Description

Enables the HTTPS listener. Requires `--tls-cert` and `--tls-key` to provide the TLS certificate and private key.

- **Without value** (`--https`): listens on `localhost:8443`
- **With value** (`--https myhost:8443`): listens on the specified `host:port`

Can be combined with `--http` and/or `--mcp`.

The corresponding field in [`synchestra-server.yaml`](../../server/synchestra-server.yaml.md) is `https`.

## Examples

```bash
# HTTPS with TLS certificates
synchestra serve --https --tls-cert cert.pem --tls-key key.pem

# Custom address + HTTP
synchestra serve --http --https 0.0.0.0:443 --tls-cert cert.pem --tls-key key.pem
```

## Outstanding Questions

None at this time.
