<template>
  <AppLayout>
    <div class="space-y-6">
      <div>
        <h1 class="page-title">{{ t('enterprise.title') }}</h1>
        <p class="page-description">{{ t('enterprise.description') }}</p>
      </div>

      <div class="card p-4">
        <div class="flex flex-wrap items-center gap-3">
          <div class="flex-1 sm:max-w-xs">
            <label class="input-label">{{ t('enterprise.platform') }}</label>
            <select v-model="platform" class="input w-full" @change="loadPool(1)">
              <option value="">{{ t('common.all') }}</option>
              <option v-for="item in platforms" :key="item" :value="item">{{ item }}</option>
            </select>
          </div>
          <button class="btn btn-secondary self-end" :disabled="loading" @click="refresh">
            {{ t('common.refresh') }}
          </button>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-200 px-4 py-3 dark:border-dark-700">
          <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('enterprise.accountPool') }}</h2>
          <p class="mt-1 text-xs text-gray-500">{{ t('enterprise.accountPoolHint') }}</p>
        </div>
        <div v-if="loading" class="p-8 text-center text-sm text-gray-500">{{ t('common.loading') }}</div>
        <div v-else-if="pool.items.length === 0" class="p-8 text-center text-sm text-gray-500">{{ t('common.noData') }}</div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th v-for="header in headers" :key="header" class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{{ header }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="account in pool.items" :key="account.id">
                <td class="whitespace-nowrap px-4 py-3 text-sm font-medium text-gray-900 dark:text-white">{{ account.name }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-600 dark:text-gray-300">{{ account.platform }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-600 dark:text-gray-300">{{ account.type }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-sm">
                  <span class="rounded-full px-2 py-1 text-xs font-medium" :class="healthClass(account.health)">{{ healthLabel(account.health) }}</span>
                </td>
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-600 dark:text-gray-300">{{ account.concurrency }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-500">{{ formatTime(account.last_used_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="flex items-center justify-between border-t border-gray-200 px-4 py-3 text-sm dark:border-dark-700">
          <span class="text-gray-500">{{ t('enterprise.total', { count: pool.total }) }}</span>
          <div class="flex gap-2">
            <button class="btn btn-secondary px-3 py-1" :disabled="page <= 1 || loading" @click="loadPool(page - 1)">{{ t('common.back') }}</button>
            <button class="btn btn-secondary px-3 py-1" :disabled="page >= pool.pages || loading" @click="loadPool(page + 1)">{{ t('common.next') }}</button>
          </div>
        </div>
      </div>

      <div class="grid gap-6 lg:grid-cols-2">
        <div v-if="canViewUsage" class="card p-5">
          <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('enterprise.usageLogs') }}</h2>
          <p class="mt-1 text-sm text-gray-500">{{ t('enterprise.usageLogsHint') }}</p>
          <div class="mt-4 flex items-center justify-between text-sm text-gray-600 dark:text-gray-300">
            <span>{{ t('enterprise.recentRecords', { count: usageCount }) }}</span>
            <router-link to="/usage" class="btn btn-secondary px-3 py-1">{{ t('common.view') }}</router-link>
          </div>
        </div>
        <div v-if="canViewLogs" class="card p-5">
          <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('enterprise.errorLogs') }}</h2>
          <p class="mt-1 text-sm text-gray-500">{{ t('enterprise.errorLogsHint') }}</p>
          <div class="mt-4 flex items-center justify-between text-sm text-gray-600 dark:text-gray-300">
            <span>{{ t('enterprise.recentRecords', { count: errorCount }) }}</span>
            <router-link to="/usage?tab=errors" class="btn btn-secondary px-3 py-1">{{ t('common.view') }}</router-link>
          </div>
        </div>
      </div>

      <div v-if="canViewUsage" class="card overflow-hidden">
        <div class="border-b border-gray-200 px-4 py-3 dark:border-dark-700">
          <h2 class="font-semibold text-gray-900 dark:text-white">{{ t('enterprise.requestRouting') }}</h2>
          <p class="mt-1 text-xs text-gray-500">{{ t('enterprise.requestRoutingHint') }}</p>
        </div>
        <div v-if="recentUsage.length === 0" class="p-8 text-center text-sm text-gray-500">{{ t('common.noData') }}</div>
        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800">
              <tr>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{{ t('enterprise.requestTime') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{{ t('enterprise.model') }}</th>
                <th class="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{{ t('enterprise.hitAccount') }}</th>
                <th class="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">{{ t('enterprise.cost') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900">
              <tr v-for="item in recentUsage" :key="item.id">
                <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-500">{{ formatTime(item.created_at) }}</td>
                <td class="max-w-[240px] truncate px-4 py-3 text-sm text-gray-700 dark:text-gray-300">{{ item.model }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-sm font-medium text-gray-900 dark:text-white">{{ item.account?.name || '****' }}</td>
                <td class="whitespace-nowrap px-4 py-3 text-right text-sm tabular-nums text-gray-600 dark:text-gray-300">${{ item.total_cost.toFixed(6) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { enterpriseAPI, type EnterpriseAccountPoolItem, type ScopedUsageLog } from '@/api'
import { useAuthStore } from '@/stores/auth'
import { Permission } from '@/utils/permissions'

const { t, locale } = useI18n()
const authStore = useAuthStore()
const loading = ref(false)
const platform = ref('')
const page = ref(1)
const pool = ref({ items: [] as EnterpriseAccountPoolItem[], total: 0, page: 1, page_size: 20, pages: 1 })
const usageCount = ref(0)
const errorCount = ref(0)
const recentUsage = ref<ScopedUsageLog[]>([])
const canViewUsage = computed(() => authStore.hasPermission(Permission.EnterpriseUsage))
const canViewLogs = computed(() => authStore.hasPermission(Permission.EnterpriseLogs))
const platforms = ['openai', 'anthropic', 'gemini', 'antigravity', 'grok']
const headers = computed(() => [t('enterprise.name'), t('enterprise.platform'), t('enterprise.type'), t('enterprise.health'), t('enterprise.capacity'), t('enterprise.lastUsed')])

async function loadPool(targetPage = page.value) {
  loading.value = true
  try {
    pool.value = await enterpriseAPI.listAccountPool({ page: targetPage, page_size: 20, platform: platform.value || undefined })
    page.value = pool.value.page
  } finally {
    loading.value = false
  }
}

async function refresh() {
  await loadPool(1)
  if (canViewUsage.value) {
    const result = await enterpriseAPI.listUsageLogs({ page: 1, page_size: 10 })
    usageCount.value = result.total
    recentUsage.value = result.items
  }
  if (canViewLogs.value) {
    const result = await enterpriseAPI.listErrorLogs({ page: 1, page_size: 1 })
    errorCount.value = result.total
  }
}

function formatTime(value?: string | null) {
  if (!value) return t('common.time.never')
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value))
}

function healthLabel(value: string) {
  return t(`enterprise.healthValues.${value}`, value)
}

function healthClass(value: string) {
  if (value === 'healthy') return 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300'
  if (value === 'degraded') return 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300'
  return 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
}

onMounted(() => { refresh().catch(() => undefined) })
</script>
