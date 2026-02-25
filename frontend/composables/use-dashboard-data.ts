import type { GroupStatistics, ItemSummary, MaintenanceEntryWithDetails, TotalsByOrganizer, TreeItem } from "~/lib/api/types/data-contracts";
import type { PluginInfo, ActivityEntry, WarrantyAlert, ValueByLocation, SystemAlert } from "~/lib/api/classes/plugins";
import type { UserClient } from "~/lib/api/user";

export interface DashboardStats {
  totalItems: number;
  totalValue: number;
  totalLocations: number;
  totalTags: number;
}

export interface LowStockItem {
  id: string;
  name: string;
  quantity: number;
  threshold: number;
}

export function useDashboardData(api: UserClient) {
  // ---- Quick Stats ----
  const {
    data: statistics,
    pending: statsLoading,
    refresh: refreshStats,
  } = useAsyncData<GroupStatistics | null>(
    "dashboard-statistics",
    async () => {
      const { data } = await api.stats.group();
      return data;
    },
    { lazy: true, server: false }
  );

  const dashboardStats = computed<DashboardStats>(() => ({
    totalItems: statistics.value?.totalItems ?? 0,
    totalValue: statistics.value?.totalItemPrice ?? 0,
    totalLocations: statistics.value?.totalLocations ?? 0,
    totalTags: statistics.value?.totalTags ?? 0,
  }));

  // ---- Recent Items ----
  const {
    data: recentItems,
    pending: recentItemsLoading,
    refresh: refreshRecentItems,
  } = useAsyncData<ItemSummary[]>(
    "dashboard-recent-items",
    async () => {
      const { data } = await api.items.getAll({
        page: 1,
        pageSize: 10,
        orderBy: "createdAt",
      });
      return data?.items ?? [];
    },
    { lazy: true, server: false }
  );

  // ---- Maintenance ----
  const {
    data: maintenanceEntries,
    pending: maintenanceLoading,
    refresh: refreshMaintenance,
  } = useAsyncData<MaintenanceEntryWithDetails[]>(
    "dashboard-maintenance",
    async () => {
      const { data } = await api.maintenance.getAll({ status: "scheduled" as any });
      return data ?? [];
    },
    { lazy: true, server: false }
  );

  // ---- Plugins ----
  const {
    data: plugins,
    pending: pluginsLoading,
    refresh: refreshPlugins,
  } = useAsyncData<PluginInfo[]>(
    "dashboard-plugins",
    async () => {
      const { data } = await api.plugins.getAll();
      return data ?? [];
    },
    { lazy: true, server: false }
  );

  // ---- Location Stats ----
  const {
    data: locationStats,
    pending: locationStatsLoading,
    refresh: refreshLocationStats,
  } = useAsyncData<TotalsByOrganizer[]>(
    "dashboard-location-stats",
    async () => {
      const { data } = await api.stats.locations();
      return data ?? [];
    },
    { lazy: true, server: false }
  );

  // ---- Location Tree ----
  const {
    data: locationTree,
    pending: locationTreeLoading,
    refresh: refreshLocationTree,
  } = useAsyncData<TreeItem[]>(
    "dashboard-location-tree",
    async () => {
      const { data } = await api.locations.getTree({ withItems: true });
      return data ?? [];
    },
    { lazy: true, server: false }
  );

  // ---- Activity Feed (via plugins API) ----
  const {
    data: activityFeed,
    pending: activityLoading,
    refresh: refreshActivity,
  } = useAsyncData<ActivityEntry[]>(
    "dashboard-activity",
    async () => {
      try {
        const { data } = await api.plugins.getActivityFeed(20);
        return data ?? [];
      } catch {
        return [];
      }
    },
    { lazy: true, server: false }
  );

  // ---- Warranty Alerts (via plugins API) ----
  const {
    data: warrantyAlerts,
    pending: warrantyLoading,
    refresh: refreshWarranty,
  } = useAsyncData<WarrantyAlert[]>(
    "dashboard-warranty",
    async () => {
      try {
        const { data } = await api.plugins.getWarrantyAlerts(90);
        return data ?? [];
      } catch {
        return [];
      }
    },
    { lazy: true, server: false }
  );

  // ---- System Alerts ----
  const {
    data: systemAlerts,
    pending: systemAlertsLoading,
    refresh: refreshSystemAlerts,
  } = useAsyncData<SystemAlert[]>(
    "dashboard-system-alerts",
    async () => {
      try {
        const { data } = await api.plugins.getSystemAlerts();
        return data ?? [];
      } catch {
        return [];
      }
    },
    { lazy: true, server: false }
  );

  // ---- Value by Location (via plugins API) ----
  const {
    data: valueByLocation,
    pending: valueByLocationLoading,
    refresh: refreshValueByLocation,
  } = useAsyncData<ValueByLocation[]>(
    "dashboard-value-by-location",
    async () => {
      try {
        const { data } = await api.plugins.getValueByLocation();
        return data ?? [];
      } catch {
        return [];
      }
    },
    { lazy: true, server: false }
  );

  // ---- Refresh All ----
  async function refreshAll() {
    await Promise.allSettled([
      refreshStats(),
      refreshRecentItems(),
      refreshMaintenance(),
      refreshPlugins(),
      refreshLocationStats(),
      refreshLocationTree(),
      refreshActivity(),
      refreshWarranty(),
      refreshSystemAlerts(),
      refreshValueByLocation(),
    ]);
  }

  return {
    // Stats
    dashboardStats,
    statsLoading,
    refreshStats,

    // Recent Items
    recentItems: computed(() => recentItems.value ?? []),
    recentItemsLoading,
    refreshRecentItems,

    // Maintenance
    maintenanceEntries: computed(() => maintenanceEntries.value ?? []),
    maintenanceLoading,
    refreshMaintenance,

    // Plugins
    plugins: computed(() => plugins.value ?? []),
    pluginsLoading,
    refreshPlugins,

    // Location Stats
    locationStats: computed(() => locationStats.value ?? []),
    locationStatsLoading,
    refreshLocationStats,

    // Location Tree
    locationTree: computed(() => locationTree.value ?? []),
    locationTreeLoading,
    refreshLocationTree,

    // Activity Feed
    activityFeed: computed(() => activityFeed.value ?? []),
    activityLoading,
    refreshActivity,

    // Warranty Alerts
    warrantyAlerts: computed(() => warrantyAlerts.value ?? []),
    warrantyLoading,
    refreshWarranty,

    // System Alerts
    systemAlerts: computed(() => systemAlerts.value ?? []),
    systemAlertsLoading,
    refreshSystemAlerts,

    // Value by Location
    valueByLocation: computed(() => valueByLocation.value ?? []),
    valueByLocationLoading,
    refreshValueByLocation,

    // Refresh All
    refreshAll,
  };
}
