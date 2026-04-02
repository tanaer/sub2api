# Provider Config Design

Date: 2026-04-03

## Summary

This design extends existing group-managed `config_templates` so one public group can define:

- structured provider metadata for a new public `GET /api/provider` endpoint
- client-specific configuration examples for the user-side "Use Key" modal
- different config content for `claude_code` and `opencode` under the same group

The design keeps backward compatibility with the current `config_templates` array format and uses existing public groups as the single source of truth.

## Goals

- Generate `GET /api/provider` from existing public groups in Group Management
- Let a group define different config examples for `claude_code` and `opencode`
- Keep current `use_key_instructions` behavior as top-level explanatory copy
- Preserve existing `config_templates` array behavior for old groups
- Avoid adding new database fields for the first version

## Non-Goals

- No visual JSON editor in admin for v1
- No automatic migration of old template data
- No separate provider-specific storage outside groups
- No automatic derivation of provider config from account credentials
- No extra query/filter parameters on `GET /api/provider` in v1

## Source Of Truth

Provider configuration is derived from existing groups that satisfy:

- `status = active`
- `is_exclusive = false`

Only groups with valid structured provider config participate in `GET /api/provider`.

This keeps one configuration source for:

- group management
- user-side "Use Key" modal
- public provider discovery

## Data Model

The existing `groups.config_templates` text field supports two formats:

### Legacy Format

```json
[
  {
    "filename": "settings.json",
    "content": "{...}"
  }
]
```

This remains supported for the "Use Key" modal and is ignored by `GET /api/provider`.

### Structured Format

```json
{
  "version": 1,
  "provider": {
    "id": "default",
    "name": "默认分组",
    "description": "适用于 Claude Code 和 OpenCode",
    "recommended": true,
    "anthropic_base_url": "{{BASE_URL}}",
    "opencode_base_url": "{{BASE_URL}}/v1",
    "model_mapping": {
      "claude-opus-4.6": "glm-5",
      "claude-sonnet-4.6": "glm-4.7",
      "claude-haiku-4-5-20251001": "glm-4.5-air"
    }
  },
  "clients": {
    "claude_code": {
      "files": [
        {
          "path": "~/.claude/settings.json",
          "language": "json",
          "content": "{\n  \"env\": {\n    \"ANTHROPIC_BASE_URL\": \"{{ANTHROPIC_BASE_URL}}\",\n    \"ANTHROPIC_API_KEY\": \"{{API_KEY}}\",\n    \"ANTHROPIC_DEFAULT_HAIKU_MODEL\": \"glm-4.5-air\",\n    \"ANTHROPIC_DEFAULT_SONNET_MODEL\": \"glm-4.7\",\n    \"ANTHROPIC_DEFAULT_OPUS_MODEL\": \"glm-5\"\n  }\n}"
        }
      ]
    },
    "opencode": {
      "files": [
        {
          "path": "~/.config/opencode/opencode.json",
          "language": "json",
          "content": "{\n  \"name\": \"{{CHANNEL}}\",\n  \"npm\": \"@ai-sdk/anthropic\",\n  \"options\": {\n    \"apiKey\": \"{{API_KEY}}\",\n    \"baseURL\": \"{{OPENCODE_BASE_URL}}\"\n  },\n  \"models\": {\n    \"glm-4.5-air\": {\n      \"name\": \"glm-4.5-air\",\n      \"options\": {\n        \"store\": false\n      },\n      \"variants\": {\n        \"low\": {},\n        \"medium\": {},\n        \"high\": {},\n        \"xhigh\": {}\n      }\n    },\n    \"glm-4.7\": {\n      \"name\": \"glm-4.7\",\n      \"options\": {\n        \"store\": false\n      },\n      \"variants\": {\n        \"low\": {},\n        \"medium\": {},\n        \"high\": {},\n        \"xhigh\": {}\n      }\n    },\n    \"glm-5\": {\n      \"name\": \"glm-5\",\n      \"options\": {\n        \"store\": false\n      },\n      \"variants\": {\n        \"low\": {},\n        \"medium\": {},\n        \"high\": {},\n        \"xhigh\": {}\n      }\n    }\n  }\n}"
        }
      ]
    }
  }
}
```

## Structured Format Semantics

### Root

- `version`: required, must be `1`
- `provider`: optional for modal-only use, required for inclusion in `GET /api/provider`
- `clients`: optional, used by the user-side "Use Key" modal

### Provider Object

- `id`: preferred stable identifier for public provider group entry
- `name`: display name for public group entry
- `description`: display description for public group entry
- `recommended`: optional boolean, defaults to `false`
- `anthropic_base_url`: optional string with placeholders
- `opencode_base_url`: optional string with placeholders
- `model_mapping`: required object when `provider` exists

Fallback behavior:

- `id`: fallback to `group-{group_id}`
- `name`: fallback to group name
- `description`: fallback to group description
- `recommended`: fallback to `false`

### Clients Object

Supported keys in v1:

- `claude_code`
- `opencode`
- `codex`
- `gemini`

Each client contains:

