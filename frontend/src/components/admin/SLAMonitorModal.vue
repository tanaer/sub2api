<template>
  <BaseDialog :show="show" :title="t('admin.sla.title')" @close="$emit('close')" width="extra-wide">
    <div class="space-y-4">
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.sla.description') }}</p>
      <!-- Time range selector -->
      <div class="flex items-center gap-2">
        <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.sla.timeRange') }}</label>
        <select v-model="hours" @change="loadReport"
          class="rounded-md border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-800 text-sm px-2 py-1">
          <option :value="1">1h</option>
          <option :value="6">6h</option>
          <option :value="24">24h</option>
          <option :value="72">3d</option>
          <option :value="168">7d</option>
        </select>
        <button @click="loadReport"
          class="p-1 rounded hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-500" :disabled="loading">
          <svg class="w-4 h-4" :class="{ 'animate-spin': loading }" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        </button>
      </div>

      <!-- Overview cards -->
      <div v-if="report" class="grid grid-cols-2 md:grid-cols-4 gap-3">
        <div class="bg-green-50 dark:bg-green-900/20 rounded-lg p-3">
          <div class="text-xs text-green-600 dark:text-green-400 font-medium">{{ t('admin.sla.successRate') }}</div>
          <div class="text-2xl font-bold" :class="successRateColor">{{ report.client_metrics.success_rate.toFixed(2) }}%</div>
          <div class="text-xs text-gray-500">{{ report.client_metrics.successful }} / {{ report.client_metrics.total_requests }}</div>
        </div>
        <div class="bg-red-50 dark:bg-red-900/20 rounded-lg p-3">
          <div class="text-xs text-red-600 dark:text-red-400 font-medium">{{ t('admin.sla.clientErrors') }}</div>
          <div class="text-2xl font-bold text-red-600 dark:text-red-400">{{ report.client_metrics.client_errors }}</div>
          <div class="text-xs text-gray-500">{{ t('admin.sla.failedToClient') }}</div>
        </div>
        <div class="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-3">
          <div class="text-xs text-blue-600 dark:text-blue-400 font-medium">{{ t('admin.sla.failoverSaved') }}</div>
          <div class="text-2xl font-bold text-blue-600 dark:text-blue-400">{{ report.client_metrics.recovered_upstream_errors }}</div>
          <div class="text-xs text-gray-500">{{ t('admin.sla.recoveredByFailover') }}</div>
        </div>
        <div class="bg-purple-50 dark:bg-purple-900/20 rounded-lg p-3">
          <div class="text-xs text-purple-600 dark:text-purple-400 font-medium">{{ t('admin.sla.failoverRate') }}</div>
          <div class="text-2xl font-bold text-purple-600 dark:text-purple-400">{{ report.failover_metrics.failover_success_rate.toFixed(1) }}%</div>
          <div class="text-xs text-gray-500">{{ report.failover_metrics.total_with_failover }} {{ t('admin.sla.failoverEvents') }}</div>
        </div>
      </div>

      <!-- Tabs -->
      <div v-if="report" class="border-b border-gray-200 dark:border-gray-700">
        <nav class="flex gap-4">
          <button v-for="tab in tabs" :key="tab"
            @click="activeTab = tab"
            class="pb-2 text-sm font-medium border-b-2 transition-colors"
            :class="activeTab === tab ? 'border-blue-500 text-blue-600 dark:text-blue-400' : 'border-transparent text-gray-500 hover:text-gray-700'">
            {{ t(`admin.sla.tab.${tab}`) }}
          </button>
        </nav>
      </div>

      <!-- Tab: Upstream Errors -->
      <div v-if="report && activeTab === 'upstream'" class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-gray-500 dark:text-gray-400 border-b dark:border-gray-700">
              <th class="py-2 px-2">{{ t('admin.sla.account') }}</th>
              <th class="py-2 px-2">{{ t('admin.sla.provider') }}</th>
              <th class="py-2 px-2 text-right">{{ t('admin.sla.upstreamStatus') }}</th>
              <th class="py-2 px-2 text-right">{{ t('admin.sla.total') }}</th>
              <th class="py-2 px-2 text-right">{{ t('admin.sla.recovered') }}</th>
              <th class="py-2 px-2 text-right">{{ t('admin.sla.clientFacing') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in report.upstream_errors" :key="i"
              class="border-b dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800">
              <td class="py-1.5 px-2 font-mono text-xs">{{ row.account }}</td>
              <td class="py-1.5 px-2">{{ row.provider }}</td>
              <td class="py-1.5 px-2 text-right">
                <span class="px-1.5 py-0.5 rounded text-xs font-medium"
                  :class="statusBadgeClass(row.upstream_status)">{{ row.upstream_status || '-' }}</span>
              </td>
              <td class="py-1.5 px-2 text-right font-mono">{{ row.total }}</td>
              <td class="py-1.5 px-2 text-right font-mono text-green-600 dark:text-green-400">{{ row.recovered }}</td>
              <td class="py-1.5 px-2 text-right font-mono" :class="row.client_facing > 0 ? 'text-red-600 dark:text-red-400 font-bold' : ''">
                {{ row.client_facing }}
              </td>
            </tr>
            <tr v-if="!report.upstream_errors?.length">
              <td colspan="6" class="py-4 text-center text-gray-400">{{ t('admin.sla.noData') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Tab: Client Errors -->
      <div v-if="report && activeTab === 'clientErrors'" class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-gray-500 dark:text-gray-400 border-b dark:border-gray-700">
              <th class="py-2 px-2">{{ t('admin.sla.statusCode') }}</th>
              <th class="py-2 px-2">{{ t('admin.sla.phase') }}</th>
              <th class="py-2 px-2">{{ t('admin.sla.errorMessage') }}</th>
              <th class="py-2 px-2 text-right">{{ t('admin.sla.count') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in report.client_errors" :key="i"
              class="border-b dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800">
              <td class="py-1.5 px-2">
                <span class="px-1.5 py-0.5 rounded text-xs font-medium"
                  :class="statusBadgeClass(row.status_code)">{{ row.status_code }}</span>
              </td>
              <td class="py-1.5 px-2">
                <span class="px-1.5 py-0.5 rounded text-xs" :class="phaseBadgeClass(row.error_phase)">{{ row.error_phase }}</span>
              </td>
              <td class="py-1.5 px-2 font-mono text-xs max-w-xs truncate" :title="row.error_message">{{ row.error_message }}</td>
              <td class="py-1.5 px-2 text-right font-mono font-bold">{{ row.count }}</td>
            </tr>
            <tr v-if="!report.client_errors?.length">
              <td colspan="4" class="py-4 text-center text-gray-400">{{ t('admin.sla.noData') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Tab: Failover Paths -->
      <div v-if="report && activeTab === 'failover'" class="overflow-x-auto">
        <div class="text-xs text-gray-500 mb-2">
          {{ t('admin.sla.failoverAvg') }}: {{ report.failover_metrics.avg_attempts.toFixed(1) }}
          {{ t('admin.sla.failoverMax') }}: {{ report.failover_metrics.max_attempts }}
        </div>
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-gray-500 dark:text-gray-400 border-b dark:border-gray-700">
              <th class="py-2 px-2">{{ t('admin.sla.time') }}</th>
              <th class="py-2 px-2">{{ t('admin.sla.model') }}</th>
              <th class="py-2 px-2">{{ t('admin.sla.finalResult') }}</th>
              <th class="py-2 px-2">{{ t('admin.sla.failoverChain') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in report.failover_paths" :key="i"
              class="border-b dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800">
              <td class="py-1.5 px-2 text-xs text-gray-500 whitespace-nowrap">{{ formatTime(row.created_at) }}</td>
              <td class="py-1.5 px-2 font-mono text-xs">{{ row.model || '-' }}</td>
              <td class="py-1.5 px-2">
                <span class="px-1.5 py-0.5 rounded text-xs font-medium"
                  :class="row.final_status === 200 ? 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300' : statusBadgeClass(row.final_status)">
                  {{ row.final_status }}
                </span>
              </td>
              <td class="py-1.5 px-2">
                <div class="flex flex-wrap gap-1">
                  <span v-for="(step, j) in parseUpstreamErrors(row.upstream_errors)" :key="j"
                    class="inline-flex items-center gap-0.5 px-1.5 py-0.5 rounded text-xs"
                    :class="step.upstream_status_code >= 400 ? 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300' : 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300'">
                    {{ step.account_name || `#${step.account_id}` }}
                    <span class="font-mono">&rarr;{{ step.upstream_status_code || 'err' }}</span>
                  </span>
                </div>
              </td>
            </tr>
            <tr v-if="!report.failover_paths?.length">
              <td colspan="4" class="py-4 text-center text-gray-400">{{ t('admin.sla.noData') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Tab: Provider Latency -->
      <div v-if="report && activeTab === 'latency'" class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-left text-gray-500 dark:text-gray-400 border-b dark:border-gray-700">
              <th class="py-2 px-2">{{ t('admin.sla.provider') }}</th>
              <th class="py-2 px-2 text-right">{{ t('admin.sla.requests') }}</th>
              <th class="py-2 px-2 text-right">P50</th>
              <th class="py-2 px-2 text-right">P90</th>
              <th class="py-2 px-2 text-right">P99</th>
              <th class="py-2 px-2 text-right">TTFB</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(row, i) in report.provider_latency" :key="i"
              class="border-b dark:border-gray-700 hover:bg-gray-50 dark:hover:bg-gray-800">
              <td class="py-1.5 px-2 font-medium">{{ row.provider }}</td>
              <td class="py-1.5 px-2 text-right font-mono">{{ row.total }}</td>
              <td class="py-1.5 px-2 text-right font-mono" :class="latencyColor(row.p50_ms)">{{ formatMs(row.p50_ms) }}</td>
              <td class="py-1.5 px-2 text-right font-mono" :class="latencyColor(row.p90_ms)">{{ formatMs(row.p90_ms) }}</td>
              <td class="py-1.5 px-2 text-right font-mono" :class="latencyColor(row.p99_ms)">{{ formatMs(row.p99_ms) }}</td>
              <td class="py-1.5 px-2 text-right font-mono" :class="latencyColor(row.ttfb_avg_ms)">{{ row.ttfb_avg_ms ? formatMs(row.ttfb_avg_ms) : '-' }}</td>
            </tr>
            <tr v-if="!report.provider_latency?.length">
              <td colspan="6" class="py-4 text-center text-gray-400">{{ t('admin.sla.noData') }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="flex justify-center py-8">
        <svg class="animate-spin h-6 w-6 text-blue-500" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
        </svg>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { getSLAReport, type SLAReport } from '@/api/admin/settings'
import { useAppStore } from '@/stores/app'

const { t } = useI18n()
const appStore = useAppStore()

const props = defineProps<{ show: boolean }>()
defineEmits<{ close: [] }>()

const hours = ref(1)
const loading = ref(false)
const report = ref<SLAReport | null>(null)
const activeTab = ref('upstream')
const tabs = ['upstream', 'clientErrors', 'failover', 'latency']

async function loadReport() {
  loading.value = true
  try {
    report.value = await getSLAReport(hours.value)
  } catch (e: any) {
    appStore.showError(e.message || 'Failed to load SLA report')
  } finally {
    loading.value = false
  }
}

watch(() => props.show, (val) => {
  if (val && !report.value) loadReport()
})

const successRateColor = ref('')
watch(() => report.value?.client_metrics.success_rate, (rate) => {
  if (rate === undefined) return
  successRateColor.value = rate >= 99 ? 'text-green-600 dark:text-green-400' :
    rate >= 95 ? 'text-yellow-600 dark:text-yellow-400' : 'text-red-600 dark:text-red-400'
}, { immediate: true })

function statusBadgeClass(code: number) {
  if (code >= 500) return 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300'
  if (code >= 400) return 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300'
  if (code >= 200 && code < 300) return 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
  return 'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300'
}

function phaseBadgeClass(phase: string) {
  const map: Record<string, string> = {
    request: 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900 dark:text-yellow-300',
    routing: 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300',
    upstream: 'bg-red-100 text-red-700 dark:bg-red-900 dark:text-red-300',
    internal: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
  }
  return map[phase] || 'bg-gray-100 text-gray-700'
}

function latencyColor(ms: number | null) {
  if (ms === null) return ''
  if (ms < 3000) return 'text-green-600 dark:text-green-400'
  if (ms < 10000) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-red-600 dark:text-red-400'
}

function formatMs(ms: number) {
  if (ms >= 1000) return (ms / 1000).toFixed(1) + 's'
  return ms + 'ms'
}

function formatTime(ts: string) {
  if (!ts) return '-'
  const d = new Date(ts)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function parseUpstreamErrors(raw: string): any[] {
  try {
    return JSON.parse(raw) || []
  } catch {
    return []
  }
}
</script>
