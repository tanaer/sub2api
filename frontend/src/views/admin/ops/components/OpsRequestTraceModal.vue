<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { useClipboard } from '@/composables/useClipboard'
import {
  opsAPI,
  type OpsRequestTraceCandidate,
  type OpsRequestTraceQueryKeyType,
  type OpsRequestTraceResponse,
  type OpsRequestTraceTimelineItem,
} from '@/api/admin/ops'
import { useAppStore } from '@/stores'
import { formatDateTime } from '@/utils/format'

interface Props {
  modelValue: boolean
  requestKey: string
  requestKeyType?: OpsRequestTraceQueryKeyType
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
}>()

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const loading = ref(false)
const trace = ref<OpsRequestTraceResponse | null>(null)
const ambiguousCandidates = ref<OpsRequestTraceCandidate[]>([])
const notFound = ref(false)
const errorMessage = ref('')
const activeKey = ref('')
const activeKeyType = ref<OpsRequestTraceQueryKeyType>('auto')

const title = computed(() => t('admin.ops.requestTrace.title'))

const requestRows = computed(() => {
  if (!trace.value) return []
  return [
    { label: t('admin.ops.requestTrace.fields.createdAt'), value: formatNullableTime(trace.value.request.created_at) },
    { label: t('admin.ops.requestTrace.fields.finishedAt'), value: formatNullableTime(trace.value.request.finished_at) },
    { label: t('admin.ops.requestTrace.fields.duration'), value: formatNullableDuration(trace.value.request.duration_ms) },
    { label: t('admin.ops.requestTrace.fields.status'), value: formatNullableValue(trace.value.request.status) },
    { label: t('admin.ops.requestTrace.fields.platform'), value: formatNullableValue(trace.value.request.platform) },
    { label: t('admin.ops.requestTrace.fields.requestPath'), value: formatNullableValue(trace.value.request.request_path) },
    { label: t('admin.ops.requestTrace.fields.inboundEndpoint'), value: formatNullableValue(trace.value.request.inbound_endpoint) },
    { label: t('admin.ops.requestTrace.fields.upstreamEndpoint'), value: formatNullableValue(trace.value.request.upstream_endpoint) },
    { label: t('admin.ops.requestTrace.fields.stream'), value: formatBoolean(trace.value.request.stream) },
    { label: t('admin.ops.requestTrace.fields.userId'), value: formatNullableValue(trace.value.request.user_id) },
    { label: t('admin.ops.requestTrace.fields.apiKeyId'), value: formatNullableValue(trace.value.request.api_key_id) },
    { label: t('admin.ops.requestTrace.fields.groupId'), value: formatNullableValue(trace.value.request.group_id) },
  ]
})

const modelRows = computed(() => {
  if (!trace.value) return []
  return [
    { label: t('admin.ops.requestTrace.fields.originalRequestedModel'), value: formatNullableValue(trace.value.models.original_requested_model) },
    { label: t('admin.ops.requestTrace.fields.groupResolvedModel'), value: formatNullableValue(trace.value.models.group_resolved_model) },
    { label: t('admin.ops.requestTrace.fields.accountSupportLookupModel'), value: formatNullableValue(trace.value.models.account_support_lookup_model) },
    { label: t('admin.ops.requestTrace.fields.finalUpstreamModel'), value: formatNullableValue(trace.value.models.final_upstream_model) },
  ]
})

const resultRows = computed(() => {
  if (!trace.value) return []
  return [
    { label: t('admin.ops.requestTrace.fields.finalStatus'), value: formatNullableValue(trace.value.result.final_status) },
    { label: t('admin.ops.requestTrace.fields.finalStatusCode'), value: formatNullableValue(trace.value.result.final_status_code) },
    {
      label: t('admin.ops.requestTrace.fields.finalAccount'),
      value: formatNullableAccount(trace.value.result.final_account_name, trace.value.result.final_account_id),
    },
    { label: t('admin.ops.requestTrace.fields.finalUpstreamModel'), value: formatNullableValue(trace.value.result.final_upstream_model) },
    { label: t('admin.ops.requestTrace.fields.traceIncomplete'), value: formatBoolean(trace.value.trace_incomplete) },
  ]
})

