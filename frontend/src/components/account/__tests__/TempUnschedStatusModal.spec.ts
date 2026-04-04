import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import TempUnschedStatusModal from '../TempUnschedStatusModal.vue'

const mockGetTempUnschedulableStatus = vi.fn()
const mockCopyToClipboard = vi.fn()
const mockPush = vi.fn()

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getTempUnschedulableStatus: (...args: any[]) => mockGetTempUnschedulableStatus(...args),
      recoverState: vi.fn(),
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn(),
  }),
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({
    copyToClipboard: (...args: any[]) => mockCopyToClipboard(...args),
  }),
}))

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: (...args: any[]) => mockPush(...args),
  }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key }),
  }
})

vi.mock('@/utils/format', () => ({
  formatDateTime: () => '2026-04-04 00:00:00',
}))

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: { show: Boolean, title: String },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>',
})

describe('TempUnschedStatusModal', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    const nowUnix = Math.floor(Date.now() / 1000)
    mockGetTempUnschedulableStatus.mockResolvedValue({
      active: true,
      state: {
        until_unix: nowUnix + 3600,
        triggered_at_unix: nowUnix - 60,
        status_code: 404,
        matched_keyword: 'Model Not Found',
        rule_index: 0,
        error_message: '{"error":{"message":"Model Not Found"}}',
        request_id: 'req-temp-ui-1',
        upstream_status_code: 404,
        upstream_error_message: 'Model Not Found',
        upstream_error_detail: '{"error":{"message":"Model Not Found"}}',
      },
    })
    mockCopyToClipboard.mockResolvedValue(true)
  })

  it('会渲染 request trace 并跳转到 Ops 请求详情', async () => {
    const wrapper = mount(TempUnschedStatusModal, {
      props: {
        show: false,
        account: { id: 7, name: 'xunfei-7', platform: 'anthropic' } as any,
      },
      global: {
        stubs: { BaseDialog: BaseDialogStub },
      },
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    expect(wrapper.text()).toContain('req-temp-ui-1')
    expect(wrapper.text()).toContain('Model Not Found')

    await wrapper.get('[data-testid="temp-unsched-open-request"]').trigger('click')

    expect(mockPush).toHaveBeenCalledWith(expect.objectContaining({
      path: '/admin/ops',
      query: expect.objectContaining({
        open_request_details: '1',
        request_id: 'req-temp-ui-1',
        tr: 'custom',
        start_time: expect.any(String),
        end_time: expect.any(String),
      }),
    }))
  })

  it('点击复制按钮会调用 copyToClipboard', async () => {
    const wrapper = mount(TempUnschedStatusModal, {
      props: {
        show: false,
        account: { id: 7, name: 'xunfei-7', platform: 'anthropic' } as any,
      },
      global: {
        stubs: { BaseDialog: BaseDialogStub },
      },
    })

    await wrapper.setProps({ show: true })
    await flushPromises()

    await wrapper.get('[data-testid="temp-unsched-copy-request"]').trigger('click')

    expect(mockCopyToClipboard).toHaveBeenCalledWith(
      'req-temp-ui-1',
      'admin.accounts.tempUnschedulable.requestIdCopied'
    )
  })
})
