#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DEFAULT_ENV_FILE="${SCRIPT_DIR}/.pg_sync.env"
DEFAULT_STATE_FILE="${SCRIPT_DIR}/.pg_sync_state.env"
DEFAULT_SOURCE_CONFIG_FILE="/etc/aiapi/config.yaml"
DEFAULT_TARGET_ADMIN_DB="postgres"
DEFAULT_OVERLAP_MINUTES="5"

ENV_FILE="${PG_SYNC_ENV_FILE:-${DEFAULT_ENV_FILE}}"
STATE_FILE="${PG_SYNC_STATE_FILE:-${DEFAULT_STATE_FILE}}"
SOURCE_CONFIG_FILE="${PG_SYNC_SOURCE_CONFIG_FILE:-${DEFAULT_SOURCE_CONFIG_FILE}}"
TARGET_ADMIN_DB="${PG_SYNC_TARGET_ADMIN_DB:-${DEFAULT_TARGET_ADMIN_DB}}"
OVERLAP_MINUTES="${PG_SYNC_OVERLAP_MINUTES:-${DEFAULT_OVERLAP_MINUTES}}"

DRY_RUN=0
INIT_ONLY=0
SYNC_ONLY=0
REBUILD_STATE_ONLY=0

log() {
  printf '[%s] %s\n' "$(date '+%Y-%m-%d %H:%M:%S')" "$*" >&2
}

die() {
  log "ERROR: $*"
  exit 1
}

require_command() {
  local command_name="$1"
  command -v "${command_name}" >/dev/null 2>&1 || die "缺少命令: ${command_name}"
}

# 状态文件只做简单键值存储，避免把同步水位写进业务库。
init_state_file() {
  local state_file="$1"
  mkdir -p "$(dirname "${state_file}")"
  touch "${state_file}"
}

state_key() {
  local table_name="$1"
  local field_name="$2"
  printf '%s__%s' "${table_name}" "${field_name}"
}

get_state_value() {
  local state_file="$1"
  local table_name="$2"
  local field_name="$3"
  local key

  key="$(state_key "${table_name}" "${field_name}")"
  [[ -f "${state_file}" ]] || return 0

  awk -F= -v key="${key}" '
    $1 == key {
      print substr($0, index($0, "=") + 1)
      found = 1
    }
    END {
      if (!found) {
        exit 0
      }
    }
  ' "${state_file}" | tail -n 1
}

set_state_value() {
  local state_file="$1"
  local table_name="$2"
  local field_name="$3"
  local value="$4"
  local key
  local tmp_file

  key="$(state_key "${table_name}" "${field_name}")"
  tmp_file="$(mktemp)"
  init_state_file "${state_file}"

  awk -F= -v key="${key}" '$1 != key { print $0 }' "${state_file}" >"${tmp_file}"
  printf '%s=%s\n' "${key}" "${value}" >>"${tmp_file}"
  mv "${tmp_file}" "${state_file}"
}

load_env_file() {
  if [[ -f "${ENV_FILE}" ]]; then
    # shellcheck disable=SC1090
    source "${ENV_FILE}"
  fi
}