const identityRows = computed(() => {
  if (!trace.value) return []
  return [
    {
      label: t('admin.ops.requestTrace.fields.queryKey'),
      value: formatNullableValue(trace.value.identity.query_key),
      copyValue: trace.value.identity.query_key,
    },
    {
      label: t('admin.ops.requestTrace.fields.queryKeyType'),
      value: formatNullableValue(trace.value.identity.query_key_type),
    },
    {
      label: t('admin.ops.requestTrace.fields.matchedBy'),
      value: formatNullableValue(trace.value.identity.matched_by),
    },
    {
      label: t('admin.ops.requestTrace.fields.clientRequestId'),
      value: formatNullableValue(trace.value.identity.client_request_id),
      copyValue: trace.value.identity.client_request_id,
    },
    {
      label: t('admin.ops.requestTrace.fields.localRequestId'),
      value: formatNullableValue(trace.value.identity.local_request_id),
      copyValue: trace.value.identity.local_request_id,
    },
    {
      label: t('admin.ops.requestTrace.fields.usageRequestId'),
      value: formatNullableValue(trace.value.identity.usage_request_id),
      copyValue: trace.value.identity.usage_request_id,
    },
  ]
})

const upstreamRequestIDs = computed(() => trace.value?.identity.upstream_request_ids ?? [])
const timeline = computed(() => trace.value?.timeline ?? [])

function close() {
  emit('update:modelValue', false)
}

function resetState() {
  trace.value = null
  ambiguousCandidates.value = []
  notFound.value = false
  errorMessage.value = ''
}

function syncQueryFromProps() {
  activeKey.value = String(props.requestKey || '').trim()
  activeKeyType.value = props.requestKeyType ?? 'auto'
}

function formatNullableValue(value: unknown): string {
  if (value === null || value === undefined) return '-'
  const text = String(value).trim()
  return text || '-'
}

function formatNullableTime(value: string | null | undefined): string {
  if (!value) return '-'
  return formatDateTime(value)
}

function formatNullableDuration(value: number | null | undefined): string {
  if (typeof value !== 'number' || !Number.isFinite(value)) return '-'
  return `${value} ms`
}

function formatBoolean(value: boolean | null | undefined): string {
  if (typeof value !== 'boolean') return '-'
  return value ? t('admin.ops.requestTrace.boolean.yes') : t('admin.ops.requestTrace.boolean.no')
}

function formatNullableAccount(accountName?: string | null, accountID?: number | null): string {
  const name = String(accountName || '').trim()
  if (name) return name
  if (typeof accountID === 'number' && Number.isFinite(accountID)) return String(accountID)
  return '-'
}

function timelineDataText(item: OpsRequestTraceTimelineItem): string {
  if (!item.data || Object.keys(item.data).length === 0) {
    return ''
  }
  return JSON.stringify(item.data, null, 2)
}

function traceStatusClass(status: string | undefined): string {
  const normalized = String(status || '').trim().toLowerCase()
  if (normalized === 'success' || normalized === 'succeeded') {
    return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
  }
  if (normalized === 'error' || normalized === 'failed') {
    return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  }
  return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
}

function phaseClass(phase: string | undefined): string {
  switch (String(phase || '').trim().toLowerCase()) {
    case 'ingress':
      return 'bg-sky-100 text-sky-700 dark:bg-sky-900/30 dark:text-sky-300'
    case 'routing':
      return 'bg-violet-100 text-violet-700 dark:bg-violet-900/30 dark:text-violet-300'
    case 'selection':
      return 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'
    case 'account':
      return 'bg-fuchsia-100 text-fuchsia-700 dark:bg-fuchsia-900/30 dark:text-fuchsia-300'
    case 'upstream':
      return 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-300'
    case 'usage':
      return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
    case 'finish':
      return 'bg-slate-200 text-slate-700 dark:bg-slate-800 dark:text-slate-300'
    default:
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300'
  }
}

