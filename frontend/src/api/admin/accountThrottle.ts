import { apiClient } from '../client'

export interface AccountThrottleRule {
  id: number
  name: string
  enabled: boolean
  priority: number
  keywords: string[]
  match_mode: 'contains' | 'exact'
  trigger_mode: 'immediate' | 'accumulated'
  accumulated_count: number
  accumulated_window: number
  action_type: 'duration' | 'scheduled_recovery'
  action_duration: number
  action_recover_hour: number
  platforms: string[]
  description: string | null
  created_at: string
  updated_at: string
}

export interface CreateThrottleRuleRequest {
  name: string
  enabled?: boolean
  priority?: number
  keywords?: string[]
  match_mode?: 'contains' | 'exact'
  trigger_mode?: 'immediate' | 'accumulated'
  accumulated_count?: number
  accumulated_window?: number
  action_type?: 'duration' | 'scheduled_recovery'
  action_duration?: number
  action_recover_hour?: number
  platforms?: string[]
  description?: string | null
}

export interface UpdateThrottleRuleRequest {
  name?: string
  enabled?: boolean
  priority?: number
  keywords?: string[]
  match_mode?: 'contains' | 'exact'
  trigger_mode?: 'immediate' | 'accumulated'
  accumulated_count?: number
  accumulated_window?: number
  action_type?: 'duration' | 'scheduled_recovery'
  action_duration?: number
  action_recover_hour?: number
  platforms?: string[]
  description?: string | null
}

export async function list(): Promise<AccountThrottleRule[]> {
  const { data } = await apiClient.get<AccountThrottleRule[]>('/admin/account-throttle-rules')
  return data
}

export async function getById(id: number): Promise<AccountThrottleRule> {
  const { data } = await apiClient.get<AccountThrottleRule>(`/admin/account-throttle-rules/${id}`)
  return data
}

export async function create(ruleData: CreateThrottleRuleRequest): Promise<AccountThrottleRule> {
  const { data } = await apiClient.post<AccountThrottleRule>('/admin/account-throttle-rules', ruleData)
  return data
}

export async function update(id: number, updates: UpdateThrottleRuleRequest): Promise<AccountThrottleRule> {
  const { data } = await apiClient.put<AccountThrottleRule>(`/admin/account-throttle-rules/${id}`, updates)
  return data
}

export async function deleteRule(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/account-throttle-rules/${id}`)
  return data
}

export async function toggleEnabled(id: number, enabled: boolean): Promise<AccountThrottleRule> {
  return update(id, { enabled })
}

export const accountThrottleAPI = {
  list,
  getById,
  create,
  update,
  delete: deleteRule,
  toggleEnabled
}

export default accountThrottleAPI
