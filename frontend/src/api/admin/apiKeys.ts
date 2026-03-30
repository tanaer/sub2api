/**
 * Admin API Keys API endpoints
 * Handles API key management for administrators
 */

import { apiClient } from '../client'
import type { ApiKey } from '@/types'

export interface UpdateApiKeyGroupResult {
  api_key: ApiKey
  auto_granted_group_access: boolean
  granted_group_id?: number
  granted_group_name?: string
}

/**
 * Update an API key's group binding
 * @param id - API Key ID
 * @param groupId - Group ID (0 to unbind, positive to bind, null/undefined to skip)
 * @returns Updated API key with auto-grant info
 */
export async function updateApiKeyGroup(id: number, groupId: number | null): Promise<UpdateApiKeyGroupResult> {
  const { data } = await apiClient.put<UpdateApiKeyGroupResult>(`/admin/api-keys/${id}`, {
    group_id: groupId === null ? 0 : groupId
  })
  return data
}

export async function updateApiKeyRequestQuota(
  id: number,
  requestQuota: number | null,
  resetRequestQuotaUsed = false
): Promise<ApiKey> {
  const payload: Record<string, number | boolean> = {
    reset_request_quota_used: resetRequestQuotaUsed
  }
  if (requestQuota !== null) {
    payload.request_quota = requestQuota
  }
  const { data } = await apiClient.put<ApiKey>(`/admin/api-keys/${id}/request-quota`, payload)
  return data
}

export const apiKeysAPI = {
  updateApiKeyGroup,
  updateApiKeyRequestQuota
}

export default apiKeysAPI
