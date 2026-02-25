<script lang="ts" setup>
  import MdiMagnify from "~icons/mdi/magnify";
  import MdiChartBar from "~icons/mdi/chart-bar";
  import MdiFileExcel from "~icons/mdi/file-excel";
  import MdiBookOpenVariant from "~icons/mdi/book-open-variant";
  import MdiFileDocument from "~icons/mdi/file-document";
  import MdiHomeAutomation from "~icons/mdi/home-automation";
  import MdiCamera from "~icons/mdi/camera";
  import MdiBell from "~icons/mdi/bell";
  import MdiHandshake from "~icons/mdi/handshake";
  import MdiWrench from "~icons/mdi/wrench";
  import MdiCart from "~icons/mdi/cart";
  import MdiEye from "~icons/mdi/eye";
  import MdiPrinter from "~icons/mdi/printer";
  import MdiPuzzle from "~icons/mdi/puzzle";
  import MdiPackageVariantClosed from "~icons/mdi/package-variant-closed";
  import MdiFilter from "~icons/mdi/filter";
  import MdiClose from "~icons/mdi/close";
  import MdiClock from "~icons/mdi/clock";
  import MdiCheck from "~icons/mdi/check";
  import MdiServerNetwork from "~icons/mdi/server-network";
  import MdiBatteryCharging from "~icons/mdi/battery-charging";
  import MdiBookOpenPageVariant from "~icons/mdi/book-open-page-variant";
  import MdiUpdate from "~icons/mdi/update";

  import { Input } from "~/components/ui/input";
  import { Button } from "~/components/ui/button";
  import { Badge } from "~/components/ui/badge";

  useHead({
    title: "HomeBoxNG | Plugins",
  });

  definePageMeta({
    middleware: ["auth"],
    layout: "default",
  });

  // Plugin metadata: maps slug to category, icon, and description
  const pluginMeta: Record<
    string,
    {
      category: string;
      icon: Component;
      description: string;
    }
  > = {
    analytics: {
      category: "Inventory & Data",
      icon: MdiChartBar,
      description: "Track inventory trends, value reports, and usage statistics",
    },
    "excel-export": {
      category: "Inventory & Data",
      icon: MdiFileExcel,
      description: "Export your inventory data to Excel spreadsheets",
    },
    catalog: {
      category: "Inventory & Data",
      icon: MdiBookOpenVariant,
      description: "Browse and search a structured product catalog",
    },
    paperless: {
      category: "Integrations",
      icon: MdiFileDocument,
      description: "Sync documents and receipts with Paperless-ngx",
    },
    "ha-bridge": {
      category: "Integrations",
      icon: MdiHomeAutomation,
      description: "Connect to Home Assistant for smart home automation",
    },
    eyefi: {
      category: "Integrations",
      icon: MdiCamera,
      description: "Auto-import photos from Eye-Fi enabled cameras",
    },
    notifications: {
      category: "Communication",
      icon: MdiBell,
      description: "Get alerts for warranty expiry, low stock, and events",
    },
    lending: {
      category: "Communication",
      icon: MdiHandshake,
      description: "Track items lent out and manage return reminders",
    },
    "maintenance-scheduler": {
      category: "Automation",
      icon: MdiWrench,
      description: "Schedule recurring maintenance tasks for your items",
    },
    shopping: {
      category: "Automation",
      icon: MdiCart,
      description: "Build shopping lists from inventory needs and reorders",
    },
    "ai-vision": {
      category: "Automation",
      icon: MdiEye,
      description: "Identify and catalog items using AI image recognition",
    },
    "label-printer": {
      category: "Tools",
      icon: MdiPrinter,
      description: "Print labels and QR codes for your inventory items",
    },
    "it-assets": {
      category: "IT & Homelab",
      icon: MdiServerNetwork,
      description: "Track servers, desktops, laptops, and network equipment with hardware details",
    },
    batteries: {
      category: "Tools",
      icon: MdiBatteryCharging,
      description: "Power tool battery, charger, and modular storage system tracking",
    },
    manuals: {
      category: "Inventory & Data",
      icon: MdiBookOpenPageVariant,
      description: "Find, link, and manage product manuals from ManualsLib and other sources",
    },
    updater: {
      category: "System",
      icon: MdiUpdate,
      description: "Manage automatic updates, backups, and version control for HomeBoxNG",
    },
  };

  // State
  const searchQuery = ref("");
  type FilterTab = "all" | "installed" | "recently-viewed" | "active" | "inactive";
  const activeFilter = ref<FilterTab>("all");

  // Recently viewed — persisted in localStorage
  const RECENT_KEY = "homeboxng-recently-viewed-plugins";

  function getRecentlyViewed(): string[] {
    try {
      return JSON.parse(localStorage.getItem(RECENT_KEY) || "[]");
    } catch {
      return [];
    }
  }

  function trackPluginView(slug: string) {
    const recent = getRecentlyViewed().filter(s => s !== slug);
    recent.unshift(slug);
    localStorage.setItem(RECENT_KEY, JSON.stringify(recent.slice(0, 20)));
  }

  const recentlyViewed = ref<string[]>(getRecentlyViewed());

  // Fetch plugin data from API
  interface PluginInfo {
    name: string;
    version: string;
    description: string;
    author: string;
    builtIn: boolean;
  }

  interface PluginStatus {
    info: PluginInfo;
    state: string;
  }

  const api = useUserApi();
  const { data: plugins } = useAsyncData<PluginStatus[]>(async () => {
    try {
      const { data } = await api.http.get<PluginStatus[]>("/api/v1/plugins");
      return data || [];
    } catch {
      return [];
    }
  });

  // Computed: filtered plugins based on active tab
  const filteredPlugins = computed(() => {
    if (!plugins.value) return [];

    let result = plugins.value;

    // Tab filter
    switch (activeFilter.value) {
      case "installed":
        result = result.filter(p => p.info.builtIn || p.state === "running" || p.state === "stopped");
        break;
      case "recently-viewed":
        result = result.filter(p => recentlyViewed.value.includes(p.info.name));
        // Sort by recently viewed order
        result.sort((a, b) => {
          const ai = recentlyViewed.value.indexOf(a.info.name);
          const bi = recentlyViewed.value.indexOf(b.info.name);
          return ai - bi;
        });
        break;
      case "active":
        result = result.filter(p => p.state === "running");
        break;
      case "inactive":
        result = result.filter(p => p.state !== "running");
        break;
    }

    // Search filter
    if (searchQuery.value) {
      const query = searchQuery.value.toLowerCase();
      result = result.filter(plugin => {
        const slug = plugin.info.name;
        const meta = pluginMeta[slug];
        const matchesName = slug.toLowerCase().includes(query) ||
          getPluginDisplayName(slug).toLowerCase().includes(query);
        const matchesDesc = (meta?.description || plugin.info.description || "").toLowerCase().includes(query);
        const matchesCategory = (meta?.category || "").toLowerCase().includes(query);
        return matchesName || matchesDesc || matchesCategory;
      });
    }

    return result;
  });

  // Stats
  const totalPlugins = computed(() => plugins.value?.length || 0);
  const activePlugins = computed(() => plugins.value?.filter(p => p.state === "running").length || 0);
  const installedCount = computed(() => plugins.value?.filter(p => p.info.builtIn || p.state === "running" || p.state === "stopped").length || 0);
  const recentCount = computed(() => {
    if (!plugins.value) return 0;
    return plugins.value.filter(p => recentlyViewed.value.includes(p.info.name)).length;
  });
  const inactiveCount = computed(() => plugins.value?.filter(p => p.state !== "running").length || 0);

  // Helpers
  function getPluginMeta(slug: string) {
    return pluginMeta[slug] || { category: "Other", icon: MdiPuzzle, description: "" };
  }

  function isPluginActive(plugin: PluginStatus) {
    return plugin.state === "running";
  }

  function getPluginDisplayName(slug: string) {
    return slug
      .split("-")
      .map(word => word.charAt(0).toUpperCase() + word.slice(1))
      .join(" ");
  }

  function navigateToPlugin(slug: string) {
    trackPluginView(slug);
    recentlyViewed.value = getRecentlyViewed();
    navigateTo(`/plugins/${slug}`);
  }

  const filterTabs: { key: FilterTab; label: string; icon?: Component; count?: () => number }[] = [
    { key: "all", label: "All Plugins", count: () => totalPlugins.value },
    { key: "installed", label: "Installed", icon: MdiCheck, count: () => installedCount.value },
    { key: "recently-viewed", label: "Recently Viewed", icon: MdiClock, count: () => recentCount.value },
    { key: "active", label: "Active", count: () => activePlugins.value },
    { key: "inactive", label: "Inactive", count: () => inactiveCount.value },
  ];