yaml_section_value() {
  local file_path="$1"
  local section_name="$2"
  local key_name="$3"

  [[ -f "${file_path}" ]] || return 0

  awk -v section_name="${section_name}" -v key_name="${key_name}" '
    $0 ~ "^[[:space:]]*" section_name ":[[:space:]]*$" {
      in_section = 1
      next
    }

    in_section && $0 ~ "^[^[:space:]]" {
      in_section = 0
    }

    in_section {
      line = $0
      sub(/^[[:space:]]+/, "", line)
      if (line ~ "^" key_name ":[[:space:]]*") {
        sub("^" key_name ":[[:space:]]*", "", line)
        gsub(/^"/, "", line)
        gsub(/"$/, "", line)
        gsub(/^'\''/, "", line)
        gsub(/'\''$/, "", line)
        print line
        exit
      }
    }
  ' "${file_path}"
}

apply_source_defaults_from_config() {
  [[ -f "${SOURCE_CONFIG_FILE}" ]] || return 0

  : "${PG_SYNC_SOURCE_HOST:=$(yaml_section_value "${SOURCE_CONFIG_FILE}" database host)}"
  : "${PG_SYNC_SOURCE_PORT:=$(yaml_section_value "${SOURCE_CONFIG_FILE}" database port)}"
  : "${PG_SYNC_SOURCE_USER:=$(yaml_section_value "${SOURCE_CONFIG_FILE}" database user)}"
  : "${PG_SYNC_SOURCE_PASSWORD:=$(yaml_section_value "${SOURCE_CONFIG_FILE}" database password)}"
  : "${PG_SYNC_SOURCE_DB:=$(yaml_section_value "${SOURCE_CONFIG_FILE}" database dbname)}"
  : "${PG_SYNC_SOURCE_SSLMODE:=$(yaml_section_value "${SOURCE_CONFIG_FILE}" database sslmode)}"
}

normalize_config() {
  : "${PG_SYNC_SOURCE_PORT:=5432}"
  : "${PG_SYNC_SOURCE_SSLMODE:=disable}"
  : "${PG_SYNC_TARGET_PORT:=5432}"
  : "${PG_SYNC_TARGET_DB:=aiapi}"
  : "${PG_SYNC_TARGET_SSLMODE:=prefer}"
  : "${PG_SYNC_TARGET_ADMIN_DB:=${TARGET_ADMIN_DB}}"
  : "${PG_SYNC_STATE_FILE:=${STATE_FILE}}"
}

validate_config() {
  [[ -n "${PG_SYNC_SOURCE_HOST:-}" ]] || die "未提供 PG_SYNC_SOURCE_HOST，且无法从 ${SOURCE_CONFIG_FILE} 读取"
  [[ -n "${PG_SYNC_SOURCE_USER:-}" ]] || die "未提供 PG_SYNC_SOURCE_USER"
  [[ -n "${PG_SYNC_SOURCE_PASSWORD:-}" ]] || die "未提供 PG_SYNC_SOURCE_PASSWORD"
  [[ -n "${PG_SYNC_SOURCE_DB:-}" ]] || die "未提供 PG_SYNC_SOURCE_DB"
  [[ -n "${PG_SYNC_TARGET_HOST:-}" ]] || die "未提供 PG_SYNC_TARGET_HOST"
  [[ -n "${PG_SYNC_TARGET_USER:-}" ]] || die "未提供 PG_SYNC_TARGET_USER"
  [[ -n "${PG_SYNC_TARGET_PASSWORD:-}" ]] || die "未提供 PG_SYNC_TARGET_PASSWORD"
  [[ -n "${PG_SYNC_TARGET_DB:-}" ]] || die "未提供 PG_SYNC_TARGET_DB"
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --env-file)
        ENV_FILE="$2"
        shift 2
        ;;
      --state-file)
        STATE_FILE="$2"
        shift 2
        ;;
      --source-config)
        SOURCE_CONFIG_FILE="$2"
        shift 2
        ;;
      --dry-run)
        DRY_RUN=1
        shift
        ;;
      --init-only)
        INIT_ONLY=1
        shift
        ;;
      --sync-only)
        SYNC_ONLY=1
        shift
        ;;
      --rebuild-state)
        REBUILD_STATE_ONLY=1
        shift
        ;;
      *)
        die "不支持的参数: $1"
        ;;
    esac
  done

  if [[ "${INIT_ONLY}" -eq 1 && "${SYNC_ONLY}" -eq 1 ]]; then
    die "--init-only 与 --sync-only 不能同时使用"
  fi
}

run_or_echo() {
  if [[ "${DRY_RUN}" -eq 1 ]]; then
    log "DRY RUN: $*"
  else
    "$@"
  fi
}

psql_source() {
  PGPASSWORD="${PG_SYNC_SOURCE_PASSWORD}" \
  PGSSLMODE="${PG_SYNC_SOURCE_SSLMODE}" \
    psql -X -v ON_ERROR_STOP=1 \
      -h "${PG_SYNC_SOURCE_HOST}" \
      -p "${PG_SYNC_SOURCE_PORT}" \
      -U "${PG_SYNC_SOURCE_USER}" \
      -d "${PG_SYNC_SOURCE_DB}" \
      "$@"
}

