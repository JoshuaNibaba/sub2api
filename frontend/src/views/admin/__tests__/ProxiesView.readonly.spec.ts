import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'
import ProxiesView from '../ProxiesView.vue'

const { list, getAllWithCount } = vi.hoisted(() => ({ list: vi.fn(), getAllWithCount: vi.fn() }))
vi.mock('@/api/admin', () => ({ adminAPI: { proxies: { list, getAllWithCount } } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }) }))
// A restricted administrator can read proxies but lacks admin.proxies.write.
vi.mock('@/stores/auth', () => ({ useAuthStore: () => ({ hasPermission: () => false }) }))
vi.mock('vue-i18n', async () => ({
  ...await vi.importActual<typeof import('vue-i18n')>('vue-i18n'),
  useI18n: () => ({ t: (key: string) => key }),
}))

const mountView = () => shallowMount(ProxiesView, {
  global: { stubs: {
    AppLayout: { template: '<div><slot /></div>' },
    TablePageLayout: { template: '<div><slot name="filters" /><slot name="table" /></div>' },
    DataTable: { props: ['data'], template: '<div v-for="row in data" :key="row.id"><slot name="cell-address" :row="row" /><slot name="cell-actions" :row="row" /></div>' },
    BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
  } },
})

let wrapper: ReturnType<typeof mountView>
afterEach(() => wrapper?.unmount())

describe('proxy list for viewers without proxy write access', () => {
  it('shows the masked endpoint but no copy or management actions', async () => {
    list.mockResolvedValue({ items: [{ id: 9, name: 'proxy', protocol: 'http', host: 'prox****mple', port: 8080, username: 'ol****er', status: 'active' }], total: 1, pages: 1 })
    getAllWithCount.mockResolvedValue([])
    wrapper = mountView(); await flushPromises()

    expect(wrapper.text()).toContain('prox****mple:8080')
    const labels = wrapper.findAll('button').map(button => button.text())
    expect(labels).not.toContain('common.edit')
    expect(labels).not.toContain('common.delete')
    expect(labels).not.toContain('admin.proxies.createProxy')
    expect(labels).not.toContain('admin.proxies.dataExport')
    expect(wrapper.find('[title="admin.proxies.copyProxyUrl"]').exists()).toBe(false)
  })
})
