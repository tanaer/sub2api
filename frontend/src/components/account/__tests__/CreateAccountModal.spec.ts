import { describe, expect, it, vi, beforeEach } from 'vitest'
import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'

const {
  createAccountMock,
  checkMixedChannelRiskMock,
  showErrorMock,
  showInfoMock,
  showSuccessMock,
  showWarningMock,
  fetchAntigravityDefaultMappingsMock,
  listTLSFingerprintProfilesMock
} = vi.hoisted(() => ({
  createAccountMock: vi.fn(),
  checkMixedChannelRiskMock: vi.fn(),
  showErrorMock: vi.fn(),
  showInfoMock: vi.fn(),
  showSuccessMock: vi.fn(),
  showWarningMock: vi.fn(),
  fetchAntigravityDefaultMappingsMock: vi.fn().mockResolvedValue([]),
  listTLSFingerprintProfilesMock: vi.fn().mockResolvedValue([])
}))

const buildOAuthClient = () => ({
  authUrl: ref(''),
  sessionId: ref(''),
  loading: ref(false),
  error: ref(''),
  state: ref(''),
  oauthState: ref(''),
  resetState: vi.fn(),
  generateAuthUrl: vi.fn(),
  buildCredentials: vi.fn((tokenInfo?: Record<string, unknown>) => ({ ...(tokenInfo || {}) })),
  buildExtraInfo: vi.fn((tokenInfo?: Record<string, unknown>) => ({ ...((tokenInfo?.extra as Record<string, unknown>) || {}) })),
  exchangeAuthCode: vi.fn(),
  validateRefreshToken: vi.fn(),
  validateSessionToken: vi.fn(),
  getCapabilities: vi.fn().mockResolvedValue({ ai_studio_oauth_enabled: true })
})

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: showErrorMock,
    showInfo: showInfoMock,
    showSuccess: showSuccessMock,
    showWarning: showWarningMock
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
      create: createAccountMock,
      checkMixedChannelRisk: checkMixedChannelRiskMock
    },
    tlsFingerprintProfiles: {
      list: listTLSFingerprintProfilesMock
    }
  }
}))

vi.mock('@/composables/useModelWhitelist', () => ({
  claudeModels: ['claude-sonnet-4'],
  getModelsByPlatform: (platform: string) => {
    if (platform === 'openai') return ['gpt-4.1']
    if (platform === 'gemini') return ['gemini-2.5-pro']
    return ['claude-sonnet-4']
  },
  getPresetMappingsByPlatform: vi.fn(() => []),
  commonErrorCodes: [],
  buildModelMappingObject: (
    mode: 'whitelist' | 'mapping',
    allowedModels: string[],
    mappings: Array<{ from: string; to: string }>
  ) => {
    if (mode === 'whitelist') {
      const filtered = allowedModels
        .map((model) => model.trim())
        .filter((model) => model.length > 0)
      if (filtered.length === 0) return undefined
      return Object.fromEntries(filtered.map((model) => [model, model]))
    }

    const filtered = mappings
      .map((mapping) => ({
        from: mapping.from.trim(),
        to: mapping.to.trim()
      }))
      .filter((mapping) => mapping.from.length > 0 && mapping.to.length > 0)
    if (filtered.length === 0) return undefined
    return Object.fromEntries(filtered.map((mapping) => [mapping.from, mapping.to]))
  },
  fetchAntigravityDefaultMappings: fetchAntigravityDefaultMappingsMock,
  isValidWildcardPattern: () => true
}))

vi.mock('@/composables/useAccountOAuth', () => ({
  useAccountOAuth: () => ({
    authUrl: ref(''),
    sessionId: ref(''),
    loading: ref(false),
    error: ref(''),
    resetState: vi.fn(),
    generateAuthUrl: vi.fn(),
    buildExtraInfo: vi.fn((tokenInfo?: Record<string, unknown>) => ({ ...((tokenInfo?.extra as Record<string, unknown>) || {}) })),
    parseSessionKeys: vi.fn(() => [])
  })
}))

vi.mock('@/composables/useOpenAIOAuth', () => ({
  useOpenAIOAuth: () => buildOAuthClient()
}))

vi.mock('@/composables/useGeminiOAuth', () => ({
  useGeminiOAuth: () => buildOAuthClient()
}))

