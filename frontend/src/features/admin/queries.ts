// Truy vấn dùng chung giữa các trang quản trị
import { useQuery } from '@tanstack/vue-query'
import { api, unwrap } from '@/lib/api/client'

export const adminKeys = {
  accounts: ['admin', 'accounts'] as const,
  readOnly: ['admin', 'read-only'] as const,
  settings: ['screening', 'settings'] as const,
  penalties: ['penalties', 'settings'] as const,
  exports: ['admin', 'exports'] as const,
}

export const useAccounts = () => useQuery({ queryKey: adminKeys.accounts, queryFn: () => unwrap(api.identity.GET('/accounts')) })
export const useReadOnly = () =>
  useQuery({ queryKey: adminKeys.readOnly, queryFn: () => unwrap(api.penalties.GET('/penalties/read-only')) })
export const useScreeningSettings = () =>
  useQuery({ queryKey: adminKeys.settings, queryFn: () => unwrap(api.screening.GET('/screening/settings')) })
export const usePenaltySettings = () =>
  useQuery({ queryKey: adminKeys.penalties, queryFn: () => unwrap(api.penalties.GET('/penalties/settings')) })
export const useExports = () => useQuery({ queryKey: adminKeys.exports, queryFn: () => unwrap(api.administration.GET('/admin/exports')) })