pg_dump_source() {
  PGPASSWORD="${PG_SYNC_SOURCE_PASSWORD}" \
  PGSSLMODE="${PG_SYNC_SOURCE_SSLMODE}" \
    pg_dump \
      -h "${PG_SYNC_SOURCE_HOST}" \
      -p "${PG_SYNC_SOURCE_PORT}" \
      -U "${PG_SYNC_SOURCE_USER}" \
      -d "${PG_SYNC_SOURCE_DB}" \
      "$@"
}

psql_target() {
  PGPASSWORD="${PG_SYNC_TARGET_PASSWORD}" \
  PGSSLMODE="${PG_SYNC_TARGET_SSLMODE}" \
    psql -X -v ON_ERROR_STOP=1 \
      -h "${PG_SYNC_TARGET_HOST}" \
      -p "${PG_SYNC_TARGET_PORT}" \
      -U "${PG_SYNC_TARGET_USER}" \
      -d "${PG_SYNC_TARGET_DB}" \
      "$@"
}

psql_target_admin() {
  PGPASSWORD="${PG_SYNC_TARGET_PASSWORD}" \
  PGSSLMODE="${PG_SYNC_TARGET_SSLMODE}" \
    psql -X -v ON_ERROR_STOP=1 \
      -h "${PG_SYNC_TARGET_HOST}" \
      -p "${PG_SYNC_TARGET_PORT}" \
      -U "${PG_SYNC_TARGET_USER}" \
      -d "${PG_SYNC_TARGET_ADMIN_DB}" \
      "$@"
}

sql_escape_literal() {
  local value="$1"
  printf "%s" "${value//\'/\'\'}"
}

table_strategy() {
  local table_name="$1"

  case "${table_name}" in
    account_groups|atlas_schema_revisions|ops_metrics_daily|ops_metrics_hourly|schema_migrations|usage_dashboard_daily|usage_dashboard_daily_users|usage_dashboard_hourly|usage_dashboard_hourly_users|user_allowed_groups)
      printf 'refresh\n'
      ;;
    accounts|announcements|api_keys|error_passthrough_rules|groups|idempotency_records|ops_alert_rules|ops_job_heartbeats|promo_codes|proxies|scheduled_test_plans|security_secrets|settings|sora_accounts|tls_fingerprint_profiles|usage_cleanup_tasks|usage_dashboard_aggregation_watermark|user_attribute_definitions|user_attribute_values|user_group_rate_multipliers|user_group_request_quota_grants|user_group_request_quotas|user_subscriptions|users)
      printf 'upsert_updated_at\n'
      ;;
    announcement_reads|billing_usage_entries|ops_alert_events|ops_error_logs|ops_retry_attempts|ops_system_log_cleanup_audits|ops_system_logs|ops_system_metrics|orphan_allowed_groups_audit|promo_code_usages|redeem_codes|scheduled_test_results|scheduler_outbox|sora_generations|usage_billing_dedup|usage_logs)
      printf 'append_id\n'
      ;;
    usage_billing_dedup_archive)
      printf 'append_created_at\n'
      ;;
    *)
      printf 'unknown\n'
      ;;
  esac
}

watermark_column_for_strategy() {
  local strategy="$1"

  case "${strategy}" in
    upsert_updated_at)
      printf 'updated_at\n'
      ;;
    append_id)
      printf 'id\n'
      ;;
    append_created_at)
      printf 'created_at\n'
      ;;
    *)
      printf '\n'
      ;;
  esac
}

list_source_tables() {
  psql_source -Atqc "select table_name from information_schema.tables where table_schema = 'public' order by table_name;"
}

table_exists_in_target() {
  local table_name="$1"

  psql_target -Atqc "select count(*) from information_schema.tables where table_schema = 'public' and table_name = '${table_name}';"
}

table_columns_csv() {
  local table_name="$1"

  psql_source -Atqc "
    select string_agg(format('%I', column_name), ', ' order by ordinal_position)
    from information_schema.columns
    where table_schema = 'public' and table_name = '${table_name}';
  "
}

table_primary_key_csv() {
  local table_name="$1"

  psql_source -Atqc "
    select string_agg(format('%I', kcu.column_name), ', ' order by kcu.ordinal_position)
    from information_schema.table_constraints tc
    join information_schema.key_column_usage kcu
      on tc.constraint_name = kcu.constraint_name
     and tc.table_schema = kcu.table_schema
    where tc.constraint_type = 'PRIMARY KEY'
      and tc.table_schema = 'public'
      and tc.table_name = '${table_name}';
  "
}

