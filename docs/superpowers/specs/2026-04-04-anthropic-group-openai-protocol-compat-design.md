# Anthropic Group OpenAI Protocol Compatibility Design

Date: 2026-04-04

## Summary

This design standardizes protocol compatibility for Anthropic groups without changing the group platform model.

An API key bound to an Anthropic group should be able to use the three common client-facing ingress styles:

- `POST /v1/messages`
- `POST /v1/chat/completions`
- `POST /v1/responses`

All three ingress paths continue to schedule only Anthropic-compatible capacity for that group:

- Anthropic accounts in the group
- existing mixed-scheduling Antigravity accounts already allowed by current Anthropic scheduling rules

This is explicitly a protocol-compatibility feature, not a multi-platform group feature.

## Goals

- Reduce user friction when the real backend group platform is Anthropic but clients send OpenAI-style requests
- Make Anthropic groups accept common OpenAI-compatible HTTP request formats by default
- Preserve the current single-platform group scheduling model
- Keep platform selection based on the bound group, not on the client protocol shape
- Prefer compatibility and successful execution over strict parameter rejection
- Support both non-streaming and SSE streaming for HTTP `POST /v1/chat/completions` and `POST /v1/responses`

## Non-Goals

- No true multi-platform group model
- No change to `group.platform` from single value to list/set
- No scheduler redesign to select OpenAI accounts for Anthropic groups
- No requirement that Anthropic groups support the OpenAI Responses WebSocket ingress in this iteration
- No user-facing configuration switch required to enable Anthropic-group protocol compatibility

## Current State

### Group Platform Semantics

`group.platform` is currently a single-valued scheduling attribute, not a display-only label.

Scheduling, account filtering, and mixed scheduling behavior all depend directly on the group platform value.
Because of that, a "one group supports multiple upstream platforms" model would require a broader architectural change than the current problem statement justifies.

### Admin UI Behavior

The admin UI currently disables editing group platform after creation.
However, backend update logic still accepts and persists a non-empty platform update.

This means "platform not editable" is currently a frontend policy rather than a hard backend invariant.
That inconsistency is noted here for context, but it is not the target of this design.

### Existing Compatibility Work In Repository

The current repository already contains most of the Anthropic-side compatibility path:

- route-level auto dispatch for `POST /v1/messages`, `POST /v1/chat/completions`, and `POST /v1/responses`
- Anthropic-side handlers for Chat Completions and Responses ingress
- conversion chains:
  - Chat Completions -> Responses -> Anthropic
  - Responses -> Anthropic

This means the desired behavior is already partially implemented in the current codebase.
The main product/design task is to formalize the target behavior, verify ingress routing end-to-end, close remaining gaps, and add regression coverage.

### Likely Production Symptom

If an Anthropic-group request still receives an error such as "use /v1/messages instead", the request is likely entering the OpenAI handler/account-selection path rather than the Anthropic compatibility path.

Operationally, that points to one of these causes:

- deployed version is older than current repository head
- an alternate ingress path bypasses the current route dispatch logic
- route registration or proxying differs between environments

## Approved Approach

Keep the current single-platform group model and formalize "multi-protocol ingress compatibility" for Anthropic groups.

For Anthropic groups:

- `POST /v1/messages` remains the native Claude-compatible ingress
- `POST /v1/chat/completions` is treated as a compatibility ingress and routed into the Anthropic compatibility chain
- `POST /v1/responses` is treated as a compatibility ingress and routed into the Anthropic compatibility chain

Exception:
if a group is explicitly configured as Claude-Code-only, that restriction remains authoritative and non-`/v1/messages` ingress may still be rejected for that group.

The client protocol does not change the group's scheduling platform.
The bound group platform remains the source of truth for scheduling.

This is the smallest change surface that matches the product goal:

1. users can keep their existing Anthropic-bound API keys
2. users can send the most common protocol variants
3. the backend still schedules only Anthropic-compatible capacity
4. no group-data migration is required

## Target Behavior

### Ingress Rules

For an API key bound to an Anthropic group:

- `POST /v1/messages` -> Anthropic native handler
- `POST /v1/chat/completions` -> Anthropic compatibility handler
- `POST /v1/responses` -> Anthropic compatibility handler

Explicit exception:
when the bound group is marked Claude-Code-only, compatibility ingress does not override that product restriction.

For an API key bound to an OpenAI group:

- existing OpenAI behavior remains unchanged

For compatibility aliases without `/v1` prefix:

- `/chat/completions`
- `/responses`

they should follow the same group-platform-based routing rule as the `/v1/...` forms.

### Scheduling Rules

Anthropic-group compatibility ingress must continue to use Anthropic-group scheduling rules only.

That means:

- use Anthropic accounts in the group
- allow existing mixed-scheduling Antigravity accounts where the current scheduler already allows them
- do not switch to OpenAI account selection merely because the ingress is OpenAI-shaped

