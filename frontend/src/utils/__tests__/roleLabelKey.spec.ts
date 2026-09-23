import { describe, expect, it } from 'vitest'

import en from '@/i18n/locales/en'
import zh from '@/i18n/locales/zh'
import { roleLabelKey, type UserRole } from '../permissions'

const ROLES: UserRole[] = ['super_admin', 'admin', 'enterprise_user', 'user']

function resolve(messages: unknown, key: string): unknown {
  return key.split('.').reduce<unknown>((current, segment) => {
    if (current === null || typeof current !== 'object') return undefined
    return (current as Record<string, unknown>)[segment]
  }, messages)
}

describe('roleLabelKey', () => {
  it('maps every backend role to a translated label in both locales', () => {
    for (const role of ROLES) {
      const key = roleLabelKey(role)
      for (const [locale, messages] of Object.entries({ en, zh })) {
        const label = resolve(messages, key)
        expect(typeof label, `${locale} is missing ${key} for role ${role}`).toBe('string')
        expect(label).not.toBe('')
      }
    }
  })

  it('never returns a raw role value as the key suffix', () => {
    expect(roleLabelKey('enterprise_user')).toBe('admin.users.roles.enterpriseUser')
    expect(roleLabelKey('super_admin')).toBe('admin.users.roles.superAdmin')
  })

  it('falls back to the plain user label for unknown roles', () => {
    expect(roleLabelKey(undefined)).toBe('admin.users.roles.user')
    expect(roleLabelKey('something_new')).toBe('admin.users.roles.user')
  })
})