table_update_assignments_csv() {
  local table_name="$1"

  psql_source -Atqc "
    with pk_columns as (
      select kcu.column_name
      from information_schema.table_constraints tc
      join information_schema.key_column_usage kcu
        on tc.constraint_name = kcu.constraint_name
       and tc.table_schema = kcu.table_schema
      where tc.constraint_type = 'PRIMARY KEY'
        and tc.table_schema = 'public'
        and tc.table_name = '${table_name}'
    )
    select string_agg(format('%1\$I = excluded.%1\$I', column_name), ', ' order by ordinal_position)
    from information_schema.columns
    where table_schema = 'public'
      and table_name = '${table_name}'
      and column_name not in (select column_name from pk_columns);
  "
}

source_add_column_ddl() {
  local table_name="$1"
  local column_name="$2"

  psql_source -Atqc "
    select format(
      'alter table public.%I add column %I %s%s%s;',
      c.relname,
      a.attname,
      pg_catalog.format_type(a.atttypid, a.atttypmod),
      case
        when ad.adbin is not null then ' default ' || pg_get_expr(ad.adbin, ad.adrelid)
        else ''
      end,
      case
        when a.attnotnull then ' not null'
        else ''
      end
    )
    from pg_attribute a
    join pg_class c
      on c.oid = a.attrelid
    join pg_namespace n
      on n.oid = c.relnamespace
    left join pg_attrdef ad
      on ad.adrelid = a.attrelid
     and ad.adnum = a.attnum
    where n.nspname = 'public'
      and c.relname = '${table_name}'
      and a.attname = '${column_name}'
      and a.attnum > 0
      and not a.attisdropped;
  "
}

missing_columns_for_table() {
  local table_name="$1"
  local src_file
  local tgt_file

  src_file="$(mktemp)"
  tgt_file="$(mktemp)"

  psql_source -Atqc "
    select column_name
    from information_schema.columns
    where table_schema = 'public'
      and table_name = '${table_name}'
    order by column_name;
  " >"${src_file}"

  psql_target -Atqc "
    select column_name
    from information_schema.columns
    where table_schema = 'public'
      and table_name = '${table_name}'
    order by column_name;
  " >"${tgt_file}"

  comm -23 "${src_file}" "${tgt_file}" | sed '/^$/d'
  rm -f "${src_file}" "${tgt_file}"
}

reconcile_table_schema() {
  local table_name="$1"
  local column_name
  local ddl

  while IFS= read -r column_name; do
    [[ -n "${column_name}" ]] || continue
    ddl="$(source_add_column_ddl "${table_name}" "${column_name}")"
    [[ -n "${ddl}" ]] || die "无法生成补列 DDL: ${table_name}.${column_name}"

    log "补齐远端列 ${table_name}.${column_name}"
    if [[ "${DRY_RUN}" -eq 1 ]]; then
      log "DRY RUN DDL: ${ddl}"
    else
      psql_target -c "${ddl}" >/dev/null
    fi
  done < <(missing_columns_for_table "${table_name}")
}

max_value_from_db() {
  local which_db="$1"
  local table_name="$2"
  local column_name="$3"
  local sql

  sql="select coalesce(max(${column_name})::text, '') from public.${table_name};"

  if [[ "${which_db}" == "source" ]]; then
    psql_source -Atqc "${sql}"
  else
    if [[ "$(table_exists_in_target "${table_name}")" == "0" ]]; then
      printf '\n'
    else
      psql_target -Atqc "${sql}"
    fi
  fi
}

target_database_exists() {
  psql_target_admin -Atqc "select count(*) from pg_database where datname = '${PG_SYNC_TARGET_DB}';"
}

target_public_table_count() {
  if [[ "$(target_database_exists)" == "0" ]]; then
    printf '0\n'
  else
    psql_target -Atqc "select count(*) from information_schema.tables where table_schema = 'public';"
  fi
}

ensure_target_database() {
  if [[ "$(target_database_exists)" != "0" ]]; then
    return 0
  fi

  log "创建远端数据库 ${PG_SYNC_TARGET_DB}"
  if [[ "${DRY_RUN}" -eq 1 ]]; then
    log "DRY RUN: create database ${PG_SYNC_TARGET_DB}"
  else
    psql_target_admin -c "create database ${PG_SYNC_TARGET_DB};" >/dev/null
  fi
}

