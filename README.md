# hook-spray
Webhook notification proxy to dispatch to multiple endpoints.
This also works for SPARQL 1.1 Update.
`hook-spray` endpoint is fixed to `/hook`.

```mermaid
graph LR
  ORIGIN --> hook-spray
  hook-spray --> webhook1
  hook-spray --> webhook2
```

This webhook proxy is quick to setup but not robust.
If you want robustness, there is kafka-backed job queue dispatcher.

https://github.com/simonireilly/kafka-webhook-dispatcher

## args

- dest : webhook endpoint. you can set multiple times.
- port : hook-spray listening address. (default=`:8080`)

Example:

```bash
hook-spray -port=:8080 -dest=http://webhook1 -dest=http://webhook2
```

## Metrics

Prometheus metrics are available at `/metrics`. In addition to the standard Go
runtime and process metrics, hook-spray exports:

- `hook_spray_in_flight_requests`
- `hook_spray_requests_total`
- `hook_spray_request_body_bytes`
- `hook_spray_request_body_read_duration_seconds`
- `hook_spray_request_duration_seconds`
- `hook_spray_upstream_requests_total`
- `hook_spray_upstream_request_duration_seconds`

