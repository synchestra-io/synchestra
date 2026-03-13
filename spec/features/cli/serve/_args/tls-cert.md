# --tls-cert

Path to the TLS certificate file for HTTPS.

| Detail | Value |
|---|---|
| Type | String (file path) |
| Required | Yes, when `--https` is used |
| Default | — |

## Supported by

| Command |
|---|
| [`serve`](../README.md) |

## Description

Specifies the path to a PEM-encoded TLS certificate file. Required when `--https` is used. If `--https` is specified without `--tls-cert`, the command exits with code `2` (invalid arguments).

The corresponding field in [`synchestra-server.yaml`](../../server/synchestra-server.yaml.md) is `tls.cert`.

## Examples

```bash
synchestra serve --https --tls-cert /etc/ssl/certs/synchestra.pem --tls-key /etc/ssl/private/synchestra.key
```

## Outstanding Questions

None at this time.
