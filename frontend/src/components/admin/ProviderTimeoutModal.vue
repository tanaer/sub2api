<template>
  <BaseDialog :show="show" :title="t('admin.providerTimeout.title')" width="extra-wide" @close="$emit('close')">
    <div class="space-y-5">
      <!-- Description + Refresh -->
      <div class="flex items-center justify-between">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.providerTimeout.description') }}
        </p>
        <div class="flex items-center gap-2">
          <select
            v-model="statsHours"
            class="input-field w-auto text-sm"
            @change="loadStats"
          >
            <option :value="1">{{ t('admin.providerTimeout.hours1') }}</option>
            <option :value="6">{{ t('admin.providerTimeout.hours6') }}</option>
            <option :value="24">{{ t('admin.providerTimeout.hours24') }}</option>
            <option :value="72">{{ t('admin.providerTimeout.hours72') }}</option>
            <option :value="168">{{ t('admin.providerTimeout.hours168') }}</option>
          </select>
          <button @click="loadStats" class="btn btn-secondary btn-sm" :disabled="loadingStats">
            <Icon name="refresh" size="sm" :class="loadingStats ? 'animate-spin' : ''" />
          </button>
        </div>
      </div>

      <!-- Latency Stats Table -->
      <div v-if="loadingStats" class="flex items-center justify-center py-6">
        <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
      </div>

      <div v-else-if="stats.length === 0" class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.providerTimeout.noStats') }}
      </div>

      <div v-else class="overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
          <thead class="bg-gray-50 dark:bg-dark-700">
            <tr>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.providerTimeout.columns.provider') }}
              </th>
              <th class="px-3 py-2 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.providerTimeout.columns.count') }}
              </th>
              <th class="px-3 py-2 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">P50</th>
              <th class="px-3 py-2 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">P90</th>
              <th class="px-3 py-2 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">P99</th>
              <th class="px-3 py-2 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">AVG</th>
              <th class="px-3 py-2 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">MAX</th>
              <th class="px-3 py-2 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.providerTimeout.columns.timeout') }}
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="s in stats" :key="s.provider" class="hover:bg-gray-50 dark:hover:bg-dark-750">
              <td class="px-3 py-2 text-sm font-medium text-gray-900 dark:text-white">
                {{ s.provider }}
              </td>
              <td class="px-3 py-2 text-right text-sm text-gray-600 dark:text-gray-300">
                {{ s.count.toLocaleString() }}
              </td>
              <td class="px-3 py-2 text-right text-sm tabular-nums" :class="latencyColor(s.p50_ms)">
                {{ formatMs(s.p50_ms) }}
              </td>
              <td class="px-3 py-2 text-right text-sm tabular-nums" :class="latencyColor(s.p90_ms)">
                {{ formatMs(s.p90_ms) }}
              </td>
              <td class="px-3 py-2 text-right text-sm tabular-nums" :class="latencyColor(s.p99_ms)">
                {{ formatMs(s.p99_ms) }}
              </td>
              <td class="px-3 py-2 text-right text-sm tabular-nums" :class="latencyColor(s.avg_ms)">
                {{ formatMs(s.avg_ms) }}
              </td>
              <td class="px-3 py-2 text-right text-sm tabular-nums" :class="latencyColor(s.max_ms)">
                {{ formatMs(s.max_ms) }}
              </td>
              <td class="px-3 py-2 text-right text-sm font-medium">
                <input
                  v-model.number="editTimeouts[s.provider]"
                  type="number"
                  min="10"
                  max="600"
                  :placeholder="t('admin.providerTimeout.defaultPlaceholder')"
                  class="input-field w-20 text-right text-sm"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Enable toggle + global default -->
      <div class="flex items-center gap-6 rounded-lg bg-gray-50 px-4 py-3 dark:bg-dark-700">
        <label class="flex cursor-pointer items-center gap-2">
          <input
            v-model="editEnabled"
            type="checkbox"
            class="toggle-checkbox"
          />
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
            {{ t('admin.providerTimeout.enableLabel') }}
          </span>
        </label>
        <div class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.providerTimeout.enableHint') }}
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex items-center justify-between">
        <span class="text-xs text-gray-400 dark:text-gray-500">
          {{ t('admin.providerTimeout.unit') }}
        </span>
        <div class="flex gap-2">
          <button @click="$emit('close')" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button @click="handleSave" :disabled="saving" class="btn btn-primary">
            <Icon v-if="saving" name="refresh" size="sm" class="mr-1 animate-spin" />
            {{ t('common.save') }}
          </button>
        </div>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import {
  getProviderTimeoutSettings,
  updateProviderTimeoutSettings,
  getProviderLatencyStats,
  type ProviderLatencyStats
} from '@/api/admin/settings'
import { useAppStore } from '@/stores/app'

const props = defineProps<{
  show: boolean
}>()

defineEmits<{
  close: []
}>()

const { t } = useI18n()
const appStore = useAppStore()

const loadingStats = ref(false)
const loadingSettings = ref(false)
const saving = ref(false)
const statsHours = ref(24)
const stats = ref<ProviderLatencyStats[]>([])
const editEnabled = ref(false)
const editTimeouts = ref<Record<string, number | undefined>>({})

function formatMs(ms: number): string {
  if (ms >= 1000) {
    return (ms / 1000).toFixed(1) + 's'
  }
  return ms + 'ms'
}

function latencyColor(ms: number): string {
  if (ms >= 60000) return 'text-red-600 dark:text-red-400 font-semibold'
  if (ms >= 30000) return 'text-orange-600 dark:text-orange-400'
  if (ms >= 10000) return 'text-yellow-600 dark:text-yellow-400'
  return 'text-gray-600 dark:text-gray-300'
}

async function loadStats() {
  loadingStats.value = true
  try {
    stats.value = await getProviderLatencyStats(statsHours.value)
  } catch {
    stats.value = []
  } finally {
    loadingStats.value = false
  }
}

async function loadSettings() {
  loadingSettings.value = true
  try {
    const settings = await getProviderTimeoutSettings()
    editEnabled.value = settings.enabled
    // Merge saved timeouts into edit state
    editTimeouts.value = { ...settings.timeouts }
  } catch {
    editEnabled.value = false
    editTimeouts.value = {}
  } finally {
    loadingSettings.value = false
  }
}

async function handleSave() {
  saving.value = true
  try {
    // Build clean timeouts map (only non-empty values)
    const timeouts: Record<string, number> = {}
    for (const [k, v] of Object.entries(editTimeouts.value)) {
      if (v != null && v > 0) {
        timeouts[k] = v
      }
    }
    await updateProviderTimeoutSettings({
      enabled: editEnabled.value,
      timeouts
    })
    appStore.showSuccess(t('admin.providerTimeout.saveSuccess'))
  } catch (e: unknown) {
    appStore.showError(e instanceof Error ? e.message : String(e))
  } finally {
    saving.value = false
  }
}

watch(
  () => props.show,
  (val) => {
    if (val) {
      loadSettings()
      loadStats()
    }
  }
)
</script>
