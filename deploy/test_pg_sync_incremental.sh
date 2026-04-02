#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SYNC_SCRIPT="${SCRIPT_DIR}/pg_sync_incremental.sh"
STATE_FILE_UNDER_TEST=""
LOCAL_TEST_TABLES=()
REMOTE_TEST_TABLES=()

# shellcheck source=/dev/null
source "${SYNC_SCRIPT}"

assert_eq() {
  local expected="$1"
  local actual="$2"
  local message="$3"
  if [[ "${expected}" != "${actual}" ]]; then
    printf 'ASSERT FAIL: %s\nexpected: %s\nactual:   %s\n' "${message}" "${expected}" "${actual}" >&2
    exit 1
  fi
}

register_local_test_table() {
  LOCAL_TEST_TABLES+=("$1")
}

register_remote_test_table() {
  REMOTE_TEST_TABLES+=("$1")
}

cleanup() {
  local table_name

  for table_name in "${LOCAL_TEST_TABLES[@]:-}"; do
    [[ -n "${table_name}" ]] || continue
    if [[ -n "${PG_SYNC_SOURCE_HOST:-}" ]]; then
      psql_source -c "drop table if exists public.${table_name};" >/dev/null 2>&1 || true
    fi
  done

  for table_name in "${REMOTE_TEST_TABLES[@]:-}"; do
    [[ -n "${table_name}" ]] || continue
    if [[ -n "${PG_SYNC_TARGET_HOST:-}" ]]; then
      psql_target -c "drop table if exists public.${table_name};" >/dev/null 2>&1 || true
    fi
  done

  if [[ -n "${STATE_FILE_UNDER_TEST}" ]]; then
    rm -f "${STATE_FILE_UNDER_TEST}"
  fi
}

assert_successful_copy_pipeline() {
  local source_config_file="/etc/aiapi/config.yaml"
  local test_table_name="sync_copy_test_$$"
  local columns_csv="account_id, group_id, priority, created_at"
  local select_sql
  local insert_sql
  local temp_table_name
  local row_count

  PG_SYNC_SOURCE_HOST="$(yaml_section_value "${source_config_file}" database host)"
  PG_SYNC_SOURCE_PORT="$(yaml_section_value "${source_config_file}" database port)"
  PG_SYNC_SOURCE_USER="$(yaml_section_value "${source_config_file}" database user)"
  PG_SYNC_SOURCE_PASSWORD="$(yaml_section_value "${source_config_file}" database password)"
  PG_SYNC_SOURCE_DB="$(yaml_section_value "${source_config_file}" database dbname)"
  PG_SYNC_SOURCE_SSLMODE="disable"

  PG_SYNC_TARGET_HOST="${PG_SYNC_SOURCE_HOST}"
  PG_SYNC_TARGET_PORT="${PG_SYNC_SOURCE_PORT}"
  PG_SYNC_TARGET_USER="${PG_SYNC_SOURCE_USER}"
  PG_SYNC_TARGET_PASSWORD="${PG_SYNC_SOURCE_PASSWORD}"
  PG_SYNC_TARGET_DB="${PG_SYNC_SOURCE_DB}"
  PG_SYNC_TARGET_SSLMODE="disable"

  psql_source -c "drop table if exists public.${test_table_name}; create table public.${test_table_name} (account_id bigint primary key, group_id bigint not null, priority bigint not null, created_at timestamptz not null); insert into public.${test_table_name} (account_id, group_id, priority, created_at) values (1, 2, 1, '2026-04-02 00:47:45.098984+08'), (3, 2, 1, '2026-03-31 13:42:19.773608+08'), (4, 2, 1, '2026-04-02 00:47:12.817744+08');" >/dev/null
  register_local_test_table "${test_table_name}"

  select_sql="select ${columns_csv} from public.${test_table_name} order by account_id"
  temp_table_name="tmp_sync_${test_table_name}_$$"
  insert_sql="delete from public.${test_table_name}; insert into public.${test_table_name} (${columns_csv}) select ${columns_csv} from ${temp_table_name};"

  run_copy_pipeline "${test_table_name}" "${select_sql}" "${insert_sql}" "${temp_table_name}" "${columns_csv}"

  row_count="$(psql_source -Atqc "select count(*) from public.${test_table_name};")"
  assert_eq "3" "${row_count}" "copy pipeline should preserve all rows"
}