vi.mock('@/composables/useAntigravityOAuth', () => ({
  useAntigravityOAuth: () => buildOAuthClient()
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

import CreateAccountModal from '../CreateAccountModal.vue'

const BaseDialogStub = defineComponent({
  name: 'BaseDialog',
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const ConfirmDialogStub = defineComponent({
  name: 'ConfirmDialog',
  template: '<div />'
})

const ModelWhitelistSelectorStub = defineComponent({
  name: 'ModelWhitelistSelector',
  template: '<div />'
})

const OAuthAuthorizationFlowStub = defineComponent({
  name: 'OAuthAuthorizationFlow',
  template: '<div />'
})

function buildApiKeyTemplateAccount() {
  return {
    id: 11,
    name: 'OpenAI API Key',
    notes: 'copied notes',
    platform: 'openai',
    type: 'apikey',
    credentials: {
      base_url: 'https://api.openai.com/v1',
      api_key: 'sk-copy',
      model_mapping: {
        'gpt-4.1': 'gpt-4.1'
      },
      pool_mode: true,
      pool_mode_retry_count: 4,
      custom_error_codes_enabled: true,
      custom_error_codes: [403]
    },
    extra: {
      quota_limit: 100,
      quota_daily_limit: 25,
      quota_weekly_limit: 70,
      quota_daily_reset_mode: 'fixed',
      quota_daily_reset_hour: 8,
      quota_weekly_reset_mode: 'fixed',
      quota_weekly_reset_day: 2,
      quota_weekly_reset_hour: 9,
      quota_reset_timezone: 'Asia/Shanghai'
    },
    proxy_id: 8,
    concurrency: 6,
    load_factor: 2,
    priority: 9,
    rate_multiplier: 1.5,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: 1719999999,
    auto_pause_on_expired: true,
    created_at: '2026-04-01T00:00:00Z',
    updated_at: '2026-04-01T00:00:00Z',
    group_ids: [3, 4],
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null
  } as any
}

function buildOAuthTemplateAccount() {
  return {
    id: 22,
    name: 'Claude OAuth',
    notes: 'oauth copy',
    platform: 'anthropic',
    type: 'setup-token',
    credentials: {
      access_token: 'at-copy',
      refresh_token: 'rt-copy',
      intercept_warmup_requests: true,
      temp_unschedulable_enabled: true,
      temp_unschedulable_rules: [
        {
          error_code: 429,
          keywords: ['rate limit'],
          duration_minutes: 10,
          description: '临时限流'
        }
      ]
    },
    extra: {},
    proxy_id: 5,
    concurrency: 3,
    load_factor: 1,
    priority: 7,
    rate_multiplier: 2,
    status: 'active',
    error_message: null,
    last_used_at: null,
    expires_at: null,
    auto_pause_on_expired: false,
    created_at: '2026-04-01T00:00:00Z',
    updated_at: '2026-04-01T00:00:00Z',
    group_ids: [7],
    schedulable: true,
    rate_limited_at: null,
    rate_limit_reset_at: null,
    overload_until: null,
    temp_unschedulable_until: null,
    temp_unschedulable_reason: null,
    session_window_start: null,
    session_window_end: null,
    session_window_status: null,
    window_cost_limit: 12,
    window_cost_sticky_reserve: 4,
    max_sessions: 2,
    session_idle_timeout_minutes: 15,
    base_rpm: 30,
    rpm_strategy: 'sticky_exempt',
    rpm_sticky_buffer: 6,
    user_msg_queue_mode: 'serialize',
    enable_tls_fingerprint: true,
    session_id_masking_enabled: true,
    cache_ttl_override_enabled: true,
    cache_ttl_override_target: '1h'
  } as any
}

function mountModal(templateAccount: any) {
  return mount(CreateAccountModal, {
    props: {
      show: true,
      templateAccount,
      proxies: [],
      groups: []
    },
    global: {
      stubs: {
        BaseDialog: BaseDialogStub,
        ConfirmDialog: ConfirmDialogStub,
        Select: true,
        Icon: true,
        ProxySelector: true,
        GroupSelector: true,
        ModelWhitelistSelector: ModelWhitelistSelectorStub,
        QuotaLimitCard: true,
        OAuthAuthorizationFlow: OAuthAuthorizationFlowStub
      }
    }
  })
}

describe('CreateAccountModal', () => {
  beforeEach(() => {
    createAccountMock.mockReset()
    checkMixedChannelRiskMock.mockReset()
    showErrorMock.mockReset()
    showInfoMock.mockReset()
    showSuccessMock.mockReset()
    showWarningMock.mockReset()
    fetchAntigravityDefaultMappingsMock.mockClear()
    listTLSFingerprintProfilesMock.mockClear()
    createAccountMock.mockResolvedValue({ id: 999 })
    checkMixedChannelRiskMock.mockResolvedValue({ has_risk: false })
  })

  it('prefills copied API key account data and submits the cloned payload directly', async () => {
    const templateAccount = buildApiKeyTemplateAccount()
    const wrapper = mountModal(templateAccount)

    await flushPromises()

    expect((wrapper.get('[data-tour="account-form-name"]').element as HTMLInputElement).value).toBe('OpenAI API Key')

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'OpenAI API Key',
        notes: 'copied notes',
        platform: 'openai',
        type: 'apikey',
        proxy_id: 8,
        concurrency: 6,
        load_factor: 2,
        priority: 9,
        rate_multiplier: 1.5,
        group_ids: [3, 4],
        expires_at: 1719999999,
        auto_pause_on_expired: true,
        credentials: expect.objectContaining({
          base_url: 'https://api.openai.com/v1',
          api_key: 'sk-copy',
          model_mapping: {
            'gpt-4.1': 'gpt-4.1'
          },
          pool_mode: true,
          pool_mode_retry_count: 4,
          custom_error_codes_enabled: true,
          custom_error_codes: [403]
        }),
        extra: expect.objectContaining({
          quota_limit: 100,
          quota_daily_limit: 25,
          quota_weekly_limit: 70,
          quota_daily_reset_mode: 'fixed',
          quota_daily_reset_hour: 8,
          quota_weekly_reset_mode: 'fixed',
          quota_weekly_reset_day: 2,
          quota_weekly_reset_hour: 9,
          quota_reset_timezone: 'Asia/Shanghai'
        })
      })
    )
  })

  it('duplicates copied OAuth accounts without forcing a new authorization step', async () => {
    const templateAccount = buildOAuthTemplateAccount()
    const wrapper = mountModal(templateAccount)

    await flushPromises()
    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(checkMixedChannelRiskMock).toHaveBeenCalledWith({
      platform: 'anthropic',
      group_ids: [7]
    })
    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock).toHaveBeenCalledWith(
      expect.objectContaining({
        name: 'Claude OAuth',
        platform: 'anthropic',
        type: 'setup-token',
        proxy_id: 5,
        group_ids: [7],
        credentials: expect.objectContaining({
          access_token: 'at-copy',
          refresh_token: 'rt-copy',
          intercept_warmup_requests: true,
          temp_unschedulable_enabled: true,
          temp_unschedulable_rules: [
            {
              error_code: 429,
              keywords: ['rate limit'],
              duration_minutes: 10,
              description: '临时限流'
            }
          ]
        }),
        extra: expect.objectContaining({
          window_cost_limit: 12,
          window_cost_sticky_reserve: 4,
          max_sessions: 2,
          session_idle_timeout_minutes: 15,
          base_rpm: 30,
          rpm_strategy: 'sticky_exempt',
          rpm_sticky_buffer: 6,
          user_msg_queue_mode: 'serialize',
          enable_tls_fingerprint: true,
          session_id_masking_enabled: true,
          cache_ttl_override_enabled: true,
          cache_ttl_override_target: '1h'
        })
      })
    )
  })

  it('在映射模式支持批量粘贴并回填提交 model_mapping', async () => {
    const templateAccount = buildApiKeyTemplateAccount()
    const wrapper = mountModal(templateAccount)

    await flushPromises()

    const mappingTab = wrapper
      .findAll('button')
      .find((button) => button.text().includes('admin.accounts.modelMapping'))
    expect(mappingTab).toBeTruthy()
    await mappingTab!.trigger('click')

    await wrapper.get('[data-testid="model-mapping-paste-toggle"]').trigger('click')
    await wrapper
      .get('[data-testid="model-mapping-paste-input"]')
      .setValue('gpt-5.2 -> gpt-5.2-2025-12-11\nclaude-3-5-sonnet,claude-sonnet-4')
    await wrapper.get('[data-testid="model-mapping-paste-apply"]').trigger('click')

    await wrapper.get('form#create-account-form').trigger('submit.prevent')
    await flushPromises()

    expect(createAccountMock).toHaveBeenCalledTimes(1)
    expect(createAccountMock.mock.calls[0]?.[0]?.credentials?.model_mapping).toEqual({
      'gpt-5.2': 'gpt-5.2-2025-12-11',
      'claude-3-5-sonnet': 'claude-sonnet-4'
    })
  })
})
