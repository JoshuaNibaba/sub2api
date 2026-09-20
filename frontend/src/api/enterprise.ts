import { apiClient } from './client'
import type { PaginatedResponse, UsageLog, UserErrorRequest } from '@/types'

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

export const enterpriseAPI = {
  listAccountPool(params: EnterpriseAccountPoolParams = {}) {
    return apiClient.get<PaginatedResponse<EnterpriseAccountPoolItem>>('/enterprise/account-pool', { params })
      .then(({ data }) => data)
  },

  listUsageLogs(params: Record<string, unknown> = {}) {
    return apiClient.get<PaginatedResponse<UsageLog>>('/enterprise/usage-logs', { params })
      .then(({ data }) => data)
  },

  listErrorLogs(params: Record<string, unknown> = {}) {
    return apiClient.get<PaginatedResponse<UserErrorRequest>>('/enterprise/error-logs', { params })
      .then(({ data }) => data)
  },
}