assert_remote_copy_pipeline_when_enabled() {
  local test_table_name="sync_remote_copy_test_$$"
  local columns_csv="account_id, group_id, priority, created_at"
  local select_sql
  local insert_sql
  local temp_table_name
  local row_count

  [[ "${PG_SYNC_ENABLE_REMOTE_INTEGRATION:-0}" == "1" ]] || return 0

  load_env_file

  PG_SYNC_SOURCE_HOST="$(yaml_section_value "/etc/aiapi/config.yaml" database host)"
  PG_SYNC_SOURCE_PORT="$(yaml_section_value "/etc/aiapi/config.yaml" database port)"
  PG_SYNC_SOURCE_USER="$(yaml_section_value "/etc/aiapi/config.yaml" database user)"
  PG_SYNC_SOURCE_PASSWORD="$(yaml_section_value "/etc/aiapi/config.yaml" database password)"
  PG_SYNC_SOURCE_DB="$(yaml_section_value "/etc/aiapi/config.yaml" database dbname)"
  PG_SYNC_SOURCE_SSLMODE="disable"
  normalize_config
  validate_config

  psql_source -c "drop table if exists public.${test_table_name}; create table public.${test_table_name} (account_id bigint primary key, group_id bigint not null, priority bigint not null, created_at timestamptz not null); insert into public.${test_table_name} (account_id, group_id, priority, created_at) values (1, 2, 1, '2026-04-02 00:47:45.098984+08'), (3, 2, 1, '2026-03-31 13:42:19.773608+08'), (4, 2, 1, '2026-04-02 00:47:12.817744+08');" >/dev/null
  psql_target -c "drop table if exists public.${test_table_name}; create table public.${test_table_name} (account_id bigint primary key, group_id bigint not null, priority bigint not null, created_at timestamptz not null);" >/dev/null
  register_local_test_table "${test_table_name}"
  register_remote_test_table "${test_table_name}"

  select_sql="select ${columns_csv} from public.${test_table_name} order by account_id"
  temp_table_name="tmp_sync_${test_table_name}_$$"
  insert_sql="delete from public.${test_table_name}; insert into public.${test_table_name} (${columns_csv}) select ${columns_csv} from ${temp_table_name};"

  run_copy_pipeline "${test_table_name}" "${select_sql}" "${insert_sql}" "${temp_table_name}" "${columns_csv}"

  row_count="$(psql_target -Atqc "select count(*) from public.${test_table_name};")"
  assert_eq "3" "${row_count}" "remote copy pipeline should preserve all rows"
}

assert_reset_sequence_skips_tables_without_id() {
  PG_SYNC_SOURCE_HOST="$(yaml_section_value "/etc/aiapi/config.yaml" database host)"
  PG_SYNC_SOURCE_PORT="$(yaml_section_value "/etc/aiapi/config.yaml" database port)"
  PG_SYNC_SOURCE_USER="$(yaml_section_value "/etc/aiapi/config.yaml" database user)"
  PG_SYNC_SOURCE_PASSWORD="$(yaml_section_value "/etc/aiapi/config.yaml" database password)"
  PG_SYNC_SOURCE_DB="$(yaml_section_value "/etc/aiapi/config.yaml" database dbname)"
  PG_SYNC_SOURCE_SSLMODE="disable"

  PG_SYNC_TARGET_HOST="${PG_SYNC_SOURCE_HOST}"
  PG_SYNC_TARGET_PORT="${PG_SYNC_SOURCE_PORT}"
  PG_SYNC_TARGET_USER="${PG_SYNC_SOURCE_USER}"
  PG_SYNC_TARGET_PASSWORD="${PG_SYNC_SOURCE_PASSWORD}"
  PG_SYNC_TARGET_DB="${PG_SYNC_SOURCE_DB}"
  PG_SYNC_TARGET_SSLMODE="disable"

  reset_table_sequence "account_groups"
}

assert_source_add_column_ddl() {
  PG_SYNC_SOURCE_HOST="$(yaml_section_value "/etc/aiapi/config.yaml" database host)"
  PG_SYNC_SOURCE_PORT="$(yaml_section_value "/etc/aiapi/config.yaml" database port)"
  PG_SYNC_SOURCE_USER="$(yaml_section_value "/etc/aiapi/config.yaml" database user)"
  PG_SYNC_SOURCE_PASSWORD="$(yaml_section_value "/etc/aiapi/config.yaml" database password)"
  PG_SYNC_SOURCE_DB="$(yaml_section_value "/etc/aiapi/config.yaml" database dbname)"
  PG_SYNC_SOURCE_SSLMODE="disable"

  assert_eq \
    "alter table public.accounts add column upstream_provider character varying(50);" \
    "$(source_add_column_ddl "accounts" "upstream_provider")" \
    "source add-column DDL should match source schema"
}

main() {
  local state_file
  state_file="$(mktemp)"
  STATE_FILE_UNDER_TEST="${state_file}"
  trap cleanup EXIT

  init_state_file "${state_file}"

  assert_eq "refresh" "$(table_strategy account_groups)" "account_groups should full refresh"
  assert_eq "upsert_updated_at" "$(table_strategy users)" "users should upsert by updated_at"
  assert_eq "append_id" "$(table_strategy usage_logs)" "usage_logs should append by id"
  assert_eq "refresh" "$(table_strategy usage_dashboard_hourly)" "usage_dashboard_hourly should full refresh"

  assert_eq "" "$(get_state_value "${state_file}" users updated_at)" "missing state should be empty"

  set_state_value "${state_file}" "users" "updated_at" "2026-04-02T00:00:00Z"
  set_state_value "${state_file}" "usage_logs" "id" "12345"

  assert_eq "2026-04-02T00:00:00Z" "$(get_state_value "${state_file}" users updated_at)" "updated_at watermark should persist"
  assert_eq "12345" "$(get_state_value "${state_file}" usage_logs id)" "id watermark should persist"

  assert_successful_copy_pipeline
  assert_remote_copy_pipeline_when_enabled
  assert_reset_sequence_skips_tables_without_id
  assert_source_add_column_ddl

  printf 'deploy/test_pg_sync_incremental.sh: PASS\n'
}

main "$@"
