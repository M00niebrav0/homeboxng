<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import PluginCard from "~/components/Plugin/PluginCard.vue";
  import type { PluginInfo } from "~/lib/api/classes/plugins";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Plugins",
  });

  const api = useUserApi();

  // --- Plugin Registry (all built-in plugins with metadata) ---

  interface PluginMeta {
    name: string;
    slug: string;
    icon: string;
    description: string;
    category: "Data" | "Analytics" | "Integration" | "Utility" | "System";
    version: string;
    builtIn: boolean;
    permissionCount: number;
  }

  const pluginRegistry: PluginMeta[] = [
    {
      name: "AI Vision",
      slug: "ai-vision",
      icon: "\uD83D\uDC41\uFE0F",
      description: "AI-powered item identification from photos using vision models. Scan items, match to inventory, and auto-tag.",
      category: "Data",
      version: "1.0.0",
      builtIn: true,
      permissionCount: 3,
    },
    {
      name: "Analytics",
      slug: "analytics",
      icon: "\uD83D\uDCCA",
      description: "Inventory analytics dashboard with value tracking, location breakdowns, activity feed, and warranty alerts.",
      category: "Analytics",
      version: "1.0.0",
      builtIn: true,
      permissionCount: 1,
    },
    {
      name: "Plugin Catalog",
      slug: "catalog",
      icon: "\uD83D\uDED2",
      description: "Browse and install community plugins from the HomeBoxNG plugin catalog. HACS-like plugin marketplace.",
      category: "System",
      version: "1.0.0",
      builtIn: true,
      permissionCount: 2,
    },
    {
      name: "Excel Export",
      slug: "excel-export",
      icon: "\uD83D\uDCC4",
      description: "Export inventory data to CSV/TSV. Import items from spreadsheets with field mapping and conflict resolution.",
      category: "Data",
      version: "1.0.0",
      builtIn: true,
      permissionCount: 2,
    },
    {
      name: "Eye-Fi Upload",
      slug: "eyefi",
      icon: "\uD83D\uDCF7",
      description: "Receive photos from Eye-Fi cards and Wi-Fi cameras. Auto-scan uploaded images with AI Vision.",
      category: "Integration",
      version: "1.0.0",
      builtIn: true,
      permissionCount: 2,
    },
    {
      name: "HA Bridge",
      slug: "ha-bridge",
      icon: "\uD83C\uDFE0",
      description: "Home Assistant integration for entity mapping, automations, and real-time sync between HA and HomeBox.",
      category: "Integration",
      version: "1.0.0",
      builtIn: true,
      permissionCount: 4,
    },
    {
      name: "Label Printer",
      slug: "label-printer",
      icon: "\uD83C\uDFF7\uFE0F",
      description: "Print QR code labels for inventory items. Supports multiple label sizes, orientations, and half-labels.",
      category: "Utility",
      version: "1.0.0",
      builtIn: true,
      permissionCount: 1,
    },
    {
      name: "Lending Tracker",
      slug: "lending",
      icon: "\uD83E\uDD1D",
      description: "Track borrowed and lent items. Checkout, return, due dates, and overdue notifications.",
      category: "Utility",
      version: "1.0.0",
      builtIn: true,
      permissionCount: 2,
    },
    {
      name: "Maintenance Scheduler",
      slug: "maintenance-scheduler",
      icon: "\uD83D\uDD27",
      description: "Schedule recurring maintenance tasks, track service providers, and log repair history with cost tracking.",
      category: "Utility",
      version: "1.0.0",
      builtIn: true,
      permissionCount: 2,
    },
    {
      name: "Notifications",
      slug: "notifications",
      icon: "\uD83D\uDD14",
      description: "Multi-platform notifications: Email, Discord, Web Push, Gotify, and ntfy. Event-based alert routing.",
      category: "System",
      version: "1.0.0",
      builtIn: true,
      permissionCount: 3,
    },
    {
      name: "Paperless-ngx",
      slug: "paperless",
      icon: "\uD83D\uDCC1",
      description: "Connect to Paperless-ngx for document-item linking. Auto-sync, rule-based matching, and manual linking.",
      category: "Integration",
      version: "1.0.0",
      builtIn: true,
      permissionCount: 3,
    },
    {
      name: "Shopping List",
      slug: "shopping",
      icon: "\uD83D\uDED2",
      description: "Manage shopping lists with auto-reorder rules. Track quantities, preferred stores, and purchase status.",
      category: "Utility",
      version: "1.0.0",
      builtIn: true,
      permissionCount: 1,
    },
  ];

  // --- API Data ---

  const { data: livePlugins, refresh: refreshPlugins } = useAsyncData("plugins", async () => {
    const { data } = await api.plugins.getAll();
    return data;
  });

  // --- Search & Filter State ---

  const searchQuery = ref("");
  const activeCategory = ref("All");
  const statusFilter = ref("All");
  const viewMode = ref<"grid" | "list">("grid");

  const categories = ["All", "Data", "Analytics", "Integration", "Utility", "System"];
  const statusOptions = ["All", "Enabled", "Disabled"];

  // --- Computed: merge registry with live data ---

  interface MergedPlugin extends PluginMeta {
    status: "enabled" | "disabled" | "error";
    liveData?: PluginInfo;
  }

  const mergedPlugins = computed<MergedPlugin[]>(() => {
    return pluginRegistry.map(meta => {
      const live = livePlugins.value?.find(p => p.name === meta.slug);
      let pluginStatus: "enabled" | "disabled" | "error" = "disabled";
      if (live) {
        if (live.status === "error") pluginStatus = "error";
        else if (live.enabled !== false) pluginStatus = "enabled";
        else pluginStatus = "disabled";
      }
      return {
        ...meta,
        status: pluginStatus,
        liveData: live || undefined,
      };
    });
  });

  // --- Computed: filtered plugins ---

  const filteredPlugins = computed(() => {
    let result = mergedPlugins.value;

    // Search filter
    if (searchQuery.value.trim()) {
      const q = searchQuery.value.toLowerCase();
      result = result.filter(
        p => p.name.toLowerCase().includes(q) || p.description.toLowerCase().includes(q) || p.slug.toLowerCase().includes(q)
      );
    }

    // Category filter
    if (activeCategory.value !== "All") {
      result = result.filter(p => p.category === activeCategory.value);
    }

    // Status filter
    if (statusFilter.value === "Enabled") {
      result = result.filter(p => p.status === "enabled");
    } else if (statusFilter.value === "Disabled") {
      result = result.filter(p => p.status !== "enabled");
    }

    return result;
  });

  // --- Computed: grouped by category ---

  const groupedPlugins = computed(() => {
    const groups: Record<string, MergedPlugin[]> = {};
    for (const plugin of filteredPlugins.value) {
      if (!groups[plugin.category]) {
        groups[plugin.category] = [];
      }
      groups[plugin.category].push(plugin);
    }
    return groups;
  });

  // --- Stats ---

  const totalPlugins = computed(() => pluginRegistry.length);
  const enabledCount = computed(() => mergedPlugins.value.filter(p => p.status === "enabled").length);
  const disabledCount = computed(() => mergedPlugins.value.filter(p => p.status !== "enabled").length);
  const updatesAvailable = ref(0); // Placeholder for future catalog integration
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between flex-wrap gap-4">
        <div>
          <h1 class="text-2xl font-bold">Plugin Hub</h1>
          <p class="text-sm opacity-70">
            Manage, configure, and extend HomeBoxNG with built-in and third-party plugins.
          </p>
        </div>
        <div class="flex gap-2 flex-wrap">
          <button class="btn btn-sm btn-outline" @click="refreshPlugins">
            Refresh
          </button>
          <NuxtLink to="/plugins/catalog" class="btn btn-sm btn-outline">
            Browse Catalog
          </NuxtLink>
          <NuxtLink to="/settings" class="btn btn-sm btn-ghost">
            Plugin Settings
          </NuxtLink>
        </div>
      </div>

      <!-- Stats Row -->
      <div class="stats shadow w-full">
        <div class="stat">
          <div class="stat-title">Total Plugins</div>
          <div class="stat-value text-primary">{{ totalPlugins }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Enabled</div>
          <div class="stat-value text-success">{{ enabledCount }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Disabled</div>
          <div class="stat-value text-base-content/50">{{ disabledCount }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Updates</div>
          <div class="stat-value" :class="updatesAvailable > 0 ? 'text-warning' : 'text-base-content/30'">
            {{ updatesAvailable }}
          </div>
          <div v-if="updatesAvailable > 0" class="stat-desc text-warning">Available</div>
        </div>
      </div>

      <!-- Search & Filter Bar -->
      <div class="flex flex-col gap-3">
        <!-- Search Input -->
        <div class="flex gap-3 flex-wrap items-center">
          <div class="form-control flex-1 min-w-[200px]">
            <input
              v-model="searchQuery"
              type="text"
              placeholder="Search plugins by name or description..."
              class="input input-bordered input-sm w-full"
            />
          </div>

          <!-- Status Filter -->
          <select v-model="statusFilter" class="select select-bordered select-sm">
            <option v-for="opt in statusOptions" :key="opt" :value="opt">
              {{ opt === "All" ? "All Status" : opt }}
            </option>
          </select>

          <!-- View Toggle -->
          <div class="join">
            <button
              class="btn btn-sm join-item"
              :class="viewMode === 'grid' ? 'btn-active' : 'btn-ghost'"
              title="Grid view"
              @click="viewMode = 'grid'"
            >
              Grid
            </button>
            <button
              class="btn btn-sm join-item"
              :class="viewMode === 'list' ? 'btn-active' : 'btn-ghost'"
              title="List view"
              @click="viewMode = 'list'"
            >
              List
            </button>
          </div>
        </div>

        <!-- Category Tabs -->
        <div class="tabs tabs-boxed bg-base-200 w-fit">
          <button
            v-for="cat in categories"
            :key="cat"
            class="tab tab-sm"
            :class="{ 'tab-active': activeCategory === cat }"
            @click="activeCategory = cat"
          >
            {{ cat }}
          </button>
        </div>
      </div>

      <!-- Results Count -->
      <div class="flex items-center justify-between">
        <p class="text-sm opacity-60">
          Showing {{ filteredPlugins.length }} of {{ totalPlugins }} plugins
        </p>
      </div>

      <!-- Plugin Grid / List grouped by category -->
      <template v-for="(categoryPlugins, category) in groupedPlugins" :key="category">
        <section>
          <Subtitle>{{ category }}</Subtitle>

          <!-- Grid View -->
          <div
            v-if="viewMode === 'grid'"
            class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4"
          >
            <PluginCard
              v-for="plugin in categoryPlugins"
              :key="plugin.slug"
              :name="plugin.name"
              :slug="plugin.slug"
              :description="plugin.description"
              :icon="plugin.icon"
              :version="plugin.version"
              :status="plugin.status"
              :permission-count="plugin.permissionCount"
              :built-in="plugin.builtIn"
              :category="plugin.category"
            />
          </div>

          <!-- List View -->
          <div v-else class="flex flex-col gap-2">
            <div
              v-for="plugin in categoryPlugins"
              :key="plugin.slug"
              class="flex items-center gap-4 p-3 rounded-lg border border-base-200 hover:border-primary/30 hover:bg-base-200/50 cursor-pointer transition-all duration-150"
              @click="$router.push(`/plugins/${plugin.slug}`)"
            >
              <div class="text-xl w-8 text-center flex-shrink-0">{{ plugin.icon }}</div>
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <span class="font-semibold text-sm">{{ plugin.name }}</span>
                  <span
                    class="w-2 h-2 rounded-full flex-shrink-0"
                    :class="{
                      'bg-success': plugin.status === 'enabled',
                      'bg-base-300': plugin.status === 'disabled',
                      'bg-error': plugin.status === 'error',
                    }"
                  />
                  <span v-if="plugin.builtIn" class="badge badge-xs badge-outline">Built-in</span>
                </div>
                <p class="text-xs opacity-50 truncate">{{ plugin.description }}</p>
              </div>
              <div class="flex items-center gap-2 flex-shrink-0">
                <span class="text-xs opacity-40 font-mono">v{{ plugin.version }}</span>
                <span class="badge badge-xs badge-ghost">{{ plugin.category }}</span>
              </div>
            </div>
          </div>
        </section>
      </template>

      <!-- Empty State -->
      <div v-if="filteredPlugins.length === 0" class="text-center py-16 opacity-50">
        <p class="text-4xl mb-4">
          {{ searchQuery ? "\uD83D\uDD0D" : "\uD83E\uDDE9" }}
        </p>
        <p class="text-lg">
          {{ searchQuery ? "No plugins match your search" : "No plugins found" }}
        </p>
        <p class="text-sm mt-2">
          {{ searchQuery ? "Try a different search term or clear filters." : "Check that the backend is running and plugins are registered." }}
        </p>
        <button
          v-if="searchQuery || activeCategory !== 'All' || statusFilter !== 'All'"
          class="btn btn-sm btn-outline mt-4"
          @click="searchQuery = ''; activeCategory = 'All'; statusFilter = 'All'"
        >
          Clear Filters
        </button>
      </div>
    </BaseContainer>
  </div>
</template>
