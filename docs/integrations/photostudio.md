# PhotoStudio integration

## Overview

After a user registers in MWork, the service triggers an async sync request to PhotoStudio. Failures are logged but do not block registration.

## Payload

The sync payload is a JSON object with the following fields:

```json
{
  "mwork_user_id": "<uuid>",
  "email": "user@example.com",
  "role": "model"
}
```

Supported roles: `model`, `employer`, `agency`, `admin`.

## Environment variables

Set the following environment variables to configure the PhotoStudio integration:

- `PHOTOSTUDIO_BASE_URL` — Base URL for the PhotoStudio API. Leave empty to disable the integration.
- `PHOTOSTUDIO_TOKEN` — API token for authenticating requests to PhotoStudio.
- `PHOTOSTUDIO_SYNC_ENABLED` — Feature flag to enable background sync jobs (`true`/`false`).
- `PHOTOSTUDIO_TIMEOUT_SECONDS` — Request timeout in seconds (recommended 5–10 seconds).

## Docker / network example

If PhotoStudio is running inside Docker Compose as the `photostudio` service on port `8090`, use:

```
PHOTOSTUDIO_BASE_URL=http://photostudio:8090
```

## Troubleshooting

- Check logs for `photostudio sync ok` and `photostudio sync failed` messages.
- `401 Unauthorized`: ensure `PHOTOSTUDIO_TOKEN` is correct and has access.
- `connection refused`: PhotoStudio is down or not reachable at `PHOTOSTUDIO_BASE_URL`.
- `timeout`: increase `PHOTOSTUDIO_TIMEOUT_SECONDS` or verify network latency.

## Manual resync (backfill)

Use the admin endpoint to trigger a backfill batch. The endpoint is protected by admin auth and requires the `users.view` permission.

```
POST /api/v1/admin/photostudio/resync?limit=500&offset=0
```

Notes:
- Batches are processed with rate limiting (~5 requests per second).
- The response includes `processed`, `success`, and `failed` counts.
- Errors are logged and counted as failures, but do not stop the batch.
