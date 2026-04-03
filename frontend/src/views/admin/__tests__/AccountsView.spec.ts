import { beforeEach, describe, expect, it, vi } from 'vitest'
import { computed, defineComponent, h, reactive, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const accountsRef = ref([
  {
    id: 1,
    name: 'Whitelist Account',
    platform: 'anthropic',
    type: 'apikey',
    credentials: {
      model_mapping: {
        'claude-sonnet-4': 'claude-sonnet-4'
      }
    },
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: false,
    created_at: '2026-04-04T00:00:00Z',
    updated_at: '2026-04-04T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null
  },
  {
    id: 2,
    name: 'Mapping Account',
    platform: 'openai',
    type: 'apikey',
    credentials: {
      model_mapping: {
        sonnet: 'glm-4.5-air'
      }
    },
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: false,
    created_at: '2026-04-04T00:00:00Z',
    updated_at: '2026-04-04T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null
  },
  {
    id: 3,
    name: 'All Models Account',
    platform: 'gemini',
    type: 'apikey',
    credentials: {},
    proxy_id: null,
    concurrency: 1,
    priority: 1,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: false,
    created_at: '2026-04-04T00:00:00Z',
    updated_at: '2026-04-04T00:00:00Z',
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null
  }
] as any[])

const {
  loadMock,
  reloadMock,
  debouncedReloadMock,
  getBatchTodayStatsMock,
  listMock,
  getAllProxiesMock,
  getAllGroupsMock
} = vi.hoisted(() => ({
  loadMock: vi.fn().mockResolvedValue(undefined),
  reloadMock: vi.fn().mockResolvedValue(undefined),
  debouncedReloadMock: vi.fn(),
  getBatchTodayStatsMock: vi.fn().mockResolvedValue({ stats: {} }),
  listMock: vi.fn(),
  getAllProxiesMock: vi.fn().mockResolvedValue([]),
  getAllGroupsMock: vi.fn().mockResolvedValue([])
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    isSimpleMode: true
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listMock,
      getBatchTodayStats: getBatchTodayStatsMock
    },
    proxies: {
      getAll: getAllProxiesMock
    },
    groups: {
      getAll: getAllGroupsMock
    }
  }
}))

vi.mock('@/composables/useTableLoader', () => ({
  useTableLoader: () => ({
    items: accountsRef,
    loading: ref(false),
    params: reactive({ platform: '', type: '', status: '', privacy_mode: '', group: '', search: '', model: '' }),
    pagination: ref({ page: 1, page_size: 20, total: accountsRef.value.length }),
    load: loadMock,
    reload: reloadMock,
    debouncedReload: debouncedReloadMock,
    handlePageChange: vi.fn(),
    handlePageSizeChange: vi.fn()
  })
}))

vi.mock('@/composables/useSwipeSelect', () => ({
  useSwipeSelect: vi.fn()
}))

vi.mock('@/composables/useTableSelection', () => ({
  useTableSelection: () => ({
    selectedIds: ref<number[]>([]),
    allVisibleSelected: computed(() => false),
    isSelected: () => false,
    setSelectedIds: vi.fn(),
    select: vi.fn(),
    deselect: vi.fn(),
    toggle: vi.fn(),
    clear: vi.fn(),
    removeMany: vi.fn(),
    toggleVisible: vi.fn(),
    selectVisible: vi.fn()
  })
}))

