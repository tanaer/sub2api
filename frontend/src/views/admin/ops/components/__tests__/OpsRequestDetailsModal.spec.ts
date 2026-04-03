import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import OpsRequestDetailsModal from '../OpsRequestDetailsModal.vue'

const mockListRequestDetails = vi.fn()
const mockShowError = vi.fn()
const mockShowWarning = vi.fn()
const mockCopyToClipboard = vi.fn()

vi.mock('@/api/admin/ops', () => ({
  opsAPI: {
    listRequestDetails: (...args: any[]) => mockListRequestDetails(...args),
  },
}))

vi.mock('@/stores', () => ({
  useAppStore: () => ({
    showError: mockShowError,
    showWarning: mockShowWarning,
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: (...args: any[]) => mockCopyToClipboard(...args),
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

const PaginationStub = defineComponent({
  name: 'Pagination',
  template: '<div class="pagination-stub" />',
})

describe('OpsRequestDetailsModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockListRequestDetails.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 10,
    })
  })

  it('SLA 预设会把 exclude_phases 透传到请求明细接口', async () => {
    const wrapper = mount(OpsRequestDetailsModal, {
      props: {
        modelValue: false,
        timeRange: '1h',
        preset: {
          title: 'SLA',
          kind: 'error',
          exclude_phases: ['auth'],
        } as any,
        platform: 'openai',
        groupId: 7,
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Pagination: PaginationStub,
        },
      },
    })

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(mockListRequestDetails).toHaveBeenCalledWith(expect.objectContaining({
      kind: 'error',
      platform: 'openai',
      group_id: 7,
      exclude_phases: ['auth'],
    }))
  })

  it('点击查看链路会透传 request_id', async () => {
    mockListRequestDetails.mockResolvedValue({
      items: [
        {
          kind: 'success',
          created_at: '2026-04-04T12:00:00Z',
          request_id: 'req-trace-1',
          platform: 'openai',
          model: 'gpt-4.1',
        },
      ],
      total: 1,
      page: 1,
      page_size: 10,
    })

    const wrapper = mount(OpsRequestDetailsModal, {
      props: {
        modelValue: false,
        timeRange: '1h',
        preset: {
          title: 'Requests',
        } as any,
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Pagination: PaginationStub,
        },
      },
    })

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    const traceButton = wrapper.findAll('button').find((node) => node.text() === 'admin.ops.requestDetails.viewTrace')
    expect(traceButton).toBeTruthy()

    await traceButton!.trigger('click')

    expect(wrapper.emitted('openRequestTrace')).toEqual([['req-trace-1']])
  })
})
