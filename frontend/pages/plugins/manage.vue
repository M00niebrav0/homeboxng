<script setup lang="ts">
  import { toast } from "@/components/ui/sonner";
  import { Button } from "@/components/ui/button";
  import { Input } from "@/components/ui/input";
  import { Switch } from "@/components/ui/switch";
  import { Badge } from "@/components/ui/badge";
  import { Label } from "@/components/ui/label";
  import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
  } from "@/components/ui/select";
  import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
  } from "@/components/ui/table";
  import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
  } from "@/components/ui/dialog";
  import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
  } from "@/components/ui/alert-dialog";
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import BaseSectionHeader from "@/components/Base/SectionHeader.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import type { PluginInfo, PluginPermission, PluginLogEntry, CatalogPlugin } from "~/lib/api/classes/plugins";
  import MdiPuzzle from "~icons/mdi/puzzle";
  import MdiRefresh from "~icons/mdi/refresh";
  import MdiMagnify from "~icons/mdi/magnify";
  import MdiCog from "~icons/mdi/cog";
  import MdiShieldCheck from "~icons/mdi/shield-check";
  import MdiDelete from "~icons/mdi/delete";
  import MdiLoading from "~icons/mdi/loading";
  import MdiDownload from "~icons/mdi/download";
  import MdiPlus from "~icons/mdi/plus";
  import MdiClose from "~icons/mdi/close";
  import MdiHistory from "~icons/mdi/history";
  import MdiAlertCircle from "~icons/mdi/alert-circle";
  import MdiCheckCircle from "~icons/mdi/check-circle";
  import MdiFilter from "~icons/mdi/filter";
  import MdiLink from "~icons/mdi/link";
  import MdiChevronRight from "~icons/mdi/chevron-right";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Plugin Management",
  });

  const api = useUserApi();

  // ==================== Plugin Overview ====================
  interface FlatPlugin {
    name: string;
    version: string;
    description: string;
    author: string;
    builtIn: boolean;
    enabled: boolean;
    status: string;
    error: string;
  }

  const { data: plugins, refresh: refreshPlugins } = useAsyncData("manage-plugins", async () => {
    const { data } = await api.plugins.getAll();
    return (data || []).map((ps: any): FlatPlugin => ({
      name: ps.info?.name || ps.name || '',
      version: ps.info?.version || ps.version || '',
      description: ps.info?.description || ps.description || '',
      author: ps.info?.author || ps.author || '',
      builtIn: ps.info?.builtIn ?? ps.builtIn ?? false,
      enabled: ps.state === 'running' || ps.state === 'started' || (ps.enabled !== undefined ? ps.enabled !== false : true),
      status: ps.state || ps.status || 'unknown',
      error: ps.error || '',
    }));
  });

  const refreshLoading = ref(false);

  async function doRefresh() {
    refreshLoading.value = true;
    await refreshPlugins();
    refreshLoading.value = false;
    toast.success("Plugin list refreshed.");
  }

  function statusDotClass(plugin: FlatPlugin): string {
    if (!plugin.enabled) return "bg-gray-400";
    if (plugin.status === "running") return "bg-green-500";
    if (plugin.status === "error") return "bg-red-500";
    return "bg-yellow-500";
  }

  function statusLabel(plugin: FlatPlugin): string {
    if (!plugin.enabled) return "Disabled";
    if (plugin.status === "running") return "Healthy";
    if (plugin.status === "error") return "Error";
    return "Warning";
  }

  function statusBadgeVariant(plugin: FlatPlugin): "default" | "secondary" | "destructive" | "outline" {
    if (!plugin.enabled) return "outline";
    if (plugin.status === "running") return "default";
    if (plugin.status === "error") return "destructive";
    return "secondary";
  }

  async function togglePlugin(plugin: FlatPlugin) {
    if (plugin.enabled) {
      await api.plugins.disable(plugin.name);
      toast.success(`Plugin "${plugin.name}" disabled.`);
    } else {
      await api.plugins.enable(plugin.name);
      toast.success(`Plugin "${plugin.name}" enabled.`);
    }
    refreshPlugins();
  }

  // ==================== Permission Review Modal ====================
  const showPermModal = ref(false);
  const permPlugin = ref<FlatPlugin | null>(null);
  const permList = ref<PluginPermission[]>([]);
  const permLoading = ref(false);

  async function openPermissions(plugin: FlatPlugin) {
    permPlugin.value = plugin;
    permLoading.value = true;
    showPermModal.value = true;

    const { data } = await api.plugins.getPermissions(plugin.name);
    permList.value = data || [];
    permLoading.value = false;
  }

  function riskLevel(perm: PluginPermission): { label: string; color: string } {
    const highRisk = ["admin", "delete", "write-all", "system", "execute"];
    const medRisk = ["write", "create", "update", "modify"];

    const permLower = perm.permission.toLowerCase();
    if (highRisk.some(h => permLower.includes(h))) {
      return { label: "High", color: "destructive" };
    }
    if (medRisk.some(m => permLower.includes(m))) {
      return { label: "Medium", color: "secondary" };
    }
    return { label: "Low", color: "outline" };
  }

  async function togglePermission(perm: PluginPermission) {
    if (!permPlugin.value) return;
    const name = permPlugin.value.name;

    if (perm.granted) {
      await api.plugins.revokePermission(name, perm.permission);
    } else {
      await api.plugins.grantPermission(name, perm.permission);
    }

    // Refresh
    const { data } = await api.plugins.getPermissions(name);
    permList.value = data || [];
  }

  async function grantAllPermissions() {
    if (!permPlugin.value) return;
    for (const perm of permList.value) {
      if (!perm.granted) {
        await api.plugins.grantPermission(permPlugin.value.name, perm.permission);
      }
    }
    const { data } = await api.plugins.getPermissions(permPlugin.value.name);
    permList.value = data || [];
    toast.success("All permissions granted.");
  }

  async function revokeAllPermissions() {
    if (!permPlugin.value) return;
    for (const perm of permList.value) {
      if (perm.granted) {
        await api.plugins.revokePermission(permPlugin.value.name, perm.permission);
      }
    }
    const { data } = await api.plugins.getPermissions(permPlugin.value.name);
    permList.value = data || [];
    toast.success("All permissions revoked.");
  }

  const hasHighRiskPerms = computed(() =>
    permList.value.some(p => riskLevel(p).label === "High")
  );

  // ==================== Plugin Sources ====================
  const showSourcesSection = ref(true);
  const sourcesList = ref<{ url: string; name: string }[]>([
    { url: "https://plugins.homeboxng.dev/catalog.json", name: "Official Repository" },
  ]);
  const newSourceUrl = ref("");
  const sourceUrlError = ref("");
  const showRemoveSourceDialog = ref(false);
  const removeSourceIndex = ref(-1);

  function validateUrl(url: string): boolean {
    try {
      const parsed = new URL(url);
      return parsed.protocol === "https:" || parsed.protocol === "http:";
    } catch {
      return false;
    }
  }

  function addSource() {
    sourceUrlError.value = "";
    if (!newSourceUrl.value.trim()) {
      sourceUrlError.value = "URL is required.";
      return;
    }
    if (!validateUrl(newSourceUrl.value)) {
      sourceUrlError.value = "Please enter a valid HTTP or HTTPS URL.";
      return;
    }
    if (sourcesList.value.some(s => s.url === newSourceUrl.value.trim())) {
      sourceUrlError.value = "This source is already added.";
      return;
    }

    sourcesList.value.push({
      url: newSourceUrl.value.trim(),
      name: new URL(newSourceUrl.value.trim()).hostname,
    });
    newSourceUrl.value = "";
    toast.success("Plugin source added.");
  }

  function confirmRemoveSource(index: number) {
    removeSourceIndex.value = index;
    showRemoveSourceDialog.value = true;
  }

  function removeSource() {
    if (removeSourceIndex.value >= 0) {
      sourcesList.value.splice(removeSourceIndex.value, 1);
      toast.success("Plugin source removed.");
    }
    showRemoveSourceDialog.value = false;
  }

  const catalogRefreshLoading = ref(false);

  async function refreshCatalog() {
    catalogRefreshLoading.value = true;
    await api.plugins.refreshCatalog();
    await loadCatalog();
    catalogRefreshLoading.value = false;
    toast.success("Plugin catalog refreshed.");
  }

  // ==================== Plugin Catalog Browser ====================
  const catalogPlugins = ref<CatalogPlugin[]>([]);
  const catalogLoading = ref(false);
  const catalogSearch = ref("");
  const catalogCategory = ref("all");
  const installLoading = ref<Record<string, boolean>>({});

  const catalogCategories = [
    { label: "All", value: "all" },
    { label: "Data", value: "data" },
    { label: "Analytics", value: "analytics" },
    { label: "Integration", value: "integration" },
    { label: "Utility", value: "utility" },
  ];

  async function loadCatalog() {
    catalogLoading.value = true;
    const { data } = await api.plugins.getCatalog();
    catalogPlugins.value = data || [];
    catalogLoading.value = false;
  }

  onMounted(() => {
    loadCatalog();
    loadAuditTrail();
  });

  const filteredCatalog = computed(() => {
    let results = catalogPlugins.value;
    if (catalogCategory.value !== "all") {
      results = results.filter(p => (p.category || "utility").toLowerCase() === catalogCategory.value);
    }
    if (catalogSearch.value.trim()) {
      const q = catalogSearch.value.toLowerCase();
      results = results.filter(p =>
        p.name.toLowerCase().includes(q) ||
        p.description.toLowerCase().includes(q) ||
        p.author.toLowerCase().includes(q)
      );
    }
    return results;
  });

  async function installPlugin(plugin: CatalogPlugin) {
    installLoading.value[plugin.name] = true;
    try {
      await api.plugins.installPlugin(plugin.name);
      toast.success(`Plugin "${plugin.name}" installed successfully.`);
      plugin.installed = true;
      refreshPlugins();
    } catch {
      toast.error(`Failed to install "${plugin.name}".`);
    }
    installLoading.value[plugin.name] = false;
  }

  // ==================== Audit Trail ====================
  interface AuditEvent {
    id: string;
    timestamp: string;
    pluginName: string;
    action: string;
    details: string;
    level: "info" | "warn" | "error";
  }

  const auditEvents = ref<AuditEvent[]>([]);
  const auditLoading = ref(false);
  const auditFilterPlugin = ref("all");
  const auditFilterAction = ref("all");

  async function loadAuditTrail() {
    auditLoading.value = true;
    // Aggregate logs from all plugins
    const events: AuditEvent[] = [];
    if (plugins.value) {
      for (const p of plugins.value.slice(0, 10)) {
        try {
          const { data } = await api.plugins.getLogs(p.name);
          if (data) {
            for (const entry of data.slice(0, 20)) {
              events.push({
                id: `${p.name}-${entry.timestamp}-${Math.random().toString(36).slice(2, 6)}`,
                timestamp: entry.timestamp,
                pluginName: p.name,
                action: entry.level,
                details: entry.message,
                level: (entry.level === "error" ? "error" : entry.level === "warn" ? "warn" : "info") as AuditEvent["level"],
              });
            }
          }
        } catch {
          // Skip plugins that error on log retrieval
        }
      }
    }
    // Sort by timestamp descending
    events.sort((a, b) => b.timestamp.localeCompare(a.timestamp));
    auditEvents.value = events;
    auditLoading.value = false;
  }

  const auditPluginOptions = computed(() => {
    const names = new Set(auditEvents.value.map(e => e.pluginName));
    return ["all", ...Array.from(names)];
  });

  const auditActionOptions = computed(() => {
    const actions = new Set(auditEvents.value.map(e => e.action));
    return ["all", ...Array.from(actions)];
  });

  const filteredAudit = computed(() => {
    let results = auditEvents.value;
    if (auditFilterPlugin.value !== "all") {
      results = results.filter(e => e.pluginName === auditFilterPlugin.value);
    }
    if (auditFilterAction.value !== "all") {
      results = results.filter(e => e.action === auditFilterAction.value);
    }
    return results.slice(0, 50);
  });

  function auditLevelIcon(level: AuditEvent["level"]) {
    if (level === "error") return MdiAlertCircle;
    if (level === "warn") return MdiAlertCircle;
    return MdiCheckCircle;
  }

  function auditLevelColor(level: AuditEvent["level"]): string {
    if (level === "error") return "text-destructive";
    if (level === "warn") return "text-yellow-500";
    return "text-green-500";
  }

  function exportAuditLog() {
    const lines = filteredAudit.value.map(e =>
      `${e.timestamp}\t${e.pluginName}\t${e.action}\t${e.details}`
    );
    const blob = new Blob([lines.join("\n")], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "plugin-audit-log.txt";
    a.click();
    URL.revokeObjectURL(url);
    toast.success("Audit log exported.");
  }

  // ==================== Uninstall Plugin ====================
  const showUninstallDialog = ref(false);
  const uninstallPluginName = ref("");

  function confirmUninstall(name: string) {
    uninstallPluginName.value = name;
    showUninstallDialog.value = true;
  }

  async function uninstallPlugin() {
    try {
      await api.http.delete({ url: `/api/v1/plugins/${uninstallPluginName.value}` });
      toast.success(`Plugin "${uninstallPluginName.value}" uninstalled.`);
      refreshPlugins();
    } catch {
      toast.error("Failed to uninstall plugin.");
    }
    showUninstallDialog.value = false;
  }
</script>

<template>
  <div>
    <!-- Permission Review Modal -->
    <Dialog v-model:open="showPermModal">
      <DialogContent class="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle class="flex items-center gap-2">
            <MdiShieldCheck class="h-5 w-5" />
            {{ permPlugin?.name }} - Permission Review
          </DialogTitle>
          <DialogDescription>
            Version {{ permPlugin?.version }} by {{ permPlugin?.author }}.
            Review and control each permission this plugin requests.
          </DialogDescription>
        </DialogHeader>

        <!-- High-risk warning banner -->
        <div
          v-if="hasHighRiskPerms"
          class="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 p-3"
        >
          <MdiAlertCircle class="h-5 w-5 flex-shrink-0 text-destructive" />
          <p class="text-sm text-destructive">
            This plugin requests high-risk permissions. Review each one carefully before granting access.
          </p>
        </div>

        <div v-if="permLoading" class="flex items-center justify-center py-12">
          <MdiLoading class="h-6 w-6 animate-spin text-muted-foreground" />
        </div>

        <div v-else class="max-h-96 space-y-2 overflow-y-auto">
          <div
            v-for="perm in permList"
            :key="perm.permission"
            class="flex items-center justify-between rounded-lg border p-3 transition-colors"
            :class="perm.granted ? 'border-primary/20 bg-primary/5' : ''"
          >
            <div class="flex-1 space-y-1">
              <div class="flex flex-wrap items-center gap-2">
                <span class="text-sm font-medium">{{ perm.permission }}</span>
                <Badge
                  :variant="riskLevel(perm).color as any"
                  class="text-[10px]"
                >
                  {{ riskLevel(perm).label }} Risk
                </Badge>
                <Badge
                  :variant="perm.required ? 'secondary' : 'outline'"
                  class="text-[10px]"
                >
                  {{ perm.required ? "Required" : "Optional" }}
                </Badge>
              </div>
              <p class="text-xs text-muted-foreground">{{ perm.reason }}</p>
            </div>
            <Switch
              :checked="perm.granted"
              @update:checked="togglePermission(perm)"
            />
          </div>

          <div v-if="permList.length === 0" class="py-8 text-center text-sm text-muted-foreground">
            This plugin has not declared any permissions.
          </div>
        </div>

        <DialogFooter class="flex-col gap-2 sm:flex-row">
          <div class="flex gap-2">
            <Button variant="outline" size="sm" @click="grantAllPermissions">
              Grant All
            </Button>
            <Button variant="destructive" size="sm" @click="revokeAllPermissions">
              Revoke All
            </Button>
          </div>
          <Button variant="outline" @click="showPermModal = false">Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Remove Source Confirmation -->
    <AlertDialog v-model:open="showRemoveSourceDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Remove Plugin Source</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to remove this plugin source?
            Plugins installed from this source will remain but may not receive updates.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            @click="removeSource"
          >
            Remove
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Uninstall Plugin Confirmation -->
    <AlertDialog v-model:open="showUninstallDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Uninstall Plugin</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to uninstall "{{ uninstallPluginName }}"?
            This will remove the plugin and all its configuration. This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            @click="uninstallPlugin"
          >
            Uninstall
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Main Content -->
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 class="text-2xl font-bold">Plugin Management</h1>
          <p class="text-sm text-muted-foreground">
            Full control over installed plugins, permissions, sources, and the catalog.
          </p>
        </div>
        <div class="flex gap-2">
          <Button variant="outline" size="sm" :disabled="refreshLoading" @click="doRefresh">
            <MdiRefresh class="mr-1 h-4 w-4" :class="{ 'animate-spin': refreshLoading }" />
            Refresh
          </Button>
          <NuxtLink to="/plugins">
            <Button variant="outline" size="sm">
              <MdiPuzzle class="mr-1 h-4 w-4" />
              Plugin Manager
            </Button>
          </NuxtLink>
        </div>
      </div>

      <!-- Stats Bar -->
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div class="rounded-lg border p-3">
          <p class="text-xs text-muted-foreground">Total Plugins</p>
          <p class="text-2xl font-bold text-primary">{{ plugins?.length || 0 }}</p>
        </div>
        <div class="rounded-lg border p-3">
          <p class="text-xs text-muted-foreground">Enabled</p>
          <p class="text-2xl font-bold text-green-500">
            {{ plugins?.filter(p => p.enabled !== false).length || 0 }}
          </p>
        </div>
        <div class="rounded-lg border p-3">
          <p class="text-xs text-muted-foreground">Built-in</p>
          <p class="text-2xl font-bold">
            {{ plugins?.filter(p => p.builtIn).length || 0 }}
          </p>
        </div>
        <div class="rounded-lg border p-3">
          <p class="text-xs text-muted-foreground">Errors</p>
          <p class="text-2xl font-bold text-destructive">
            {{ plugins?.filter(p => p.status === 'error').length || 0 }}
          </p>
        </div>
      </div>

      <!-- ==================== Plugin Overview Grid ==================== -->
      <BaseCard>
        <template #title>
          <BaseSectionHeader>
            <MdiPuzzle class="-mt-1 mr-2" />
            <span>Installed Plugins</span>
            <template #description>Manage your installed plugins, toggle them on or off, and review their permissions.</template>
          </BaseSectionHeader>
        </template>

        <div class="grid grid-cols-1 gap-3 p-4 md:grid-cols-2 xl:grid-cols-3">
          <div
            v-for="plugin in (plugins || [])"
            :key="plugin.name"
            class="relative flex flex-col rounded-lg border transition-colors"
            :class="plugin.enabled !== false ? 'border-border' : 'border-dashed opacity-70'"
          >
            <!-- Plugin header -->
            <div class="flex items-start gap-3 p-4">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2">
                  <!-- Status dot -->
                  <span
                    class="h-2.5 w-2.5 rounded-full"
                    :class="statusDotClass(plugin)"
                  />
                  <h3 class="truncate text-sm font-semibold">{{ plugin.name }}</h3>
                </div>
                <p class="mt-0.5 text-xs text-muted-foreground">
                  v{{ plugin.version }} by {{ plugin.author }}
                </p>
                <p class="mt-2 line-clamp-2 text-sm">{{ plugin.description }}</p>
              </div>
              <div class="flex flex-col items-end gap-1">
                <Badge :variant="statusBadgeVariant(plugin)" class="text-[10px]">
                  {{ statusLabel(plugin) }}
                </Badge>
                <Switch
                  :checked="plugin.enabled !== false"
                  @update:checked="togglePlugin(plugin)"
                  class="mt-1"
                />
              </div>
            </div>

            <!-- Plugin actions -->
            <div class="flex gap-1 border-t p-2">
              <NuxtLink :to="`/plugins/${plugin.name}`">
                <Button variant="ghost" size="sm" class="h-7 text-xs">
                  <MdiCog class="mr-1 h-3 w-3" />
                  Configure
                </Button>
              </NuxtLink>
              <Button variant="ghost" size="sm" class="h-7 text-xs" @click="openPermissions(plugin)">
                <MdiShieldCheck class="mr-1 h-3 w-3" />
                Permissions
              </Button>
              <Button
                v-if="!plugin.builtIn"
                variant="ghost"
                size="sm"
                class="ml-auto h-7 text-xs text-destructive hover:text-destructive"
                @click="confirmUninstall(plugin.name)"
              >
                <MdiDelete class="h-3 w-3" />
              </Button>
            </div>
          </div>
        </div>

        <div v-if="!plugins || plugins.length === 0" class="py-12 text-center text-sm text-muted-foreground">
          <MdiPuzzle class="mx-auto mb-2 h-8 w-8 opacity-50" />
          <p>No plugins installed.</p>
          <p class="mt-1">Browse the catalog below to discover and install plugins.</p>
        </div>
      </BaseCard>

      <!-- ==================== Plugin Sources ==================== -->
      <BaseCard :collapsable="true">
        <template #title>
          <BaseSectionHeader>
            <MdiLink class="-mt-1 mr-2" />
            <span>Plugin Sources</span>
            <template #description>Manage repositories where HomeBoxNG looks for plugins.</template>
          </BaseSectionHeader>
        </template>

        <div class="space-y-4 p-4">
          <!-- Existing sources -->
          <div class="space-y-2">
            <div
              v-for="(source, idx) in sourcesList"
              :key="source.url"
              class="flex items-center justify-between rounded-lg border p-3"
            >
              <div>
                <p class="text-sm font-medium">{{ source.name }}</p>
                <p class="text-xs text-muted-foreground font-mono">{{ source.url }}</p>
              </div>
              <Button
                variant="ghost"
                size="sm"
                class="text-destructive hover:text-destructive"
                @click="confirmRemoveSource(idx)"
              >
                <MdiClose class="h-4 w-4" />
              </Button>
            </div>

            <div v-if="sourcesList.length === 0" class="py-4 text-center text-sm text-muted-foreground">
              No plugin sources configured.
            </div>
          </div>

          <!-- Add new source -->
          <div class="flex gap-2">
            <div class="flex-1">
              <Input
                v-model="newSourceUrl"
                placeholder="https://plugins.example.com/catalog.json"
                :class="{ 'border-destructive': sourceUrlError }"
              />
              <p v-if="sourceUrlError" class="mt-1 text-xs text-destructive">{{ sourceUrlError }}</p>
            </div>
            <Button variant="outline" size="sm" @click="addSource">
              <MdiPlus class="mr-1 h-4 w-4" />
              Add Source
            </Button>
          </div>

          <div class="flex justify-end">
            <Button variant="outline" size="sm" :disabled="catalogRefreshLoading" @click="refreshCatalog">
              <MdiRefresh class="mr-1 h-4 w-4" :class="{ 'animate-spin': catalogRefreshLoading }" />
              Refresh Catalog
            </Button>
          </div>
        </div>
      </BaseCard>

      <!-- ==================== Plugin Catalog Browser ==================== -->
      <BaseCard>
        <template #title>
          <BaseSectionHeader>
            <MdiDownload class="-mt-1 mr-2" />
            <span>Plugin Catalog</span>
            <template #description>Browse and install plugins from configured sources.</template>
          </BaseSectionHeader>
        </template>

        <div class="space-y-4 p-4">
          <!-- Search and filters -->
          <div class="flex flex-col gap-3 sm:flex-row">
            <div class="relative flex-1">
              <MdiMagnify class="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                v-model="catalogSearch"
                placeholder="Search plugins..."
                class="pl-9"
              />
            </div>
            <div class="flex flex-wrap gap-1">
              <Button
                v-for="cat in catalogCategories"
                :key="cat.value"
                :variant="catalogCategory === cat.value ? 'default' : 'outline'"
                size="sm"
                @click="catalogCategory = cat.value"
              >
                {{ cat.label }}
              </Button>
            </div>
          </div>

          <!-- Loading state -->
          <div v-if="catalogLoading" class="flex items-center justify-center py-12">
            <MdiLoading class="h-6 w-6 animate-spin text-muted-foreground" />
          </div>

          <!-- Plugin grid -->
          <div v-else class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
            <div
              v-for="plugin in filteredCatalog"
              :key="plugin.name"
              class="flex flex-col rounded-lg border p-4"
            >
              <div class="flex-1">
                <div class="flex items-center justify-between">
                  <h4 class="text-sm font-semibold">{{ plugin.name }}</h4>
                  <Badge v-if="plugin.category" variant="outline" class="text-[10px]">
                    {{ plugin.category }}
                  </Badge>
                </div>
                <p class="mt-0.5 text-xs text-muted-foreground">
                  v{{ plugin.version }} by {{ plugin.author }}
                </p>
                <p class="mt-2 line-clamp-3 text-sm">{{ plugin.description }}</p>
              </div>

              <div class="mt-3 flex items-center justify-between">
                <a
                  v-if="plugin.repository"
                  :href="plugin.repository"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-xs text-primary underline"
                >
                  Repository
                </a>
                <span v-else />
                <Button
                  v-if="!plugin.installed"
                  size="sm"
                  :disabled="!!installLoading[plugin.name]"
                  @click="installPlugin(plugin)"
                >
                  <MdiLoading v-if="installLoading[plugin.name]" class="mr-1 h-3 w-3 animate-spin" />
                  <MdiDownload v-else class="mr-1 h-3 w-3" />
                  Install
                </Button>
                <Badge v-else variant="secondary" class="text-[10px]">Installed</Badge>
              </div>
            </div>
          </div>

          <div
            v-if="!catalogLoading && filteredCatalog.length === 0"
            class="py-8 text-center text-sm text-muted-foreground"
          >
            No plugins found matching your search criteria.
          </div>
        </div>
      </BaseCard>

      <!-- ==================== Audit Trail ==================== -->
      <BaseCard :collapsable="true">
        <template #title>
          <BaseSectionHeader>
            <MdiHistory class="-mt-1 mr-2" />
            <span>Audit Trail</span>
            <template #description>Recent plugin-related events and activity.</template>
          </BaseSectionHeader>
        </template>

        <div class="space-y-4 p-4">
          <!-- Filters -->
          <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <div class="flex gap-2">
              <Select v-model="auditFilterPlugin">
                <SelectTrigger class="w-40">
                  <SelectValue placeholder="Filter by plugin" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="opt in auditPluginOptions" :key="opt" :value="opt">
                    {{ opt === 'all' ? 'All Plugins' : opt }}
                  </SelectItem>
                </SelectContent>
              </Select>

              <Select v-model="auditFilterAction">
                <SelectTrigger class="w-32">
                  <SelectValue placeholder="Filter by level" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="opt in auditActionOptions" :key="opt" :value="opt">
                    {{ opt === 'all' ? 'All Levels' : opt }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <Button variant="outline" size="sm" @click="exportAuditLog">
              <MdiDownload class="mr-1 h-4 w-4" />
              Export Log
            </Button>
          </div>

          <!-- Loading state -->
          <div v-if="auditLoading" class="flex items-center justify-center py-8">
            <MdiLoading class="h-6 w-6 animate-spin text-muted-foreground" />
          </div>

          <!-- Timeline -->
          <div v-else class="relative max-h-96 space-y-0 overflow-y-auto">
            <div
              v-for="event in filteredAudit"
              :key="event.id"
              class="flex gap-3 border-l-2 py-3 pl-4"
              :class="{
                'border-destructive': event.level === 'error',
                'border-yellow-500': event.level === 'warn',
                'border-green-500': event.level === 'info',
              }"
            >
              <component
                :is="auditLevelIcon(event.level)"
                class="mt-0.5 h-4 w-4 flex-shrink-0"
                :class="auditLevelColor(event.level)"
              />
              <div class="flex-1 min-w-0">
                <div class="flex flex-wrap items-center gap-2">
                  <Badge variant="outline" class="text-[10px]">{{ event.pluginName }}</Badge>
                  <span class="text-xs text-muted-foreground">{{ event.timestamp }}</span>
                </div>
                <p class="mt-1 text-sm">{{ event.details }}</p>
              </div>
            </div>

            <div v-if="filteredAudit.length === 0" class="py-8 text-center text-sm text-muted-foreground">
              No audit events found matching the current filters.
            </div>
          </div>
        </div>
      </BaseCard>
    </BaseContainer>
  </div>
</template>
