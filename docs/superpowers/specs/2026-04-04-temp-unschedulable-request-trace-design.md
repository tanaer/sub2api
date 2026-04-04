# Temp Unschedulable Request Trace Design

Date: 2026-04-04

## Summary

This design extends the existing account-management "Temp Unschedulable" status modal so operators can see the exact runtime context that caused the temporary scheduling block:

- platform-internal `request_id`
- actual upstream status code
- upstream error message / detail
- the existing temp-unsched metadata (`status_code`, `matched_keyword`, `until`, `triggered_at`)

The design also adds a direct jump from account management into the existing Ops request-details workflow, filtered by the stored internal `request_id`.

## Goals

- Show the internal platform `request_id` in the account-management temp-unschedulable modal
- Show the actual upstream status code that triggered the temp-unsched state
- Show upstream error content in a readable form inside the modal
- Allow operators to jump directly to the existing Ops request-details view for that request
- Preserve backward compatibility for old temp-unsched records already stored in the database

## Non-Goals

- No new standalone request detail page in account management
- No duplication of Ops request-detail table logic inside account management
- No best-effort lookup by timestamp/account when `request_id` was never persisted
- No migration/backfill job for old `temp_unschedulable_reason` rows
- No attempt to expose provider-side request ids in this iteration

## Current State

### Backend

The existing endpoint `GET /api/v1/admin/accounts/:id/temp-unschedulable` returns:

- `active`
- `state.until_unix`
- `state.triggered_at_unix`
- `state.status_code`
- `state.matched_keyword`
- `state.rule_index`
- `state.error_message`

The state is loaded from:

- Redis temp-unsched cache when present
- `accounts.temp_unschedulable_reason` JSON fallback from database

The runtime state currently does not persist the internal request correlation id or upstream HTTP context.

### Frontend

The existing modal at `frontend/src/components/account/TempUnschedStatusModal.vue` already displays:

- account name
- trigger time
- expiry time
- remaining time
- status code
- matched keyword
- rule order
- error message

It does not currently show:

- internal `request_id`
- actual upstream status code
- structured upstream error text
- direct navigation into Ops request drill-down

## Approved Approach

Reuse the existing temp-unsched modal in account management and the existing Ops request-details modal.

The account-management modal becomes the first-level diagnosis view.
The Ops request-details modal remains the full drill-down target.

This avoids duplicating request-detail UI while keeping the operator flow fast:

1. Open account status
2. See the exact request id and upstream error summary
3. Click through into Ops for the matching request

## Data Model Changes

### TempUnschedState

Extend `backend/internal/service/temp_unsched.go` `TempUnschedState` with the following optional fields:

- `request_id`: internal platform request id from `ctxkey.RequestID`
- `upstream_status_code`: actual upstream HTTP status when available
- `upstream_error_message`: summarized upstream error message
- `upstream_error_detail`: sanitized upstream error detail/body excerpt

Resulting semantics:

- `status_code`: the status code used by temp-unsched / throttle matching logic
- `upstream_status_code`: the real upstream status returned by provider-side call, when known
- `request_id`: internal server-side correlation id for the triggering request

All new fields are optional so old stored JSON remains valid.

### Backward Compatibility

Old rows may contain only the legacy fields.

Behavior for old rows:

- `request_id` renders as empty / `-`
- `upstream_status_code` renders as empty / `-`
- `upstream_error_message` and `upstream_error_detail` render as empty / `-`
- the modal still works without any migration

## Backend Design

### Persist Request Trace At Trigger Time

When temp-unschedulable state is created from runtime error handling:

- read internal request id from `ctx.Value(ctxkey.RequestID)`
- preserve actual upstream status code from the caller path
- preserve upstream error summary/detail from the same response body already being processed

The extended state is then serialized into:

- Redis temp-unsched cache
- `accounts.temp_unschedulable_reason`

### Scope Of Capture

This applies to temp-unsched creation paths that already serialize `TempUnschedState`, including:

