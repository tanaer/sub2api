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
  emits: ['update:modelValue'],
  template: `
    <div data-testid="request-modal">
      {{ JSON.stringify({ modelValue, preset, timeRange, customStartTime, customEndTime }) }}
      <button data-testid="close-request-modal" type="button" @click="$emit('update:modelValue', false)">close</button>
    </div>
  `,
})

describe('OpsDashboard deep link', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    route.query = {
      tr: 'custom',
      start_time: '2026-04-04T01:00:00.000Z',
      end_time: '2026-04-04T03:00:00.000Z',
      open_request_details: '1',
      request_id: 'req-temp-ui-1',
    }
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
    expect(wrapper.get('[data-testid="request-modal"]').text()).toContain('"customStartTime":"2026-04-04T01:00:00.000Z"')
    expect(wrapper.get('[data-testid="request-modal"]').text()).toContain('"customEndTime":"2026-04-04T03:00:00.000Z"')
  })

  it('关闭请求详情弹窗后会回收 deep link query', async () => {
    vi.useFakeTimers()
    route.query = {
      tr: 'custom',
      start_time: '2026-04-04T01:00:00.000Z',
      end_time: '2026-04-04T03:00:00.000Z',
      open_request_details: '1',
      request_id: 'req-temp-ui-1',
      open_alert_rules: '1',
      alert_rule_id: '23',
    }

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
    await wrapper.get('[data-testid="close-request-modal"]').trigger('click')
    await vi.advanceTimersByTimeAsync(300)

    expect(mockReplace).toHaveBeenLastCalledWith({
      query: {
        tr: 'custom',
        start_time: '2026-04-04T01:00:00.000Z',
        end_time: '2026-04-04T03:00:00.000Z',
        open_alert_rules: '1',
        alert_rule_id: '23',
      },
    })

    vi.useRealTimers()
  })

  it('route query 变化后会更新 request trace 的 request_id 和自定义时间范围', async () => {
    route.query = {}

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

    route.query = {
      tr: 'custom',
      start_time: '2026-04-04T05:00:00.000Z',
      end_time: '2026-04-04T06:30:00.000Z',
      open_request_details: '1',
      request_id: 'req-temp-ui-2',
    }
    await flushPromises()

    expect(wrapper.get('[data-testid="request-modal"]').text()).toContain('"modelValue":true')
    expect(wrapper.get('[data-testid="request-modal"]').text()).toContain('"request_id":"req-temp-ui-2"')
    expect(wrapper.get('[data-testid="request-modal"]').text()).toContain('"customStartTime":"2026-04-04T05:00:00.000Z"')
    expect(wrapper.get('[data-testid="request-modal"]').text()).toContain('"customEndTime":"2026-04-04T06:30:00.000Z"')
  })
})
