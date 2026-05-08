# terraform-provider-flokinet (MVP)

Кастомный Terraform provider для floki/cPanel.

## Что уже реализовано

- Provider: `flokinet`
- Resource: `flokinet_addon_domain`
  - создание/удаление addon domain через cPanel API 2 (`AddonDomain::addaddondomain`)

## Конфиг

```hcl
provider "flokinet" {
  host      = var.floki_cpanel_host
  port      = 2083
  username  = var.floki_cpanel_username
  api_token = var.floki_cpanel_api_token
}

resource "flokinet_addon_domain" "site" {
  domain    = "example.com"
  subdomain = "example"
  docroot   = "/home/user/root/sites/example.com"
}
```

Также поддерживаются env:
- `FLOKI_CPANEL_HOST`
- `FLOKI_CPANEL_PORT`
- `FLOKI_CPANEL_USERNAME`
- `FLOKI_CPANEL_API_TOKEN`
