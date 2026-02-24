<script setup lang="ts">
  import { useWidgetLayout, WIDGET_DEFINITIONS, type WidgetType } from "~/composables/use-widget-layout";
  import { useDashboardData } from "~/composables/use-dashboard-data";
  import { fmtDate } from "~/composables/use-formatters";
  import BaseContainer from "@/components/Base/Container.vue";
  import { Badge } from "@/components/ui/badge";
  import { Button } from "@/components/ui/button";
  import { Skeleton } from "@/components/ui/skeleton";
  import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
  import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
  import MdiCog from "~icons/mdi/cog";
  import MdiClose from "~icons/mdi/close";
  import MdiPlus from "~icons/mdi/plus";
  import MdiRefresh from "~icons/mdi/refresh";
  import MdiRestore from "~icons/mdi/restore";
  import MdiChevronUp from "~icons/mdi/chevron-up";
  import MdiChevronDown from "~icons/mdi/chevron-down";
  import MdiPackageVariant from "~icons/mdi/package-variant";
  import MdiCurrencyUsd from "~icons/mdi/currency-usd";
  import MdiMapMarker from "~icons/mdi/map-marker";
  import MdiTag from "~icons/mdi/tag";
  import MdiClock from "~icons/mdi/clock";
  import MdiShieldCheck from "~icons/mdi/shield-check";
  import MdiWrench from "~icons/mdi/wrench";
  import MdiPuzzle from "~icons/mdi/puzzle";
  import MdiLightningBolt from "~icons/mdi/lightning-bolt";
  import MdiFormatListBulleted from "~icons/mdi/format-list-bulleted";
  import MdiChartBar from "~icons/mdi/chart-bar";
  import MdiFileTree from "~icons/mdi/file-tree";
  import MdiAlertCircle from "~icons/mdi/alert-circle";
  import MdiBarcodeScan from "~icons/mdi/barcode-scan";
  import MdiFileImport from "~icons/mdi/file-import";
  import MdiFileExport from "~icons/mdi/file-export";
  import MdiPrinter from "~icons/mdi/printer";
  import MdiBookOpenVariant from "~icons/mdi/book-open-variant";
  import MdiCircle from "~icons/mdi/circle";
  import MdiArrowRight from "~icons/mdi/arrow-right";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Dashboard",
  });

  const api = useUserApi();
  const router = useRouter();
  const formatCurrency = await useFormatCurrency();

  const {
    visibleWidgets,
    hiddenWidgets,
    isCustomizing,
    addWidget,
    removeWidget,
    moveWidget,
    resetLayout,
    saveLayout,
  } = useWidgetLayout();

  const {
    dashboardStats,
    statsLoading,
    recentItems,
    recentItemsLoading,
    maintenanceEntries,
    maintenanceLoading,
    plugins,
    pluginsLoading,
    locationStats,
    locationStatsLoading,
    locationTree,
    locationTreeLoading,
    activityFeed,
    activityLoading,
    warrantyAlerts,
    warrantyLoading,
    valueByLocation,
    valueByLocationLoading,
    refreshAll,
  } = useDashboardData(api);

  const showAddPanel = ref(false);
  const isRefreshing = ref(false);

  async function handleRefresh() {
    isRefreshing.value = true;
    await refreshAll();
    isRefreshing.value = false;
  }

  function toggleCustomize() {
    isCustomizing.value = !isCustomizing.value;
    if (!isCustomizing.value) {
      showAddPanel.value = false;
      saveLayout();
    }
  }

  function handleRemoveWidget(id: string) {
    removeWidget(id);
  }

  function handleAddWidget(type: WidgetType) {
    addWidget(type);
    if (hiddenWidgets.value.length === 0) {
      showAddPanel.value = false;
    }
  }

  function handleResetLayout() {
    resetLayout();
    showAddPanel.value = false;
  }

  function goTo(path: string) {
    router.push(path);
  }

  // ---- Warranty helpers ----
  function warrantyBadgeVariant(daysRemaining: number): "destructive" | "default" | "secondary" | "outline" {
    if (daysRemaining <= 7) return "destructive";
    if (daysRemaining <= 30) return "default";
    return "secondary";
  }

  // ---- Maintenance helpers ----
  function isOverdue(scheduledDate: Date | string): boolean {
    const d = typeof scheduledDate === "string" ? new Date(scheduledDate) : scheduledDate;
    return d < new Date();
  }

  // ---- Plugin status helpers ----
  function pluginStatusColor(plugin: { enabled?: boolean; status?: string }): string {
    if (!plugin.enabled) return "text-gray-400";
    if (plugin.status === "error") return "text-red-500";
    if (plugin.status === "warning") return "text-yellow-500";
    return "text-green-500";
  }

  // ---- Value by Location helpers ----
  function maxLocationValue(): number {
    const vals = valueByLocation.value;
    if (!vals || vals.length === 0) return 1;
    return Math.max(...vals.map(v => v.totalValue), 1);
  }

  function barWidthPercent(value: number): number {
    return Math.max(2, (value / maxLocationValue()) * 100);
  }

  // ---- Location tree helpers ----
  const expandedNodes = ref<Set<string>>(new Set());

  function toggleTreeNode(id: string) {
    if (expandedNodes.value.has(id)) {
      expandedNodes.value.delete(id);
    } else {
      expandedNodes.value.add(id);
    }
  }

  // ---- Activity feed helpers ----
  function activityBadgeVariant(action: string): "destructive" | "default" | "secondary" | "outline" {
    switch (action) {
      case "added": return "default";
      case "updated": return "secondary";
      case "moved": return "outline";
      case "deleted": return "destructive";
      default: return "secondary";
    }
  }

  // ---- Low stock (computed from items with quantity info) ----
  const lowStockItems = computed(() => {
    const items = recentItems.value ?? [];
    return items
      .filter(item => item.quantity !== undefined && item.quantity <= 1 && item.quantity >= 0)
      .slice(0, 10)
      .map(item => ({
        id: item.id,
        name: item.name,
        quantity: item.quantity,
        threshold: 1,
      }));
  });

  // ---- Drag and drop ----
  const draggedWidgetId = ref<string | null>(null);

  function onDragStart(event: DragEvent, widgetId: string) {
    draggedWidgetId.value = widgetId;
    if (event.dataTransfer) {
      event.dataTransfer.effectAllowed = "move";
      event.dataTransfer.setData("text/plain", widgetId);
    }
  }

  function onDragOver(event: DragEvent) {
    event.preventDefault();
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = "move";
    }
  }

  function onDrop(event: DragEvent, targetWidgetId: string) {
    event.preventDefault();
    const sourceId = draggedWidgetId.value;
    if (!sourceId || sourceId === targetWidgetId) return;

    const allWidgets = visibleWidgets.value;
    const sourceIdx = allWidgets.findIndex(w => w.id === sourceId);
    const targetIdx = allWidgets.findIndex(w => w.id === targetWidgetId);
    if (sourceIdx === -1 || targetIdx === -1) return;

    const source = allWidgets[sourceIdx];
    const target = allWidgets[targetIdx];
    if (!source || !target) return;

    const tmpRow = source.row;
    const tmpCol = source.col;
    source.row = target.row;
    source.col = target.col;
    target.row = tmpRow;
    target.col = tmpCol;

    saveLayout();
    draggedWidgetId.value = null;
  }

  function onDragEnd() {
    draggedWidgetId.value = null;
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-4">
      <!-- Header -->
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h1 class="text-2xl font-bold">Dashboard</h1>
        <div class="flex items-center gap-2">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger as-child>
                <Button
                  variant="outline"
                  size="sm"
                  :disabled="isRefreshing"
                  @click="handleRefresh"
                >
                  <MdiRefresh :class="{ 'animate-spin': isRefreshing }" class="size-4" />
                  <span class="hidden sm:inline">Refresh</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>Refresh all widget data</TooltipContent>
            </Tooltip>
          </TooltipProvider>

          <Button
            :variant="isCustomizing ? 'default' : 'outline'"
            size="sm"
            @click="toggleCustomize"
          >
            <MdiCog class="size-4" />
            <span class="hidden sm:inline">{{ isCustomizing ? "Done" : "Customize" }}</span>
          </Button>

          <Button
            v-if="isCustomizing"
            variant="outline"
            size="sm"
            @click="handleResetLayout"
          >
            <MdiRestore class="size-4" />
            <span class="hidden sm:inline">Reset Layout</span>
          </Button>
        </div>
      </div>

      <!-- Customize Mode: Add Widget Panel -->
      <div
        v-if="isCustomizing && hiddenWidgets.length > 0"
        class="rounded-lg border-2 border-dashed border-blue-300 bg-blue-50/50 p-4"
      >
        <div class="mb-3 flex items-center justify-between">
          <h3 class="text-sm font-semibold text-blue-700">Add Widgets</h3>
          <Button variant="ghost" size="sm" @click="showAddPanel = !showAddPanel">
            <MdiPlus class="size-4" />
            {{ showAddPanel ? "Hide" : "Show Available" }}
          </Button>
        </div>
        <div v-if="showAddPanel" class="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
          <button
            v-for="widget in hiddenWidgets"
            :key="widget.id"
            class="flex items-center gap-3 rounded-md border border-blue-200 bg-white p-3 text-left transition-colors hover:border-blue-400 hover:bg-blue-50"
            @click="handleAddWidget(widget.type)"
          >
            <MdiPlus class="size-5 shrink-0 text-blue-500" />
            <div>
              <p class="text-sm font-medium">{{ WIDGET_DEFINITIONS[widget.type].title }}</p>
              <p class="text-xs text-gray-500">{{ WIDGET_DEFINITIONS[widget.type].description }}</p>
            </div>
          </button>
        </div>
      </div>

      <!-- Widget Grid -->
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
        <div
          v-for="widget in visibleWidgets"
          :key="widget.id"
          :class="{
            'rounded-lg border-2 border-dashed border-blue-300': isCustomizing,
            'opacity-50': draggedWidgetId === widget.id,
          }"
          :draggable="isCustomizing"
          @dragstart="onDragStart($event, widget.id)"
          @dragover="onDragOver"
          @drop="onDrop($event, widget.id)"
          @dragend="onDragEnd"
        >
          <!-- Widget Customize Header -->
          <div
            v-if="isCustomizing"
            class="flex items-center justify-between rounded-t-lg bg-blue-100 px-3 py-1.5"
          >
            <div class="flex items-center gap-2">
              <span class="cursor-grab text-blue-500" title="Drag to reorder">
                <svg class="size-4" viewBox="0 0 24 24" fill="currentColor">
                  <circle cx="9" cy="5" r="1.5" /><circle cx="15" cy="5" r="1.5" />
                  <circle cx="9" cy="10" r="1.5" /><circle cx="15" cy="10" r="1.5" />
                  <circle cx="9" cy="15" r="1.5" /><circle cx="15" cy="15" r="1.5" />
                  <circle cx="9" cy="20" r="1.5" /><circle cx="15" cy="20" r="1.5" />
                </svg>
              </span>
              <span class="text-xs font-medium text-blue-700">{{ widget.title }}</span>
            </div>
            <div class="flex items-center gap-1">
              <button
                class="rounded p-0.5 text-blue-500 hover:bg-blue-200"
                title="Move up"
                @click="moveWidget(widget.id, 'up')"
              >
                <MdiChevronUp class="size-4" />
              </button>
              <button
                class="rounded p-0.5 text-blue-500 hover:bg-blue-200"
                title="Move down"
                @click="moveWidget(widget.id, 'down')"
              >
                <MdiChevronDown class="size-4" />
              </button>
              <button
                class="rounded p-0.5 text-red-500 hover:bg-red-100"
                title="Remove widget"
                @click="handleRemoveWidget(widget.id)"
              >
                <MdiClose class="size-4" />
              </button>
            </div>
          </div>

          <!-- Quick Stats Widget -->
          <Card v-if="widget.type === 'quick-stats'" class="overflow-hidden shadow">
            <CardHeader class="pb-2">
              <CardTitle class="flex items-center gap-2 text-base">
                <MdiChartBar class="size-5 text-primary" />
                Quick Stats
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div v-if="statsLoading" class="grid grid-cols-2 gap-3">
                <Skeleton v-for="i in 4" :key="i" class="h-20 rounded-lg" />
              </div>
              <div v-else class="grid grid-cols-2 gap-3">
                <div class="flex flex-col items-center rounded-lg bg-secondary p-3 text-center">
                  <MdiPackageVariant class="mb-1 size-6 text-primary" />
                  <span class="text-xl font-bold">{{ dashboardStats.totalItems }}</span>
                  <span class="text-xs text-muted-foreground">Items</span>
                </div>
                <div class="flex flex-col items-center rounded-lg bg-secondary p-3 text-center">
                  <MdiCurrencyUsd class="mb-1 size-6 text-green-600" />
                  <span class="text-xl font-bold">{{ formatCurrency(dashboardStats.totalValue) }}</span>
                  <span class="text-xs text-muted-foreground">Total Value</span>
                </div>
                <div class="flex flex-col items-center rounded-lg bg-secondary p-3 text-center">
                  <MdiMapMarker class="mb-1 size-6 text-blue-600" />
                  <span class="text-xl font-bold">{{ dashboardStats.totalLocations }}</span>
                  <span class="text-xs text-muted-foreground">Locations</span>
                </div>
                <div class="flex flex-col items-center rounded-lg bg-secondary p-3 text-center">
                  <MdiTag class="mb-1 size-6 text-purple-600" />
                  <span class="text-xl font-bold">{{ dashboardStats.totalTags }}</span>
                  <span class="text-xs text-muted-foreground">Tags</span>
                </div>
              </div>
            </CardContent>
          </Card>

          <!-- Recent Items Widget -->
          <Card v-else-if="widget.type === 'recent-items'" class="overflow-hidden shadow">
            <CardHeader class="pb-2">
              <CardTitle class="flex items-center gap-2 text-base">
                <MdiClock class="size-5 text-primary" />
                Recent Items
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div v-if="recentItemsLoading" class="space-y-2">
                <Skeleton v-for="i in 5" :key="i" class="h-10 rounded" />
              </div>
              <div v-else-if="recentItems.length === 0" class="py-4 text-center text-sm text-muted-foreground">
                No items found
              </div>
              <div v-else class="divide-y">
                <button
                  v-for="item in recentItems"
                  :key="item.id"
                  class="flex w-full items-center justify-between px-1 py-2 text-left transition-colors hover:bg-muted/50"
                  @click="goTo(`/item/${item.id}`)"
                >
                  <div class="min-w-0 flex-1">
                    <p class="truncate text-sm font-medium">{{ item.name }}</p>
                    <p class="truncate text-xs text-muted-foreground">
                      {{ item.location?.name || "No location" }}
                      <span class="mx-1">&middot;</span>
                      {{ fmtDate(item.createdAt, "relative") }}
                    </p>
                  </div>
                  <MdiArrowRight class="ml-2 size-4 shrink-0 text-muted-foreground" />
                </button>
              </div>
            </CardContent>
          </Card>

          <!-- Low Stock Alert Widget -->
          <Card v-else-if="widget.type === 'low-stock'" class="overflow-hidden shadow">
            <CardHeader class="pb-2">
              <CardTitle class="flex items-center gap-2 text-base">
                <MdiAlertCircle class="size-5 text-orange-500" />
                Low Stock Alerts
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div v-if="recentItemsLoading" class="space-y-2">
                <Skeleton v-for="i in 3" :key="i" class="h-10 rounded" />
              </div>
              <div v-else-if="lowStockItems.length === 0" class="py-4 text-center text-sm text-muted-foreground">
                No low stock items
              </div>
              <div v-else class="divide-y">
                <div
                  v-for="item in lowStockItems"
                  :key="item.id"
                  class="flex items-center justify-between px-1 py-2"
                >
                  <div class="min-w-0 flex-1">
                    <p class="truncate text-sm font-medium">{{ item.name }}</p>
                    <p class="text-xs text-muted-foreground">
                      Qty: <span class="font-semibold text-orange-600">{{ item.quantity }}</span>
                      / Reorder at: {{ item.threshold }}
                    </p>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    class="ml-2 shrink-0"
                    @click="goTo(`/item/${item.id}`)"
                  >
                    Reorder
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>

          <!-- Warranty Tracker Widget -->
          <Card v-else-if="widget.type === 'warranty-tracker'" class="overflow-hidden shadow">
            <CardHeader class="pb-2">
              <CardTitle class="flex items-center gap-2 text-base">
                <MdiShieldCheck class="size-5 text-blue-600" />
                Warranty Tracker
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div v-if="warrantyLoading" class="space-y-2">
                <Skeleton v-for="i in 3" :key="i" class="h-12 rounded" />
              </div>
              <div v-else-if="warrantyAlerts.length === 0" class="py-4 text-center text-sm text-muted-foreground">
                No warranties expiring soon
              </div>
              <div v-else class="divide-y">
                <div
                  v-for="alert in warrantyAlerts"
                  :key="alert.itemId"
                  class="flex items-center justify-between px-1 py-2"
                >
                  <div class="min-w-0 flex-1">
                    <button
                      class="truncate text-sm font-medium hover:underline"
                      @click="goTo(`/item/${alert.itemId}`)"
                    >
                      {{ alert.itemName }}
                    </button>
                    <p class="text-xs text-muted-foreground">
                      Expires: {{ fmtDate(alert.warrantyExpires, "short") }}
                    </p>
                  </div>
                  <Badge :variant="warrantyBadgeVariant(alert.daysRemaining)" class="ml-2 shrink-0">
                    {{ alert.daysRemaining }}d
                  </Badge>
                </div>
              </div>
            </CardContent>
          </Card>

          <!-- Maintenance Due Widget -->
          <Card v-else-if="widget.type === 'maintenance-due'" class="overflow-hidden shadow">
            <CardHeader class="pb-2">
              <CardTitle class="flex items-center gap-2 text-base">
                <MdiWrench class="size-5 text-amber-600" />
                Maintenance Due
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div v-if="maintenanceLoading" class="space-y-2">
                <Skeleton v-for="i in 3" :key="i" class="h-12 rounded" />
              </div>
              <div
                v-else-if="maintenanceEntries.length === 0"
                class="py-4 text-center text-sm text-muted-foreground"
              >
                No scheduled maintenance
              </div>
              <div v-else class="divide-y">
                <div
                  v-for="entry in maintenanceEntries.slice(0, 10)"
                  :key="entry.id"
                  class="flex items-center justify-between px-1 py-2"
                  :class="{ 'bg-red-50/50': isOverdue(entry.scheduledDate) }"
                >
                  <div class="min-w-0 flex-1">
                    <button
                      class="truncate text-sm font-medium hover:underline"
                      @click="goTo(`/item/${entry.itemID}`)"
                    >
                      {{ entry.itemName }}
                    </button>
                    <p class="text-xs text-muted-foreground">
                      {{ entry.name }} &middot; {{ entry.description }}
                    </p>
                  </div>
                  <div class="ml-2 shrink-0 text-right">
                    <p
                      class="text-xs font-medium"
                      :class="isOverdue(entry.scheduledDate) ? 'text-red-600' : 'text-muted-foreground'"
                    >
                      {{ fmtDate(entry.scheduledDate, "short") }}
                    </p>
                    <Badge v-if="isOverdue(entry.scheduledDate)" variant="destructive" class="mt-0.5 text-[10px]">
                      Overdue
                    </Badge>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <!-- Plugin Status Widget -->
          <Card v-else-if="widget.type === 'plugin-status'" class="overflow-hidden shadow">
            <CardHeader class="pb-2">
              <CardTitle class="flex items-center gap-2 text-base">
                <MdiPuzzle class="size-5 text-indigo-600" />
                Plugin Status
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div v-if="pluginsLoading" class="space-y-2">
                <Skeleton v-for="i in 4" :key="i" class="h-8 rounded" />
              </div>
              <div v-else-if="plugins.length === 0" class="py-4 text-center text-sm text-muted-foreground">
                No plugins installed
              </div>
              <div v-else class="divide-y">
                <div
                  v-for="plugin in plugins"
                  :key="plugin.name"
                  class="flex items-center justify-between px-1 py-2"
                >
                  <div class="flex items-center gap-2">
                    <MdiCircle class="size-3" :class="pluginStatusColor(plugin)" />
                    <span class="text-sm">{{ plugin.name }}</span>
                  </div>
                  <span class="text-xs text-muted-foreground">
                    {{ plugin.enabled ? (plugin.status || "Active") : "Disabled" }}
                  </span>
                </div>
                <div class="pt-2">
                  <Button variant="ghost" size="sm" class="w-full text-xs" @click="goTo('/plugins')">
                    Manage Plugins
                    <MdiArrowRight class="ml-1 size-3" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>

          <!-- Quick Actions Widget -->
          <Card v-else-if="widget.type === 'quick-actions'" class="overflow-hidden shadow">
            <CardHeader class="pb-2">
              <CardTitle class="flex items-center gap-2 text-base">
                <MdiLightningBolt class="size-5 text-yellow-500" />
                Quick Actions
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div class="grid grid-cols-2 gap-2">
                <Button
                  variant="outline"
                  class="flex h-auto flex-col items-center gap-1.5 p-3"
                  @click="goTo('/items?create=true')"
                >
                  <MdiPlus class="size-5 text-green-600" />
                  <span class="text-xs">Add Item</span>
                </Button>
                <Button
                  variant="outline"
                  class="flex h-auto flex-col items-center gap-1.5 p-3"
                  @click="goTo('/plugins/ai-vision')"
                >
                  <MdiBarcodeScan class="size-5 text-blue-600" />
                  <span class="text-xs">Scan Barcode</span>
                </Button>
                <Button
                  variant="outline"
                  class="flex h-auto flex-col items-center gap-1.5 p-3"
                  @click="goTo('/plugins/excel-export')"
                >
                  <MdiFileImport class="size-5 text-purple-600" />
                  <span class="text-xs">Import CSV</span>
                </Button>
                <Button
                  variant="outline"
                  class="flex h-auto flex-col items-center gap-1.5 p-3"
                  @click="goTo('/plugins/excel-export')"
                >
                  <MdiFileExport class="size-5 text-teal-600" />
                  <span class="text-xs">Export Data</span>
                </Button>
                <Button
                  variant="outline"
                  class="flex h-auto flex-col items-center gap-1.5 p-3"
                  @click="goTo('/plugins/label-printer')"
                >
                  <MdiPrinter class="size-5 text-orange-600" />
                  <span class="text-xs">Print Labels</span>
                </Button>
                <Button
                  variant="outline"
                  class="flex h-auto flex-col items-center gap-1.5 p-3"
                  @click="goTo('/plugins/catalog')"
                >
                  <MdiBookOpenVariant class="size-5 text-indigo-600" />
                  <span class="text-xs">Browse Catalog</span>
                </Button>
              </div>
            </CardContent>
          </Card>

          <!-- Activity Feed Widget -->
          <Card v-else-if="widget.type === 'activity-feed'" class="overflow-hidden shadow">
            <CardHeader class="pb-2">
              <CardTitle class="flex items-center gap-2 text-base">
                <MdiFormatListBulleted class="size-5 text-cyan-600" />
                Activity Feed
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div v-if="activityLoading" class="space-y-2">
                <Skeleton v-for="i in 5" :key="i" class="h-10 rounded" />
              </div>
              <div v-else-if="activityFeed.length === 0" class="py-4 text-center text-sm text-muted-foreground">
                No recent activity
              </div>
              <div v-else class="divide-y">
                <div
                  v-for="entry in activityFeed.slice(0, 15)"
                  :key="entry.id"
                  class="flex items-start gap-2 px-1 py-2"
                >
                  <Badge :variant="activityBadgeVariant(entry.action)" class="mt-0.5 shrink-0 text-[10px]">
                    {{ entry.action }}
                  </Badge>
                  <div class="min-w-0 flex-1">
                    <p class="truncate text-sm">{{ entry.itemName }}</p>
                    <p v-if="entry.details" class="truncate text-xs text-muted-foreground">{{ entry.details }}</p>
                    <p class="text-xs text-muted-foreground">{{ fmtDate(entry.timestamp, "relative") }}</p>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <!-- Value by Location Chart Widget -->
          <Card v-else-if="widget.type === 'value-by-location'" class="overflow-hidden shadow">
            <CardHeader class="pb-2">
              <CardTitle class="flex items-center gap-2 text-base">
                <MdiChartBar class="size-5 text-emerald-600" />
                Value by Location
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div v-if="valueByLocationLoading || locationStatsLoading" class="space-y-3">
                <Skeleton v-for="i in 5" :key="i" class="h-8 rounded" />
              </div>
              <template v-else>
                <!-- Prefer valueByLocation (plugin API), fall back to locationStats (core) -->
                <div v-if="valueByLocation.length > 0" class="space-y-2">
                  <button
                    v-for="loc in valueByLocation.slice(0, 10)"
                    :key="loc.locationId"
                    class="flex w-full items-center gap-2 rounded p-1 text-left transition-colors hover:bg-muted/50"
                    @click="goTo(`/location/${loc.locationId}`)"
                  >
                    <span class="w-28 shrink-0 truncate text-xs font-medium">{{ loc.locationName }}</span>
                    <div class="flex-1">
                      <div
                        class="h-5 rounded bg-primary/80 transition-all"
                        :style="{ width: barWidthPercent(loc.totalValue) + '%' }"
                      />
                    </div>
                    <span class="w-20 shrink-0 text-right text-xs text-muted-foreground">
                      {{ formatCurrency(loc.totalValue) }}
                    </span>
                  </button>
                </div>
                <div v-else-if="locationStats.length > 0" class="space-y-2">
                  <button
                    v-for="loc in locationStats.slice(0, 10)"
                    :key="loc.id"
                    class="flex w-full items-center gap-2 rounded p-1 text-left transition-colors hover:bg-muted/50"
                    @click="goTo(`/location/${loc.id}`)"
                  >
                    <span class="w-28 shrink-0 truncate text-xs font-medium">{{ loc.name }}</span>
                    <div class="flex-1">
                      <div
                        class="h-5 rounded bg-primary/80 transition-all"
                        :style="{ width: Math.max(2, (loc.total / Math.max(...locationStats.map(l => l.total), 1)) * 100) + '%' }"
                      />
                    </div>
                    <span class="w-16 shrink-0 text-right text-xs text-muted-foreground">
                      {{ loc.total }} items
                    </span>
                  </button>
                </div>
                <div v-else class="py-4 text-center text-sm text-muted-foreground">
                  No location data available
                </div>
              </template>
            </CardContent>
          </Card>

          <!-- Storage Map Widget -->
          <Card v-else-if="widget.type === 'storage-map'" class="overflow-hidden shadow">
            <CardHeader class="pb-2">
              <CardTitle class="flex items-center gap-2 text-base">
                <MdiFileTree class="size-5 text-teal-600" />
                Storage Map
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div v-if="locationTreeLoading" class="space-y-2">
                <Skeleton v-for="i in 5" :key="i" class="h-6 rounded" />
              </div>
              <div v-else-if="locationTree.length === 0" class="py-4 text-center text-sm text-muted-foreground">
                No locations found
              </div>
              <div v-else class="max-h-80 overflow-y-auto">
                <template v-for="node in locationTree" :key="node.id">
                  <StorageTreeNode
                    :node="node"
                    :depth="0"
                    :expanded-nodes="expandedNodes"
                    @toggle="toggleTreeNode"
                    @navigate="goTo"
                  />
                </template>
              </div>
            </CardContent>
          </Card>
        </div>
      </div>

      <!-- Empty State -->
      <div
        v-if="visibleWidgets.length === 0"
        class="flex flex-col items-center justify-center rounded-lg border-2 border-dashed border-gray-300 py-16"
      >
        <MdiPackageVariant class="mb-4 size-12 text-gray-400" />
        <p class="mb-2 text-lg font-medium text-gray-600">No widgets on your dashboard</p>
        <p class="mb-4 text-sm text-gray-500">Click Customize to add widgets to your dashboard.</p>
        <Button variant="default" @click="toggleCustomize">
          <MdiCog class="size-4" />
          Customize Dashboard
        </Button>
      </div>
    </BaseContainer>
  </div>