</script>

<template>
  <div class="mx-auto max-w-7xl px-4 py-6 sm:px-6 lg:px-8">
    <!-- Header Banner -->
    <div class="mb-6 overflow-hidden rounded-xl bg-gradient-to-r from-indigo-500 via-purple-500 to-indigo-600 p-6 text-white shadow-lg">
      <div class="flex items-center justify-between">
        <div>
          <div class="flex items-center gap-3">
            <MdiPackageVariantClosed class="size-8" />
            <h1 class="text-2xl font-bold">Plugin Manager</h1>
          </div>
          <p class="mt-1 text-sm text-indigo-100">
            {{ totalPlugins }} plugins available
            <span v-if="activePlugins > 0" class="ml-2 inline-flex items-center gap-1 rounded-full bg-white/20 px-2 py-0.5 text-xs font-medium">
              {{ activePlugins }} active
            </span>
          </p>
        </div>
        <div class="hidden gap-2 sm:flex">
          <NuxtLink to="/plugins/catalog">
            <Button variant="secondary" size="sm" class="bg-white/20 text-white hover:bg-white/30">
              <MdiBookOpenVariant class="mr-1 size-4" />
              Browse Catalog
            </Button>
          </NuxtLink>
        </div>
      </div>
    </div>

    <!-- Tab Bar -->
    <div class="mb-4 flex items-center gap-1 overflow-x-auto border-b border-border pb-0">
      <button
        v-for="tab in filterTabs"
        :key="tab.key"
        class="flex shrink-0 items-center gap-1.5 border-b-2 px-4 py-2.5 text-sm font-medium transition-colors"
        :class="
          activeFilter === tab.key
            ? 'border-primary text-primary'
            : 'border-transparent text-muted-foreground hover:border-muted hover:text-foreground'
        "
        @click="activeFilter = tab.key"
      >
        <component :is="tab.icon" v-if="tab.icon" class="size-4" />
        {{ tab.label }}
        <Badge
          v-if="tab.count"
          :variant="activeFilter === tab.key ? 'default' : 'secondary'"
          class="ml-0.5 h-5 min-w-[20px] px-1.5 text-[10px]"
        >
          {{ tab.count() }}
        </Badge>
      </button>
    </div>

    <!-- Search -->
    <div class="mb-6">
      <div class="relative">
        <MdiMagnify class="absolute left-3 top-1/2 size-5 -translate-y-1/2 text-muted-foreground" />
        <Input
          v-model="searchQuery"
          class="h-10 pl-10 pr-10"
          placeholder="Search plugins by name, category, or description..."
          type="search"
        />
        <button
          v-if="searchQuery"
          class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
          @click="searchQuery = ''"
        >
          <MdiClose class="size-4" />
        </button>
      </div>
    </div>

    <!-- Empty State -->
    <div
      v-if="filteredPlugins.length === 0 && plugins !== null"
      class="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-muted py-16"
    >
      <MdiPuzzle class="mb-4 size-12 text-muted-foreground" />
      <p class="text-lg font-medium text-muted-foreground">
        <template v-if="activeFilter === 'recently-viewed'">
          No recently viewed plugins
        </template>
        <template v-else>
          No plugins found
        </template>
      </p>
      <p class="mt-1 text-sm text-muted-foreground">
        <template v-if="searchQuery">
          No plugins match "{{ searchQuery }}"
        </template>
        <template v-else-if="activeFilter === 'recently-viewed'">
          Visit a plugin page and it will appear here.
        </template>
        <template v-else>
          No plugins match the current filter.
        </template>
      </p>
      <Button
        v-if="searchQuery || activeFilter !== 'all'"
        variant="outline"
        size="sm"
        class="mt-4"
        @click="
          searchQuery = '';
          activeFilter = 'all';
        "
      >
        Show all plugins
      </Button>
    </div>

    <!-- Loading State -->
    <div v-else-if="plugins === null" class="flex items-center justify-center py-16">
      <div class="flex flex-col items-center gap-3">
        <div class="size-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        <p class="text-sm text-muted-foreground">Loading plugins...</p>
      </div>
    </div>

    <!-- Plugin Cards Grid — All plugins flat (no category grouping) -->
    <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      <div
        v-for="plugin in filteredPlugins"
        :key="plugin.info.name"
        class="group cursor-pointer rounded-xl border border-border bg-card p-4 shadow-sm transition-all duration-200 hover:border-primary/40 hover:shadow-md"
        @click="navigateToPlugin(plugin.info.name)"
      >
        <!-- Card Header: Icon + Status -->
        <div class="mb-3 flex items-start justify-between">
          <div class="flex size-11 items-center justify-center rounded-lg bg-primary/10 text-primary transition-colors group-hover:bg-primary/20">
            <component :is="getPluginMeta(plugin.info.name).icon" class="size-6" />
          </div>
          <div class="flex items-center gap-1.5">
            <span
              class="size-2.5 rounded-full"
              :class="isPluginActive(plugin) ? 'bg-green-500' : 'bg-gray-400'"
            />
            <span class="text-xs text-muted-foreground">
              {{ isPluginActive(plugin) ? "Active" : "Inactive" }}
            </span>
          </div>
        </div>

        <!-- Card Body: Name + Description -->
        <h3 class="mb-1 text-sm font-semibold text-foreground group-hover:text-primary">
          {{ getPluginDisplayName(plugin.info.name) }}
        </h3>
        <p class="mb-2 text-xs text-muted-foreground">
          {{ getPluginMeta(plugin.info.name).category || "Other" }}
        </p>
        <p class="mb-3 line-clamp-2 text-xs leading-relaxed text-muted-foreground">
          {{ getPluginMeta(plugin.info.name).description || plugin.info.description }}
        </p>

        <!-- Card Footer: Version + Badges -->
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-2">
            <Badge
              v-if="plugin.info.builtIn"
              variant="secondary"
              class="text-[10px]"
            >
              Built-in
            </Badge>
            <span class="text-[10px] text-muted-foreground">
              v{{ plugin.info.version }}
            </span>
          </div>
          <Button
            size="sm"
            variant="ghost"
            class="h-7 px-2.5 text-xs opacity-0 transition-opacity group-hover:opacity-100"
            @click.stop="navigateToPlugin(plugin.info.name)"
          >
            Open
          </Button>
        </div>
      </div>
    </div>
  </div>
</template>
