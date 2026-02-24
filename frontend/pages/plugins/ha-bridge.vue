<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import PluginHealthIndicator from "~/components/Plugin/PluginHealthIndicator.vue";
  import { route } from "~/lib/api/base";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | HA Bridge",
  });

  const api = useUserApi();

  // --- Types ---

  interface HABridgeStatus {
    connected: boolean;
    lastSync: string;
    entitiesSynced: number;
    automationsCount: number;
    health: "healthy" | "degraded" | "error" | "disabled" | "unknown";
    healthMessage: string;
  }

  interface HAEntityMapping {
    id: string;
    entityId: string;
    entityName: string;
    entityDomain: string;
    itemId: string;
    itemName: string;
    linkType: "controls" | "monitors" | "is";
    syncStatus: "synced" | "pending" | "error";
    lastSynced: string;
  }

  interface HAAutomation {
    id: string;
    name: string;
    description: string;
    trigger: string;
    action: string;
    enabled: boolean;
    lastTriggered?: string;
  }

  interface HADiscoveredEntity {
    entityId: string;
    name: string;
    domain: string;
    state: string;
    attributes: Record<string, unknown>;
  }

  interface SyncLogEntry {
    timestamp: string;
    level: "info" | "warn" | "error";
    message: string;
  }

  // --- State ---

  const haUrl = ref("");
  const haToken = ref("");
  const connectionStatus = ref<"idle" | "testing" | "success" | "error">("idle");
  const connectionMessage = ref("");
  const loading = ref(false);
  const syncing = ref(false);

  const status = ref<HABridgeStatus>({
    connected: false,
    lastSync: "Never",
    entitiesSynced: 0,
    automationsCount: 0,
    health: "unknown",
    healthMessage: "",
  });

  const entityMappings = ref<HAEntityMapping[]>([]);
  const automations = ref<HAAutomation[]>([]);
  const discoveredEntities = ref<HADiscoveredEntity[]>([]);
  const syncLog = ref<SyncLogEntry[]>([]);
  const discovering = ref(false);

  // Auto-sync settings
  const autoSync = reactive({
    enabled: false,
    interval: "15min",
  });

  const intervalOptions = [
    { value: "5min", label: "Every 5 minutes" },
    { value: "15min", label: "Every 15 minutes" },
    { value: "30min", label: "Every 30 minutes" },
    { value: "1hr", label: "Every hour" },
  ];

  // New mapping form
  const newMapping = reactive({
    entitySearch: "",
    itemSearch: "",
    selectedEntityId: "",
    selectedItemId: "",
    linkType: "monitors" as "controls" | "monitors" | "is",
  });
  const entitySearchResults = ref<HADiscoveredEntity[]>([]);
  const itemSearchResults = ref<Array<{ id: string; name: string }>>([]);

  // New automation form
  const showAddAutomation = ref(false);
  const newAutomation = reactive({
    name: "",
    trigger: "",
    action: "",
  });

  const triggerOptions = [
    { value: "item_moved", label: "When item moves location" },
    { value: "ha_sensor_trigger", label: "When HA sensor triggers" },
    { value: "warranty_expires", label: "When item warranty expires" },
    { value: "maintenance_due", label: "When maintenance is due" },
    { value: "item_quantity_low", label: "When item quantity is low" },
  ];

  const actionOptions = [
    { value: "update_ha_entity", label: "Update HA entity" },
    { value: "add_maintenance_entry", label: "Add maintenance entry" },
    { value: "ha_notification", label: "Send HA notification" },
    { value: "ha_service_call", label: "Call HA service" },
    { value: "update_item_field", label: "Update item field" },
  ];

  const linkTypeOptions = [
    { value: "controls", label: "Controls" },
    { value: "monitors", label: "Monitors" },
    { value: "is", label: "Is" },
  ];

  // --- Data Loading ---

  useAsyncData("ha-bridge-status", async () => {
    try {
      const { data } = await api.http.get<HABridgeStatus>({ url: route("/plugins/ha-bridge/status") });
      if (data) status.value = data;
      return data;
    } catch {
      return null;
    }
  });

  useAsyncData("ha-bridge-config", async () => {
    try {
      const { data } = await api.plugins.getConfig("ha-bridge");
      if (data) {
        for (const field of data) {
          if (field.key === "ha_url") haUrl.value = field.value || field.default || "";
          if (field.key === "ha_token") haToken.value = field.value || "";
          if (field.key === "auto_sync_enabled") autoSync.enabled = field.value === "true";
          if (field.key === "auto_sync_interval") autoSync.interval = field.value || "15min";
        }
      }
      return data;
    } catch {
      return null;
    }
  });

  useAsyncData("ha-bridge-entities", async () => {
    try {
      const { data } = await api.http.get<HAEntityMapping[]>({ url: route("/plugins/ha-bridge/entities") });
      entityMappings.value = data || [];
      return data;
    } catch {
      return [];
    }
  });

  useAsyncData("ha-bridge-automations", async () => {
    try {
      const { data } = await api.http.get<HAAutomation[]>({ url: route("/plugins/ha-bridge/automations") });
      automations.value = data || [];
      return data;
    } catch {
      return [];
    }
  });

  // --- Methods ---

  async function testConnection() {
    connectionStatus.value = "testing";
    connectionMessage.value = "";
    try {
      const { data, error } = await api.http.post<{ url: string; token: string }, { success: boolean; message: string }>({
        url: route("/plugins/ha-bridge/test-connection"),
        body: { url: haUrl.value, token: haToken.value },
      });
      if (error || !data?.success) {
        connectionStatus.value = "error";
        connectionMessage.value = data?.message || "Connection failed";
      } else {
        connectionStatus.value = "success";
        connectionMessage.value = data.message || "Connected to Home Assistant";
      }
    } catch {
      connectionStatus.value = "error";
      connectionMessage.value = "Failed to test connection";
    }
  }

  async function saveConfig() {
    loading.value = true;
    try {
      await api.plugins.saveConfig("ha-bridge", {
        ha_url: haUrl.value,
        ha_token: haToken.value,
        auto_sync_enabled: autoSync.enabled ? "true" : "false",
        auto_sync_interval: autoSync.interval,
      });
    } finally {
      loading.value = false;
    }
  }

  async function syncNow() {
    syncing.value = true;
    try {
      await api.http.post<void, void>({ url: route("/plugins/ha-bridge/sync") });
      // Refresh status and log
      const { data: newStatus } = await api.http.get<HABridgeStatus>({ url: route("/plugins/ha-bridge/status") });
      if (newStatus) status.value = newStatus;
      const { data: newMappings } = await api.http.get<HAEntityMapping[]>({ url: route("/plugins/ha-bridge/entities") });
      entityMappings.value = newMappings || [];
      await refreshSyncLog();
    } finally {
      syncing.value = false;
    }
  }

  async function refreshSyncLog() {
    try {
      const { data } = await api.plugins.getLogs("ha-bridge");
      syncLog.value = (data || []).slice(0, 50).map(entry => ({
        timestamp: entry.timestamp,
        level: entry.level as "info" | "warn" | "error",
        message: entry.message,
      }));
    } catch {
      // ignore
    }
  }

  async function discoverEntities() {
    discovering.value = true;
    try {
      const { data } = await api.http.get<HADiscoveredEntity[]>({ url: route("/plugins/ha-bridge/discover") });
      discoveredEntities.value = data || [];
    } finally {
      discovering.value = false;
    }
  }

  async function searchEntities() {
    if (!newMapping.entitySearch.trim()) return;
    // Filter discovered entities locally or fetch from server
    if (discoveredEntities.value.length > 0) {
      const query = newMapping.entitySearch.toLowerCase();
      entitySearchResults.value = discoveredEntities.value.filter(
        e => e.name.toLowerCase().includes(query) || e.entityId.toLowerCase().includes(query)
      );
    } else {
      try {
        const { data } = await api.http.get<HADiscoveredEntity[]>({
          url: route("/plugins/ha-bridge/discover", { q: newMapping.entitySearch }),
        });
        entitySearchResults.value = data || [];
      } catch {
        entitySearchResults.value = [];
      }
    }
  }

  async function searchItems() {
    if (!newMapping.itemSearch.trim()) return;
    try {
      const { data } = await api.http.get<Array<{ id: string; name: string }>>({
        url: route("/plugins/ha-bridge/search/items", { q: newMapping.itemSearch }),
      });
      itemSearchResults.value = data || [];
    } catch {
      itemSearchResults.value = [];
    }
  }

  async function createMapping() {
    if (!newMapping.selectedEntityId || !newMapping.selectedItemId) return;
    loading.value = true;
    try {
      await api.http.post<{ entityId: string; itemId: string; linkType: string }, void>({
        url: route("/plugins/ha-bridge/entities/map"),
        body: {
          entityId: newMapping.selectedEntityId,
          itemId: newMapping.selectedItemId,
          linkType: newMapping.linkType,
        },
      });
      // Refresh mappings
      const { data } = await api.http.get<HAEntityMapping[]>({ url: route("/plugins/ha-bridge/entities") });
      entityMappings.value = data || [];
      // Reset form
      newMapping.entitySearch = "";
      newMapping.itemSearch = "";
      newMapping.selectedEntityId = "";
      newMapping.selectedItemId = "";
      newMapping.linkType = "monitors";
      entitySearchResults.value = [];
      itemSearchResults.value = [];
    } finally {
      loading.value = false;
    }
  }

  async function removeMapping(id: string) {
    loading.value = true;
    try {
      await api.http.delete<void>({ url: route(`/plugins/ha-bridge/entities/${id}`) });
      entityMappings.value = entityMappings.value.filter(m => m.id !== id);
    } finally {
      loading.value = false;
    }
  }

  async function quickMapEntity(entity: HADiscoveredEntity) {
    newMapping.selectedEntityId = entity.entityId;
    newMapping.entitySearch = entity.name;
    entitySearchResults.value = [entity];
    // Scroll to mapping form (let user pick an item)
    const el = document.getElementById("entity-mapping-section");
    if (el) el.scrollIntoView({ behavior: "smooth" });
  }

  async function createAutomation() {
    if (!newAutomation.name || !newAutomation.trigger || !newAutomation.action) return;
    loading.value = true;
    try {
      await api.http.post<{ name: string; trigger: string; action: string }, void>({
        url: route("/plugins/ha-bridge/automations"),
        body: {
          name: newAutomation.name,
          trigger: newAutomation.trigger,
          action: newAutomation.action,
        },
      });
      // Refresh
      const { data } = await api.http.get<HAAutomation[]>({ url: route("/plugins/ha-bridge/automations") });
      automations.value = data || [];
      newAutomation.name = "";
      newAutomation.trigger = "";
      newAutomation.action = "";
      showAddAutomation.value = false;
    } finally {
      loading.value = false;
    }
  }

  async function toggleAutomation(automation: HAAutomation) {
    loading.value = true;
    try {
      await api.http.put<{ enabled: boolean }, void>({
        url: route(`/plugins/ha-bridge/automations/${automation.id}`),
        body: { enabled: !automation.enabled },
      });
      automation.enabled = !automation.enabled;
    } finally {
      loading.value = false;
    }
  }

  async function deleteAutomation(id: string) {
    loading.value = true;
    try {
      await api.http.delete<void>({ url: route(`/plugins/ha-bridge/automations/${id}`) });
      automations.value = automations.value.filter(a => a.id !== id);
    } finally {
      loading.value = false;
    }
  }

  function connectionBadgeClass(): string {
    if (connectionStatus.value === "success") return "badge-success";
    if (connectionStatus.value === "error") return "badge-error";
    if (connectionStatus.value === "testing") return "badge-warning";
    return "badge-ghost";
  }

  function connectionBadgeText(): string {
    if (connectionStatus.value === "success") return "Connected";
    if (connectionStatus.value === "error") return "Error";
    if (connectionStatus.value === "testing") return "Testing...";
    return "Not Tested";
  }

  function syncStatusBadge(syncStatus: string): string {
    if (syncStatus === "synced") return "badge-success";
    if (syncStatus === "pending") return "badge-warning";
    if (syncStatus === "error") return "badge-error";
    return "badge-ghost";
  }

  function logLevelClass(level: string): string {
    if (level === "error") return "bg-error/10 text-error";
    if (level === "warn") return "bg-warning/10 text-warning";
    return "bg-base-200";
  }

  // Load sync log on mount
  onMounted(() => {
    refreshSyncLog();
  });
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">HA Bridge - Home Assistant Integration</h1>
          <p class="text-sm opacity-70">
            Connect HomeBoxNG to Home Assistant for entity mapping, automations, and real-time sync.
          </p>
        </div>
        <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
          Back to Plugins
        </NuxtLink>
      </div>

      <!-- Status Stats Row -->
      <div class="stats shadow w-full">
        <div class="stat">
          <div class="stat-title">Connection</div>
          <div class="stat-value text-sm">
            <PluginHealthIndicator
              :status="status.health"
              :last-check="status.lastSync"
              :message="status.healthMessage"
            />
          </div>
          <div class="stat-desc">{{ status.connected ? "Connected" : "Disconnected" }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Last Sync</div>
          <div class="stat-value text-sm">{{ status.lastSync || "Never" }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Entities Synced</div>
          <div class="stat-value text-primary">{{ status.entitiesSynced }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Automations</div>
          <div class="stat-value text-accent">{{ status.automationsCount }}</div>
        </div>
      </div>

      <!-- Connection Settings -->
      <section>
        <Subtitle>Connection Settings</Subtitle>
        <BaseCard>
          <div class="p-4">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Home Assistant URL</span>
                </label>
                <input
                  v-model="haUrl"
                  type="url"
                  placeholder="http://homeassistant.local:8123"
                  class="input input-bordered input-sm w-full"
                />
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Long-Lived Access Token</span>
                </label>
                <input
                  v-model="haToken"
                  type="password"
                  placeholder="Enter HA access token"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label">
                  <span class="label-text-alt opacity-50">Generate in HA: Profile > Security > Long-Lived Access Tokens</span>
                </label>
              </div>
            </div>
            <div class="flex items-center gap-3 mt-4">
              <button class="btn btn-sm btn-primary" :disabled="loading" @click="saveConfig">
                Save Settings
              </button>
              <button
                class="btn btn-sm btn-outline"
                :disabled="connectionStatus === 'testing' || !haUrl"
                @click="testConnection"
              >
                Test Connection
              </button>
              <span class="badge" :class="connectionBadgeClass()">
                {{ connectionBadgeText() }}
              </span>
              <span v-if="connectionMessage" class="text-sm opacity-70">{{ connectionMessage }}</span>
            </div>
          </div>
        </BaseCard>
      </section>

      <!-- Entity Mapping -->
      <section id="entity-mapping-section">
        <Subtitle>Entity Mapping</Subtitle>

        <!-- Add mapping form -->
        <BaseCard class="mb-4">
          <div class="p-4">
            <p class="text-sm font-medium mb-3">Map an HA entity to a HomeBox item</p>
            <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
              <!-- Entity Search -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text">HA Entity</span>
                </label>
                <div class="flex gap-2">
                  <input
                    v-model="newMapping.entitySearch"
                    type="text"
                    placeholder="Search HA entities..."
                    class="input input-bordered input-sm flex-1"
                    @keyup.enter="searchEntities"
                  />
                  <button class="btn btn-sm btn-outline" @click="searchEntities">Search</button>
                </div>
                <select
                  v-if="entitySearchResults.length > 0"
                  v-model="newMapping.selectedEntityId"
                  class="select select-bordered select-sm w-full mt-2"
                >
                  <option value="">Select entity</option>
                  <option v-for="entity in entitySearchResults" :key="entity.entityId" :value="entity.entityId">
                    {{ entity.name }} ({{ entity.entityId }})
                  </option>
                </select>
              </div>

              <!-- Link Type -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Link Type</span>
                </label>
                <select v-model="newMapping.linkType" class="select select-bordered select-sm w-full">
                  <option v-for="opt in linkTypeOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </option>
                </select>
                <label class="label">
                  <span class="label-text-alt opacity-50">How the item relates to the entity</span>
                </label>
              </div>

              <!-- Item Search -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text">HomeBox Item</span>
                </label>
                <div class="flex gap-2">
                  <input
                    v-model="newMapping.itemSearch"
                    type="text"
                    placeholder="Search items..."
                    class="input input-bordered input-sm flex-1"
                    @keyup.enter="searchItems"
                  />
                  <button class="btn btn-sm btn-outline" @click="searchItems">Search</button>
                </div>
                <select
                  v-if="itemSearchResults.length > 0"
                  v-model="newMapping.selectedItemId"
                  class="select select-bordered select-sm w-full mt-2"
                >
                  <option value="">Select item</option>
                  <option v-for="item in itemSearchResults" :key="item.id" :value="item.id">
                    {{ item.name }}
                  </option>
                </select>
              </div>
            </div>
            <div class="flex justify-end mt-4">
              <button
                class="btn btn-sm btn-primary"
                :disabled="loading || !newMapping.selectedEntityId || !newMapping.selectedItemId"
                @click="createMapping"
              >
                Create Mapping
              </button>
            </div>
          </div>
        </BaseCard>

        <!-- Mapping Table -->
        <BaseCard>
          <div v-if="entityMappings.length === 0" class="p-6 text-center opacity-50">
            <p>No entity mappings configured. Use the form above or discover entities below.</p>
          </div>
          <div v-else class="overflow-x-auto">
            <table class="table table-sm w-full">
              <thead>
                <tr>
                  <th>HA Entity</th>
                  <th>Link</th>
                  <th>HomeBox Item</th>
                  <th>Sync Status</th>
                  <th>Last Synced</th>
                  <th class="w-16">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="mapping in entityMappings" :key="mapping.id">
                  <td>
                    <div>
                      <span class="font-medium text-sm">{{ mapping.entityName }}</span>
                      <br />
                      <span class="text-xs opacity-50 font-mono">{{ mapping.entityId }}</span>
                    </div>
                  </td>
                  <td>
                    <span class="badge badge-sm badge-outline">{{ mapping.linkType }}</span>
                  </td>
                  <td class="font-medium text-sm">{{ mapping.itemName }}</td>
                  <td>
                    <span class="badge badge-sm" :class="syncStatusBadge(mapping.syncStatus)">
                      {{ mapping.syncStatus }}
                    </span>
                  </td>
                  <td class="text-xs opacity-60">{{ mapping.lastSynced }}</td>
                  <td>
                    <button class="btn btn-xs btn-ghost text-error" @click="removeMapping(mapping.id)">
                      Remove
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </BaseCard>
      </section>

      <!-- Automations -->
      <section>
        <div class="flex items-center justify-between mb-3">
          <Subtitle>Automations</Subtitle>
          <button class="btn btn-xs btn-primary" @click="showAddAutomation = !showAddAutomation">
            {{ showAddAutomation ? "Cancel" : "Add Automation" }}
          </button>
        </div>

        <!-- Add automation form -->
        <BaseCard v-if="showAddAutomation" class="mb-4">
          <div class="p-4">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Name</span>
                </label>
                <input
                  v-model="newAutomation.name"
                  type="text"
                  placeholder="Automation name"
                  class="input input-bordered input-sm w-full"
                />
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Trigger</span>
                </label>
                <select v-model="newAutomation.trigger" class="select select-bordered select-sm w-full">
                  <option value="" disabled>Select trigger</option>
                  <option v-for="opt in triggerOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </option>
                </select>
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Action</span>
                </label>
                <select v-model="newAutomation.action" class="select select-bordered select-sm w-full">
                  <option value="" disabled>Select action</option>
                  <option v-for="opt in actionOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </option>
                </select>
              </div>
            </div>
            <div class="flex justify-end mt-4">
              <button
                class="btn btn-sm btn-primary"
                :disabled="loading || !newAutomation.name || !newAutomation.trigger || !newAutomation.action"
                @click="createAutomation"
              >
                Create Automation
              </button>
            </div>
          </div>
        </BaseCard>

        <!-- Automations list -->
        <BaseCard>
          <div v-if="automations.length === 0" class="p-6 text-center opacity-50">
            <p>No automations configured. Click "Add Automation" to create one.</p>
          </div>
          <div v-else class="overflow-x-auto">
            <table class="table table-sm w-full">
              <thead>
                <tr>
                  <th>Name</th>
                  <th>Trigger</th>
                  <th>Action</th>
                  <th>Last Triggered</th>
                  <th class="w-20">Status</th>
                  <th class="w-24">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="automation in automations" :key="automation.id">
                  <td>
                    <div>
                      <span class="font-medium text-sm">{{ automation.name }}</span>
                      <br v-if="automation.description" />
                      <span v-if="automation.description" class="text-xs opacity-50">{{ automation.description }}</span>
                    </div>
                  </td>
                  <td class="text-sm">{{ automation.trigger }}</td>
                  <td class="text-sm">{{ automation.action }}</td>
                  <td class="text-xs opacity-60">{{ automation.lastTriggered || "Never" }}</td>
                  <td>
                    <input
                      type="checkbox"
                      class="toggle toggle-primary toggle-sm"
                      :checked="automation.enabled"
                      @change="toggleAutomation(automation)"
                    />
                  </td>
                  <td>
                    <button class="btn btn-xs btn-ghost text-error" @click="deleteAutomation(automation.id)">
                      Delete
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </BaseCard>
      </section>

      <!-- Sync Controls -->
      <section>
        <Subtitle>Sync Controls</Subtitle>
        <BaseCard>
          <div class="p-4">
            <div class="flex flex-col gap-4 md:flex-row md:items-end">
              <!-- Manual sync -->
              <div class="flex gap-3 items-center">
                <button class="btn btn-sm btn-primary" :disabled="syncing || !status.connected" @click="syncNow">
                  {{ syncing ? "Syncing..." : "Sync Now" }}
                </button>
                <span v-if="syncing" class="badge badge-warning">Sync in progress</span>
              </div>

              <!-- Auto-sync toggle -->
              <div class="flex gap-3 items-end flex-1">
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Auto-Sync</span>
                  </label>
                  <div class="flex items-center gap-2">
                    <input
                      v-model="autoSync.enabled"
                      type="checkbox"
                      class="toggle toggle-primary toggle-sm"
                    />
                    <span class="text-sm">{{ autoSync.enabled ? "Enabled" : "Disabled" }}</span>
                  </div>
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Interval</span>
                  </label>
                  <select
                    v-model="autoSync.interval"
                    class="select select-bordered select-sm"
                    :disabled="!autoSync.enabled"
                  >
                    <option v-for="opt in intervalOptions" :key="opt.value" :value="opt.value">
                      {{ opt.label }}
                    </option>
                  </select>
                </div>
                <button class="btn btn-sm btn-outline" :disabled="loading" @click="saveConfig">
                  Save Sync Settings
                </button>
              </div>
            </div>

            <!-- Sync log -->
            <div v-if="syncLog.length > 0" class="mt-4">
              <p class="text-sm font-medium mb-2">Sync Log (last 50 entries)</p>
              <div class="overflow-y-auto max-h-48 space-y-1 rounded-lg border border-base-200 p-2">
                <div
                  v-for="(entry, i) in syncLog"
                  :key="i"
                  class="text-xs font-mono p-1.5 rounded"
                  :class="logLevelClass(entry.level)"
                >
                  <span class="opacity-50">{{ entry.timestamp }}</span>
                  <span class="ml-2 font-semibold uppercase">{{ entry.level }}</span>
                  <span class="ml-2">{{ entry.message }}</span>
                </div>
              </div>
            </div>
          </div>
        </BaseCard>
      </section>

      <!-- Discovery -->
      <section>
        <Subtitle>Entity Discovery</Subtitle>
        <BaseCard>
          <div class="p-4">
            <div class="flex items-center gap-3 mb-4">
              <button
                class="btn btn-sm btn-primary"
                :disabled="discovering || !status.connected"
                @click="discoverEntities"
              >
                {{ discovering ? "Discovering..." : "Discover HA Entities" }}
              </button>
              <span v-if="discoveredEntities.length > 0" class="text-sm opacity-70">
                {{ discoveredEntities.length }} entities found
              </span>
            </div>

            <div v-if="discoveredEntities.length === 0 && !discovering" class="text-center py-6 opacity-50">
              <p>Click "Discover HA Entities" to scan your Home Assistant instance.</p>
            </div>

            <div v-else-if="discoveredEntities.length > 0" class="grid grid-cols-1 gap-2 md:grid-cols-2 lg:grid-cols-3">
              <div
                v-for="entity in discoveredEntities"
                :key="entity.entityId"
                class="flex items-center justify-between p-3 rounded-lg border border-base-200 hover:border-primary/30 transition-colors"
              >
                <div class="min-w-0 flex-1">
                  <p class="text-sm font-medium truncate">{{ entity.name }}</p>
                  <p class="text-xs opacity-50 font-mono truncate">{{ entity.entityId }}</p>
                  <div class="flex gap-1 mt-1">
                    <span class="badge badge-xs badge-outline">{{ entity.domain }}</span>
                    <span class="badge badge-xs badge-ghost">{{ entity.state }}</span>
                  </div>
                </div>
                <button
                  class="btn btn-xs btn-outline btn-primary ml-2 flex-shrink-0"
                  :disabled="entityMappings.some(m => m.entityId === entity.entityId)"
                  @click="quickMapEntity(entity)"
                >
                  {{ entityMappings.some(m => m.entityId === entity.entityId) ? "Mapped" : "Map" }}
                </button>
              </div>
            </div>
          </div>
        </BaseCard>
      </section>
    </BaseContainer>
  </div>
</template>
