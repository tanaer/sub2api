import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import UserDashboardStats from '../UserDashboardStats.vue'

const messages: Record<string, string> = {
  'dashboard.balance': 'Balance',
  'dashboard.apiKeys': 'API Keys',
  'dashboard.todayRequests': 'Today Requests',
  'dashboard.todayCost': 'Today Cost',
  'dashboard.todayTokens': 'Today Tokens',
  'dashboard.totalTokens': 'Total Tokens',
  'dashboard.performance': 'Performance',
  'dashboard.avgResponse': 'Avg Response',
  'dashboard.averageTime': 'Average Time',
  'dashboard.actual': 'Actual',
  'dashboard.standard': 'Standard',
  'dashboard.input': 'Input',
  'dashboard.output': 'Output',
  'dashboard.groupRequestQuota': 'Group Request Quota',
  'dashboard.groupRequestQuotaHint': 'Used / Remaining for each group',
  'common.available': 'Available',
  'common.active': 'Active',
  'common.total': 'Total',
  'keyUsage.requestQuota': 'Request Quota',
  'keyUsage.requestQuotaUsed': 'Used Requests',
  'keyUsage.requestQuotaRemaining': 'Remaining Requests',
}

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

describe('UserDashboardStats', () => {
  it('renders per-group request quota usage and remaining counts', () => {
    const wrapper = mount(UserDashboardStats, {
      props: {
        balance: 10,
        isSimple: false,
        stats: {
          total_api_keys: 1,
          active_api_keys: 1,
          total_requests: 12,
          total_input_tokens: 100,
          total_output_tokens: 50,
          total_cache_creation_tokens: 0,
          total_cache_read_tokens: 0,
          total_tokens: 150,
          total_cost: 1,
          total_actual_cost: 1,
          today_requests: 2,
          today_input_tokens: 10,
          today_output_tokens: 5,
          today_cache_creation_tokens: 0,
          today_cache_read_tokens: 0,
          today_tokens: 15,
          today_cost: 0.1,
          today_actual_cost: 0.1,
          average_duration_ms: 120,
          rpm: 3,
          tpm: 30,
          group_request_quotas: [
            {
              group_id: 11,
              group_name: 'GLM-5',
              platform: 'openai',
              request_quota: 5,
              request_quota_used: 2,
              request_quota_remaining: 3,
            },
          ],
        },
      },
      global: {
        stubs: {
          Icon: true,
        },
      },
    })

    const text = wrapper.text()
    expect(text).toContain('Group Request Quota')
    expect(text).toContain('GLM-5')
    expect(text).toContain('Used Requests')
    expect(text).toContain('2')
    expect(text).toContain('Remaining Requests')
    expect(text).toContain('3')
  })
})
