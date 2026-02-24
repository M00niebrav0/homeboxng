<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import type { PluginConfig } from "~/lib/api/classes/plugins";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | AI Vision",
  });

  const api = useUserApi();

  // Plugin configuration
  const { data: pluginConfig } = useAsyncData("ai-vision-config", async () => {
    const { data } = await api.plugins.getConfig("ai-vision");
    return data || [];
  });

  const configValues = ref<Record<string, string>>({});

  watch(pluginConfig, (cfg) => {
    if (!cfg) return;
    for (const field of cfg) {
      configValues.value[field.key] = field.value || field.default || "";
    }
  }, { immediate: true });

  const showConfigModal = ref(false);

  async function saveConfig() {
    await api.plugins.saveConfig("ai-vision", configValues.value);
    showConfigModal.value = false;
  }

  // File upload state
  interface ScanResult {
    id: string;
    filename: string;
    thumbnailUrl: string;
    itemName: string;
    confidence: number;
    suggestedLocation: string;
    timestamp: string;
    status: "pending" | "complete" | "error";
  }

  const isDragging = ref(false);
  const uploading = ref(false);
  const scanResults = ref<ScanResult[]>([]);
  const recentScans = ref<ScanResult[]>([]);

  // Load recent scans
  const { data: recentScansData } = useAsyncData("ai-vision-recent", async () => {
    const { data } = await api.http.get<ScanResult[]>({ url: "/api/v1/plugins/ai-vision/scans" });
    return data;
  });

  watch(recentScansData, (data) => {
    if (data) {
      recentScans.value = data;
    }
  }, { immediate: true });

  function onDragOver(e: DragEvent) {
    e.preventDefault();
    isDragging.value = true;
  }

  function onDragLeave() {
    isDragging.value = false;
  }

  function onDrop(e: DragEvent) {
    e.preventDefault();
    isDragging.value = false;
    const files = e.dataTransfer?.files;
    if (files && files.length > 0) {
      handleFiles(files);
    }
  }

  function onFileSelect(e: Event) {
    const target = e.target as HTMLInputElement;
    if (target.files && target.files.length > 0) {
      handleFiles(target.files);
    }
  }

  async function handleFiles(files: FileList) {
    uploading.value = true;

    for (const file of Array.from(files)) {
      const formData = new FormData();
      formData.append("file", file);

      try {
        const response = await fetch("/api/v1/plugins/ai-vision/scan", {
          method: "POST",
          body: formData,
        });
        const result = await response.json();
        scanResults.value.push({
          id: result.id || crypto.randomUUID(),
          filename: file.name,
          thumbnailUrl: result.thumbnailUrl || URL.createObjectURL(file),
          itemName: result.itemName || "Unknown",
          confidence: result.confidence || 0,
          suggestedLocation: result.suggestedLocation || "Unassigned",
          timestamp: new Date().toISOString(),
          status: result.itemName ? "complete" : "error",
        });
      } catch {
        scanResults.value.push({
          id: crypto.randomUUID(),
          filename: file.name,
          thumbnailUrl: URL.createObjectURL(file),
          itemName: "Scan failed",
          confidence: 0,
          suggestedLocation: "-",
          timestamp: new Date().toISOString(),
          status: "error",
        });
      }
    }

    uploading.value = false;
  }

  async function matchItem(result: ScanResult) {
    await api.http.post<{ scanId: string; action: string }, void>({
      url: "/api/v1/plugins/ai-vision/scans/match",
      body: { scanId: result.id, action: "match" },
    });
    recentScans.value.unshift(result);
    scanResults.value = scanResults.value.filter(r => r.id !== result.id);
  }

  async function createItem(result: ScanResult) {
    await api.http.post<{ scanId: string; action: string }, void>({
      url: "/api/v1/plugins/ai-vision/scans/create",
      body: { scanId: result.id, action: "create" },
    });
    recentScans.value.unshift(result);
    scanResults.value = scanResults.value.filter(r => r.id !== result.id);
  }

  function confidenceColor(confidence: number): string {
    if (confidence >= 0.8) return "bg-success";
    if (confidence >= 0.5) return "bg-warning";
    return "bg-error";
  }

  function formatTimestamp(ts: string): string {
    return new Date(ts).toLocaleString();
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">AI Vision</h1>
          <p class="text-sm opacity-70">
            Identify inventory items from photos using AI image recognition.
          </p>
        </div>
        <div class="flex gap-2">
          <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
            Back to Plugins
          </NuxtLink>
          <button class="btn btn-sm btn-ghost" @click="showConfigModal = true">
            Configure
          </button>
        </div>
      </div>

      <!-- Upload Section -->
      <section>
        <Subtitle>Upload Photos</Subtitle>
        <div
          class="border-2 border-dashed rounded-lg p-12 text-center transition-colors cursor-pointer"
          :class="{
            'border-primary bg-primary/5': isDragging,
            'border-base-300 hover:border-primary/50': !isDragging,
          }"
          @dragover="onDragOver"
          @dragleave="onDragLeave"
          @drop="onDrop"
          @click="($refs.fileInput as HTMLInputElement)?.click()"
        >
          <input
            ref="fileInput"
            type="file"
            accept="image/*"
            multiple
            class="hidden"
            @change="onFileSelect"
          />
          <div v-if="uploading" class="flex flex-col items-center gap-2">
            <span class="loading loading-spinner loading-lg text-primary" />
            <p class="text-sm opacity-70">Analyzing images...</p>
          </div>
          <div v-else class="flex flex-col items-center gap-2">
            <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 opacity-30" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
            </svg>
            <p class="font-medium">Drop photos here or click to browse</p>
            <p class="text-xs opacity-50">Supports JPG, PNG, WebP. Multiple files allowed.</p>
          </div>
        </div>
      </section>

      <!-- Scan Results -->
      <section v-if="scanResults.length > 0">
        <Subtitle>Scan Results</Subtitle>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          <BaseCard v-for="result in scanResults" :key="result.id">
            <div class="p-4">
              <!-- Thumbnail -->
              <div class="aspect-square w-full overflow-hidden rounded-lg bg-base-200 mb-3">
                <img
                  :src="result.thumbnailUrl"
                  :alt="result.filename"
                  class="h-full w-full object-cover"
                />
              </div>

              <!-- Result Info -->
              <div class="space-y-2">
                <div class="flex items-center justify-between">
                  <h3 class="font-semibold truncate">{{ result.itemName }}</h3>
                  <span
                    class="badge badge-xs"
                    :class="{
                      'badge-success': result.status === 'complete',
                      'badge-error': result.status === 'error',
                      'badge-warning': result.status === 'pending',
                    }"
                  >
                    {{ result.status }}
                  </span>
                </div>

                <!-- Confidence Bar -->
                <div>
                  <div class="flex justify-between text-xs opacity-60 mb-1">
                    <span>Confidence</span>
                    <span>{{ (result.confidence * 100).toFixed(0) }}%</span>
                  </div>
                  <div class="w-full bg-base-200 rounded-full h-2">
                    <div
                      class="h-2 rounded-full transition-all"
                      :class="confidenceColor(result.confidence)"
                      :style="{ width: `${result.confidence * 100}%` }"
                    />
                  </div>
                </div>

                <p class="text-xs opacity-60">
                  Location: <span class="font-medium">{{ result.suggestedLocation }}</span>
                </p>

                <!-- Actions -->
                <div class="flex gap-2 pt-2">
                  <button
                    class="btn btn-xs btn-primary flex-1"
                    :disabled="result.status === 'error'"
                    @click="matchItem(result)"
                  >
                    Match Existing
                  </button>
                  <button
                    class="btn btn-xs btn-success flex-1"
                    :disabled="result.status === 'error'"
                    @click="createItem(result)"
                  >
                    Create New
                  </button>
                </div>
              </div>
            </div>
          </BaseCard>
        </div>
      </section>

      <!-- Recent Scans -->
      <section>
        <Subtitle>Recent Scans</Subtitle>
        <div v-if="recentScans.length === 0" class="text-center py-8 opacity-50">
          <p>No recent scans. Upload a photo to get started.</p>
        </div>
        <div v-else class="overflow-x-auto">
          <table class="table table-sm w-full">
            <thead>
              <tr>
                <th>Image</th>
                <th>Identified Item</th>
                <th>Confidence</th>
                <th>Location</th>
                <th>Timestamp</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="scan in recentScans" :key="scan.id">
                <td>
                  <div class="w-10 h-10 rounded overflow-hidden bg-base-200">
                    <img
                      :src="scan.thumbnailUrl"
                      :alt="scan.filename"
                      class="h-full w-full object-cover"
                    />
                  </div>
                </td>
                <td class="font-medium">{{ scan.itemName }}</td>
                <td>
                  <div class="flex items-center gap-2">
                    <div class="w-16 bg-base-200 rounded-full h-1.5">
                      <div
                        class="h-1.5 rounded-full"
                        :class="confidenceColor(scan.confidence)"
                        :style="{ width: `${scan.confidence * 100}%` }"
                      />
                    </div>
                    <span class="text-xs">{{ (scan.confidence * 100).toFixed(0) }}%</span>
                  </div>
                </td>
                <td>{{ scan.suggestedLocation }}</td>
                <td class="text-xs opacity-60">{{ formatTimestamp(scan.timestamp) }}</td>
                <td>
                  <span
                    class="badge badge-xs"
                    :class="{
                      'badge-success': scan.status === 'complete',
                      'badge-error': scan.status === 'error',
                    }"
                  >
                    {{ scan.status }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </BaseContainer>

    <!-- Configuration Modal -->
    <dialog class="modal" :class="{ 'modal-open': showConfigModal }">
      <div class="modal-box max-w-lg">
        <h3 class="font-bold text-lg">AI Vision - Configuration</h3>
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

          <div v-if="!pluginConfig || pluginConfig.length === 0" class="text-center py-4 opacity-50">
            <p>No configuration options available.</p>
          </div>
        </div>
        <div class="modal-action">
          <button class="btn btn-sm" @click="showConfigModal = false">Cancel</button>
          <button class="btn btn-sm btn-primary" @click="saveConfig">Save</button>
        </div>
      </div>
      <div class="modal-backdrop" @click="showConfigModal = false" />
    </dialog>
  </div>
</template>