async function handleCopy(value: string | undefined) {
  const text = String(value || '').trim()
  if (!text) return
  const ok = await copyToClipboard(text, t('admin.ops.requestTrace.copied'))
  if (!ok) {
    appStore.showWarning(t('admin.ops.requestTrace.copyFailed'))
  }
}

async function fetchTrace() {
  const key = String(activeKey.value || '').trim()
  if (!props.modelValue || !key) {
    resetState()
    return
  }

  loading.value = true
  resetState()

  try {
    trace.value = await opsAPI.getRequestTrace(key, activeKeyType.value)
  } catch (err: any) {
    const status = Number(err?.response?.status || 0)
    if (status === 404) {
      notFound.value = true
      return
    }
    if (status === 409) {
      ambiguousCandidates.value = Array.isArray(err?.response?.data?.error?.candidates)
        ? err.response.data.error.candidates
        : []
      errorMessage.value = err?.response?.data?.error?.message || t('admin.ops.requestTrace.ambiguousHint')
      return
    }

    const message = err?.response?.data?.error?.message || err?.message || t('admin.ops.requestTrace.failedToLoad')
    errorMessage.value = message
    appStore.showError(message)
  } finally {
    loading.value = false
  }
}

function handleUseCandidate(candidate: OpsRequestTraceCandidate) {
  activeKey.value = String(candidate.client_request_id || '').trim()
  activeKeyType.value = 'client_request_id'
  void fetchTrace()
}

watch(
  () => [props.modelValue, props.requestKey, props.requestKeyType] as const,
  ([open]) => {
    if (!open) {
      resetState()
      return
    }
    syncQueryFromProps()
    void fetchTrace()
  },
  { immediate: true }
)
</script>