assert_known_table_strategies() {
  local missing=0
  local table_name
  while IFS= read -r table_name; do
    [[ -n "${table_name}" ]] || continue
    if [[ "$(table_strategy "${table_name}")" == "unknown" ]]; then
      log "未配置同步策略的表: ${table_name}"
      missing=1
    fi
  done < <(list_source_tables)

  [[ "${missing}" -eq 0 ]] || die "存在未覆盖的 public 表，请先补齐同步策略"
}

full_init_remote() {
  log "远端库为空，执行首轮全量初始化"
  if [[ "${DRY_RUN}" -eq 1 ]]; then
    log "DRY RUN: pg_dump source | psql target"
    return 0
  fi

  pg_dump_source --no-owner --no-acl --clean --if-exists | psql_target >/dev/null
}

bootstrap_state_from_database() {
  local which_db="$1"
  local table_name
  local strategy
  local watermark_column
  local watermark_value

  init_state_file "${STATE_FILE}"

  while IFS= read -r table_name; do
    [[ -n "${table_name}" ]] || continue
    strategy="$(table_strategy "${table_name}")"
    watermark_column="$(watermark_column_for_strategy "${strategy}")"
    [[ -n "${watermark_column}" ]] || continue

    watermark_value="$(max_value_from_db "${which_db}" "${table_name}" "${watermark_column}")"
    if [[ -n "${watermark_value}" ]]; then
      set_state_value "${STATE_FILE}" "${table_name}" "${watermark_column}" "${watermark_value}"
    fi
  done < <(list_source_tables)
}

copy_select_to_stdout() {
  local select_sql="$1"
  psql_source -c "\\copy (${select_sql}) to stdout with (format csv)"
}

copy_select_to_csv_file() {
  local select_sql="$1"
  local csv_file_path="$2"

  psql_source -c "\\copy (${select_sql}) to '${csv_file_path}' with (format csv)"
}

run_copy_pipeline() {
  local table_name="$1"
  local select_sql="$2"
  local insert_sql="$3"
  local temp_table_name="$4"
  local columns_csv="$5"
  local csv_file_path

  if [[ "${DRY_RUN}" -eq 1 ]]; then
    log "DRY RUN: 同步 ${table_name}"
    log "DRY RUN SQL SELECT: ${select_sql}"
    log "DRY RUN SQL INSERT: ${insert_sql}"
    return 0
  fi

  csv_file_path="$(mktemp)"
  copy_select_to_csv_file "${select_sql}" "${csv_file_path}"

  {
    printf 'begin;\n'
    printf 'create temp table %s (like public.%s including all);\n' "${temp_table_name}" "${table_name}"
    printf "\\copy %s (%s) from '%s' with (format csv)\n" "${temp_table_name}" "${columns_csv}" "${csv_file_path}"
    printf '%s\n' "${insert_sql}"
    printf 'commit;\n'
  } | psql_target >/dev/null

  rm -f "${csv_file_path}"
}

refresh_table() {
  local table_name="$1"
  local columns_csv
  local select_sql
  local insert_sql
  local temp_table_name

  columns_csv="$(table_columns_csv "${table_name}")"
  select_sql="select ${columns_csv} from public.${table_name}"
  temp_table_name="tmp_sync_${table_name}_$$"
  insert_sql="delete from public.${table_name}; insert into public.${table_name} (${columns_csv}) select ${columns_csv} from ${temp_table_name};"

  log "整表刷新 ${table_name}"
  run_copy_pipeline "${table_name}" "${select_sql}" "${insert_sql}" "${temp_table_name}" "${columns_csv}"
}

