<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import type { PluginConfig } from "~/lib/api/classes/plugins";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Label Printer",
  });

  const api = useUserApi();

  // Printer configuration
  const { data: printerConfig, refresh: refreshConfig } = useAsyncData("label-printer-config", async () => {
    const { data } = await api.plugins.getConfig("label-printer");
    return data || [];
  });

  const configValues = ref<Record<string, string>>({});

  watch(printerConfig, (cfg) => {
    if (!cfg) return;
    for (const field of cfg) {
      configValues.value[field.key] = field.value || field.default || "";
    }
  }, { immediate: true });

  const savingConfig = ref(false);

  async function saveConfig() {
    savingConfig.value = true;
    try {
      await api.plugins.saveConfig("label-printer", configValues.value);
      refreshConfig();
    } finally {
      savingConfig.value = false;
    }
  }

  // Label size options
  const labelSizes = ["62mm", "29mm", "12mm"];
  const orientations = ["horizontal", "vertical"];

  const selectedLabelSize = ref("62mm");
  const selectedOrientation = ref("horizontal");
  const halfLabelMode = ref(false);

  // Quick print search
  const searchQuery = ref("");
  const searchResults = ref<Array<{ id: string; name: string; location: string }>>([]);
  const selectedItem = ref<{ id: string; name: string; location: string } | null>(null);
  const searching = ref(false);
  const printing = ref(false);

  async function searchItems() {
    if (!searchQuery.value || searchQuery.value.length < 2) {
      searchResults.value = [];
      return;
    }

    searching.value = true;
    try {
      const { data } = await api.items.getAll({
        q: searchQuery.value,
        page: 1,
        pageSize: 10,
      });
      searchResults.value = (data?.items || []).map((item: any) => ({
        id: item.id,
        name: item.name,
        location: item.location?.name || "No location",
      }));
    } finally {
      searching.value = false;
    }
  }

  function selectItem(item: { id: string; name: string; location: string }) {
    selectedItem.value = item;
    searchQuery.value = item.name;
    searchResults.value = [];
  }

  async function printLabel() {
    if (!selectedItem.value) return;
    printing.value = true;
    try {
      await api.http.post<{
        itemId: string;
        labelSize: string;
        orientation: string;
        halfLabel: boolean;
      }, void>({
        url: "/api/v1/plugins/label-printer/print",
        body: {
          itemId: selectedItem.value.id,
          labelSize: selectedLabelSize.value,
          orientation: selectedOrientation.value,
          halfLabel: halfLabelMode.value,
        },
      });
      refreshHistory();
    } finally {
      printing.value = false;
    }
  }

  // Print history
  interface PrintJob {
    id: string;
    itemName: string;
    labelSize: string;
    timestamp: string;
    status: "success" | "error" | "pending";
  }

  const { data: printHistory, refresh: refreshHistory } = useAsyncData("label-printer-history", async () => {
    const { data } = await api.http.get<PrintJob[]>({
      url: "/api/v1/plugins/label-printer/history",
    });
    return data || [];
  });

  // Preview data
  const previewItem = computed(() => {
    return selectedItem.value || { name: "Item Name", location: "Location / Shelf" };
  });

  const previewDate = computed(() => {
    return new Date().toLocaleDateString();
  });

  function formatTimestamp(ts: string): string {
    return new Date(ts).toLocaleString();
  }

  function statusBadge(status: string): string {
    if (status === "success") return "badge-success";
    if (status === "error") return "badge-error";
    return "badge-warning";
  }

  // Debounce search
  let searchTimeout: ReturnType<typeof setTimeout>;
  watch(searchQuery, () => {
    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(searchItems, 300);
  });
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Label Printer</h1>
          <p class="text-sm opacity-70">
            Configure your label printer and print inventory labels with QR codes.
          </p>
        </div>
        <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
          Back to Plugins
        </NuxtLink>
      </div>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <!-- Left Column: Config + Quick Print -->
        <div class="flex flex-col gap-6">
          <!-- Printer Configuration -->
          <section>
            <Subtitle>Printer Configuration</Subtitle>
            <BaseCard>
              <div class="p-4 space-y-4">
                <div v-for="field in printerConfig" :key="field.key" class="form-control">
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
                    v-else
                    v-model="configValues[field.key]"
                    :type="field.type === 'secret' ? 'password' : field.type === 'number' ? 'number' : 'text'"
                    :placeholder="field.default"
                    class="input input-bordered input-sm w-full"
                  />
                </div>

                <!-- Label Size -->
                <div class="form-control">
                  <label class="label">
                    <span class="label-text font-medium">Label Size</span>
                  </label>
                  <div class="flex gap-2">
                    <button
                      v-for="size in labelSizes"
                      :key="size"
                      class="btn btn-sm"
                      :class="selectedLabelSize === size ? 'btn-primary' : 'btn-ghost'"
                      @click="selectedLabelSize = size"
                    >
                      {{ size }}
                    </button>
                  </div>
                </div>

                <!-- Orientation -->
                <div class="form-control">
                  <label class="label">
                    <span class="label-text font-medium">Orientation</span>
                  </label>
                  <div class="flex gap-2">
                    <button
                      v-for="orient in orientations"
                      :key="orient"
                      class="btn btn-sm"
                      :class="selectedOrientation === orient ? 'btn-primary' : 'btn-ghost'"
                      @click="selectedOrientation = orient"
                    >
                      {{ orient }}
                    </button>
                  </div>
                </div>

                <!-- Half-Label Mode -->
                <div class="form-control">
                  <label class="label cursor-pointer">
                    <span class="label-text font-medium">Half-Label Mode</span>
                    <input
                      v-model="halfLabelMode"
                      type="checkbox"
                      class="toggle toggle-primary"
                    />
                  </label>
                  <p class="text-xs opacity-60">Print 2 labels per page in a split layout.</p>
                </div>

                <button
                  class="btn btn-sm btn-primary w-full"
                  :disabled="savingConfig"
                  @click="saveConfig"
                >
                  {{ savingConfig ? "Saving..." : "Save Configuration" }}
                </button>
              </div>
            </BaseCard>
          </section>

          <!-- Quick Print -->
          <section>
            <Subtitle>Quick Print</Subtitle>
            <BaseCard>
              <div class="p-4 space-y-4">
                <!-- Item Search -->
                <div class="form-control">
                  <label class="label">
                    <span class="label-text font-medium">Search Item</span>
                  </label>
                  <div class="relative">
                    <input
                      v-model="searchQuery"
                      type="text"
                      placeholder="Type to search items..."
                      class="input input-bordered input-sm w-full"
                    />
                    <span v-if="searching" class="absolute right-3 top-2">
                      <span class="loading loading-spinner loading-xs" />
                    </span>
                  </div>

                  <!-- Search Dropdown -->
                  <div
                    v-if="searchResults.length > 0"
                    class="mt-1 border border-base-300 rounded-lg bg-base-100 shadow-lg max-h-48 overflow-y-auto"
                  >
                    <button
                      v-for="item in searchResults"
                      :key="item.id"
                      class="w-full text-left px-3 py-2 hover:bg-base-200 transition-colors"
                      @click="selectItem(item)"
                    >
                      <p class="text-sm font-medium">{{ item.name }}</p>
                      <p class="text-xs opacity-50">{{ item.location }}</p>
                    </button>
                  </div>
                </div>

                <!-- Selected Item -->
                <div v-if="selectedItem" class="bg-base-200 rounded-lg p-3">
                  <p class="text-sm font-medium">{{ selectedItem.name }}</p>
                  <p class="text-xs opacity-60">{{ selectedItem.location }}</p>
                </div>

                <button
                  class="btn btn-primary btn-sm w-full"
                  :disabled="!selectedItem || printing"
                  @click="printLabel"
                >
                  {{ printing ? "Printing..." : "Print Label" }}
                </button>
              </div>
            </BaseCard>
          </section>
        </div>

        <!-- Right Column: Preview -->
        <div class="flex flex-col gap-6">
          <!-- Print Preview -->
          <section>
            <Subtitle>Label Preview</Subtitle>
            <BaseCard>
              <div class="p-4">
                <!-- Single Label Preview -->
                <div
                  v-if="!halfLabelMode"
                  class="border-2 border-dashed border-base-300 rounded-lg p-4 bg-white text-black"
                  :class="{
                    'aspect-[3/1]': selectedOrientation === 'horizontal',
                    'aspect-[1/2]': selectedOrientation === 'vertical',
                  }"
                >
                  <div
                    class="h-full flex gap-3"
                    :class="{
                      'flex-row items-center': selectedOrientation === 'horizontal',
                      'flex-col items-center justify-center': selectedOrientation === 'vertical',
                    }"
                  >
                    <!-- QR Code Placeholder -->
                    <div class="flex-shrink-0 w-16 h-16 border border-gray-300 rounded flex items-center justify-center bg-gray-50">
                      <svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
                      </svg>
                    </div>
                    <!-- Label Text -->
                    <div class="flex-1 min-w-0">
                      <p class="font-bold text-sm truncate">{{ previewItem.name }}</p>
                      <p class="text-xs text-gray-500 truncate">{{ previewItem.location }}</p>
                      <p class="text-xs text-gray-400 mt-1">{{ previewDate }}</p>
                    </div>
                  </div>
                </div>

                <!-- Half-Label Preview -->
                <div v-else class="flex gap-2">
                  <div
                    v-for="i in 2"
                    :key="i"
                    class="flex-1 border-2 border-dashed border-base-300 rounded-lg p-3 bg-white text-black"
                  >
                    <div class="flex flex-col items-center gap-2">
                      <div class="w-12 h-12 border border-gray-300 rounded flex items-center justify-center bg-gray-50">
                        <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
                        </svg>
                      </div>
                      <div class="text-center">
                        <p class="font-bold text-xs truncate">{{ previewItem.name }}</p>
                        <p class="text-xs text-gray-500 truncate">{{ previewItem.location }}</p>
                      </div>
                    </div>
                  </div>
                </div>

                <div class="mt-3 flex items-center justify-between text-xs opacity-50">
                  <span>{{ selectedLabelSize }} - {{ selectedOrientation }}</span>
                  <span v-if="halfLabelMode">Half-label mode (2 per page)</span>
                </div>
              </div>
            </BaseCard>
          </section>

          <!-- Print History -->
          <section>
            <Subtitle>Print History</Subtitle>
            <div v-if="!printHistory || printHistory.length === 0" class="text-center py-8 opacity-50">
              <p>No print jobs yet.</p>
            </div>
            <div v-else class="overflow-x-auto">
              <table class="table table-sm w-full">
                <thead>
                  <tr>
                    <th>Item</th>
                    <th>Label Size</th>
                    <th>Time</th>
                    <th>Status</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="job in printHistory" :key="job.id">
                    <td class="font-medium">{{ job.itemName }}</td>
                    <td>{{ job.labelSize }}</td>
                    <td class="text-xs opacity-60">{{ formatTimestamp(job.timestamp) }}</td>
                    <td>
                      <span class="badge badge-xs" :class="statusBadge(job.status)">
                        {{ job.status }}
                      </span>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
        </div>
      </div>
    </BaseContainer>
  </div>
</template>
