<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Analytics Dashboard",
  });

  const api = useUserApi();

  // Date range filter
  const dateRange = ref("30");
  const dateRangeOptions = [
    { label: "7 days", value: "7" },
    { label: "30 days", value: "30" },
    { label: "90 days", value: "90" },
    { label: "1 year", value: "365" },
    { label: "All time", value: "all" },
  ];

  // Analytics data interfaces
  interface AnalyticsStats {
    totalItems: number;
    totalValue: number;
    itemsAddedThisMonth: number;
    locationsCount: number;
  }

  interface LocationValue {
    name: string;
    value: number;
    itemCount: number;
  }

  interface CategoryBreakdown {
    name: string;
    count: number;
    color: string;
  }

  interface ActivityEntry {
    id: string;
    action: "added" | "moved" | "updated" | "deleted";
    itemName: string;
    details: string;
    timestamp: string;
  }

  interface WarrantyItem {
    id: string;
    name: string;
    warrantyExpires: string;
    daysRemaining: number;
    location: string;
  }

  // Fetch analytics data
  const { data: stats } = useAsyncData("analytics-stats", async () => {
    const { data } = await api.http.get<AnalyticsStats>({
      url: "/api/v1/plugins/analytics/stats",
    });
    return data;
  });

  const { data: locationValues } = useAsyncData("analytics-locations", async () => {
    const { data } = await api.http.get<LocationValue[]>({
      url: "/api/v1/plugins/analytics/value-by-location",
    });
    return data || [];
  });

  const { data: categoryBreakdown } = useAsyncData("analytics-categories", async () => {
    const { data } = await api.http.get<CategoryBreakdown[]>({
      url: "/api/v1/plugins/analytics/categories",
    });
    return data || [];
  });

  const { data: activityFeed, refresh: refreshActivity } = useAsyncData("analytics-activity", async () => {
    const { data } = await api.http.get<ActivityEntry[]>({
      url: `/api/v1/plugins/analytics/activity?days=${dateRange.value}`,
    });
    return data || [];
  });

  const { data: warrantyItems } = useAsyncData("analytics-warranties", async () => {
    const { data } = await api.http.get<WarrantyItem[]>({
      url: "/api/v1/plugins/analytics/warranties-expiring",
    });
    return data || [];
  });

  // Refresh activity when date range changes
  watch(dateRange, () => {
    refreshActivity();
  });

  // Computed helpers
  const maxLocationValue = computed(() => {
    if (!locationValues.value || locationValues.value.length === 0) return 1;
    return Math.max(...locationValues.value.map(l => l.value));
  });

  function locationBarWidth(value: number): string {
    const pct = maxLocationValue.value > 0 ? (value / maxLocationValue.value) * 100 : 0;
    return `${Math.max(pct, 2)}%`;
  }

  function formatCurrency(value: number): string {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency: "USD",
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  }

  function formatTimestamp(ts: string): string {
    const date = new Date(ts);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays < 7) return `${diffDays}d ago`;
    return date.toLocaleDateString();
  }

  function actionColor(action: string): string {
    const colors: Record<string, string> = {
      added: "badge-success",
      moved: "badge-info",
      updated: "badge-warning",
      deleted: "badge-error",
    };
    return colors[action] || "badge-ghost";
  }

  function warrantyUrgency(days: number): string {
    if (days <= 7) return "text-error font-bold";
    if (days <= 30) return "text-warning font-semibold";
    if (days <= 60) return "text-info";
    return "opacity-70";
  }

  const categoryColors = [
    "bg-primary", "bg-secondary", "bg-accent", "bg-info",
    "bg-success", "bg-warning", "bg-error", "bg-primary/60",
    "bg-secondary/60", "bg-accent/60", "bg-info/60", "bg-success/60",
  ];
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Analytics Dashboard</h1>
          <p class="text-sm opacity-70">
            Insights and trends across your inventory.
          </p>
        </div>
        <div class="flex gap-2 items-center">
          <select v-model="dateRange" class="select select-bordered select-sm">
            <option v-for="opt in dateRangeOptions" :key="opt.value" :value="opt.value">
              {{ opt.label }}
            </option>
          </select>
          <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
            Back to Plugins
          </NuxtLink>
        </div>
      </div>

      <!-- Stats Row -->
      <div class="stats shadow w-full">
        <div class="stat">
          <div class="stat-title">Total Items</div>
          <div class="stat-value text-primary">{{ stats?.totalItems ?? 0 }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Total Value</div>
          <div class="stat-value text-secondary">{{ formatCurrency(stats?.totalValue ?? 0) }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Added This Month</div>
          <div class="stat-value text-accent">{{ stats?.itemsAddedThisMonth ?? 0 }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Locations</div>
          <div class="stat-value">{{ stats?.locationsCount ?? 0 }}</div>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <!-- Value by Location -->
        <section>
          <Subtitle>Value by Location</Subtitle>
          <BaseCard>
            <div class="p-4">
              <div v-if="!locationValues || locationValues.length === 0" class="text-center py-8 opacity-50">
                <p>No location data available.</p>
              </div>
              <div v-else class="space-y-3">
                <div v-for="loc in locationValues" :key="loc.name" class="space-y-1">
                  <div class="flex justify-between text-sm">
                    <span class="font-medium truncate flex-1">{{ loc.name }}</span>
                    <span class="text-xs opacity-60 ml-2">{{ loc.itemCount }} items</span>
                    <span class="font-semibold ml-3 min-w-[80px] text-right">{{ formatCurrency(loc.value) }}</span>
                  </div>
                  <div class="w-full bg-base-200 rounded-full h-3">
                    <div
                      class="bg-primary h-3 rounded-full transition-all duration-500"
                      :style="{ width: locationBarWidth(loc.value) }"
                    />
                  </div>
                </div>
              </div>
            </div>
          </BaseCard>
        </section>

        <!-- Category Breakdown -->
        <section>
          <Subtitle>Category Breakdown</Subtitle>
          <div v-if="!categoryBreakdown || categoryBreakdown.length === 0" class="text-center py-8 opacity-50">
            <p>No category data available.</p>
          </div>
          <div v-else class="grid grid-cols-2 gap-3 sm:grid-cols-3">
            <BaseCard v-for="(cat, idx) in categoryBreakdown" :key="cat.name">
              <div class="p-3 text-center">
                <div
                  class="w-8 h-8 rounded-full mx-auto mb-2 flex items-center justify-center text-white text-xs font-bold"
                  :class="categoryColors[idx % categoryColors.length]"
                >
                  {{ cat.count }}
                </div>
                <p class="text-sm font-medium truncate">{{ cat.name }}</p>
                <p class="text-xs opacity-50">{{ cat.count }} items</p>
              </div>
            </BaseCard>
          </div>
        </section>
      </div>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <!-- Activity Feed -->
        <section>
          <Subtitle>Recent Activity</Subtitle>
          <BaseCard>
            <div class="p-4">
              <div v-if="!activityFeed || activityFeed.length === 0" class="text-center py-8 opacity-50">
                <p>No recent activity.</p>
              </div>
              <div v-else class="max-h-96 overflow-y-auto space-y-2">
                <div
                  v-for="entry in activityFeed"
                  :key="entry.id"
                  class="flex items-start gap-3 p-2 rounded-lg hover:bg-base-200 transition-colors"
                >
                  <span class="badge badge-xs mt-1.5" :class="actionColor(entry.action)">
                    {{ entry.action }}
                  </span>
                  <div class="flex-1 min-w-0">
                    <p class="text-sm">
                      <span class="font-medium">{{ entry.itemName }}</span>
                      <span class="opacity-60"> {{ entry.details }}</span>
                    </p>
                    <p class="text-xs opacity-40">{{ formatTimestamp(entry.timestamp) }}</p>
                  </div>
                </div>
              </div>
            </div>
          </BaseCard>
        </section>

        <!-- Warranty Expiring -->
        <section>
          <Subtitle>Warranties Expiring Soon</Subtitle>
          <BaseCard>
            <div class="p-4">
              <div v-if="!warrantyItems || warrantyItems.length === 0" class="text-center py-8 opacity-50">
                <p>No warranties expiring in the next 90 days.</p>
              </div>
              <div v-else class="space-y-2">
                <div
                  v-for="item in warrantyItems"
                  :key="item.id"
                  class="flex items-center justify-between p-3 rounded-lg bg-base-200/50"
                >
                  <div class="min-w-0 flex-1">
                    <p class="font-medium text-sm truncate">{{ item.name }}</p>
                    <p class="text-xs opacity-50">{{ item.location }}</p>
                  </div>
                  <div class="text-right ml-3">
                    <p class="text-sm" :class="warrantyUrgency(item.daysRemaining)">
                      {{ item.daysRemaining }} days
                    </p>
                    <p class="text-xs opacity-40">
                      {{ new Date(item.warrantyExpires).toLocaleDateString() }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </BaseCard>
        </section>
      </div>
    </BaseContainer>
  </div>
</template>