sync_append_id_table() {
  local table_name="$1"
  local columns_csv
  local pk_csv
  local watermark
  local select_sql
  local insert_sql
  local temp_table_name
  local latest_source_id

  columns_csv="$(table_columns_csv "${table_name}")"
  pk_csv="$(table_primary_key_csv "${table_name}")"
  watermark="$(get_state_value "${STATE_FILE}" "${table_name}" id)"
  latest_source_id="$(max_value_from_db source "${table_name}" id)"

  if [[ -z "${latest_source_id}" ]]; then
    log "跳过 ${table_name}，源表为空"
    return 0
  fi

  if [[ -n "${watermark}" && "${latest_source_id}" -le "${watermark}" ]]; then
    log "跳过 ${table_name}，无新增 id"
    return 0
  fi

  if [[ -n "${watermark}" ]]; then
    select_sql="select ${columns_csv} from public.${table_name} where id > ${watermark} order by id"
  else
    select_sql="select ${columns_csv} from public.${table_name} order by id"
  fi

  temp_table_name="tmp_sync_${table_name}_$$"
  insert_sql="insert into public.${table_name} (${columns_csv}) select ${columns_csv} from ${temp_table_name} on conflict (${pk_csv}) do nothing;"

  log "按 id 追加 ${table_name}，当前水位 ${watermark:-<empty>} -> ${latest_source_id}"
  run_copy_pipeline "${table_name}" "${select_sql}" "${insert_sql}" "${temp_table_name}" "${columns_csv}"
  set_state_value "${STATE_FILE}" "${table_name}" id "${latest_source_id}"
}

sync_append_created_at_table() {
  local table_name="$1"
  local columns_csv
  local pk_csv
  local watermark
  local latest_source_value
  local escaped_watermark
  local select_sql
  local insert_sql
  local temp_table_name

  columns_csv="$(table_columns_csv "${table_name}")"
  pk_csv="$(table_primary_key_csv "${table_name}")"
  watermark="$(get_state_value "${STATE_FILE}" "${table_name}" created_at)"
  latest_source_value="$(max_value_from_db source "${table_name}" created_at)"

  if [[ -z "${latest_source_value}" ]]; then
    log "跳过 ${table_name}，源表为空"
    return 0
  fi

  if [[ -n "${watermark}" ]]; then
    escaped_watermark="$(sql_escape_literal "${watermark}")"
    select_sql="select ${columns_csv} from public.${table_name} where created_at >= timestamptz '${escaped_watermark}' - interval '${OVERLAP_MINUTES} minutes' order by created_at"
  else
    select_sql="select ${columns_csv} from public.${table_name} order by created_at"
  fi

  temp_table_name="tmp_sync_${table_name}_$$"
  insert_sql="insert into public.${table_name} (${columns_csv}) select ${columns_csv} from ${temp_table_name} on conflict (${pk_csv}) do nothing;"

  log "按 created_at 追加 ${table_name}，当前水位 ${watermark:-<empty>} -> ${latest_source_value}"
  run_copy_pipeline "${table_name}" "${select_sql}" "${insert_sql}" "${temp_table_name}" "${columns_csv}"
  set_state_value "${STATE_FILE}" "${table_name}" created_at "${latest_source_value}"
}

sync_upsert_updated_at_table() {
  local table_name="$1"
  local columns_csv
  local pk_csv
  local update_assignments
  local watermark
  local latest_source_value
  local escaped_watermark
  local select_sql
  local insert_sql
  local temp_table_name

  columns_csv="$(table_columns_csv "${table_name}")"
  pk_csv="$(table_primary_key_csv "${table_name}")"
  update_assignments="$(table_update_assignments_csv "${table_name}")"
  watermark="$(get_state_value "${STATE_FILE}" "${table_name}" updated_at)"
  latest_source_value="$(max_value_from_db source "${table_name}" updated_at)"

  if [[ -z "${latest_source_value}" ]]; then
    log "跳过 ${table_name}，源表为空"
    return 0
  fi

  if [[ -n "${watermark}" ]]; then
    escaped_watermark="$(sql_escape_literal "${watermark}")"
    select_sql="select ${columns_csv} from public.${table_name} where updated_at >= timestamptz '${escaped_watermark}' - interval '${OVERLAP_MINUTES} minutes' order by updated_at"
  else
    select_sql="select ${columns_csv} from public.${table_name} order by updated_at"
  fi

  temp_table_name="tmp_sync_${table_name}_$$"
  insert_sql="insert into public.${table_name} (${columns_csv}) select ${columns_csv} from ${temp_table_name} on conflict (${pk_csv}) do update set ${update_assignments};"

  log "按 updated_at UPSERT ${table_name}，当前水位 ${watermark:-<empty>} -> ${latest_source_value}"
  run_copy_pipeline "${table_name}" "${select_sql}" "${insert_sql}" "${temp_table_name}" "${columns_csv}"
  set_state_value "${STATE_FILE}" "${table_name}" updated_at "${latest_source_value}"
}

