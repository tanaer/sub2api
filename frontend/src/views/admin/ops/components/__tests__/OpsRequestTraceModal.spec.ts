import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import OpsRequestTraceModal from '../OpsRequestTraceModal.vue'

const mockGetRequestTrace = vi.fn()
const mockCopyToClipboard = vi.fn()
const mockShowError = vi.fn()
const mockShowWarning = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    getRequestTrace: (...args: any[]) => mockGetRequestTrace(...args),
  },
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: (...args: any[]) => mockCopyToClipboard(...args),
  }),
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: mockShowError,
    showWarning: mockShowWarning,
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: { type: Boolean, default: false },
    title: { type: String, default: '' },
  },
  template: '<div v-if="show"><slot /></div>',
})

describe('OpsRequestTraceModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockGetRequestTrace.mockResolvedValue({
      identity: {
        query_key: 'req-1',
        query_key_type: 'auto',
        matched_by: 'client_request_id',
        client_request_id: 'client-1',
        local_request_id: 'local-1',
        usage_request_id: 'usage-1',
        upstream_request_ids: ['upstream-1'],
        upstream_request_ids_truncated: false,
        upstream_request_ids_total: 1,
      },
      models: {
        original_requested_model: 'sonnet',
        group_resolved_model: 'glm-4.5-air',
        account_support_lookup_model: 'glm-4.5-air',
        final_upstream_model: 'glm-4.5-air',
      },
      request: {
        created_at: '2026-04-04T12:00:00Z',
        finished_at: '2026-04-04T12:00:02Z',
        duration_ms: 2000,
        status: 'success',
        platform: 'openai',
        request_path: '/v1/messages',
        inbound_endpoint: '/v1/messages',
        upstream_endpoint: '/chat/completions',
        stream: false,
      },
      timeline: [
        {
          ts: '2026-04-04T12:00:00Z',
          type: 'group_model_resolved',
          phase: 'routing',
          summary: 'sonnet -> glm-4.5-air',
          data: {
            from_model: 'sonnet',
            to_model: 'glm-4.5-air',
          },
        },
      ],
      usage: {
        usage_request_id: 'usage-1',
      },
      result: {
        final_status: 'success',
        final_status_code: 200,
        final_account_id: 9,
        final_account_name: 'glm-account-9',
        final_upstream_model: 'glm-4.5-air',
      },
      trace_incomplete: false,
    })
  })

  it('打开后会按请求标识拉取并展示追踪数据', async () => {
    const wrapper = mount(OpsRequestTraceModal, {
      props: {
        modelValue: true,
        requestKey: 'req-1',
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
        },
      },
    })

    await flushPromises()

    expect(mockGetRequestTrace).toHaveBeenCalledWith('req-1', 'auto')
    expect(wrapper.text()).toContain('client-1')
    expect(wrapper.text()).toContain('glm-4.5-air')
    expect(wrapper.text()).toContain('group_model_resolved')
  })
})