### Protocol Conversion Rules

For Anthropic-group `POST /v1/chat/completions`:

1. parse Chat Completions request
2. convert to Responses request shape
3. convert to Anthropic request shape
4. forward to Anthropic upstream
5. convert upstream result back to Chat Completions response shape

For Anthropic-group `POST /v1/responses`:

1. parse Responses request
2. convert to Anthropic request shape
3. forward to Anthropic upstream
4. convert upstream result back to Responses response shape

### Streaming Scope

This iteration formally supports:

- non-streaming `POST /v1/chat/completions`
- SSE streaming `POST /v1/chat/completions` with `stream=true`
- non-streaming `POST /v1/responses`
- SSE streaming `POST /v1/responses` with `stream=true`

This iteration does not require full Anthropic-group compatibility for:

- `GET /responses` WebSocket ingress

Reason:
the current codebase uses the OpenAI Responses WebSocket ingress handler for `GET /responses`, and that path is a separate transport/proxy design from the Anthropic HTTP compatibility handlers.

If WebSocket compatibility for Anthropic groups is needed later, it should be treated as a separate enhancement with explicit transport-level design and tests.

## Compatibility Policy

### Philosophy

Compatibility should be "wide accept, safe degrade".

The platform should prefer:

- accepting common client request formats
- preserving the core request intent
- ignoring unsupported non-critical fields

over:

- rejecting requests early because a field does not map perfectly

### Request Validation

Hard request rejection remains appropriate only for structurally invalid requests, for example:

- empty request body
- invalid JSON
- missing `model`
- message/input structure so malformed that user intent cannot be derived

### Field Mapping Policy

For Anthropic-group compatibility ingress:

- fields with stable mappings should be converted normally
- fields that are non-critical and cannot be mapped reliably should be ignored by default
- unsupported fields should not automatically produce protocol-guidance errors
- protocol differences should be treated as compatibility concerns first, not as fatal validation errors

### Error Messaging Policy

Do not return user guidance that effectively says:

- "this Anthropic group only supports `/v1/messages`"

when the product behavior is intended to support compatible OpenAI-style HTTP ingress.

Real runtime failures should still surface normally, including:

- authentication failure
- billing or balance failure
- request concurrency limits
- upstream account unavailability
- provider-side upstream failures

If a parameter truly cannot be honored and proceeding would create materially wrong behavior, return a normal API error for that parameter.
Do not redirect the user to a different protocol unless that protocol change is the only valid recovery path.

## Observability And Traceability

Compatibility must stay transparent to operators even when it is invisible to users.

Request tracing should preserve:

- inbound endpoint actually used by the client, such as `/v1/chat/completions`
- resolved group platform, such as `anthropic`
- upstream endpoint actually used internally
- selected account platform and account id

This lets support and operators answer both questions at once:

- what protocol did the client send?
- what upstream platform and path did the gateway actually use?

## Testing Strategy

### Routing Tests

Add regression tests that prove:

- Anthropic-group `POST /v1/chat/completions` reaches the Anthropic compatibility handler
- Anthropic-group `POST /v1/responses` reaches the Anthropic compatibility handler
- OpenAI-group behavior remains unchanged
- alias routes without `/v1` follow the same dispatch rule

### Compatibility Tests

Add tests that prove:

- Anthropic-group Chat Completions requests convert and forward successfully
- Anthropic-group Responses requests convert and forward successfully
- non-critical unsupported fields are ignored rather than rejected where intended

### Streaming Tests

Add tests that prove:

- Anthropic-group Chat Completions SSE streaming works
- Anthropic-group Responses SSE streaming works
- stream and non-stream behavior do not accidentally diverge in handler selection

### Negative And Regression Tests

Add tests that prove:

- OpenAI groups still use OpenAI handlers
- Anthropic-group compatibility requests do not fall back into OpenAI account selection
- existing `/v1/messages` behavior is unchanged
- Claude-code-only restrictions continue to apply where intended

## Rollout Notes

This design is intentionally low-risk because it does not change:

- group schema
- scheduler ownership model
- account platform semantics

Implementation should focus on:

1. verifying all ingress paths in the actual deployed routing stack
2. correcting any path that still enters OpenAI selection for Anthropic groups
3. expanding automated tests around handler dispatch and streaming behavior
4. documenting this as supported product behavior

## Success Criteria

The feature is considered complete when all of the following are true:

- a key bound to an Anthropic group can successfully call:
  - `POST /v1/messages`
  - `POST /v1/chat/completions`
  - `POST /v1/responses`
- both non-streaming and SSE streaming work for the two OpenAI-style HTTP endpoints above
- users do not need to rebind keys or create OpenAI groups just to use OpenAI-style HTTP clients
- Anthropic-group compatibility requests stay on Anthropic-compatible scheduling
- OpenAI-group behavior does not regress