</template>

<!-- Recursive Storage Tree Node Component -->
<script lang="ts">
  import { computed, defineComponent, type PropType } from "vue";
  import type { TreeItem } from "~/lib/api/types/data-contracts";

  const StorageTreeNode = defineComponent({
    name: "StorageTreeNode",
    props: {
      node: {
        type: Object as PropType<TreeItem>,
        required: true,
      },
      depth: {
        type: Number,
        default: 0,
      },
      expandedNodes: {
        type: Object as PropType<Set<string>>,
        required: true,
      },
    },
    emits: ["toggle", "navigate"],
    setup(props, { emit }) {
      const hasChildren = computed(() => props.node.children && props.node.children.length > 0);
      const isExpanded = computed(() => props.expandedNodes.has(props.node.id));

      function countItems(node: TreeItem): number {
        let count = node.type === "item" ? 1 : 0;
        if (node.children) {
          for (const child of node.children) {
            count += countItems(child);
          }
        }
        return count;
      }

      const itemCount = computed(() => countItems(props.node));

      return { hasChildren, isExpanded, itemCount, emit };
    },
    template: `
      <div>
        <div
          class="flex items-center gap-1 rounded py-1 hover:bg-muted/50"
          :style="{ paddingLeft: (depth * 16 + 4) + 'px' }"
        >
          <button
            v-if="hasChildren"
            class="shrink-0 rounded p-0.5 hover:bg-muted"
            @click="emit('toggle', node.id)"
          >
            <svg
              class="size-3.5 transition-transform"
              :class="{ 'rotate-90': isExpanded }"
              viewBox="0 0 24 24"
              fill="currentColor"
            >
              <path d="M8.59 16.59L13.17 12 8.59 7.41 10 6l6 6-6 6z" />
            </svg>
          </button>
          <span v-else class="w-[18px] shrink-0" />
          <button
            class="flex min-w-0 flex-1 items-center gap-1.5 text-left text-sm hover:underline"
            @click="emit('navigate', '/location/' + node.id)"
          >
            <span class="truncate">{{ node.name }}</span>
          </button>
          <span class="shrink-0 text-xs text-muted-foreground">{{ itemCount }}</span>
        </div>
        <template v-if="hasChildren && isExpanded">
          <StorageTreeNode
            v-for="child in node.children"
            :key="child.id"
            :node="child"
            :depth="depth + 1"
            :expanded-nodes="expandedNodes"
            @toggle="(id: string) => emit('toggle', id)"
            @navigate="(path: string) => emit('navigate', path)"
          />
        </template>
      </div>
    `,
  });
</script>
