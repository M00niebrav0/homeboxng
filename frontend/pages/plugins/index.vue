<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import type { PluginInfo } from "~/lib/api/classes/plugins";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Plugins",
  });

  const api = useUserApi();

  const { data: plugins, refresh: refreshPlugins } = useAsyncData("plugins", async () => {
    const { data } = await api.plugins.getAll();
    return data;
  });

  const selectedPlugin = ref<PluginInfo | null>(null);
  const showConfigModal = ref(false);
  const pluginConfig = ref<any[]>([]);
  const pluginLogs = ref<any[]>([]);
  const showLogsModal = ref(false);
  const configValues = ref<Record<string, string>>({});

  async function openConfig(plugin: PluginInfo) {
    selectedPlugin.value = plugin;
    const { data } = await api.plugins.getConfig(plugin.name);
    pluginConfig.value = data || [];
    configValues.value = {};
    for (const field of pluginConfig.value) {
      configValues.value[field.key] = field.value || field.default || "";
    }
    showConfigModal.value = true;
  }

  async function saveConfig() {
    if (!selectedPlugin.value) return;
    await api.plugins.saveConfig(selectedPlugin.value.name, configValues.value);
    showConfigModal.value = false;
  }

  async function openLogs(plugin: PluginInfo) {
    selectedPlugin.value = plugin;
    const { data } = await api.plugins.getLogs(plugin.name);
    pluginLogs.value = data || [];
    showLogsModal.value = true;
  }

  async function togglePlugin(plugin: PluginInfo) {
    if (plugin.enabled) {
      await api.plugins.disable(plugin.name);
    } else {
      await api.plugins.enable(plugin.name);
    }
    refreshPlugins();
  }

  const pluginCategories = computed(() => {
    if (!plugins.value) return {};

    const cats: Record<string, PluginInfo[]> = {
      "AI & Vision": [],
      "Notifications": [],
      "Import & Export": [],
      "Integrations": [],
      "Inventory Management": [],
    };

    for (const p of plugins.value) {
      if (p.name === "ai-vision") cats["AI & Vision"].push(p);
      else if (p.name.includes("notify-") || p.name === "email" || p.name === "discord" || p.name === "webpush" || p.name === "gotify" || p.name === "ntfy")
        cats["Notifications"].push(p);
      else if (p.name === "excel-export" || p.name === "eyefi" || p.name === "paperless")
        cats["Import & Export"].push(p);
      else if (p.name === "ha-bridge" || p.name === "label-printer")
        cats["Integrations"].push(p);
      else cats["Inventory Management"].push(p);
    }

    // Remove empty categories
    for (const key of Object.keys(cats)) {
      if (cats[key].length === 0) delete cats[key];
    }

    return cats;
  });

  function pluginIcon(name: string): string {
    const icons: Record<string, string> = {
      "ai-vision": "mdi-eye",
      "label-printer": "mdi-printer",
      "eyefi": "mdi-camera-wireless",
      "excel-export": "mdi-file-excel",
      "paperless": "mdi-file-document",
      "analytics": "mdi-chart-bar",
      "ha-bridge": "mdi-home-assistant",
      "lending": "mdi-hand-extended",
      "maintenance": "mdi-wrench-clock",
      "shopping": "mdi-cart",
      "example": "mdi-code-braces",
    };
    return icons[name] || "mdi-puzzle";
  }

  function statusColor(plugin: PluginInfo): string {
    if (!plugin.enabled) return "badge-ghost";
    if (plugin.status === "running") return "badge-success";
    if (plugin.status === "error") return "badge-error";
    return "badge-info";
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Plugin Manager</h1>
          <p class="text-sm opacity-70">
            Manage built-in and third-party plugins. Enable, disable, and configure plugins to extend HomeBoxNG.
          </p>
        </div>
        <div class="flex gap-2">
          <button class="btn btn-sm btn-outline" @click="refreshPlugins">
            Refresh
          </button>
          <NuxtLink to="/plugins/catalog" class="btn btn-sm btn-primary">
            Browse Catalog
          </NuxtLink>
        </div>
      </div>

      <!-- Plugin Stats -->
      <div class="stats shadow w-full">
        <div class="stat">
          <div class="stat-title">Total Plugins</div>
          <div class="stat-value text-primary">{{ plugins?.length || 0 }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Built-in</div>
          <div class="stat-value">{{ plugins?.filter(p => p.builtIn).length || 0 }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Enabled</div>
          <div class="stat-value text-success">{{ plugins?.filter(p => p.enabled !== false).length || 0 }}</div>
        </div>
      </div>

      <!-- Plugins by Category -->
      <template v-for="(categoryPlugins, category) in pluginCategories" :key="category">
        <section>
          <Subtitle>{{ category }}</Subtitle>
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
            <BaseCard
              v-for="plugin in categoryPlugins"
              :key="plugin.name"
              class="relative"
            >
              <div class="flex items-start gap-3 p-4">
                <!-- Plugin Info -->
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-2">
                    <h3 class="font-semibold truncate">{{ plugin.name }}</h3>
                    <span class="badge badge-xs" :class="statusColor(plugin)">
                      {{ plugin.enabled !== false ? "enabled" : "disabled" }}
                    </span>
                    <span v-if="plugin.builtIn" class="badge badge-xs badge-outline">built-in</span>
                  </div>
                  <p class="text-xs opacity-60 mt-1">v{{ plugin.version }} by {{ plugin.author }}</p>
                  <p class="text-sm mt-2 line-clamp-2">{{ plugin.description }}</p>
                </div>
              </div>

              <!-- Actions -->
              <div class="flex gap-1 p-2 pt-0 justify-end">
                <button class="btn btn-xs btn-ghost" @click="openLogs(plugin)">Logs</button>
                <button class="btn btn-xs btn-ghost" @click="openConfig(plugin)">Config</button>
                <button
                  class="btn btn-xs"
                  :class="plugin.enabled !== false ? 'btn-warning' : 'btn-success'"
                  @click="togglePlugin(plugin)"
                >
                  {{ plugin.enabled !== false ? "Disable" : "Enable" }}
                </button>
              </div>
            </BaseCard>
          </div>
        </section>
      </template>

      <!-- Empty State -->
      <div v-if="!plugins || plugins.length === 0" class="text-center py-12 opacity-50">
        <p class="text-lg">No plugins found</p>
        <p class="text-sm mt-2">Check that the backend is running and plugins are registered.</p>
      </div>
    </BaseContainer>

    <!-- Config Modal -->
    <dialog class="modal" :class="{ 'modal-open': showConfigModal }">
      <div class="modal-box max-w-lg">
        <h3 class="font-bold text-lg">{{ selectedPlugin?.name }} - Configuration</h3>
        <div class="py-4 space-y-4">
          <div v-for="field in pluginConfig" :key="field.key" class="form-control">
            <label class="label">
              <span class="label-text font-medium">{{ field.label }}</span>
              <span v-if="field.required" class="label-text-alt text-error">Required</span>
            </label>
            <p class="text-xs opacity-60 mb-1">{{ field.description }}</p>

            <select
              v-if="field.type === 'select'"
              v-model="configValues[field.key]"
              class="select select-bordered select-sm w-full"
            >
              <option v-for="opt in field.options" :key="opt" :value="opt">{{ opt }}</option>
            </select>

            <input
              v-else-if="field.type === 'boolean'"
              v-model="configValues[field.key]"
              type="checkbox"
              class="toggle toggle-primary"
              true-value="true"
              false-value="false"
            />

            <input
              v-else
              v-model="configValues[field.key]"
              :type="field.type === 'secret' ? 'password' : field.type === 'number' ? 'number' : 'text'"
              :placeholder="field.default"
              class="input input-bordered input-sm w-full"
            />
          </div>
        </div>
        <div class="modal-action">
          <button class="btn btn-sm" @click="showConfigModal = false">Cancel</button>
          <button class="btn btn-sm btn-primary" @click="saveConfig">Save</button>
        </div>
      </div>
      <div class="modal-backdrop" @click="showConfigModal = false" />
    </dialog>

    <!-- Logs Modal -->
    <dialog class="modal" :class="{ 'modal-open': showLogsModal }">
      <div class="modal-box max-w-2xl">
        <h3 class="font-bold text-lg">{{ selectedPlugin?.name }} - Logs</h3>
        <div class="py-4">
          <div v-if="pluginLogs.length === 0" class="text-center opacity-50 py-8">
            No log entries
          </div>
          <div v-else class="overflow-y-auto max-h-96 space-y-1">
            <div
              v-for="(entry, i) in pluginLogs"
              :key="i"
              class="text-xs font-mono p-2 rounded"
              :class="{
                'bg-error/10 text-error': entry.level === 'error',
                'bg-warning/10 text-warning': entry.level === 'warn',
                'bg-base-200': entry.level === 'info',
                'opacity-60': entry.level === 'debug',
              }"
            >
              <span class="opacity-50">{{ entry.timestamp }}</span>
              <span class="ml-2 font-semibold uppercase">{{ entry.level }}</span>
              <span class="ml-2">{{ entry.message }}</span>
            </div>
          </div>
        </div>
        <div class="modal-action">
          <button class="btn btn-sm" @click="showLogsModal = false">Close</button>
        </div>
      </div>
      <div class="modal-backdrop" @click="showLogsModal = false" />
    </dialog>
  </div>
</template>
