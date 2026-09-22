import { apiClient } from './client'
import type { PaginatedResponse, UsageLog, UsageQueryParams, UserErrorRequest } from '@/types'

export interface EnterpriseAccountPoolItem {
  id: number
  name: string
  owned_by_viewer: boolean
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
  type?: string
  status?: string
  group?: string
  sort_by?: string
  sort_order?: 'asc' | 'desc'
  search?: string
}

export interface ScopedUsageLog extends UsageLog {
  account?: { id: number; name: string } | null
  account_owned_by_user: boolean
}

export const enterpriseAPI = {
  listAccountPool(params: EnterpriseAccountPoolParams = {}) {
    return apiClient.get<PaginatedResponse<EnterpriseAccountPoolItem>>('/enterprise/account-pool', { params })
      .then(({ data }) => data)
  },

  listUsageLogs(params: UsageQueryParams & { sort_by?: string; sort_order?: 'asc' | 'desc' } = {}) {
    return apiClient.get<PaginatedResponse<ScopedUsageLog>>('/enterprise/usage-logs', { params })
      .then(({ data }) => data)
  },

  listErrorLogs(params: Record<string, unknown> = {}) {
    return apiClient.get<PaginatedResponse<UserErrorRequest>>('/enterprise/error-logs', { params })
      .then(({ data }) => data)
  },
}
