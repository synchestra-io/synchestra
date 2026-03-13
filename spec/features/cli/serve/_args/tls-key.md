# --tls-key

Path to the TLS private key file for HTTPS.

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

Specifies the path to a PEM-encoded TLS private key file. Required when `--https` is used. If `--https` is specified without `--tls-key`, the command exits with code `2` (invalid arguments).

The corresponding field in [`synchestra-server.yaml`](../../server/synchestra-server.yaml.md) is `tls.key`.

## Examples

```bash
synchestra serve --https --tls-cert cert.pem --tls-key key.pem
```

## Outstanding Questions

None at this time.
