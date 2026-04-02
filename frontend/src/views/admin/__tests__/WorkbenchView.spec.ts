import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import WorkbenchView from '../WorkbenchView.vue'

const {
  lookupAPIKeys,
  getRedeemPresets,
  getRedeemTemplates,
  generateRedeemPreset,
  deleteApiKey,
  toggleUserStatus,
  getAllGroups,
  showSuccess,
  showError,
} = vi.hoisted(() => ({
  lookupAPIKeys: vi.fn(),
  getRedeemPresets: vi.fn(),
  getRedeemTemplates: vi.fn(),
  generateRedeemPreset: vi.fn(),
  deleteApiKey: vi.fn(),
  toggleUserStatus: vi.fn(),
  getAllGroups: vi.fn(),
  showSuccess: vi.fn(),
  showError: vi.fn(),
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    tools: {
      lookupAPIKeys,
      getRedeemPresets,
      getRedeemTemplates,
      generateRedeemPreset,
    },
    apiKeys: {
      delete: deleteApiKey,
    },
    users: {
      toggleStatus: toggleUserStatus,
    },
    groups: {
      getAll: getAllGroups,
    },
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showSuccess,
    showError,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('WorkbenchView', () => {
  beforeEach(() => {
    lookupAPIKeys.mockReset()
    getRedeemPresets.mockReset()
    getRedeemTemplates.mockReset()
    generateRedeemPreset.mockReset()
    deleteApiKey.mockReset()
    toggleUserStatus.mockReset()
    getAllGroups.mockReset()
    showSuccess.mockReset()
    showError.mockReset()

    getRedeemPresets.mockResolvedValue([
      {
        id: 'preset-balance',
        name: '50额度',
        enabled: true,
        sort_order: 1,
        type: 'balance',
        value: 50,
        validity_days: 30,
        template_id: 'template-basic',
      },
    ])
    getRedeemTemplates.mockResolvedValue([
      {
        id: 'template-basic',
        name: '默认话术',
        enabled: true,
        sort_order: 1,
        content: '密钥：{{code}}',
      },
    ])
    getAllGroups.mockResolvedValue([])

    Object.assign(navigator, {
      clipboard: {
        writeText: vi.fn().mockResolvedValue(undefined),
        readText: vi.fn().mockResolvedValue('密钥：sk-match-1234567890abcdef'),
      },
    })
  })

  it('supports paste lookup, shows last success time, deletes unused api keys, and toggles user status', async () => {
    lookupAPIKeys.mockResolvedValue({
      extracted_keys: ['sk-match-1234567890abcdef', 'sk-miss-1234567890abcdef'],
      matched_count: 1,
      unmatched_count: 1,
      items: [
        {
          extracted_key: 'sk-match-1234567890abcdef',
          matched: true,
          api_key: 'sk-match-1234567890abcdef',
          api_key_id: 11,
          user_id: 5,
          user_email: 'lookup@example.com',
          username: 'lookup-user',
          user_status: 'active',
          latest_redeem_at: '2026-03-25T08:00:00Z',
          last_success_at: '2026-04-01T12:30:00Z',
          success_call_count: 19,
          api_keys: [
            {
              id: 11,
              key: 'sk-match-1234567890abcdef',
              created_at: '2026-04-01T10:00:00Z',
              last_success_at: '2026-04-01T12:30:00Z',
              success_call_count: 12,
              status: 'active',
            },
            {
              id: 12,
              key: 'sk-second-1234567890abcdef',
              created_at: '2026-04-01T11:00:00Z',
              last_success_at: null,
              success_call_count: 0,
              status: 'disabled',
            },
          ],
        },
        {
          extracted_key: 'sk-miss-1234567890abcdef',
          matched: false,
          last_success_at: null,
          success_call_count: 0,
          api_keys: [],
        },
      ],
    })
    toggleUserStatus.mockResolvedValue({ status: 'disabled' })
    deleteApiKey.mockResolvedValue({ message: 'deleted' })

    const wrapper = mount(WorkbenchView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.get('[data-testid="lookup-input"]').attributes('placeholder')).toContain('API Key 或兑换码')
    expect(wrapper.get('[data-testid="lookup-submit"]').text()).toContain('粘贴识别')

    await wrapper.get('[data-testid="lookup-input"]').setValue('旧内容')
    await wrapper.get('[data-testid="lookup-submit"]').trigger('click')
    await flushPromises()

    expect(lookupAPIKeys).toHaveBeenCalledWith('密钥：sk-match-1234567890abcdef')
    expect(navigator.clipboard.readText).toHaveBeenCalled()
    expect((wrapper.get('[data-testid="lookup-input"]').element as HTMLTextAreaElement).value).toBe(
      '密钥：sk-match-1234567890abcdef',
    )
    expect(wrapper.text()).toContain('lookup@example.com')
    expect(wrapper.text()).toContain('最后成功调用时间')
    expect(wrapper.text()).toContain('成功调用次数')
    expect(wrapper.text()).toContain('19')
    expect(wrapper.text()).toContain('sk-second-1234567890abcdef')
    expect(wrapper.text()).toContain('未匹配')
    expect(wrapper.find('[data-testid="delete-user-api-key-11"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="delete-user-api-key-12"]').text()).toContain('删除')

    await wrapper.get('[data-testid="delete-user-api-key-12"]').trigger('click')
    await flushPromises()

    expect(deleteApiKey).toHaveBeenCalledWith(12)
    expect(wrapper.text()).not.toContain('sk-second-1234567890abcdef')

    await wrapper.get('[data-testid="copy-all-results"]').trigger('click')
    expect(navigator.clipboard.writeText).toHaveBeenCalled()

    await wrapper.get('[data-testid="toggle-user-status-5"]').trigger('click')
    await flushPromises()

    expect(toggleUserStatus).toHaveBeenCalledWith(5, 'disabled')
    expect(wrapper.text()).toContain('启用')
  })

  it('loads preset buttons and generates a copyable scripted message', async () => {
    generateRedeemPreset.mockResolvedValue({
      code: 'R-TEST-001',
      rendered_message: '密钥：R-TEST-001',
      preset: {
        id: 'preset-balance',
        name: '50额度',
        enabled: true,
        sort_order: 1,
        type: 'balance',
        value: 50,
        validity_days: 30,
        template_id: 'template-basic',
      },
    })

    const wrapper = mount(WorkbenchView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
        },
      },
    })

    await flushPromises()

    expect(wrapper.text()).toContain('50额度')

    await wrapper.get('[data-testid="preset-generate-preset-balance"]').trigger('click')
    await flushPromises()

    expect(generateRedeemPreset).toHaveBeenCalledWith('preset-balance')
    expect(wrapper.text()).toContain('R-TEST-001')
    expect(wrapper.text()).toContain('密钥：R-TEST-001')

    await wrapper.get('[data-testid="copy-generated-message"]').trigger('click')
    expect(navigator.clipboard.writeText).toHaveBeenLastCalledWith('密钥：R-TEST-001')
  })

  it('manages templates separately and lets presets choose one template', async () => {
    const wrapper = mount(WorkbenchView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Teleport: true,
        },
      },
    })

    await flushPromises()

    const managePresetButton = wrapper.findAll('button').find((button) => button.text() === '管理预设')
    expect(managePresetButton).toBeTruthy()
    await managePresetButton!.trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('选择话术模板')
    expect(wrapper.text()).toContain('默认话术')
    expect(wrapper.text()).toContain('管理模板')
  })
})
