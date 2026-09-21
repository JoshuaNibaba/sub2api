import { apiClient } from './client'
import type { PaginatedResponse, UserErrorRequest } from '@/types'

export interface EnterpriseAccountPoolItem {
  id: number
  name: string
  platform: string
  type: string
  status: string
  schedulable: boolean
  concurrency: number
  load_factor?: number | null
  group_ids?: number[]
  last_used_at?: string | null
  rate_limited: boolean
  temporarily_disabled: boolean
  health: 'healthy' | 'degraded' | 'unavailable' | string
}

export interface EnterpriseAccountPoolParams {
  page?: number
  page_size?: number
  platform?: string
  search?: string
}

export interface ScopedUsageLog {
  id: number
  account_id: number
  request_id: string
  model: string
  input_tokens: number
  output_tokens: number
  total_cost: number
  created_at: string
  account?: { id: number; name: string } | null
  account_owned_by_user: boolean
}

export const enterpriseAPI = {
  listAccountPool(params: EnterpriseAccountPoolParams = {}) {
    return apiClient.get<PaginatedResponse<EnterpriseAccountPoolItem>>('/enterprise/account-pool', { params })
      .then(({ data }) => data)
  },

  listUsageLogs(params: Record<string, unknown> = {}) {
    return apiClient.get<PaginatedResponse<ScopedUsageLog>>('/enterprise/usage-logs', { params })
      .then(({ data }) => data)
  },

  listErrorLogs(params: Record<string, unknown> = {}) {
    return apiClient.get<PaginatedResponse<UserErrorRequest>>('/enterprise/error-logs', { params })
      .then(({ data }) => data)
  },
}
