<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-xl font-bold text-gray-900 dark:text-white">{{ t('admin.subscriptionPlans.title') }}</h1>
          <p class="text-sm text-gray-500 dark:text-gray-400">{{ t('admin.subscriptionPlans.description') }}</p>
        </div>
        <button @click="openCreateModal" class="btn btn-primary">
          {{ t('admin.subscriptionPlans.create') }}
        </button>
      </div>

      <!-- Plans List -->
      <div class="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="plan in plans"
          :key="plan.id"
          class="card p-5 transition-shadow hover:shadow-md"
        >
          <div class="flex items-start justify-between">
            <div>
              <h3 class="font-semibold text-gray-900 dark:text-white">{{ plan.name }}</h3>
              <p v-if="plan.description" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ plan.description }}</p>
            </div>
            <span
              class="rounded-full px-2 py-0.5 text-xs font-medium"
              :class="plan.status === 'active' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' : 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'"
            >
              {{ plan.status === 'active' ? t('admin.subscriptionPlans.active') : t('admin.subscriptionPlans.archived') }}
            </span>
          </div>

          <div class="mt-3 space-y-2">
            <div v-if="plan.group" class="flex items-center gap-2 text-sm">
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.subscriptionPlans.group') }}:</span>
              <span class="font-medium text-gray-700 dark:text-gray-300">{{ plan.group.name }}</span>
              <span class="text-xs text-gray-400">({{ plan.group.platform }})</span>
            </div>
            <div class="flex items-center gap-2 text-sm">
              <span class="rounded bg-purple-100 px-1.5 py-0.5 text-xs font-medium text-purple-700 dark:bg-purple-900/30 dark:text-purple-400">
                {{ plan.billing_mode === 'per_request' ? t('admin.subscriptionPlans.perRequest') : t('admin.subscriptionPlans.perUsd') }}
              </span>
              <span class="text-gray-600 dark:text-gray-300">
                <template v-if="plan.billing_mode === 'per_request'">
                  {{ plan.request_quota.toLocaleString() }} {{ t('admin.subscriptionPlans.requests') }}
                </template>
                <template v-else>
                  <span v-if="plan.daily_limit_usd">{{ t('admin.subscriptionPlans.daily') }} ${{ plan.daily_limit_usd }}</span>
                  <span v-if="plan.weekly_limit_usd"> / {{ t('admin.subscriptionPlans.weekly') }} ${{ plan.weekly_limit_usd }}</span>
                  <span v-if="plan.monthly_limit_usd"> / {{ t('admin.subscriptionPlans.monthly') }} ${{ plan.monthly_limit_usd }}</span>
                </template>
              </span>
            </div>
            <div class="text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.subscriptionPlans.validityDays') }}: {{ plan.validity_days }} {{ t('admin.subscriptionPlans.days') }}
            </div>
          </div>

          <div class="mt-4 flex gap-2 border-t pt-3 dark:border-dark-600">
            <button @click="openEditModal(plan)" class="btn btn-secondary btn-sm flex-1">
              {{ t('common.edit') }}
            </button>
            <button @click="handleDelete(plan)" class="btn btn-danger btn-sm">
              {{ t('common.delete') }}
            </button>
          </div>
        </div>
      </div>

      <div v-if="plans.length === 0 && !loading" class="card p-10 text-center">
        <p class="text-gray-500 dark:text-gray-400">{{ t('admin.subscriptionPlans.empty') }}</p>
      </div>
    </div>

    <!-- Create/Edit Modal -->
    <BaseDialog
      :show="showModal"
      :title="editingPlan ? t('admin.subscriptionPlans.editPlan') : t('admin.subscriptionPlans.createPlan')"
      width="narrow"
      @close="closeModal"
    >
      <form id="plan-form" @submit.prevent="handleSubmit" class="space-y-4">
        <div>
          <label class="input-label">{{ t('admin.subscriptionPlans.planName') }}</label>
          <input v-model="form.name" type="text" required maxlength="100" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.subscriptionPlans.planDescription') }}</label>
          <input v-model="form.description" type="text" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.subscriptionPlans.group') }}</label>
          <select v-model="form.group_id" class="input">
            <option :value="undefined">{{ t('admin.subscriptionPlans.noGroup') }}</option>
            <option v-for="g in groups" :key="g.id" :value="g.id">{{ g.name }} ({{ g.platform }})</option>
          </select>
        </div>
        <div>
          <label class="input-label">{{ t('admin.subscriptionPlans.billingMode') }}</label>
          <select v-model="form.billing_mode" class="input">
            <option value="per_request">{{ t('admin.subscriptionPlans.perRequest') }}</option>
            <option value="per_usd">{{ t('admin.subscriptionPlans.perUsd') }}</option>
          </select>
        </div>

        <!-- Per-request fields -->
        <div v-if="form.billing_mode === 'per_request'">
          <label class="input-label">{{ t('admin.subscriptionPlans.requestQuota') }}</label>
          <input v-model.number="form.request_quota" type="number" min="1" required class="input" />
        </div>

        <!-- Per-USD fields -->
        <template v-if="form.billing_mode === 'per_usd'">
          <div>
            <label class="input-label">{{ t('admin.subscriptionPlans.dailyLimit') }}</label>
            <input v-model.number="form.daily_limit_usd" type="number" step="0.01" min="0" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.subscriptionPlans.weeklyLimit') }}</label>
            <input v-model.number="form.weekly_limit_usd" type="number" step="0.01" min="0" class="input" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.subscriptionPlans.monthlyLimit') }}</label>
            <input v-model.number="form.monthly_limit_usd" type="number" step="0.01" min="0" class="input" />
          </div>
        </template>

        <div>
          <label class="input-label">{{ t('admin.subscriptionPlans.validityDays') }}</label>
          <input v-model.number="form.validity_days" type="number" min="1" max="36500" required class="input" />
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeModal" type="button" class="btn btn-secondary">{{ t('common.cancel') }}</button>
          <button type="submit" form="plan-form" class="btn btn-primary" :disabled="submitting">
            {{ submitting ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { subscriptionPlansAPI, type SubscriptionPlan, type CreateSubscriptionPlanRequest } from '@/api/admin/subscriptionPlans'
import { useAppStore } from '@/stores'
import { getAll as getAllGroups } from '@/api/admin/groups'

const { t } = useI18n()
const appStore = useAppStore()

const plans = ref<SubscriptionPlan[]>([])
const groups = ref<Array<{ id: number; name: string; platform: string }>>([])
const loading = ref(false)
const submitting = ref(false)
const showModal = ref(false)
const editingPlan = ref<SubscriptionPlan | null>(null)

const form = reactive<CreateSubscriptionPlanRequest & { group_id?: number }>({
  name: '',
  description: '',
  group_id: undefined,
  billing_mode: 'per_request',
  request_quota: 2000,
  daily_limit_usd: 0,
  weekly_limit_usd: 0,
  monthly_limit_usd: 0,
  validity_days: 30,
})

async function loadPlans() {
  loading.value = true
  try {
    plans.value = await subscriptionPlansAPI.list()
  } catch {
    appStore.showError('Failed to load subscription plans')
  } finally {
    loading.value = false
  }
}

async function loadGroups() {
  try {
    const all = await getAllGroups()
    groups.value = all.map(g => ({ id: g.id, name: g.name, platform: g.platform }))
  } catch { /* ignore */ }
}

function openCreateModal() {
  editingPlan.value = null
  Object.assign(form, {
    name: '', description: '', group_id: undefined,
    billing_mode: 'per_request', request_quota: 2000,
    daily_limit_usd: 0, weekly_limit_usd: 0, monthly_limit_usd: 0,
    validity_days: 30,
  })
  showModal.value = true
}

function openEditModal(plan: SubscriptionPlan) {
  editingPlan.value = plan
  Object.assign(form, {
    name: plan.name,
    description: plan.description || '',
    group_id: plan.group_id || undefined,
    billing_mode: plan.billing_mode,
    request_quota: plan.request_quota,
    daily_limit_usd: plan.daily_limit_usd,
    weekly_limit_usd: plan.weekly_limit_usd,
    monthly_limit_usd: plan.monthly_limit_usd,
    validity_days: plan.validity_days,
  })
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingPlan.value = null
}

async function handleSubmit() {
  submitting.value = true
  try {
    if (editingPlan.value) {
      await subscriptionPlansAPI.update(editingPlan.value.id, form)
      appStore.showSuccess(t('admin.subscriptionPlans.updated'))
    } else {
      await subscriptionPlansAPI.create(form)
      appStore.showSuccess(t('admin.subscriptionPlans.created'))
    }
    closeModal()
    loadPlans()
  } catch (e: any) {
    appStore.showError(e.response?.data?.message || 'Failed')
  } finally {
    submitting.value = false
  }
}

async function handleDelete(plan: SubscriptionPlan) {
  if (!confirm(t('admin.subscriptionPlans.confirmDelete', { name: plan.name }))) return
  try {
    await subscriptionPlansAPI.remove(plan.id)
    appStore.showSuccess(t('admin.subscriptionPlans.deleted'))
    loadPlans()
  } catch (e: any) {
    appStore.showError(e.response?.data?.message || 'Failed to delete')
  }
}

onMounted(() => {
  loadPlans()
  loadGroups()
})
</script>
