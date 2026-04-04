import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, reactive } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import OpsDashboard from '../OpsDashboard.vue'

vi.hoisted(() => {
  Object.defineProperty(globalThis, 'localStorage', {
    value: {
      getItem: vi.fn(),
      setItem: vi.fn(),
      removeItem: vi.fn(),
    },
    configurable: true,
  })
})

const route = reactive({
  query: {
    tr: 'custom',
    start_time: '2026-04-04T01:00:00.000Z',
    end_time: '2026-04-04T03:00:00.000Z',
    open_request_details: '1',
    request_id: 'req-temp-ui-1',
  },
})
const mockReplace = vi.fn()

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => ({
    replace: (...args: any[]) => mockReplace(...args),
  }),
}))

vi.mock('@/stores', () => ({
  useAdminSettingsStore: () => ({
    opsMonitoringEnabled: true,
    opsQueryModeDefault: 'auto',
    fetch: vi.fn().mockResolvedValue(undefined),
  }),
  useAppStore: () => ({
    showError: vi.fn(),
  }),
}))

vi.mock('@/api/admin/ops', () => {
  const opsAPIMock = {
    getAdvancedSettings: vi.fn().mockResolvedValue({
      display_alert_events: true,
      display_openai_token_stats: false,
      auto_refresh_enabled: false,
      auto_refresh_interval_seconds: 30,
    }),
    getDashboardSnapshotV2: vi.fn().mockResolvedValue({
      overview: null,
      throughput_trend: null,
      error_trend: null,
    }),
    getLatencyHistogram: vi.fn().mockResolvedValue(null),
    getErrorDistribution: vi.fn().mockResolvedValue(null),
    getMetricThresholds: vi.fn().mockResolvedValue(null),
    getThroughputTrend: vi.fn().mockResolvedValue(null),
  }

  return {
    opsAPI: opsAPIMock,
    default: opsAPIMock,
  }
})

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const passthrough = defineComponent({ template: '<div><slot /></div>' })
const requestModalStub = defineComponent({
  props: ['modelValue', 'preset', 'timeRange', 'customStartTime', 'customEndTime'],
  template: '<div data-testid="request-modal">{{ JSON.stringify({ modelValue, preset, timeRange, customStartTime, customEndTime }) }}</div>',
})

describe('OpsDashboard deep link', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('会根据 query 自动打开请求详情并注入 request_id', async () => {
    const wrapper = mount(OpsDashboard, {
      global: {
        stubs: {
          AppLayout: passthrough,
          BaseDialog: passthrough,
          OpsDashboardHeader: passthrough,
          OpsDashboardSkeleton: passthrough,
          OpsConcurrencyCard: passthrough,
          OpsSwitchRateTrendChart: passthrough,
          OpsThroughputTrendChart: passthrough,
          OpsLatencyChart: passthrough,
          OpsErrorDistributionChart: passthrough,
          OpsErrorTrendChart: passthrough,
          OpsAlertEventsCard: passthrough,
          OpsOpenAITokenStatsCard: passthrough,
          OpsSystemLogTable: passthrough,
          OpsSettingsDialog: passthrough,
          OpsAlertRulesCard: passthrough,
          OpsErrorDetailsModal: passthrough,
          OpsErrorDetailModal: passthrough,
          OpsRequestDetailsModal: requestModalStub,
        },
      },
    })

    await flushPromises()

    expect(wrapper.get('[data-testid="request-modal"]').text()).toContain('"modelValue":true')
    expect(wrapper.get('[data-testid="request-modal"]').text()).toContain('"request_id":"req-temp-ui-1"')
    expect(wrapper.get('[data-testid="request-modal"]').text()).toContain('"timeRange":"custom"')
  })
})
