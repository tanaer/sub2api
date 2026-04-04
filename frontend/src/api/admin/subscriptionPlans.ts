import { apiClient } from '../client'

export interface SubscriptionPlan {
  id: number
  name: string
  description?: string
  group_id?: number
  billing_mode: 'per_request' | 'per_usd'
  request_quota: number
  daily_limit_usd: number
  weekly_limit_usd: number
  monthly_limit_usd: number
  validity_days: number
  status: 'active' | 'archived'
  group?: {
    id: number
    name: string
    platform: string
  }
}

export interface CreateSubscriptionPlanRequest {
  name: string
  description?: string
  group_id?: number
  billing_mode: 'per_request' | 'per_usd'
  request_quota?: number
  daily_limit_usd?: number
  weekly_limit_usd?: number
  monthly_limit_usd?: number
  validity_days: number
}

export interface UpdateSubscriptionPlanRequest {
  name?: string
  description?: string
  group_id?: number
  billing_mode?: 'per_request' | 'per_usd'
  request_quota?: number
  daily_limit_usd?: number
  weekly_limit_usd?: number
  monthly_limit_usd?: number
  validity_days?: number
  status?: 'active' | 'archived'
}

export const subscriptionPlansAPI = {
  async list(status?: string): Promise<SubscriptionPlan[]> {
    const { data } = await apiClient.get<SubscriptionPlan[]>('/admin/subscription-plans', {
      params: status ? { status } : undefined,
    })
    return data || []
  },

  async create(req: CreateSubscriptionPlanRequest): Promise<SubscriptionPlan> {
    const { data } = await apiClient.post<SubscriptionPlan>('/admin/subscription-plans', req)
    return data
  },

  async update(id: number, req: UpdateSubscriptionPlanRequest): Promise<SubscriptionPlan> {
    const { data } = await apiClient.put<SubscriptionPlan>(`/admin/subscription-plans/${id}`, req)
    return data
  },

  async remove(id: number): Promise<void> {
    await apiClient.delete(`/admin/subscription-plans/${id}`)
  },
}
