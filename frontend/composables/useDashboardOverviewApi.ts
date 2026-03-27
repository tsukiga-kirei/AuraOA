import type { DashboardOverview, PlatformDashboardOverview } from '~/types/dashboard-overview'

export const useDashboardOverviewApi = () => {
  const { authFetch } = useAuth()

  async function fetchDashboardOverview(): Promise<DashboardOverview> {
    return await authFetch<DashboardOverview>('/api/tenant/settings/dashboard-overview')
  }

  async function fetchPlatformDashboardOverview(): Promise<PlatformDashboardOverview> {
    return await authFetch<PlatformDashboardOverview>('/api/admin/dashboard-overview')
  }

  return { fetchDashboardOverview, fetchPlatformDashboardOverview }
}
