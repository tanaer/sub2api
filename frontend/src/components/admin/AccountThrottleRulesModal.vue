<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accountThrottle.title')"
    width="extra-wide"
    @close="$emit('close')"
  >
    <div class="space-y-4">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.accountThrottle.description') }}
        </p>
        <button @click="showCreateModal = true" class="btn btn-primary btn-sm">
          <Icon name="plus" size="sm" class="mr-1" />
          {{ t('admin.accountThrottle.createRule') }}
        </button>
      </div>

      <!-- Rules Table -->
      <div v-if="loading" class="flex items-center justify-center py-8">
        <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
      </div>

      <div v-else-if="rules.length === 0" class="py-8 text-center">
        <div class="mx-auto mb-4 flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 dark:bg-dark-700">
          <Icon name="bolt" size="lg" class="text-gray-400" />
        </div>
        <h4 class="mb-1 text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.accountThrottle.noRules') }}
        </h4>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.accountThrottle.createFirstRule') }}
        </p>
      </div>

      <div v-else class="max-h-96 overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
          <thead class="sticky top-0 bg-gray-50 dark:bg-dark-700">
            <tr>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.accountThrottle.columns.priority') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.accountThrottle.columns.name') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.accountThrottle.columns.keywords') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.accountThrottle.columns.trigger') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.accountThrottle.columns.action') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.accountThrottle.columns.status') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.accountThrottle.columns.actions') }}
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-800">
            <tr v-for="rule in rules" :key="rule.id" class="hover:bg-gray-50 dark:hover:bg-dark-700">
              <td class="whitespace-nowrap px-3 py-2">
                <span class="inline-flex h-5 w-5 items-center justify-center rounded bg-gray-100 text-xs font-medium text-gray-700 dark:bg-dark-600 dark:text-gray-300">
                  {{ rule.priority }}
                </span>
              </td>
              <td class="px-3 py-2">
                <div class="font-medium text-gray-900 dark:text-white text-sm">{{ rule.name }}</div>
                <div v-if="rule.description" class="mt-0.5 text-xs text-gray-500 dark:text-gray-400 max-w-xs truncate">
                  {{ rule.description }}
                </div>
                <div v-if="rule.platforms.length > 0" class="mt-0.5 flex flex-wrap gap-1">
                  <span
                    v-for="platform in rule.platforms.slice(0, 2)"
                    :key="platform"
                    class="badge badge-primary text-xs"
                  >
                    {{ platform }}
                  </span>
                  <span v-if="rule.platforms.length > 2" class="text-xs text-gray-500">
                    +{{ rule.platforms.length - 2 }}
                  </span>
                </div>
              </td>
              <td class="px-3 py-2">
                <div v-if="rule.error_codes && rule.error_codes.length > 0" class="mb-1 flex flex-wrap gap-1">
                  <span
                    v-for="code in rule.error_codes.slice(0, 3)"
                    :key="code"
                    class="badge badge-warning text-xs"
                  >
                    {{ code }}
                  </span>
                  <span v-if="rule.error_codes.length > 3" class="text-xs text-gray-500">
                    +{{ rule.error_codes.length - 3 }}
                  </span>
                </div>
                <div v-else class="mb-1">
                  <span class="text-xs text-gray-400 dark:text-gray-500">{{ t('admin.accountThrottle.allErrorCodes') }}</span>
                </div>
                <div class="flex flex-wrap gap-1 max-w-48">
                  <span
                    v-for="keyword in rule.keywords.slice(0, 2)"
                    :key="keyword"
                    class="badge badge-gray text-xs"
                  >
                    "{{ keyword.length > 12 ? keyword.substring(0, 12) + '...' : keyword }}"
                  </span>
                  <span v-if="rule.keywords.length > 2" class="text-xs text-gray-500">
                    +{{ rule.keywords.length - 2 }}
                  </span>
                </div>
                <div class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.accountThrottle.matchMode.' + rule.match_mode) }}
                </div>
              </td>
              <td class="px-3 py-2">
                <div class="text-xs">
                  <span v-if="rule.trigger_mode === 'immediate'" class="badge badge-warning">
                    {{ t('admin.accountThrottle.triggerMode.immediate') }}
                  </span>
                  <div v-else class="space-y-0.5">
                    <span class="badge badge-info">
                      {{ t('admin.accountThrottle.triggerMode.accumulated') }}
                    </span>
                    <div class="text-gray-500 dark:text-gray-400">
                      {{ rule.accumulated_count }}{{ t('admin.accountThrottle.timesIn') }}{{ rule.accumulated_window }}{{ t('admin.accountThrottle.seconds') }}
                    </div>
                  </div>
                </div>
              </td>
              <td class="px-3 py-2">
                <div class="text-xs">
                  <span v-if="rule.action_type === 'duration'" class="text-gray-700 dark:text-gray-300">
                    {{ t('admin.accountThrottle.actionType.duration') }}: {{ formatDuration(rule.action_duration) }}
                  </span>
                  <span v-else class="text-gray-700 dark:text-gray-300">
                    {{ t('admin.accountThrottle.actionType.scheduledRecovery') }}: {{ rule.action_recover_hour }}:00
                  </span>
                </div>
              </td>
              <td class="px-3 py-2">
                <button
                  @click="toggleEnabled(rule)"
                  :class="[
                    'relative inline-flex h-4 w-7 flex-shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
                    rule.enabled ? 'bg-primary-600' : 'bg-gray-200 dark:bg-dark-600'
                  ]"
                >
                  <span
                    :class="[
                      'pointer-events-none inline-block h-3 w-3 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out',
                      rule.enabled ? 'translate-x-3' : 'translate-x-0'
                    ]"
                  />
                </button>
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center gap-1">
                  <button
                    @click="handleEdit(rule)"
                    class="p-1 text-gray-500 hover:text-primary-600 dark:hover:text-primary-400"
                    :title="t('common.edit')"
                  >
                    <Icon name="edit" size="sm" />
                  </button>
                  <button
                    @click="handleDelete(rule)"
                    class="p-1 text-gray-500 hover:text-red-600 dark:hover:text-red-400"
                    :title="t('common.delete')"
                  >
                    <Icon name="trash" size="sm" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button @click="$emit('close')" class="btn btn-secondary">
          {{ t('common.close') }}
        </button>
      </div>
    </template>

    <!-- Create/Edit Modal -->
    <BaseDialog
      :show="showCreateModal || showEditModal"
      :title="showEditModal ? t('admin.accountThrottle.editRule') : t('admin.accountThrottle.createRule')"
      width="wide"
      @close="closeFormModal"
    >
      <form @submit.prevent="handleSubmit" class="space-y-4">
        <!-- Basic Info -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.accountThrottle.form.name') }}</label>
            <input
              v-model="form.name"
              type="text"
              required
              class="input"
              :placeholder="t('admin.accountThrottle.form.namePlaceholder')"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.accountThrottle.form.priority') }}</label>
            <input
              v-model.number="form.priority"
              type="number"
              min="0"
              class="input"
            />
            <p class="input-hint">{{ t('admin.accountThrottle.form.priorityHint') }}</p>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.accountThrottle.form.description') }}</label>
          <input
            v-model="form.description"
            type="text"
            class="input"
            :placeholder="t('admin.accountThrottle.form.descriptionPlaceholder')"
          />
        </div>

        <!-- Match Conditions -->
        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <h4 class="mb-2 text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.accountThrottle.form.matchConditions') }}
          </h4>

          <div class="mb-3">
            <label class="input-label text-xs">{{ t('admin.accountThrottle.form.errorCodes') }}</label>
            <input
              v-model="errorCodesInput"
              type="text"
              class="input text-sm"
              :placeholder="t('admin.accountThrottle.form.errorCodesPlaceholder')"
            />
            <p class="input-hint text-xs">{{ t('admin.accountThrottle.form.errorCodesHint') }}</p>
          </div>

          <div>
            <label class="input-label text-xs">{{ t('admin.accountThrottle.form.keywords') }}</label>
            <textarea
              v-model="keywordsInput"
              rows="3"
              class="input font-mono text-xs"
              :placeholder="t('admin.accountThrottle.form.keywordsPlaceholder')"
            />
            <p class="input-hint text-xs">{{ t('admin.accountThrottle.form.keywordsHint') }}</p>
          </div>

          <div class="mt-3">
            <label class="input-label text-xs">{{ t('admin.accountThrottle.form.matchMode') }}</label>
            <div class="mt-1 flex gap-4">
              <label
                v-for="option in matchModeOptions"
                :key="option.value"
                class="flex items-center gap-2 cursor-pointer"
              >
                <input
                  type="radio"
                  :value="option.value"
                  v-model="form.match_mode"
                  class="h-3.5 w-3.5 border-gray-300 text-primary-600 focus:ring-primary-500"
                />
                <span class="text-xs text-gray-700 dark:text-gray-300">{{ option.label }}</span>
              </label>
            </div>
          </div>

          <div class="mt-3">
            <label class="input-label text-xs">{{ t('admin.accountThrottle.form.platforms') }}</label>
            <div class="flex flex-wrap gap-3">
              <label
                v-for="platform in platformOptions"
                :key="platform.value"
                class="inline-flex items-center gap-1.5"
              >
                <input
                  type="checkbox"
                  :value="platform.value"
                  v-model="form.platforms"
                  class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                />
                <span class="text-xs text-gray-700 dark:text-gray-300">{{ platform.label }}</span>
              </label>
            </div>
            <p class="input-hint text-xs mt-1">{{ t('admin.accountThrottle.form.platformsHint') }}</p>
          </div>
        </div>

        <!-- Trigger Mode -->
        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <h4 class="mb-2 text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.accountThrottle.form.triggerConfig') }}
          </h4>

          <div class="flex gap-4">
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                value="immediate"
                v-model="form.trigger_mode"
                class="h-3.5 w-3.5 border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              <div>
                <span class="text-xs font-medium text-gray-700 dark:text-gray-300">{{ t('admin.accountThrottle.triggerMode.immediate') }}</span>
                <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accountThrottle.form.immediateHint') }}</p>
              </div>
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                value="accumulated"
                v-model="form.trigger_mode"
                class="h-3.5 w-3.5 border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              <div>
                <span class="text-xs font-medium text-gray-700 dark:text-gray-300">{{ t('admin.accountThrottle.triggerMode.accumulated') }}</span>
                <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.accountThrottle.form.accumulatedHint') }}</p>
              </div>
            </label>
          </div>

          <div v-if="form.trigger_mode === 'accumulated'" class="mt-3 grid grid-cols-2 gap-3">
            <div>
              <label class="input-label text-xs">{{ t('admin.accountThrottle.form.accumulatedCount') }}</label>
              <input
                v-model.number="form.accumulated_count"
                type="number"
                min="1"
                class="input text-sm"
              />
            </div>
            <div>
              <label class="input-label text-xs">{{ t('admin.accountThrottle.form.accumulatedWindow') }}</label>
              <input
                v-model.number="form.accumulated_window"
                type="number"
                min="1"
                class="input text-sm"
              />
              <p class="input-hint text-xs">{{ t('admin.accountThrottle.form.accumulatedWindowUnit') }}</p>
            </div>
          </div>
        </div>

        <!-- Action -->
        <div class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
          <h4 class="mb-2 text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.accountThrottle.form.actionConfig') }}
          </h4>

          <div class="flex gap-4">
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                value="duration"
                v-model="form.action_type"
                class="h-3.5 w-3.5 border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              <span class="text-xs font-medium text-gray-700 dark:text-gray-300">{{ t('admin.accountThrottle.actionType.duration') }}</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input
                type="radio"
                value="scheduled_recovery"
                v-model="form.action_type"
                class="h-3.5 w-3.5 border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              <span class="text-xs font-medium text-gray-700 dark:text-gray-300">{{ t('admin.accountThrottle.actionType.scheduledRecovery') }}</span>
            </label>
          </div>

          <div v-if="form.action_type === 'duration'" class="mt-3">
            <label class="input-label text-xs">{{ t('admin.accountThrottle.form.actionDuration') }}</label>
            <input
              v-model.number="form.action_duration"
              type="number"
              min="1"
              class="input text-sm"
            />
            <p class="input-hint text-xs">{{ t('admin.accountThrottle.form.actionDurationUnit') }}</p>
          </div>

          <div v-if="form.action_type === 'scheduled_recovery'" class="mt-3">
            <label class="input-label text-xs">{{ t('admin.accountThrottle.form.actionRecoverHour') }}</label>
            <select v-model.number="form.action_recover_hour" class="input text-sm">
              <option v-for="h in 24" :key="h - 1" :value="h - 1">{{ (h - 1).toString().padStart(2, '0') }}:00</option>
            </select>
          </div>
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-2">
          <button @click="closeFormModal" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button @click="handleSubmit" class="btn btn-primary" :disabled="submitting">
            <Icon v-if="submitting" name="refresh" size="sm" class="mr-1 animate-spin" />
            {{ showEditModal ? t('common.save') : t('common.create') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, reactive, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { accountThrottleAPI, type AccountThrottleRule } from '@/api/admin/accountThrottle'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  show: boolean
}>()

defineEmits(['close'])

const { t } = useI18n()
const appStore = useAppStore()

const loading = ref(false)
const submitting = ref(false)
const rules = ref<AccountThrottleRule[]>([])
const showCreateModal = ref(false)
const showEditModal = ref(false)
const editingRuleId = ref<number | null>(null)
const keywordsInput = ref('')
const errorCodesInput = ref('')

const form = reactive({
  name: '',
  priority: 0,
  description: '' as string | null,
  error_codes: [] as number[],
  keywords: [] as string[],
  match_mode: 'contains' as 'contains' | 'exact',
  trigger_mode: 'immediate' as 'immediate' | 'accumulated',
  accumulated_count: 3,
  accumulated_window: 60,
  action_type: 'duration' as 'duration' | 'scheduled_recovery',
  action_duration: 300,
  action_recover_hour: 0,
  platforms: [] as string[]
})

const matchModeOptions = computed(() => [
  { value: 'contains', label: t('admin.accountThrottle.matchMode.contains') },
  { value: 'exact', label: t('admin.accountThrottle.matchMode.exact') }
])

const platformOptions = [
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' }
]

function formatDuration(seconds: number): string {
  if (seconds >= 3600) {
    const h = Math.floor(seconds / 3600)
    const m = Math.floor((seconds % 3600) / 60)
    return m > 0 ? `${h}h${m}m` : `${h}h`
  }
  if (seconds >= 60) {
    const m = Math.floor(seconds / 60)
    const s = seconds % 60
    return s > 0 ? `${m}m${s}s` : `${m}m`
  }
  return `${seconds}s`
}

async function loadRules() {
  loading.value = true
  try {
    rules.value = await accountThrottleAPI.list()
  } catch (err: any) {
    appStore.showError(err.message || 'Failed to load rules')
  } finally {
    loading.value = false
  }
}

function resetForm() {
  form.name = ''
  form.priority = 0
  form.description = null
  form.error_codes = []
  form.keywords = []
  form.match_mode = 'contains'
  form.trigger_mode = 'immediate'
  form.accumulated_count = 3
  form.accumulated_window = 60
  form.action_type = 'duration'
  form.action_duration = 300
  form.action_recover_hour = 0
  form.platforms = []
  keywordsInput.value = ''
  errorCodesInput.value = ''
  editingRuleId.value = null
}

function closeFormModal() {
  showCreateModal.value = false
  showEditModal.value = false
  resetForm()
}

function handleEdit(rule: AccountThrottleRule) {
  editingRuleId.value = rule.id
  form.name = rule.name
  form.priority = rule.priority
  form.description = rule.description
  form.error_codes = [...rule.error_codes]
  form.keywords = [...rule.keywords]
  form.match_mode = rule.match_mode
  form.trigger_mode = rule.trigger_mode
  form.accumulated_count = rule.accumulated_count
  form.accumulated_window = rule.accumulated_window
  form.action_type = rule.action_type
  form.action_duration = rule.action_duration
  form.action_recover_hour = rule.action_recover_hour
  form.platforms = [...rule.platforms]
  keywordsInput.value = rule.keywords.join('\n')
  errorCodesInput.value = rule.error_codes.length > 0 ? rule.error_codes.join(', ') : ''
  showEditModal.value = true
}

async function handleSubmit() {
  const keywords = keywordsInput.value
    .split('\n')
    .map(k => k.trim())
    .filter(k => k.length > 0)

  if (keywords.length === 0) {
    appStore.showError(t('admin.accountThrottle.form.keywordsRequired'))
    return
  }

  const errorCodes = errorCodesInput.value
    .split(/[,，\s]+/)
    .map(s => parseInt(s.trim(), 10))
    .filter(n => !isNaN(n) && n > 0)

  submitting.value = true
  try {
    const data = {
      name: form.name,
      priority: form.priority,
      description: form.description || null,
      error_codes: errorCodes,
      keywords,
      match_mode: form.match_mode,
      trigger_mode: form.trigger_mode,
      accumulated_count: form.accumulated_count,
      accumulated_window: form.accumulated_window,
      action_type: form.action_type,
      action_duration: form.action_duration,
      action_recover_hour: form.action_recover_hour,
      platforms: form.platforms
    }

    if (showEditModal.value && editingRuleId.value) {
      await accountThrottleAPI.update(editingRuleId.value, data)
      appStore.showSuccess(t('admin.accountThrottle.updateSuccess'))
    } else {
      await accountThrottleAPI.create(data)
      appStore.showSuccess(t('admin.accountThrottle.createSuccess'))
    }
    closeFormModal()
    await loadRules()
  } catch (err: any) {
    appStore.showError(err.message || 'Failed to save rule')
  } finally {
    submitting.value = false
  }
}

async function toggleEnabled(rule: AccountThrottleRule) {
  try {
    await accountThrottleAPI.toggleEnabled(rule.id, !rule.enabled)
    rule.enabled = !rule.enabled
  } catch (err: any) {
    appStore.showError(err.message || 'Failed to toggle')
  }
}

async function handleDelete(rule: AccountThrottleRule) {
  if (!confirm(t('admin.accountThrottle.deleteConfirm', { name: rule.name }))) return
  try {
    await accountThrottleAPI.delete(rule.id)
    appStore.showSuccess(t('admin.accountThrottle.deleteSuccess'))
    await loadRules()
  } catch (err: any) {
    appStore.showError(err.message || 'Failed to delete')
  }
}

watch(() => props.show, (val) => {
  if (val) loadRules()
})
</script>
