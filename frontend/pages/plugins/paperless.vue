<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import { route } from "~/lib/api/base";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Paperless-ngx Integration",
  });

  const api = useUserApi();

  // --- Types ---

  interface PaperlessConfig {
    url: string;
    apiToken: string;
    autoLink: boolean;
  }

  interface SyncStatus {
    lastSync: string;
    documentsSynced: number;
    itemsLinked: number;
    errors: number;
    running: boolean;
  }

  interface LinkedDocument {
    id: string;
    itemId: string;
    itemName: string;
    documentId: number;
    documentTitle: string;
    documentType: string;
    linkedAt: string;
  }

  interface AutoLinkRule {
    id: string;
    matchField: string;
    enabled: boolean;
    description: string;
  }

  interface SyncSchedule {
    enabled: boolean;
    interval: string;
  }

  // --- State ---

  const config = reactive<PaperlessConfig>({
    url: "",
    apiToken: "",
    autoLink: false,
  });

  const syncStatus = ref<SyncStatus>({
    lastSync: "Never",
    documentsSynced: 0,
    itemsLinked: 0,
    errors: 0,
    running: false,
  });

  const linkedDocuments = ref<LinkedDocument[]>([]);
  const autoLinkRules = ref<AutoLinkRule[]>([]);
  const syncSchedule = reactive<SyncSchedule>({
    enabled: false,
    interval: "daily",
  });

  const connectionStatus = ref<"idle" | "testing" | "success" | "error">("idle");
  const connectionMessage = ref("");
  const loading = ref(false);
  const syncing = ref(false);
  const quickStartDismissed = ref(false);

  // Manual link state
  const manualLink = reactive({
    itemSearch: "",
    documentSearch: "",
    selectedItemId: "",
    selectedDocumentId: "",
  });
  const itemSearchResults = ref<Array<{ id: string; name: string }>>([]);
  const documentSearchResults = ref<Array<{ id: number; title: string }>>([]);

  const intervalOptions = [
    { value: "hourly", label: "Every Hour" },
    { value: "daily", label: "Daily" },
    { value: "weekly", label: "Weekly" },
    { value: "manual", label: "Manual Only" },
  ];

  // --- Computed ---

  const isConfigured = computed(() => !!config.url && !!config.apiToken);
  const setupProgress = computed(() => {
    let steps = 0;
    if (config.url) steps++;
    if (config.apiToken) steps++;
    if (connectionStatus.value === "success") steps++;
    if (syncStatus.value.documentsSynced > 0) steps++;
    return steps;
  });

  // --- Data Loading ---

  useAsyncData("paperless-config", async () => {
    try {
      const { data } = await api.plugins.getConfig("paperless");
      if (data) {
        for (const field of data) {
          if (field.key === "paperless_url" || field.key === "url") config.url = field.value || field.default || "";
          if (field.key === "paperless_token" || field.key === "api_token") config.apiToken = field.value || "";
          if (field.key === "auto_link") config.autoLink = field.value === "true";
        }
      }
      return data;
    } catch {
      return null;
    }
  });

  useAsyncData("paperless-sync-status", async () => {
    try {
      const { data } = await api.http.get<SyncStatus>({ url: route("/plugins/paperless/sync/status") });
      if (data) syncStatus.value = data;
      return data;
    } catch {
      return null;
    }
  });

  useAsyncData("paperless-linked-docs", async () => {
    try {
      const { data } = await api.http.get<LinkedDocument[]>({ url: route("/plugins/paperless/links") });
      linkedDocuments.value = data || [];
      return data;
    } catch {
      return [];
    }
  });

  useAsyncData("paperless-autolink-rules", async () => {
    try {
      const { data } = await api.http.get<AutoLinkRule[]>({ url: route("/plugins/paperless/autolink-rules") });
      autoLinkRules.value = data || [];
      return data;
    } catch {
      return [];
    }
  });

  useAsyncData("paperless-schedule", async () => {
    try {
      const { data } = await api.http.get<SyncSchedule>({ url: route("/plugins/paperless/sync/schedule") });
      if (data) {
        syncSchedule.enabled = data.enabled;
        syncSchedule.interval = data.interval;
      }
      return data;
    } catch {
      return null;
    }
  });

  // --- Methods ---

  async function testConnection() {
    connectionStatus.value = "testing";
    connectionMessage.value = "";
    try {
      const { data, error } = await api.http.post<PaperlessConfig, { success: boolean; message: string }>({
        url: route("/plugins/paperless/test-connection"),
        body: { url: config.url, apiToken: config.apiToken },
      });
      if (error || !data?.success) {
        connectionStatus.value = "error";
        connectionMessage.value = data?.message || "Connection failed";
      } else {
        connectionStatus.value = "success";
        connectionMessage.value = data.message || "Connected successfully";
      }
    } catch {
      connectionStatus.value = "error";
      connectionMessage.value = "Failed to test connection";
    }
  }

  async function saveConfig() {
    loading.value = true;
    try {
      await api.plugins.saveConfig("paperless", {
        paperless_url: config.url,
        paperless_token: config.apiToken,
        auto_link: String(config.autoLink),
      });
    } finally {
      loading.value = false;
    }
  }

  async function syncNow() {
    syncing.value = true;
    syncStatus.value.running = true;
    try {
      await api.http.post<void, void>({ url: route("/plugins/paperless/sync") });
      const { data } = await api.http.get<SyncStatus>({ url: route("/plugins/paperless/sync/status") });
      if (data) syncStatus.value = data;
    } finally {
      syncing.value = false;
      syncStatus.value.running = false;
    }
  }

  async function saveSchedule() {
    loading.value = true;
    try {
      await api.http.post<SyncSchedule, void>({
        url: route("/plugins/paperless/sync/schedule"),
        body: { enabled: syncSchedule.enabled, interval: syncSchedule.interval },
      });
    } finally {
      loading.value = false;
    }
  }

  async function toggleAutoLinkRule(rule: AutoLinkRule) {
    loading.value = true;
    try {
      await api.http.post<{ enabled: boolean }, void>({
        url: route(`/plugins/paperless/autolink-rules/${rule.id}/toggle`),
        body: { enabled: !rule.enabled },
      });
      rule.enabled = !rule.enabled;
    } finally {
      loading.value = false;
    }
  }

  async function searchItems() {
    if (!manualLink.itemSearch.trim()) return;
    try {
      const { data } = await api.http.get<Array<{ id: string; name: string }>>({
        url: route("/plugins/paperless/search/items", { q: manualLink.itemSearch }),
      });
      itemSearchResults.value = data || [];
    } catch {
      itemSearchResults.value = [];
    }
  }

  async function searchDocuments() {
    if (!manualLink.documentSearch.trim()) return;
    try {
      const { data } = await api.http.get<Array<{ id: number; title: string }>>({
        url: route("/plugins/paperless/search/documents", { q: manualLink.documentSearch }),
      });
      documentSearchResults.value = data || [];
    } catch {
      documentSearchResults.value = [];
    }
  }

  async function createManualLink() {
    if (!manualLink.selectedItemId || !manualLink.selectedDocumentId) return;
    loading.value = true;
    try {
      await api.http.post<{ itemId: string; documentId: string }, void>({
        url: route("/plugins/paperless/links"),
        body: { itemId: manualLink.selectedItemId, documentId: manualLink.selectedDocumentId },
      });
      manualLink.itemSearch = "";
      manualLink.documentSearch = "";
      manualLink.selectedItemId = "";
      manualLink.selectedDocumentId = "";
      itemSearchResults.value = [];
      documentSearchResults.value = [];
      const { data } = await api.http.get<LinkedDocument[]>({ url: route("/plugins/paperless/links") });
      linkedDocuments.value = data || [];
    } finally {
      loading.value = false;
    }
  }

  async function removeLink(linkId: string) {
    loading.value = true;
    try {
      await api.http.delete<void>({ url: route(`/plugins/paperless/links/${linkId}`) });
      linkedDocuments.value = linkedDocuments.value.filter(l => l.id !== linkId);
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
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Paperless-ngx Integration</h1>
          <p class="text-sm opacity-70">
            Connect HomeBoxNG to Paperless-ngx for document-item linking and automatic synchronization.
          </p>
        </div>
        <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
          Back to Plugins
        </NuxtLink>
      </div>

      <!-- Status Overview Bar -->
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div class="rounded-lg border p-3">
          <p class="text-xs opacity-50">Connection</p>
          <div class="flex items-center gap-2 mt-1">
            <span
              class="w-2.5 h-2.5 rounded-full"
              :class="connectionStatus === 'success' ? 'bg-success' : connectionStatus === 'error' ? 'bg-error' : 'bg-base-300'"
            />
            <span class="text-sm font-semibold">{{ connectionBadgeText() }}</span>
          </div>
        </div>
        <div class="rounded-lg border p-3">
          <p class="text-xs opacity-50">Documents Synced</p>
          <p class="text-2xl font-bold text-primary mt-0.5">{{ syncStatus.documentsSynced }}</p>
        </div>
        <div class="rounded-lg border p-3">
          <p class="text-xs opacity-50">Items Linked</p>
          <p class="text-2xl font-bold text-success mt-0.5">{{ syncStatus.itemsLinked }}</p>
        </div>
        <div class="rounded-lg border p-3">
          <p class="text-xs opacity-50">Last Sync</p>
          <p class="text-sm font-semibold mt-1">{{ syncStatus.lastSync }}</p>
        </div>
      </div>

      <!-- Quick Start Guide (dismissible) -->
      <section v-if="!quickStartDismissed && !isConfigured">
        <div class="rounded-lg border-2 border-primary/20 bg-primary/5 p-5">
          <div class="flex items-start justify-between">
            <div>
              <h2 class="text-lg font-bold flex items-center gap-2">
                Quick Start Guide
                <span class="badge badge-sm badge-primary">{{ setupProgress }}/4 complete</span>
              </h2>
              <p class="text-sm opacity-70 mt-1">Link your Paperless-ngx instance to HomeBoxNG in a few steps.</p>
            </div>
            <button class="btn btn-ghost btn-xs" @click="quickStartDismissed = true">Dismiss</button>
          </div>
          <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <!-- Step 1 -->
            <div class="rounded-lg border bg-base-100 p-3" :class="config.url ? 'border-success/30' : ''">
              <div class="flex items-center gap-2 mb-2">
                <span
                  class="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                  :class="config.url ? 'bg-success text-success-content' : 'bg-base-200'"
                >1</span>
                <span class="font-medium text-sm">Enter Paperless URL</span>
              </div>
              <p class="text-xs opacity-60">
                Provide the base URL of your Paperless-ngx instance, for example
                <code class="bg-base-200 px-1 rounded text-[10px]">http://192.168.1.249:8000</code>.
              </p>
            </div>
            <!-- Step 2 -->
            <div class="rounded-lg border bg-base-100 p-3" :class="config.apiToken ? 'border-success/30' : ''">
              <div class="flex items-center gap-2 mb-2">
                <span
                  class="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                  :class="config.apiToken ? 'bg-success text-success-content' : 'bg-base-200'"
                >2</span>
                <span class="font-medium text-sm">Add API Token</span>
              </div>
              <p class="text-xs opacity-60">
                Generate an API token in Paperless at
                <code class="bg-base-200 px-1 rounded text-[10px]">Settings > API Tokens</code>,
                or use an existing admin token.
              </p>
            </div>
            <!-- Step 3 -->
            <div class="rounded-lg border bg-base-100 p-3" :class="connectionStatus === 'success' ? 'border-success/30' : ''">
              <div class="flex items-center gap-2 mb-2">
                <span
                  class="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                  :class="connectionStatus === 'success' ? 'bg-success text-success-content' : 'bg-base-200'"
                >3</span>
                <span class="font-medium text-sm">Test Connection</span>
              </div>
              <p class="text-xs opacity-60">
                Save settings and click "Test Connection" to verify HomeBoxNG can reach your Paperless instance.
              </p>
            </div>
            <!-- Step 4 -->
            <div class="rounded-lg border bg-base-100 p-3" :class="syncStatus.documentsSynced > 0 ? 'border-success/30' : ''">
              <div class="flex items-center gap-2 mb-2">
                <span
                  class="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                  :class="syncStatus.documentsSynced > 0 ? 'bg-success text-success-content' : 'bg-base-200'"
                >4</span>
                <span class="font-medium text-sm">Run First Sync</span>
              </div>
              <p class="text-xs opacity-60">
                Click "Sync Now" to pull documents from Paperless and begin linking them to your inventory items.
              </p>
            </div>
          </div>
        </div>
      </section>

      <!-- Connection Settings -->
      <section>
        <Subtitle>Connection Settings</Subtitle>
        <BaseCard>
          <div class="p-4">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
              <!-- Paperless URL -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">
                    Paperless-ngx URL
                    <span class="text-error ml-0.5">*</span>
                  </span>
                </label>
                <input
                  v-model="config.url"
                  type="url"
                  placeholder="http://192.168.1.249:8000"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label pb-0">
                  <span class="label-text-alt opacity-50">Full URL including port, no trailing slash</span>
                </label>
                <label class="label pt-0">
                  <span class="label-text-alt opacity-40 font-mono text-[11px]">
                    or set <code class="bg-base-200 px-1 py-0.5 rounded text-[10px]">HBOX_PAPERLESS_URL</code>
                  </span>
                </label>
              </div>

              <!-- API Token -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">
                    API Token
                    <span class="text-error ml-0.5">*</span>
                  </span>
                </label>
                <input
                  v-model="config.apiToken"
                  type="password"
                  placeholder="Enter Paperless API token"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label pb-0">
                  <span class="label-text-alt opacity-50">Found at Paperless Settings > API Tokens</span>
                </label>
                <label class="label pt-0">
                  <span class="label-text-alt opacity-40 font-mono text-[11px]">
                    or set <code class="bg-base-200 px-1 py-0.5 rounded text-[10px]">HBOX_PAPERLESS_TOKEN</code>
                  </span>
                </label>
              </div>
            </div>

            <!-- Auto-Link toggle -->
            <div class="form-control mt-4">
              <label class="label">
                <span class="label-text font-medium">Auto-Link Documents</span>
              </label>
              <div class="flex items-center gap-2">
                <input
                  v-model="config.autoLink"
                  type="checkbox"
                  class="toggle toggle-primary toggle-sm"
                />
                <span class="text-sm">{{ config.autoLink ? "Enabled" : "Disabled" }}</span>
              </div>
              <label class="label">
                <span class="label-text-alt opacity-50">Automatically link new receipts and invoices to matching inventory items</span>
              </label>
            </div>

            <div class="flex items-center gap-3 mt-4">
              <button class="btn btn-sm btn-primary" :disabled="loading" @click="saveConfig">
                Save Settings
              </button>
              <button
                class="btn btn-sm btn-outline"
                :disabled="connectionStatus === 'testing' || !config.url"
                @click="testConnection"
              >
                {{ connectionStatus === "testing" ? "Testing..." : "Test Connection" }}
              </button>
              <span class="badge" :class="connectionBadgeClass()">
                {{ connectionBadgeText() }}
              </span>
              <span v-if="connectionMessage" class="text-sm opacity-70">{{ connectionMessage }}</span>
            </div>
          </div>
        </BaseCard>
      </section>

      <!-- Sync Status -->
      <section>
        <Subtitle>Sync Status</Subtitle>
        <div class="stats shadow w-full">
          <div class="stat">
            <div class="stat-title">Last Sync</div>
            <div class="stat-value text-sm">{{ syncStatus.lastSync }}</div>
          </div>
          <div class="stat">
            <div class="stat-title">Documents Synced</div>
            <div class="stat-value text-primary">{{ syncStatus.documentsSynced }}</div>
          </div>
          <div class="stat">
            <div class="stat-title">Items Linked</div>
            <div class="stat-value text-success">{{ syncStatus.itemsLinked }}</div>
          </div>
          <div class="stat">
            <div class="stat-title">Errors</div>
            <div class="stat-value" :class="syncStatus.errors > 0 ? 'text-error' : ''">{{ syncStatus.errors }}</div>
          </div>
        </div>
      </section>

      <!-- Sync Actions -->
      <section>
        <Subtitle>Sync Actions</Subtitle>
        <BaseCard>
          <div class="p-4 flex flex-col gap-4 md:flex-row md:items-end">
            <div class="flex gap-3 items-center">
              <button class="btn btn-sm btn-primary" :disabled="syncing || !isConfigured" @click="syncNow">
                {{ syncing ? "Syncing..." : "Sync Now" }}
              </button>
              <span v-if="syncStatus.running" class="badge badge-warning">Sync in progress</span>
            </div>
            <div class="flex gap-3 items-end flex-1">
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Scheduled Sync</span>
                </label>
                <div class="flex items-center gap-2">
                  <input
                    v-model="syncSchedule.enabled"
                    type="checkbox"
                    class="toggle toggle-primary toggle-sm"
                  />
                  <span class="text-sm">{{ syncSchedule.enabled ? "Enabled" : "Disabled" }}</span>
                </div>
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Interval</span>
                </label>
                <select
                  v-model="syncSchedule.interval"
                  class="select select-bordered select-sm"
                  :disabled="!syncSchedule.enabled"
                >
                  <option v-for="opt in intervalOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                </select>
              </div>
              <button class="btn btn-sm btn-outline" :disabled="loading" @click="saveSchedule">
                Save Schedule
              </button>
            </div>
          </div>
        </BaseCard>
      </section>

      <!-- Auto-Link Rules -->
      <section>
        <Subtitle>Auto-Link Rules</Subtitle>
        <BaseCard>
          <div v-if="autoLinkRules.length === 0" class="p-6 text-center opacity-50">
            <p>No auto-link rules configured.</p>
          </div>
          <div v-else class="overflow-x-auto">
            <table class="table table-sm w-full">
              <thead>
                <tr>
                  <th>Match Field</th>
                  <th>Description</th>
                  <th class="w-24">Status</th>
                  <th class="w-20">Toggle</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="rule in autoLinkRules" :key="rule.id">
                  <td class="font-medium">{{ rule.matchField }}</td>
                  <td class="text-sm opacity-70">{{ rule.description }}</td>
                  <td>
                    <span class="badge badge-sm" :class="rule.enabled ? 'badge-success' : 'badge-ghost'">
                      {{ rule.enabled ? "Active" : "Inactive" }}
                    </span>
                  </td>
                  <td>
                    <input
                      type="checkbox"
                      class="toggle toggle-primary toggle-sm"
                      :checked="rule.enabled"
                      @change="toggleAutoLinkRule(rule)"
                    />
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </BaseCard>
      </section>

      <!-- Manual Link -->
      <section>
        <Subtitle>Manual Link</Subtitle>
        <BaseCard>
          <div class="p-4">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
              <!-- Item Search -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Search for an Item</span>
                </label>
                <div class="flex gap-2">
                  <input
                    v-model="manualLink.itemSearch"
                    type="text"
                    placeholder="Search items..."
                    class="input input-bordered input-sm flex-1"
                    @keyup.enter="searchItems"
                  />
                  <button class="btn btn-sm btn-outline" @click="searchItems">Search</button>
                </div>
                <select
                  v-if="itemSearchResults.length > 0"
                  v-model="manualLink.selectedItemId"
                  class="select select-bordered select-sm w-full mt-2"
                >
                  <option value="">Select an item</option>
                  <option v-for="item in itemSearchResults" :key="item.id" :value="item.id">
                    {{ item.name }}
                  </option>
                </select>
              </div>

              <!-- Document Search -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Search for a Document</span>
                </label>
                <div class="flex gap-2">
                  <input
                    v-model="manualLink.documentSearch"
                    type="text"
                    placeholder="Search documents..."
                    class="input input-bordered input-sm flex-1"
                    @keyup.enter="searchDocuments"
                  />
                  <button class="btn btn-sm btn-outline" @click="searchDocuments">Search</button>
                </div>
                <select
                  v-if="documentSearchResults.length > 0"
                  v-model="manualLink.selectedDocumentId"
                  class="select select-bordered select-sm w-full mt-2"
                >
                  <option value="">Select a document</option>
                  <option v-for="doc in documentSearchResults" :key="doc.id" :value="String(doc.id)">
                    {{ doc.title }}
                  </option>
                </select>
              </div>
            </div>
            <div class="flex justify-end mt-4">
              <button
                class="btn btn-sm btn-primary"
                :disabled="loading || !manualLink.selectedItemId || !manualLink.selectedDocumentId"
                @click="createManualLink"
              >
                Create Link
              </button>
            </div>
          </div>
        </BaseCard>
      </section>

      <!-- Linked Documents -->
      <section>
        <Subtitle>Linked Documents</Subtitle>
        <BaseCard>
          <div v-if="linkedDocuments.length === 0" class="p-6 text-center opacity-50">
            <p>No documents linked to items yet. Use the manual link above or configure auto-link rules.</p>
          </div>
          <div v-else class="overflow-x-auto">
            <table class="table table-sm w-full">
              <thead>
                <tr>
                  <th>Item</th>
                  <th>Document Title</th>
                  <th>Document Type</th>
                  <th>Linked</th>
                  <th class="w-16">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="link in linkedDocuments" :key="link.id">
                  <td class="font-medium">{{ link.itemName }}</td>
                  <td>{{ link.documentTitle }}</td>
                  <td>
                    <span class="badge badge-sm badge-outline">{{ link.documentType }}</span>
                  </td>
                  <td class="text-sm opacity-70">{{ link.linkedAt }}</td>
                  <td>
                    <button class="btn btn-xs btn-ghost text-error" @click="removeLink(link.id)">
                      Unlink
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </BaseCard>
      </section>

      <!-- Environment Variables Reference -->
      <section>
        <Subtitle>Environment Variables</Subtitle>
        <BaseCard>
          <div class="p-4">
            <p class="text-sm opacity-70 mb-3">
              All Paperless connection settings can be configured via environment variables. These take priority over default values
              but are overridden by values set through this UI.
            </p>
            <div class="overflow-x-auto">
              <table class="table table-sm w-full">
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                    <th>Required</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td class="font-mono text-xs"><code class="bg-base-200 px-1.5 py-0.5 rounded">HBOX_PAPERLESS_URL</code></td>
                    <td class="text-sm">Base URL of your Paperless-ngx instance</td>
                    <td><span class="badge badge-xs badge-error">Required</span></td>
                  </tr>
                  <tr>
                    <td class="font-mono text-xs"><code class="bg-base-200 px-1.5 py-0.5 rounded">HBOX_PAPERLESS_TOKEN</code></td>
                    <td class="text-sm">API authentication token for Paperless-ngx</td>
                    <td><span class="badge badge-xs badge-error">Required</span></td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </BaseCard>
      </section>
    </BaseContainer>
  </div>
</template>
