# WildVault

Obtains a wildcard TLS certificate from Let's Encrypt via DNS-01 challenge and stores it in HashiCorp Vault.

## How it works

1. Registers a new ACME account with Let's Encrypt using an ephemeral ECDSA key
2. Fetches `BUNNY_API_KEY` from Vault (`kv/rsafe-ovh/dns/bunny`) and sets it in the environment
3. Completes the DNS-01 challenge using the Bunny DNS provider
4. Obtains a bundled wildcard certificate for `*.rsafe.ovh`
5. Parses the certificate to extract metadata (serial, validity dates, domains)
6. Writes the certificate and metadata to Vault (`kv/rsafe-ovh/tls/rsafe.ovh`)

## Prerequisites

- Go 1.25+
- HashiCorp Vault with KV v2 enabled
- `BUNNY_API_KEY` stored in Vault at `kv/rsafe-ovh/dns/bunny`

## Environment variables

| Variable | Description |
|---|---|
| `VAULT_ADDR` | Vault server address (e.g. `https://vault.example.com:8200`) |
| `VAULT_TOKEN` | Vault authentication token |

## Vault paths

| Path | Operation | Description |
|---|---|---|
| `kv/rsafe-ovh/dns/bunny` | Read | Bunny DNS API key (`BUNNY_API_KEY`) |
| `kv/rsafe-ovh/tls/rsafe.ovh` | Write | Issued certificate and metadata |

### Secret schema written to `kv/rsafe-ovh/tls/rsafe.ovh`

| Key | Description |
|---|---|
| `certificate` | Full PEM certificate bundle (cert + intermediates) |
| `private_key` | PEM private key |
| `domains` | JSON array of DNS SANs |
| `issued_at` | Validity start in RFC 3339 UTC (e.g. `2025-01-01T00:00:00Z`) |
| `expires_at` | Validity end in RFC 3339 UTC |
| `serial` | Certificate serial number (colon-separated hex) |

## Build

```sh
go build -o wildvault .
```

## Usage

```sh
# Production
./wildvault

# Staging (Let's Encrypt staging CA — untrusted certificate)
./wildvault -staging
```
