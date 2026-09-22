import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'

import AccountsView from '../AccountsView.vue'

// Enterprise customers read the real account table on this page. The rows come
// from /enterprise/account-pool already redacted, so the assertions here are
// about the page never reaching for a staff-only endpoint and never offering a
// control the customer cannot use.
const {
  listAccounts,
  listWithEtag,
  getAllProxies,
  getAllGroups,
  listAccountPool,
  getAvailableGroups
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn(),
  listAccountPool: vi.fn(),
  getAvailableGroups: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getById: vi.fn(),
      getBatchTodayStats: vi.fn(),
      getUpstreamBillingProbeSettings: vi.fn(),
      delete: vi.fn(),
      batchClearError: vi.fn(),
      batchRefresh: vi.fn(),
      toggleSchedulable: vi.fn(),
      refreshCredentials: vi.fn()
    },
    proxies: { getAll: getAllProxies },
    groups: { getAll: getAllGroups }
  }
}))

vi.mock('@/api', () => ({
  enterpriseAPI: { listAccountPool },
  userGroupsAPI: { getAvailable: getAvailableGroups }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showWarning: vi.fn(), showSuccess: vi.fn(), showInfo: vi.fn() })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    token: 'test-token',
    isSimpleMode: false,
    isAdmin: false,
    isSuperAdmin: false,
    user: { id: 31 },
    hasPermission: (permission: string) => permission === 'enterprise.account_pool.read'
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return { ...actual, useI18n: () => ({ t: (key: string) => key }) }
})

const DataTableStub = defineComponent({
  props: { columns: { type: Array, default: () => [] }, data: { type: Array, default: () => [] } },
  template: `
    <div>
      <span data-test="columns">{{ columns.map(column => column.key).join(',') }}</span>
      <div v-for="row in data" :key="row.id" :data-account-name="row.name">
        <slot name="cell-groups" :row="row" />
        <slot name="cell-actions" :row="row" />
      </div>
    </div>
  `
})

const AccountGroupsCellStub = defineComponent({
  props: { groups: { type: Array, default: () => [] } },
  template: '<span data-test="account-groups">{{ groups.map(group => group.name).join(",") }}</span>'
})

const AccountTableFiltersStub = defineComponent({
  props: { groups: { type: Array, default: () => [] }, searchable: { type: Boolean, default: true }, privacyFilter: { type: Boolean, default: true } },
  template: `
    <div>
      <span data-test="filter-groups">{{ groups.map(group => group.name).join(',') }}</span>
      <span data-test="filter-searchable">{{ String(searchable) }}</span>
      <span data-test="filter-privacy">{{ String(privacyFilter) }}</span>
    </div>
  `
})

function mountView() {
  return mount(AccountsView, {
    attachTo: document.body,
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>' },
        DataTable: DataTableStub,
        AccountTableActions: { template: '<div><slot name="after" /></div>' },
        AccountTableFilters: AccountTableFiltersStub,
        AccountBulkActionsBar: true,
        Pagination: true,
        ConfirmDialog: true,
        AccountActionMenu: true,
        ImportDataModal: true,
        ReAuthAccountModal: true,
        AccountTestModal: true,
        AccountStatsModal: true,
        ScheduledTestsPanel: true,
        SyncFromCrsModal: true,
        TempUnschedStatusModal: true,
        ErrorPassthroughRulesModal: true,
        TLSFingerprintProfilesModal: true,
        CreateAccountModal: true,
        EditAccountModal: true,
        BulkEditAccountModal: true,
        PlatformTypeBadge: true,
        AccountCapacityCell: true,
        AccountStatusIndicator: true,
        AccountTodayStatsCell: true,
        AccountGroupsCell: AccountGroupsCellStub,
        AccountUsageCell: true,
        UpstreamBillingRateCell: true,
        HelpTooltip: true,
        Icon: true,
        Teleport: true
      }
    }
  })
}

// A redacted pool row as the backend sends it: masked name, no owner, no proxy,
// no credentials, no billing multiplier.
const maskedRow = {
  id: 42,
  name: 'sh****ai',
  platform: 'openai',
  type: 'oauth',
  status: 'active',
  schedulable: true,
  concurrency: 8,
  priority: 1,
  group_ids: [7],
  current_concurrency: 3
}

describe('admin AccountsView as an enterprise customer', () => {
  beforeEach(() => {
    localStorage.clear()
    listAccounts.mockReset()
    listWithEtag.mockReset()
    getAllProxies.mockReset()
    getAllGroups.mockReset()
    listAccountPool.mockReset().mockResolvedValue({ items: [maskedRow], total: 1, page: 1, page_size: 20, pages: 1 })
    getAvailableGroups.mockReset().mockResolvedValue([{ id: 7, name: 'codex', platform: 'openai' }])
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('reads the pool through the enterprise endpoint and never the admin ones', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(listAccountPool).toHaveBeenCalledTimes(1)
    expect(listAccounts).not.toHaveBeenCalled()
    expect(listWithEtag).not.toHaveBeenCalled()
    expect(getAllProxies).not.toHaveBeenCalled()
    expect(getAllGroups).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('does not send a name search, because pool names arrive masked', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(listAccountPool.mock.calls[0][0]).not.toHaveProperty('search')
    expect(wrapper.get('[data-test="filter-searchable"]').text()).toBe('false')
    expect(wrapper.get('[data-test="filter-privacy"]').text()).toBe('false')
    wrapper.unmount()
  })

  it('renders the masked row as sent, with group names from the customer’s own groups', async () => {
    const wrapper = mountView()
    await flushPromises()

    expect(wrapper.find('[data-account-name="sh****ai"]').exists()).toBe(true)
    expect(wrapper.get('[data-test="account-groups"]').text()).toBe('codex')
    expect(wrapper.get('[data-test="filter-groups"]').text()).toBe('codex')
    wrapper.unmount()
  })

  it('drops the columns a read-only customer can never fill', async () => {
    const wrapper = mountView()
    await flushPromises()

    const columns = wrapper.get('[data-test="columns"]').text().split(',')
    for (const hidden of ['select', 'actions', 'proxy', 'notes', 'rate_multiplier', 'scheduler_score', 'upstream_billing_rate', 'today_stats']) {
      expect(columns).not.toContain(hidden)
    }
    // The operational state the pool view exists for stays on screen.
    for (const kept of ['name', 'platform_type', 'capacity', 'status', 'schedulable', 'groups', 'usage']) {
      expect(columns).toContain(kept)
    }
    wrapper.unmount()
  })
})
