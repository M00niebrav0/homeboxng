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

  // --- Data Loading ---

  useAsyncData("paperless-config", async () => {
    try {
      const { data } = await api.plugins.getConfig("paperless");
      if (data) {
        for (const field of data) {
          if (field.key === "url") config.url = field.value || field.default || "";
          if (field.key === "api_token") config.apiToken = field.value || "";
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
        url: config.url,
        api_token: config.apiToken,
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

      <!-- Connection Settings -->
      <section>
        <Subtitle>Connection Settings</Subtitle>
        <BaseCard>
          <div class="p-4">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Paperless URL</span>
                </label>
                <input
                  v-model="config.url"
                  type="url"
                  placeholder="http://192.168.1.249:8000"
                  class="input input-bordered input-sm w-full"
                />
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text">API Token</span>
                </label>
                <input
                  v-model="config.apiToken"
                  type="password"
                  placeholder="Enter Paperless API token"
                  class="input input-bordered input-sm w-full"
                />
              </div>
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
              <button class="btn btn-sm btn-primary" :disabled="syncing" @click="syncNow">
                {{ syncing ? "Syncing..." : "Sync Now" }}
              </button>
              <span v-if="syncStatus.running" class="badge badge-warning">Sync in progress</span>
            </div>
            <div class="flex gap-3 items-end flex-1">
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Scheduled Sync</span>
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
                  <span class="label-text">Interval</span>
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
                  <span class="label-text">Search for an Item</span>
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
                  <span class="label-text">Search for a Document</span>
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
    </BaseContainer>
  </div>
</template>