reset_table_sequence() {
  local table_name="$1"

  if [[ "${DRY_RUN}" -eq 1 ]]; then
    log "DRY RUN: reset sequence for ${table_name}"
    return 0
  fi

  psql_target -Atqc "
    do \$\$
    declare
      sequence_name text;
      max_id bigint;
      has_id_column boolean;
    begin
      select exists (
        select 1
        from information_schema.columns
        where table_schema = 'public'
          and table_name = '${table_name}'
          and column_name = 'id'
      ) into has_id_column;

      if not has_id_column then
        return;
      end if;

      select pg_get_serial_sequence('public.${table_name}', 'id') into sequence_name;
      if sequence_name is null then
        return;
      end if;

      execute format('select coalesce(max(id), 0) from public.%I', '${table_name}') into max_id;
      if max_id > 0 then
        execute format('select setval(%L, %s, true)', sequence_name, max_id);
      else
        execute format('select setval(%L, 1, false)', sequence_name);
      end if;
    end
    \$\$;
  " >/dev/null
}

sync_table() {
  local table_name="$1"
  local strategy

  reconcile_table_schema "${table_name}"
  strategy="$(table_strategy "${table_name}")"
  case "${strategy}" in
    refresh)
      refresh_table "${table_name}"
      ;;
    upsert_updated_at)
      sync_upsert_updated_at_table "${table_name}"
      ;;
    append_id)
      sync_append_id_table "${table_name}"
      ;;
    append_created_at)
      sync_append_created_at_table "${table_name}"
      ;;
    *)
      die "未知同步策略: ${table_name}"
      ;;
  esac

  reset_table_sequence "${table_name}"
}

rebuild_state_if_needed() {
  init_state_file "${STATE_FILE}"
  if [[ -s "${STATE_FILE}" ]]; then
    return 0
  fi

  log "状态文件为空，按远端库当前水位重建"
  bootstrap_state_from_database target
}

run_incremental_sync() {
  local table_name
  while IFS= read -r table_name; do
    [[ -n "${table_name}" ]] || continue
    sync_table "${table_name}"
  done < <(list_source_tables)
}

print_usage_hint() {
  cat >&2 <<'EOF'
用法:
  bash deploy/pg_sync_incremental.sh [--dry-run] [--init-only] [--sync-only] [--rebuild-state]

默认行为:
  1. 从 deploy/.pg_sync.env 读取远端配置
  2. 从 /etc/aiapi/config.yaml 自动读取本地源库配置
  3. 远端空库时先全量初始化，之后执行混合增量同步
EOF
}

main() {
  parse_args "$@"
  require_command psql
  require_command pg_dump

  load_env_file
  apply_source_defaults_from_config
  normalize_config
  validate_config
  init_state_file "${STATE_FILE}"

  ensure_target_database
  assert_known_table_strategies

  if [[ "${REBUILD_STATE_ONLY}" -eq 1 ]]; then
    bootstrap_state_from_database target
    log "已按远端库重建状态文件 ${STATE_FILE}"
    return 0
  fi

  if [[ "$(target_public_table_count)" == "0" ]]; then
    if [[ "${SYNC_ONLY}" -eq 1 ]]; then
      die "远端库为空，--sync-only 无法执行，请先初始化"
    fi
    full_init_remote
    bootstrap_state_from_database source
    log "首轮初始化完成，状态文件已更新到源库当前水位"
    if [[ "${INIT_ONLY}" -eq 1 ]]; then
      return 0
    fi
    return 0
  fi

  if [[ "${INIT_ONLY}" -eq 1 ]]; then
    log "远端库已存在表结构，--init-only 不再执行任何写入"
    rebuild_state_if_needed
    return 0
  fi

  rebuild_state_if_needed
  run_incremental_sync
  log "增量同步完成"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  if [[ "${1:-}" == "--help" || "${1:-}" == "-h" ]]; then
    print_usage_hint
    exit 0
  fi
  main "$@"
fi
