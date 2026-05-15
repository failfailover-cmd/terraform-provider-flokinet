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

  # optional: privileged fallback for delete via WHM API
  whm_host      = var.floki_whm_host
  whm_port      = 2087
  whm_username  = var.floki_whm_username
  whm_api_token = var.floki_whm_api_token
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
- `FLOKI_WHM_HOST` (optional)
- `FLOKI_WHM_PORT` (optional, default `2087`)
- `FLOKI_WHM_USERNAME` (optional)
- `FLOKI_WHM_API_TOKEN` (optional)


Retry tuning:
- `max_retries`
- `base_backoff_ms`
- `max_backoff_ms`
- `request_timeout_ms`

## Удаление проблемных addon-доменов

Иногда cPanel user-level API не может удалить addon-домен и возвращает ошибки вида:
- `You do not have control of the subdomain ...`
- `... does not correspond to ...`
- `... parked on top of it ...`

В этом случае провайдер автоматически делает fallback на WHM API `delete_domain`, если заданы `whm_*` параметры или `FLOKI_WHM_*` env.