- per-account temp-unsched rules
- global account throttle rules
- other temp-unsched producers that already write structured JSON state

If a path cannot provide a value:

- leave that field empty
- do not fabricate data by querying ops tables later

### API Response

Keep the existing endpoint shape:

```json
{
  "active": true,
  "state": {
    "until_unix": 1775318400,
    "triggered_at_unix": 1775271095,
    "status_code": 500,
    "matched_keyword": "Xunfei claude request failed with Sid",
    "rule_index": -1,
    "error_message": "...",
    "request_id": "01f4bd18-9238-427a-8972-4cbd16d09b7b",
    "upstream_status_code": 500,
    "upstream_error_message": "Xunfei claude request failed with Sid ...",
    "upstream_error_detail": "{...}"
  }
}
```

The endpoint contract remains backward compatible because all new fields are additive.

## Frontend Design

### Temp Unschedulable Modal

Enhance `frontend/src/components/account/TempUnschedStatusModal.vue` to display:

- internal `request_id`
- actual upstream status code
- upstream error message
- upstream error detail

Display rules:

- show `-` when the field is absent
- keep the existing cards/grid layout
- use monospace styling for `request_id`
- keep the existing raw `error_message` block for the stored temp-unsched message payload

### Request Actions

For `request_id`:

- add copy action
- add a `查看请求详情` action

The `查看请求详情` action should:

- navigate to the existing Ops dashboard route
- encode query params that instruct the dashboard to auto-open request details
- pass the stored `request_id` as the request filter

If `request_id` is absent:

- hide or disable the jump action
- keep copy action unavailable

### Ops Dashboard Deep Link

Extend `frontend/src/views/admin/ops/OpsDashboard.vue` query-driven state to support:

- opening the request-details modal on page load
- pre-populating `request_id` filter

The existing request-details modal stays the rendering target.
No new duplicate modal is introduced in account management.

## UX Flow

### Happy Path

1. Operator opens account list
2. Operator clicks temp-unsched status
3. Modal shows:
   - trigger time
   - expiry time
   - matching rule info
   - upstream status
   - upstream error summary/detail
   - internal request id
4. Operator clicks `查看请求详情`
5. System navigates to Ops dashboard
6. Request-details modal opens already filtered to that request id

### Missing Trace Data

If the temp-unsched record predates this feature:

1. Operator still sees legacy fields
2. `request_id` and upstream runtime fields show `-`
3. No broken navigation occurs

## Error Handling

### Backend

- Failure to read `ctxkey.RequestID` is non-fatal; store empty string
- Missing upstream status/message/detail is non-fatal
- JSON unmarshal of old reason payload remains supported

### Frontend

- If the temp-unsched status API returns no new fields, render fallback placeholders
- If navigation to Ops occurs but the request is no longer present in the selected time window, the Ops request-details modal shows an empty state instead of erroring

## Testing

### Backend Tests

Add unit coverage for:

- serializing extended `TempUnschedState` with `request_id` and upstream fields
- `GetTempUnschedStatus` returning the additive fields from DB JSON fallback
- temp-unsched creation paths preserving the new fields when provided

### Frontend Tests

Add component tests for:

- rendering `request_id`, upstream status, upstream message, and upstream detail in `TempUnschedStatusModal`
- hiding fallback-only fields correctly when absent
- triggering navigation / deep-link open behavior when `查看请求详情` is clicked

### Integration-Level UI Behavior

Add request-details deep-link coverage for:

- Ops dashboard opening the request-details modal from route query
- request-details modal applying `request_id` filter automatically

## Implementation Notes

- Prefer additive type/interface changes only
- Reuse existing admin ops APIs and modal components
- Do not query ops tables from the temp-unsched endpoint for v1
- Do not change the meaning of existing `status_code` field

## Open Questions Resolved

- Use internal platform `request_id`, not provider request id
- Use direct jump into existing Ops request-details flow, not a second account-management detail screen
- Keep old temp-unsched rows readable without data migration
