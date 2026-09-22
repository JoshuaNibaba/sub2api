import { apiClient } from './client'
import type { AccountListItem, PaginatedResponse, UsageLog, UsageQueryParams, UserErrorRequest } from '@/types'

/**
 * The account pool is the real account table, not a separate screen: the
 * endpoint answers with the same rows `GET /admin/accounts?lite=1` returns, so
 * the account page can render enterprise viewers without inventing values.
 *
 * Rows the viewer does not own arrive redacted by the backend — the name is
 * partially masked and proxy, credentials, notes, raw errors and the billing
 * multiplier are absent. Redacted rows are recognised by the missing
 * `owner_user_id`; the frontend must not try to reconstruct what was dropped.
 */
export type EnterpriseAccountPoolItem = AccountListItem & {
  current_concurrency?: number
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
