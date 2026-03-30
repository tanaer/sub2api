import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import KeysView from '../KeysView.vue'

const {
  list,
  getDashboardApiKeysUsage,
  getAvailable,
  getUserGroupRates,
  getPublicSettings,
  showError,
  showSuccess,
  nextStep,
  isCurrentStep,
} = vi.hoisted(() => ({
  list: vi.fn(),
  getDashboardApiKeysUsage: vi.fn(),
  getAvailable: vi.fn(),
  getUserGroupRates: vi.fn(),
  getPublicSettings: vi.fn(),
  showError: vi.fn(),
  showSuccess: vi.fn(),
  nextStep: vi.fn(),
  isCurrentStep: vi.fn(() => false),
}))

const messages: Record<string, string> = {
  'common.name': 'Name',
  'keys.apiKey': 'API Key',
  'keys.group': 'Group',
  'keys.usage': 'Usage',
  'keys.rateLimitColumn': 'Rate Limit',
  'keys.expiresAt': 'Expires At',
  'common.status': 'Status',
  'keys.lastUsedAt': 'Last Used',
  'keys.created': 'Created',
  'common.actions': 'Actions',
  'keys.today': 'Today',
  'keys.total': 'Total',
  'keys.quota': 'Quota',
  'keyUsage.requestQuota': 'Request Quota',
  'keyUsage.requestQuotaRemaining': 'Remaining Requests',
  'keyUsage.requestQuotaUsed': 'Used Requests',
  'keys.failedToLoad': 'Failed to load',
}

vi.mock('@/api', () => ({
  keysAPI: {
    list,
  },
  usageAPI: {
    getDashboardApiKeysUsage,
  },
  userGroupsAPI: {
    getAvailable,
    getUserGroupRates,
  },
  authAPI: {
    getPublicSettings,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError, showSuccess }),
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    nextStep,
    isCurrentStep,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => messages[key] ?? key,
    }),
  }
})

const AppLayoutStub = { template: '<div><slot /></div>' }
const TablePageLayoutStub = {
  template: '<div><slot name="filters" /><slot name="actions" /><slot name="table" /></div>',
}
const DataTableStub = {
  props: ['data'],
  template: `
    <div>
      <div v-for="row in data" :key="row.id">
        <slot name="cell-usage" :row="row" :value="row.usage" />
      </div>
    </div>
  `,
}

describe('user KeysView request quota display', () => {
  beforeEach(() => {
    list.mockReset()
    getDashboardApiKeysUsage.mockReset()
    getAvailable.mockReset()
    getUserGroupRates.mockReset()
    getPublicSettings.mockReset()
    showError.mockReset()
    showSuccess.mockReset()
    nextStep.mockReset()
    isCurrentStep.mockReset()
    isCurrentStep.mockReturnValue(false)

    ;(globalThis as any).ResizeObserver = class {
      observe() {}
      disconnect() {}
    }
  })

  it('shows request quota usage and remaining counts for each key', async () => {
    list.mockResolvedValue({
      items: [
        {
          id: 1,
          name: 'quota-key',
          key: 'sk-test-1234567890abcdef',
          request_quota: 12,
          request_quota_used: 5,
          quota: 0,
          quota_used: 0,
          status: 'active',
          group_id: null,
          ip_whitelist: [],
          ip_blacklist: [],
          last_used_at: null,
          expires_at: null,
          created_at: '2026-03-27T00:00:00Z',
          updated_at: '2026-03-27T00:00:00Z',
          rate_limit_5h: 0,
          rate_limit_1d: 0,
          rate_limit_7d: 0,
          usage_5h: 0,
          usage_1d: 0,
          usage_7d: 0,
          window_5h_start: null,
          window_1d_start: null,
          window_7d_start: null,
          reset_5h_at: null,
          reset_1d_at: null,
          reset_7d_at: null,
        },
      ],
      total: 1,
      pages: 1,
    })
    getDashboardApiKeysUsage.mockResolvedValue({ stats: {} })
    getAvailable.mockResolvedValue([])
    getUserGroupRates.mockResolvedValue({})
    getPublicSettings.mockResolvedValue({})

    const wrapper = mount(KeysView, {
      global: {
        stubs: {
          AppLayout: AppLayoutStub,
          TablePageLayout: TablePageLayoutStub,
          DataTable: DataTableStub,
          Pagination: true,
          BaseDialog: true,
          ConfirmDialog: true,
          EmptyState: true,
          Select: true,
          SearchInput: true,
          Icon: true,
          UseKeyModal: true,
          GroupBadge: true,
          GroupOptionItem: true,
          Teleport: true,
        },
      },
    })

    await flushPromises()

    const text = wrapper.text()
    expect(text).toContain('Request Quota')
    expect(text).toContain('Used Requests')
    expect(text).toContain('5 / 12')
    expect(text).toContain('Remaining Requests')
    expect(text).toContain('7')
  })
})
