import { apiClient } from './client'
import type { SubscriptionPlan } from '@/api/admin/subscriptionPlans'

export async function listAvailablePlans(): Promise<SubscriptionPlan[]> {
  const { data } = await apiClient.get<SubscriptionPlan[]>('/subscription-plans', {
    params: { status: 'active' }
  })
  return data || []
}