<template>
  <BaseDialog :show="modelValue" :title="title" width="full" @close="close">
    <div class="flex min-h-[640px] flex-col p-6">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div class="space-y-1">
          <div class="text-xs font-bold uppercase tracking-wider text-gray-400">
            {{ t('admin.ops.requestTrace.fields.queryKey') }}
          </div>
          <div class="font-mono text-sm text-gray-700 dark:text-gray-200">
            {{ activeKey || '-' }}
          </div>
        </div>
        <button
          type="button"
          class="btn btn-secondary btn-sm"
          @click="fetchTrace"
        >
          {{ t('common.refresh') }}
        </button>
      </div>

      <div v-if="loading" class="flex flex-1 items-center justify-center py-16">
        <div class="flex flex-col items-center gap-3">
          <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
          <div class="text-sm text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</div>
        </div>
      </div>

      <div v-else-if="ambiguousCandidates.length > 0" class="space-y-4">
        <div class="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-900/40 dark:bg-amber-900/20 dark:text-amber-200">
          <div class="font-bold">{{ t('admin.ops.requestTrace.ambiguousTitle') }}</div>
          <div class="mt-1 text-xs">{{ errorMessage || t('admin.ops.requestTrace.ambiguousHint') }}</div>
        </div>

        <div class="space-y-3">
          <div
            v-for="candidate in ambiguousCandidates"
            :key="candidate.client_request_id"
            class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800"
          >
            <div class="flex flex-wrap items-start justify-between gap-3">
              <div class="space-y-2">
                <div class="font-mono text-sm font-medium text-gray-900 dark:text-white">
                  {{ candidate.client_request_id }}
                </div>
                <div class="flex flex-wrap gap-2 text-xs text-gray-500 dark:text-gray-400">
                  <span>{{ t('admin.ops.requestTrace.fields.createdAt') }}: {{ formatNullableTime(candidate.created_at) }}</span>
                  <span>{{ t('admin.ops.requestTrace.fields.status') }}: {{ formatNullableValue(candidate.status) }}</span>
                  <span>{{ t('admin.ops.requestTrace.fields.finalAccountId') }}: {{ formatNullableValue(candidate.final_account_id) }}</span>
                </div>
              </div>
              <button
                type="button"
                class="rounded-lg bg-primary-50 px-3 py-1.5 text-xs font-bold text-primary-700 hover:bg-primary-100 dark:bg-primary-900/20 dark:text-primary-300 dark:hover:bg-primary-900/30"
                @click="handleUseCandidate(candidate)"
              >
                {{ t('admin.ops.requestTrace.useCandidate') }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="notFound" class="flex flex-1 items-center justify-center">
        <div class="rounded-xl border border-dashed border-gray-200 px-8 py-10 text-center dark:border-dark-700">
          <div class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ t('admin.ops.requestTrace.notFound') }}</div>
          <div class="mt-1 text-xs text-gray-400">{{ t('admin.ops.requestTrace.notFoundHint') }}</div>
        </div>
      </div>

      <div v-else-if="trace" class="space-y-6 overflow-y-auto pr-1">
        <div class="flex flex-wrap items-center gap-2">
          <span class="rounded-full px-3 py-1 text-xs font-bold" :class="traceStatusClass(trace.result.final_status)">
            {{ trace.result.final_status || trace.request.status || '-' }}
          </span>
          <span
            v-if="trace.trace_incomplete"
            class="rounded-full bg-amber-100 px-3 py-1 text-xs font-bold text-amber-700 dark:bg-amber-900/30 dark:text-amber-300"
          >
            {{ t('admin.ops.requestTrace.incomplete') }}
          </span>
        </div>

        <section class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <div class="mb-4 text-sm font-bold text-gray-900 dark:text-white">{{ t('admin.ops.requestTrace.sections.identity') }}</div>
          <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <div
              v-for="row in identityRows"
              :key="row.label"
              class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900"
            >
              <div class="text-[11px] font-bold uppercase tracking-wider text-gray-400">{{ row.label }}</div>
              <div class="mt-1 flex items-start justify-between gap-3">
                <div class="break-all font-mono text-sm text-gray-800 dark:text-gray-100">{{ row.value }}</div>
                <button
                  v-if="row.copyValue"
                  type="button"
                  class="shrink-0 rounded-md bg-gray-100 px-2 py-1 text-[10px] font-bold text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-300 dark:hover:bg-dark-600"
                  @click="handleCopy(row.copyValue)"
                >
                  {{ t('admin.ops.requestDetails.copy') }}
                </button>
              </div>
            </div>
          </div>

          <div class="mt-4 rounded-xl bg-gray-50 p-4 dark:bg-dark-900">
            <div class="text-[11px] font-bold uppercase tracking-wider text-gray-400">{{ t('admin.ops.requestTrace.fields.upstreamRequestIds') }}</div>
            <div v-if="upstreamRequestIDs.length > 0" class="mt-2 flex flex-wrap gap-2">
              <button
                v-for="requestID in upstreamRequestIDs"
                :key="requestID"
                type="button"
                class="rounded-lg border border-gray-200 bg-white px-3 py-1.5 font-mono text-xs text-gray-700 hover:border-primary-300 hover:text-primary-700 dark:border-dark-700 dark:bg-dark-800 dark:text-gray-200 dark:hover:border-primary-600 dark:hover:text-primary-300"
                @click="handleCopy(requestID)"
              >
                {{ requestID }}
              </button>
            </div>
            <div v-else class="mt-2 text-sm text-gray-400">-</div>
          </div>
        </section>

        <section class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <div class="mb-4 text-sm font-bold text-gray-900 dark:text-white">{{ t('admin.ops.requestTrace.sections.models') }}</div>
          <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
            <div
              v-for="row in modelRows"
              :key="row.label"
              class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900"
            >
              <div class="text-[11px] font-bold uppercase tracking-wider text-gray-400">{{ row.label }}</div>
              <div class="mt-2 break-all font-mono text-sm text-gray-800 dark:text-gray-100">{{ row.value }}</div>
            </div>
          </div>
        </section>

        <div class="grid grid-cols-1 gap-6 xl:grid-cols-2">
          <section class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
            <div class="mb-4 text-sm font-bold text-gray-900 dark:text-white">{{ t('admin.ops.requestTrace.sections.request') }}</div>
            <div class="space-y-3">
              <div
                v-for="row in requestRows"
                :key="row.label"
                class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900"
              >
                <div class="text-[11px] font-bold uppercase tracking-wider text-gray-400">{{ row.label }}</div>
                <div class="mt-1 break-all text-sm text-gray-800 dark:text-gray-100">{{ row.value }}</div>
              </div>
            </div>
          </section>

          <section class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
            <div class="mb-4 text-sm font-bold text-gray-900 dark:text-white">{{ t('admin.ops.requestTrace.sections.result') }}</div>
            <div class="space-y-3">
              <div
                v-for="row in resultRows"
                :key="row.label"
                class="rounded-xl bg-gray-50 p-4 dark:bg-dark-900"
              >
                <div class="text-[11px] font-bold uppercase tracking-wider text-gray-400">{{ row.label }}</div>
                <div class="mt-1 break-all text-sm text-gray-800 dark:text-gray-100">{{ row.value }}</div>
              </div>
            </div>
          </section>
        </div>

        <section class="rounded-2xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-800">
          <div class="mb-4 flex items-center justify-between gap-3">
            <div class="text-sm font-bold text-gray-900 dark:text-white">{{ t('admin.ops.requestTrace.sections.timeline') }}</div>
            <div class="text-xs text-gray-400">{{ timeline.length }} {{ t('admin.ops.requestTrace.timelineCount') }}</div>
          </div>

          <div v-if="timeline.length === 0" class="rounded-xl border border-dashed border-gray-200 px-6 py-10 text-center text-sm text-gray-500 dark:border-dark-700 dark:text-gray-400">
            {{ t('admin.ops.requestTrace.timelineEmpty') }}
          </div>

          <div v-else class="space-y-4">
            <div
              v-for="(item, index) in timeline"
              :key="`${item.ts}-${item.type}-${index}`"
              class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900"
            >
              <div class="flex flex-wrap items-center gap-2">
                <span class="rounded-full px-2.5 py-1 text-[10px] font-bold uppercase tracking-wider" :class="phaseClass(item.phase)">
                  {{ item.phase || '-' }}
                </span>
                <span class="font-mono text-xs font-bold text-gray-700 dark:text-gray-200">{{ item.type }}</span>
                <span class="text-xs text-gray-400">{{ formatNullableTime(item.ts) }}</span>
              </div>

              <div class="mt-3 grid grid-cols-1 gap-3 lg:grid-cols-[minmax(0,220px),1fr]">
                <div class="rounded-lg bg-white px-3 py-2 text-sm text-gray-700 dark:bg-dark-800 dark:text-gray-200">
                  {{ item.summary || item.type }}
                </div>
                <pre
                  v-if="timelineDataText(item)"
                  class="max-h-[220px] overflow-auto rounded-lg bg-white p-3 text-xs text-gray-800 dark:bg-dark-800 dark:text-gray-100"
                ><code>{{ timelineDataText(item) }}</code></pre>
                <div v-else class="rounded-lg bg-white px-3 py-2 text-xs text-gray-400 dark:bg-dark-800 dark:text-gray-500">
                  {{ t('admin.ops.requestTrace.noEventData') }}
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>

      <div v-else class="flex flex-1 items-center justify-center">
        <div class="rounded-xl border border-dashed border-gray-200 px-8 py-10 text-center dark:border-dark-700">
          <div class="text-sm font-medium text-gray-700 dark:text-gray-200">{{ errorMessage || t('admin.ops.requestTrace.empty') }}</div>
        </div>
      </div>
    </div>
  </BaseDialog>
</template>