vi.mock('@/utils/format', () => ({
  formatDateTime: vi.fn(() => '-'),
  formatRelativeTime: vi.fn(() => 'common.time.never')
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

import AccountsView from '../AccountsView.vue'

const AppLayoutStub = defineComponent({
  name: 'AppLayout',
  template: '<div><slot /></div>'
})

const TablePageLayoutStub = defineComponent({
  name: 'TablePageLayout',
  template: '<div><slot name="filters" /><slot name="table" /></div>'
})

const AccountTableActionsStub = defineComponent({
  name: 'AccountTableActions',
  template: '<div><slot name="beforeCreate" /><slot /><slot name="after" /></div>'
})

const DataTableStub = defineComponent({
  name: 'DataTable',
  props: {
    columns: {
      type: Array,
      default: () => []
    },
    data: {
      type: Array,
      default: () => []
    }
  },
  setup(props, { slots }) {
    return () =>
      h('div', [
        h(
          'div',
          { 'data-testid': 'column-labels' },
          (props.columns as Array<{ key: string; label: string }>).map((column) =>
            h('span', { key: column.key, 'data-testid': `column-${column.key}` }, column.label)
          )
        ),
        ...(props.data as Array<Record<string, any>>).map((row) =>
          h(
            'div',
            { key: row.id, 'data-testid': `row-${row.id}` },
            (props.columns as Array<{ key: string }>).map((column) =>
              h(
                'div',
                { key: column.key, 'data-testid': `cell-${column.key}-${row.id}` },
                slots[`cell-${column.key}`]
                  ? slots[`cell-${column.key}`]!({ row, value: row[column.key] })
                  : String(row[column.key] ?? '')
              )
            )
          )
        )
      ])
  }
})

function mountView() {
  return mount(AccountsView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        Pagination: true,
        ConfirmDialog: true,
        CreateAccountModal: true,
        EditAccountModal: true,
        BulkEditAccountModal: true,
        SyncFromCrsModal: true,
        TempUnschedStatusModal: true,
        AccountTableActions: AccountTableActionsStub,
        AccountTableFilters: true,
        AccountBulkActionsBar: true,
        AccountActionMenu: true,
        ImportDataModal: true,
        ReAuthAccountModal: true,
        AccountTestModal: true,
        AccountStatsModal: true,
        ScheduledTestsPanel: true,
        AccountStatusIndicator: true,
        AccountUsageCell: true,
        AccountTodayStatsCell: true,
        AccountGroupsCell: true,
        AccountCapacityCell: true,
        PlatformTypeBadge: true,
        ErrorPassthroughRulesModal: true,
        AccountThrottleRulesModal: true,
        TLSFingerprintProfilesModal: true,
        ProviderTimeoutModal: true,
        SLAMonitorModal: true,
        Icon: true
      }
    }
  })
}

describe('AccountsView', () => {
  beforeEach(() => {
    loadMock.mockClear()
    reloadMock.mockClear()
    debouncedReloadMock.mockClear()
    getBatchTodayStatsMock.mockClear()
    getBatchTodayStatsMock.mockResolvedValue({ stats: {} })
    localStorage.clear()
  })

  it('显示白名单、映射和全部模型三种模型限制摘要', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.get('[data-testid="cell-model_restriction-1"]').text()).toContain(
      'admin.accounts.modelWhitelist: claude-sonnet-4'
    )
    expect(wrapper.get('[data-testid="cell-model_restriction-2"]').text()).toContain(
      'admin.accounts.modelMapping: sonnet -> glm-4.5-air'
    )
    expect(wrapper.get('[data-testid="cell-model_restriction-3"]').text()).toContain(
      'admin.accounts.supportsAllModels'
    )
  })

  it('支持在列设置中隐藏并重新显示 model_restriction 列', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-testid="column-model_restriction"]').exists()).toBe(true)

    await wrapper.get('button[title="admin.users.columnSettings"]').trigger('click')
    const toggle = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.columns.modelRestriction'))
    expect(toggle).toBeTruthy()

    await toggle!.trigger('click')
    expect(wrapper.find('[data-testid="column-model_restriction"]').exists()).toBe(false)

    let toggleAgain = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.columns.modelRestriction'))
    if (!toggleAgain) {
      await wrapper.get('button[title="admin.users.columnSettings"]').trigger('click')
      toggleAgain = wrapper
        .findAll('button')
        .find((button) => button.text().includes('admin.accounts.columns.modelRestriction'))
    }
    expect(toggleAgain).toBeTruthy()

    await toggleAgain!.trigger('click')
    expect(wrapper.find('[data-testid="column-model_restriction"]').exists()).toBe(true)
  })
})
