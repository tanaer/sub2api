<template>
  <!-- Row 1: Core Stats -->
  <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <!-- Balance -->
    <div v-if="!isSimple" class="card p-4">
      <div class="flex items-center gap-3">
        <div class="rounded-lg bg-emerald-100 p-2 dark:bg-emerald-900/30">
          <svg class="h-5 w-5 text-emerald-600 dark:text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.25 18.75a60.07 60.07 0 0115.797 2.101c.727.198 1.453-.342 1.453-1.096V18.75M3.75 4.5v.75A.75.75 0 013 6h-.75m0 0v-.375c0-.621.504-1.125 1.125-1.125H20.25M2.25 6v9m18-10.5v.75c0 .414.336.75.75.75h.75m-1.5-1.5h.375c.621 0 1.125.504 1.125 1.125v9.75c0 .621-.504 1.125-1.125 1.125h-.375m1.5-1.5H21a.75.75 0 00-.75.75v.75m0 0H3.75m0 0h-.375a1.125 1.125 0 01-1.125-1.125V15m1.5 1.5v-.75A.75.75 0 003 15h-.75M15 10.5a3 3 0 11-6 0 3 3 0 016 0zm3 0h.008v.008H18V10.5zm-12 0h.008v.008H6V10.5z" />
          </svg>
        </div>
        <div>
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.balance') }}</p>
          <p class="text-xl font-bold text-emerald-600 dark:text-emerald-400">${{ formatBalance(balance) }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('common.available') }}</p>
        </div>
      </div>
    </div>

    <!-- API Keys -->
    <div class="card p-4">
      <div class="flex items-center gap-3">
        <div class="rounded-lg bg-blue-100 p-2 dark:bg-blue-900/30">
          <Icon name="key" size="md" class="text-blue-600 dark:text-blue-400" :stroke-width="2" />
        </div>
        <div>
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.apiKeys') }}</p>
          <p class="text-xl font-bold text-gray-900 dark:text-white">{{ stats?.total_api_keys || 0 }}</p>
          <p class="text-xs text-green-600 dark:text-green-400">{{ stats?.active_api_keys || 0 }} {{ t('common.active') }}</p>
        </div>
      </div>
    </div>

    <!-- Today Requests -->
    <div class="card p-4">
      <div class="flex items-center gap-3">
        <div class="rounded-lg bg-green-100 p-2 dark:bg-green-900/30">
          <Icon name="chart" size="md" class="text-green-600 dark:text-green-400" :stroke-width="2" />
        </div>
        <div>
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.todayRequests') }}</p>
          <p class="text-xl font-bold text-gray-900 dark:text-white">{{ stats?.today_requests || 0 }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('common.total') }}: {{ formatNumber(stats?.total_requests || 0) }}</p>
        </div>
      </div>
    </div>

    <!-- Today Cost -->
    <div class="card p-4">
      <div class="flex items-center gap-3">
        <div class="rounded-lg bg-purple-100 p-2 dark:bg-purple-900/30">
          <Icon name="dollar" size="md" class="text-purple-600 dark:text-purple-400" :stroke-width="2" />
        </div>
        <div>
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.todayCost') }}</p>
          <p class="text-xl font-bold text-gray-900 dark:text-white">
            <span class="text-purple-600 dark:text-purple-400" :title="t('dashboard.actual')">${{ formatCost(stats?.today_actual_cost || 0) }}</span>
            <span class="text-sm font-normal text-gray-400 dark:text-gray-500" :title="t('dashboard.standard')"> / ${{ formatCost(stats?.today_cost || 0) }}</span>
          </p>
          <p class="text-xs">
            <span class="text-gray-500 dark:text-gray-400">{{ t('common.total') }}: </span>
            <span class="text-purple-600 dark:text-purple-400" :title="t('dashboard.actual')">${{ formatCost(stats?.total_actual_cost || 0) }}</span>
            <span class="text-gray-400 dark:text-gray-500" :title="t('dashboard.standard')"> / ${{ formatCost(stats?.total_cost || 0) }}</span>
          </p>
        </div>
      </div>
    </div>
  </div>

  <!-- Row 2: Token Stats -->
  <div class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <!-- Today Tokens -->
    <div class="card p-4">
      <div class="flex items-center gap-3">
        <div class="rounded-lg bg-amber-100 p-2 dark:bg-amber-900/30">
          <Icon name="cube" size="md" class="text-amber-600 dark:text-amber-400" :stroke-width="2" />
        </div>
        <div>
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.todayTokens') }}</p>
          <p class="text-xl font-bold text-gray-900 dark:text-white">{{ formatTokens(stats?.today_tokens || 0) }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.input') }}: {{ formatTokens(stats?.today_input_tokens || 0) }} / {{ t('dashboard.output') }}: {{ formatTokens(stats?.today_output_tokens || 0) }}</p>
        </div>
      </div>
    </div>

    <!-- Total Tokens -->
    <div class="card p-4">
      <div class="flex items-center gap-3">
        <div class="rounded-lg bg-indigo-100 p-2 dark:bg-indigo-900/30">
          <Icon name="database" size="md" class="text-indigo-600 dark:text-indigo-400" :stroke-width="2" />
        </div>
        <div>
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.totalTokens') }}</p>
          <p class="text-xl font-bold text-gray-900 dark:text-white">{{ formatTokens(stats?.total_tokens || 0) }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.input') }}: {{ formatTokens(stats?.total_input_tokens || 0) }} / {{ t('dashboard.output') }}: {{ formatTokens(stats?.total_output_tokens || 0) }}</p>
        </div>
      </div>
    </div>

    <!-- Performance (RPM/TPM) -->
    <div class="card p-4">
      <div class="flex items-center gap-3">
        <div class="rounded-lg bg-violet-100 p-2 dark:bg-violet-900/30">
          <Icon name="bolt" size="md" class="text-violet-600 dark:text-violet-400" :stroke-width="2" />
        </div>
        <div class="flex-1">
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.performance') }}</p>
          <div class="flex items-baseline gap-2">
            <p class="text-xl font-bold text-gray-900 dark:text-white">{{ formatTokens(stats?.rpm || 0) }}</p>
            <span class="text-xs text-gray-500 dark:text-gray-400">RPM</span>
          </div>
          <div class="flex items-baseline gap-2">
            <p class="text-sm font-semibold text-violet-600 dark:text-violet-400">{{ formatTokens(stats?.tpm || 0) }}</p>
            <span class="text-xs text-gray-500 dark:text-gray-400">TPM</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Avg Response Time -->
    <div class="card p-4">
      <div class="flex items-center gap-3">
        <div class="rounded-lg bg-rose-100 p-2 dark:bg-rose-900/30">
          <Icon name="clock" size="md" class="text-rose-600 dark:text-rose-400" :stroke-width="2" />
        </div>
        <div>
          <p class="text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.avgResponse') }}</p>
          <p class="text-xl font-bold text-gray-900 dark:text-white">{{ formatDuration(stats?.average_duration_ms || 0) }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.averageTime') }}</p>
        </div>
      </div>
    </div>
  </div>

  <div v-if="stats?.group_request_quotas?.length" class="card p-4">
    <div class="mb-4 flex items-start justify-between gap-3">
      <div>
        <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('dashboard.groupRequestQuota') }}</p>
        <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('dashboard.groupRequestQuotaHint') }}</p>
      </div>
      <div class="rounded-lg bg-cyan-100 p-2 dark:bg-cyan-900/30">
        <Icon name="grid" size="md" class="text-cyan-600 dark:text-cyan-400" :stroke-width="2" />
      </div>
    </div>

    <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      <div
        v-for="groupQuota in stats.group_request_quotas"
        :key="groupQuota.group_id"
        class="rounded-xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-600 dark:bg-dark-800/70"
      >
        <!-- 头部：分组名 + 剩余次数 -->
        <div class="flex items-start justify-between gap-3">
          <div>
            <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ groupQuota.group_name }}</p>
            <p class="text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ groupQuota.platform }}</p>
          </div>
          <div class="text-right">
            <p class="text-lg font-bold" :class="groupQuota.request_quota_remaining > 0 ? 'text-emerald-600 dark:text-emerald-400' : 'text-red-500'">
              {{ groupQuota.request_quota_remaining }}
            </p>
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('keyUsage.requestQuotaRemaining') }}</p>
          </div>
        </div>

        <!-- 总计进度条 -->
        <div class="mt-3">
          <div class="mb-1 flex items-center justify-between text-xs text-gray-500 dark:text-gray-400">
            <span>{{ t('keyUsage.requestQuotaUsed') }}: {{ groupQuota.request_quota_used }}</span>
            <span>{{ t('keyUsage.requestQuota') }}: {{ groupQuota.request_quota }}</span>
          </div>
          <div class="h-2 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
            <div
              class="h-full rounded-full transition-all"
              :class="quotaProgressColor(groupQuota.request_quota_used, groupQuota.request_quota)"
              :style="{ width: quotaProgressWidth(groupQuota.request_quota_used, groupQuota.request_quota) }"
            />
          </div>
        </div>

        <!-- 额度明细：仅当有多个来源时展示，帮助用户了解各笔到期时间 -->
        <div v-if="hasMultipleSources(groupQuota) || hasActiveGrants(groupQuota)" class="mt-3">
          <p class="mb-1.5 text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('dashboard.quotaBreakdown') }}</p>
          <div class="space-y-1">
            <!-- 永久额度（仅多来源时展示） -->
            <div
              v-if="groupQuota.permanent_quota > 0 && hasActiveGrants(groupQuota)"
              class="flex items-center justify-between rounded-lg bg-white px-3 py-1.5 text-xs dark:bg-dark-700/80"
            >
              <div class="flex items-center gap-1.5">
                <span class="inline-block h-1.5 w-1.5 rounded-full bg-blue-500" />
                <span class="text-gray-600 dark:text-gray-300">{{ t('dashboard.permanentQuota') }}</span>
              </div>
              <span class="font-medium text-gray-900 dark:text-white">
                {{ Math.max(groupQuota.permanent_quota - groupQuota.permanent_quota_used, 0) }}
                <span class="text-gray-400"> / {{ groupQuota.permanent_quota }}</span>
              </span>
            </div>
            <!-- 次数包（只显示未过期的） -->
            <div
              v-for="grant in activeGrants(groupQuota)"
              :key="grant.expires_at"
              class="rounded-lg bg-white px-3 py-1.5 text-xs dark:bg-dark-700/80"
            >
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-1.5">
                  <span class="inline-block h-1.5 w-1.5 rounded-full" :class="isExpiringSoon(grant.expires_at) ? 'bg-amber-500' : 'bg-emerald-500'" />
                  <span class="text-gray-600 dark:text-gray-300">{{ grant.request_quota_total }} {{ t('dashboard.grantUnit') }}</span>
                </div>
                <span class="font-medium text-gray-900 dark:text-white">
                  {{ t('keyUsage.requestQuotaRemaining') }}
                  {{ Math.max(grant.request_quota_total - grant.request_quota_used, 0) }}
                </span>
              </div>
              <div class="mt-0.5 flex items-center gap-1 pl-3">
                <Icon name="clock" size="xs" :class="isExpiringSoon(grant.expires_at) ? 'text-amber-500' : 'text-gray-400'" />
                <span :class="isExpiringSoon(grant.expires_at) ? 'text-amber-500' : 'text-gray-400'">
                  {{ t('dashboard.expiresAt', { time: formatDate(grant.expires_at) }) }}
                </span>
              </div>
            </div>
          </div>
        </div>

        <!-- 单笔次数包的到期提示（只有一笔 grant 且无永久额度时，简化为一行提示） -->
        <div
          v-else-if="singleActiveGrant(groupQuota)"
          class="mt-2 flex items-center gap-1 text-xs"
        >
          <Icon name="clock" size="xs" :class="isExpiringSoon(singleActiveGrant(groupQuota)!.expires_at) ? 'text-amber-500' : 'text-gray-400'" />
          <span :class="isExpiringSoon(singleActiveGrant(groupQuota)!.expires_at) ? 'text-amber-500' : 'text-gray-400'">
            {{ t('dashboard.expiresAt', { time: formatDate(singleActiveGrant(groupQuota)!.expires_at) }) }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { UserDashboardStats as UserStatsType } from '@/api/usage'

defineProps<{
  stats: UserStatsType
  balance: number
  isSimple: boolean
}>()
const { t } = useI18n()

const formatBalance = (b: number) =>
  new Intl.NumberFormat('en-US', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(b)

const formatNumber = (n: number) => n.toLocaleString()
const formatCost = (c: number) => c.toFixed(4)
const formatTokens = (t: number) => {
  if (t >= 1_000_000) return `${(t / 1_000_000).toFixed(1)}M`
  if (t >= 1000) return `${(t / 1000).toFixed(1)}K`
  return t.toString()
}
const formatDuration = (ms: number) => ms >= 1000 ? `${(ms / 1000).toFixed(2)}s` : `${ms.toFixed(0)}ms`
const formatDate = (dateStr: string) => {
  const d = new Date(dateStr)
  return d.toLocaleDateString(undefined, { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}
const isExpiringSoon = (dateStr: string) => {
  const d = new Date(dateStr)
  const now = new Date()
  const diffDays = (d.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)
  return diffDays > 0 && diffDays <= 3
}
const quotaProgressWidth = (used: number, total: number) => {
  if (total <= 0) return '0%'
  return `${Math.min((used / total) * 100, 100)}%`
}
const quotaProgressColor = (used: number, total: number) => {
  if (total <= 0) return 'bg-gray-400'
  const pct = used / total
  if (pct >= 0.9) return 'bg-red-500'
  if (pct >= 0.7) return 'bg-amber-500'
  return 'bg-cyan-500'
}

type GroupQuota = NonNullable<UserStatsType['group_request_quotas']>[number]
type Grant = NonNullable<GroupQuota['grants']>[number]

const activeGrants = (gq: GroupQuota): Grant[] =>
  gq.grants?.filter((g) => !g.expired) ?? []

const hasActiveGrants = (gq: GroupQuota): boolean =>
  activeGrants(gq).length > 0

const hasMultipleSources = (gq: GroupQuota): boolean => {
  const sources = (gq.permanent_quota > 0 ? 1 : 0) + activeGrants(gq).length
  return sources > 1
}

const singleActiveGrant = (gq: GroupQuota): Grant | null => {
  const grants = activeGrants(gq)
  if (grants.length === 1 && gq.permanent_quota <= 0) return grants[0]
  return null
}
</script>
