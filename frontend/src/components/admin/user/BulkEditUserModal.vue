<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.bulkEdit.title')"
    @close="handleClose"
  >
    <form id="bulk-edit-user-form" class="space-y-5" @submit.prevent="handleSubmit">
      <!-- Info -->
      <div class="rounded-lg bg-blue-50 p-4 dark:bg-blue-900/20">
        <p class="text-sm text-blue-700 dark:text-blue-400">
          <svg class="mr-1.5 inline h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          {{ t('admin.users.bulkEdit.selectionInfo', { count: userIds.length }) }}
        </p>
      </div>

      <!-- Status -->
      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between">
          <label
            id="bulk-edit-user-status-label"
            class="input-label mb-0"
          >
            {{ t('common.status') }}
          </label>
          <input
            v-model="enableStatus"
            type="checkbox"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
        </div>
        <div :class="!enableStatus && 'pointer-events-none opacity-50'">
          <Select
            v-model="status"
            :options="statusOptions"
            aria-labelledby="bulk-edit-user-status-label"
          />
        </div>
      </div>

      <!-- Concurrency -->
      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between">
          <label
            id="bulk-edit-user-concurrency-label"
            class="input-label mb-0"
          >
            {{ t('admin.users.columns.concurrency') }}
          </label>
          <input
            v-model="enableConcurrency"
            type="checkbox"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
        </div>
        <input
          v-model.number="concurrency"
          type="number"
          min="0"
          :disabled="!enableConcurrency"
          class="input"
          :class="!enableConcurrency && 'cursor-not-allowed opacity-50'"
          :placeholder="t('admin.users.bulkEdit.concurrencyPlaceholder')"
          aria-labelledby="bulk-edit-user-concurrency-label"
        />
        <p class="input-hint">{{ t('admin.users.bulkEdit.concurrencyHint') }}</p>
      </div>

      <!-- Notes -->
      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between">
          <label
            id="bulk-edit-user-notes-label"
            class="input-label mb-0"
          >
            {{ t('admin.users.columns.notes') }}
          </label>
          <input
            v-model="enableNotes"
            type="checkbox"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
        </div>
        <textarea
          v-model="notes"
          :disabled="!enableNotes"
          class="input min-h-[80px]"
          :class="!enableNotes && 'cursor-not-allowed opacity-50'"
          :placeholder="t('admin.users.bulkEdit.notesPlaceholder')"
          aria-labelledby="bulk-edit-user-notes-label"
        ></textarea>
      </div>

      <!-- Allowed Groups -->
      <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="mb-3 flex items-center justify-between">
          <label
            id="bulk-edit-user-groups-label"
            class="input-label mb-0"
          >
            {{ t('admin.users.columns.groups') }}
          </label>
          <input
            v-model="enableGroups"
            type="checkbox"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
        </div>
        <div :class="!enableGroups && 'pointer-events-none opacity-50'">
          <GroupSelector
            v-model="groupIds"
            :groups="groups"
            aria-labelledby="bulk-edit-user-groups-label"
          />
          <p class="input-hint">{{ t('admin.users.bulkEdit.groupsHint') }}</p>
        </div>
      </div>

      <!-- Footer -->
      <div class="flex justify-end gap-3 border-t border-gray-200 pt-4 dark:border-dark-600">
        <button type="button" class="btn btn-secondary" @click="handleClose">
          {{ t('common.cancel') }}
        </button>
        <button
          type="submit"
          class="btn btn-primary"
          :disabled="loading || !hasChanges"
        >
          <span v-if="loading" class="animate-spin mr-2 inline-block h-4 w-4 border-2 border-white border-t-transparent rounded-full"></span>
          {{ t('common.save') }}
        </button>
      </div>
    </form>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import GroupSelector from '@/components/common/GroupSelector.vue'
import { adminAPI } from '@/api/admin'
import type { AdminGroup } from '@/types'

const props = defineProps<{
  show: boolean
  userIds: number[]
}>()

const emit = defineEmits<{
  close: []
  success: [updated: number]
}>()

const { t } = useI18n()
const loading = ref(false)
const groups = ref<AdminGroup[]>([])
const loadGroups = async () => {
  if (groups.value.length > 0) return
  try {
    groups.value = await adminAPI.groups.getAll()
  } catch (e) {
    console.error('Failed to load groups:', e)
  }
}

// Field toggles
const enableStatus = ref(false)
const enableConcurrency = ref(false)
const enableNotes = ref(false)
const enableGroups = ref(false)

// Field values
const status = ref<'active' | 'disabled'>('active')
const concurrency = ref<number>(0)
const notes = ref('')
const groupIds = ref<number[]>([])

const statusOptions = [
  { value: 'active', label: t('common.active') },
  { value: 'disabled', label: t('admin.users.disabled') }
]

// Reset fields when modal opens
watch(() => props.show, (val) => {
  if (val) {
    loadGroups()
    enableStatus.value = false
    enableConcurrency.value = false
    enableNotes.value = false
    enableGroups.value = false
    status.value = 'active'
    concurrency.value = 0
    notes.value = ''
    groupIds.value = []
  }
})

const hasChanges = computed(() =>
  enableStatus.value || enableConcurrency.value || enableNotes.value || enableGroups.value
)

async function handleSubmit() {
  if (!hasChanges.value) return

  loading.value = true
  try {
    const fields: Record<string, unknown> = {}
    if (enableStatus.value) fields.status = status.value
    if (enableConcurrency.value) fields.concurrency = concurrency.value
    if (enableNotes.value) fields.notes = notes.value
    if (enableGroups.value) fields.allowed_groups = groupIds.value

    const result = await adminAPI.users.batchUpdate(props.userIds, fields as Parameters<typeof adminAPI.users.batchUpdate>[1])

    if (result.errors.length > 0) {
      console.warn('Batch update errors:', result.errors)
    }

    emit('success', result.updated)
    handleClose()
  } catch (err) {
    console.error('Batch update failed:', err)
  } finally {
    loading.value = false
  }
}

function handleClose() {
  emit('close')
}
</script>