```json
{
  "files": [
    {
      "path": "~/.config/example.json",
      "language": "json",
      "content": "{...}"
    }
  ]
}
```

Rules:

- `files` is required when a client exists
- each file requires `path` and `content`
- `language` is optional and used for frontend display only

## Placeholder Expansion

Both modal rendering and provider generation support placeholder replacement.

Supported placeholders in v1:

- `{{API_KEY}}`
- `{{BASE_URL}}`
- `{{ANTHROPIC_BASE_URL}}`
- `{{OPENCODE_BASE_URL}}`
- `{{CHANNEL}}`
- `{{GROUP_NAME}}`

Expansion sources:

- `API_KEY`: selected API key
- `BASE_URL`: current public API base URL or current origin fallback
- `ANTHROPIC_BASE_URL`: resolved provider value or base URL fallback
- `OPENCODE_BASE_URL`: resolved provider value or `BASE_URL + "/v1"` fallback
- `CHANNEL`: current request host or derived public API host
- `GROUP_NAME`: group name

## Public API

### Endpoint

- `GET /api/provider`

### Selection Rules

- include only groups with `status = active`
- include only groups with `is_exclusive = false`
- include only groups whose `config_templates` is valid structured format with a `provider` object

### Ordering

- `sort_order ASC`
- `id ASC`

### Response Shape

```json
{
  "version": 1,
  "channel": "aiapi.muskpay.top",
  "updated_at": "2026-04-03T12:00:00Z",
  "groups": [
    {
      "id": "default",
      "name": "默认分组",
      "description": "适用于 Claude Code 和 OpenCode",
      "recommended": true,
      "anthropic_base_url": "https://aiapi.muskpay.top",
      "opencode_base_url": "https://aiapi.muskpay.top/v1",
      "model_mapping": {
        "claude-opus-4.6": "glm-5",
        "claude-sonnet-4.6": "glm-4.7",
        "claude-haiku-4-5-20251001": "glm-4.5-air"
      }
    }
  ]
}
```

### Derived Fields

- `channel`: first non-empty of `X-Forwarded-Host`, request `Host`, host parsed from public `api_base_url`
- `updated_at`: max `group.updated_at` among returned groups, serialized as UTC RFC3339

### Error Contract

For service failure, empty usable group set, or invalid usable provider set:

```json
{
  "version": 1,
  "error": {
    "code": "UNAVAILABLE",
    "message": "provider list unavailable"
  }
}
```

Recommended status code:

- `503 Service Unavailable`

## User-Side "Use Key" Modal

### Current Behavior To Preserve

- `use_key_instructions` remains the top description block
- old array-style `config_templates` remains supported
- built-in default examples remain available as fallback

### New Behavior

When `config_templates` is structured format:

- resolve `clients.claude_code` for the Claude Code tab
- resolve `clients.opencode` for the OpenCode tab
- resolve `clients.codex` for the Codex tab
- resolve `clients.gemini` for the Gemini tab

If a tab has custom files:

- show those files
- do not show built-in defaults for that tab

If a tab has no custom files:

- fall back to existing built-in generation for that tab

This allows partial overrides. A group can customize only `claude_code` and `opencode` while leaving other tabs unchanged.

## Admin UX

The existing `config_templates` textarea in Group Management remains the entry point.

Text updates for v1:

- label changes from "配置文件模板" to "配置映射示例"
- hint explains:
  - legacy array format is still supported
  - structured `{ version, provider, clients }` format is supported
  - structured format drives both the "Use Key" modal and `GET /api/provider`

No visual editor is added in v1.

## Validation

Validation occurs on create/update group requests.

Accepted cases:

- empty string
- valid legacy array format
- valid structured format

Rejected cases:

- invalid JSON
- structured format with `version != 1`
- `provider.model_mapping` missing or not an object when `provider` exists
- client object without `files`
- file entry missing `path` or `content`

Example error messages:

- `config_templates must be valid JSON`
- `config_templates.version must equal 1`
- `config_templates.provider.model_mapping must be an object`
- `config_templates.clients.opencode.files[0].content is required`

## Implementation Plan Boundary

V1 implementation should include:

1. shared parser/validator for legacy and structured `config_templates`
2. group create/update validation for structured format
3. modal support for client-specific files with placeholder expansion
4. public `GET /api/provider` endpoint
5. tests for parsing, validation, modal behavior, success response, and unavailable response

V1 should not include:

- schema migration
- visual JSON editor
- automatic data migration
- provider endpoint filters
- account-derived mapping generation

## Testing Requirements

Backend:

- create/update group accepts valid structured format
- create/update group rejects malformed structured format
- `GET /api/provider` returns ordered public groups with expanded fields
- `GET /api/provider` returns `503` unavailable payload when no usable groups exist

Frontend:

- modal parses structured format
- `claude_code` tab uses custom files when present
- `opencode` tab uses custom files when present
- tabs without structured client config still use built-in defaults
- legacy array format still renders correctly

## Open Questions Resolved

- Provider source: existing public groups only
- Storage: reuse `config_templates`
- Public group rule: active and non-exclusive
- Error contract: single unavailable payload
- Client-specific config: structured `clients` object
