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

  import { Input } from "~/components/ui/input";
  import { Button } from "~/components/ui/button";

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
  };

  // Category display order and icons
  const categoryOrder = ["Inventory & Data", "Integrations", "Communication", "Automation", "Tools"];

  const categoryIcons: Record<string, Component> = {
    "Inventory & Data": MdiChartBar,
    Integrations: MdiHomeAutomation,
    Communication: MdiBell,
    Automation: MdiWrench,
    Tools: MdiPrinter,
  };

  // State
  const searchQuery = ref("");
  const activeFilter = ref<"all" | "active" | "inactive" | "built-in">("all");

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

  // Computed: filtered plugins
  const filteredPlugins = computed(() => {
    if (!plugins.value) return [];

    return plugins.value.filter(plugin => {
      const slug = plugin.info.name;
      const meta = pluginMeta[slug];

      // Search filter
      if (searchQuery.value) {
        const query = searchQuery.value.toLowerCase();
        const matchesName = slug.toLowerCase().includes(query);
        const matchesDesc = (meta?.description || plugin.info.description || "").toLowerCase().includes(query);
        const matchesCategory = (meta?.category || "").toLowerCase().includes(query);
        if (!matchesName && !matchesDesc && !matchesCategory) return false;
      }

      // Status filter
      if (activeFilter.value === "active" && plugin.state !== "running") return false;
      if (activeFilter.value === "inactive" && plugin.state === "running") return false;
      if (activeFilter.value === "built-in" && !plugin.info.builtIn) return false;

      return true;
    });
  });

  // Computed: plugins grouped by category
  const groupedPlugins = computed(() => {
    const groups: Record<string, PluginStatus[]> = {};

    for (const category of categoryOrder) {
      groups[category] = [];
    }
    groups["Other"] = [];

    for (const plugin of filteredPlugins.value) {
      const meta = pluginMeta[plugin.info.name];
      const category = meta?.category || "Other";
      if (!groups[category]) {
        groups[category] = [];
      }
      groups[category].push(plugin);
    }

    return groups;
  });

  // Computed: visible categories (non-empty)
  const visibleCategories = computed(() => {
    return [...categoryOrder, "Other"].filter(cat => (groupedPlugins.value[cat]?.length || 0) > 0);
  });

  // Stats
  const totalPlugins = computed(() => plugins.value?.length || 0);
  const activePlugins = computed(() => plugins.value?.filter(p => p.state === "running").length || 0);

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
    navigateTo(`/plugins/${slug}`);
  }

  const filterOptions = [
    { key: "all" as const, label: "All" },
    { key: "active" as const, label: "Active" },
    { key: "inactive" as const, label: "Inactive" },
    { key: "built-in" as const, label: "Built-in" },
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
            {{ totalPlugins }} plugins loaded
            <span v-if="activePlugins > 0" class="ml-2 inline-flex items-center gap-1 rounded-full bg-white/20 px-2 py-0.5 text-xs font-medium">
              {{ activePlugins }} active
            </span>
          </p>
        </div>
        <div class="hidden text-right text-xs text-indigo-200 sm:block">
          <p>HomeBoxNG Plugin System</p>
          <p>v1.0.0</p>
        </div>
      </div>
    </div>

    <!-- Search + Filters -->
    <div class="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center">
      <div class="relative flex-1">
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
      <div class="flex items-center gap-1.5">
        <MdiFilter class="size-4 text-muted-foreground" />
        <button
          v-for="option in filterOptions"
          :key="option.key"
          class="rounded-full px-3 py-1.5 text-xs font-medium transition-all duration-200"
          :class="
            activeFilter === option.key
              ? 'bg-primary text-primary-foreground shadow-sm'
              : 'bg-secondary text-secondary-foreground hover:bg-secondary/80'
          "
          @click="activeFilter = option.key"
        >
          {{ option.label }}
        </button>
      </div>
    </div>

    <!-- Empty State -->
    <div
      v-if="filteredPlugins.length === 0 && plugins !== null"
      class="flex flex-col items-center justify-center rounded-xl border-2 border-dashed border-muted py-16"
    >
      <MdiPuzzle class="mb-4 size-12 text-muted-foreground" />
      <p class="text-lg font-medium text-muted-foreground">No plugins found</p>
      <p class="mt-1 text-sm text-muted-foreground">
        <template v-if="searchQuery">
          No plugins match "{{ searchQuery }}"
        </template>
        <template v-else>
          No plugins match the current filter
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
        Clear filters
      </Button>
    </div>

    <!-- Loading State -->
    <div v-else-if="plugins === null" class="flex items-center justify-center py-16">
      <div class="flex flex-col items-center gap-3">
        <div class="size-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
        <p class="text-sm text-muted-foreground">Loading plugins...</p>
      </div>
    </div>

    <!-- Category Sections -->
    <div v-else class="space-y-8">
      <section v-for="category in visibleCategories" :key="category">
        <!-- Category Header -->
        <div class="mb-4 flex items-center gap-2 border-b border-border pb-2">
          <component
            :is="categoryIcons[category] || MdiPuzzle"
            class="size-5 text-primary"
          />
          <h2 class="text-lg font-semibold text-foreground">{{ category }}</h2>
          <span class="rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
            {{ groupedPlugins[category]?.length || 0 }}
          </span>
        </div>

        <!-- Plugin Cards Grid -->
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          <div
            v-for="plugin in groupedPlugins[category]"
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
            <p class="mb-3 line-clamp-2 text-xs leading-relaxed text-muted-foreground">
              {{ getPluginMeta(plugin.info.name).description || plugin.info.description }}
            </p>

            <!-- Card Footer: Version + Open Button -->
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2">
                <span
                  v-if="plugin.info.builtIn"
                  class="rounded bg-indigo-100 px-1.5 py-0.5 text-[10px] font-medium text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300"
                >
                  Built-in
                </span>
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
      </section>
    </div>
  </div>
</template>
