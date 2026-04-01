import { apiClient } from '../client'
import type {
  WorkbenchLookupResponse,
  WorkbenchRedeemPreset,
  WorkbenchRedeemPresetGenerateResponse,
  WorkbenchRedeemTemplate,
} from '@/types'

export async function lookupAPIKeys(rawText: string): Promise<WorkbenchLookupResponse> {
  const { data } = await apiClient.post<WorkbenchLookupResponse>('/admin/tools/api-key-lookup', {
    raw_text: rawText,
  })
  return data
}

export async function getRedeemPresets(): Promise<WorkbenchRedeemPreset[]> {
  const { data } = await apiClient.get<WorkbenchRedeemPreset[]>('/admin/tools/redeem-presets')
  return data
}

export async function updateRedeemPresets(
  presets: WorkbenchRedeemPreset[],
): Promise<WorkbenchRedeemPreset[]> {
  const { data } = await apiClient.put<WorkbenchRedeemPreset[]>('/admin/tools/redeem-presets', presets)
  return data
}

export async function getRedeemTemplates(): Promise<WorkbenchRedeemTemplate[]> {
  const { data } = await apiClient.get<WorkbenchRedeemTemplate[]>('/admin/tools/redeem-templates')
  return data
}

export async function updateRedeemTemplates(
  templates: WorkbenchRedeemTemplate[],
): Promise<WorkbenchRedeemTemplate[]> {
  const { data } = await apiClient.put<WorkbenchRedeemTemplate[]>(
    '/admin/tools/redeem-templates',
    templates,
  )
  return data
}

export async function generateRedeemPreset(
  id: string,
): Promise<WorkbenchRedeemPresetGenerateResponse> {
  const { data } = await apiClient.post<WorkbenchRedeemPresetGenerateResponse>(
    `/admin/tools/redeem-presets/${id}/generate`,
  )
  return data
}

export const toolsAPI = {
  lookupAPIKeys,
  getRedeemPresets,
  updateRedeemPresets,
  getRedeemTemplates,
  updateRedeemTemplates,
  generateRedeemPreset,
}

export default toolsAPI
