<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import { route } from "~/lib/api/base";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Eye-Fi Camera Integration",
  });

  const api = useUserApi();

  // --- Types ---

  interface CameraStatus {
    connected: boolean;
    lastUpload: string;
    photosReceived: number;
    battery: string | null;
  }

  interface SoapServerConfig {
    port: number;
    uploadKey: string;
    uploadDirectory: string;
  }

  interface FileWatcherConfig {
    watchDirectory: string;
    autoProcess: boolean;
    extensions: string;
  }

  interface UploadEntry {
    id: string;
    filename: string;
    timestamp: string;
    size: string;
    status: "pending" | "processing" | "completed" | "error";
  }

  interface PipelineConfig {
    enabled: boolean;
    model: string;
  }

  // --- State ---

  const cameraStatus = ref<CameraStatus>({
    connected: false,
    lastUpload: "Never",
    photosReceived: 0,
    battery: null,
  });

  const soapConfig = reactive<SoapServerConfig>({
    port: 59278,
    uploadKey: "",
    uploadDirectory: "",
  });

  const watcherConfig = reactive<FileWatcherConfig>({
    watchDirectory: "",
    autoProcess: false,
    extensions: ".jpg,.jpeg,.png,.raw,.cr2,.nef",
  });

  const pipelineConfig = reactive<PipelineConfig>({
    enabled: false,
    model: "minicpm-v",
  });

  const uploadHistory = ref<UploadEntry[]>([]);
  const loading = ref(false);
  const serverRunning = ref(false);
  const testResult = ref<"idle" | "testing" | "success" | "error">("idle");
  const testMessage = ref("");

  // --- Data Loading ---

  useAsyncData("eyefi-status", async () => {
    try {
      const { data } = await api.http.get<CameraStatus>({ url: route("/plugins/eyefi/status") });
      if (data) cameraStatus.value = data;
      return data;
    } catch {
      return null;
    }
  });

  useAsyncData("eyefi-config", async () => {
    try {
      const { data } = await api.plugins.getConfig("eyefi");
      if (data) {
        for (const field of data) {
          if (field.key === "soap_port") soapConfig.port = parseInt(field.value || field.default || "59278");
          if (field.key === "upload_key") soapConfig.uploadKey = field.value || "";
          if (field.key === "upload_directory") soapConfig.uploadDirectory = field.value || "";
          if (field.key === "watch_directory") watcherConfig.watchDirectory = field.value || "";
          if (field.key === "auto_process") watcherConfig.autoProcess = field.value === "true";
          if (field.key === "file_extensions") watcherConfig.extensions = field.value || ".jpg,.jpeg,.png,.raw,.cr2,.nef";
          if (field.key === "pipeline_enabled") pipelineConfig.enabled = field.value === "true";
          if (field.key === "pipeline_model") pipelineConfig.model = field.value || "minicpm-v";
        }
      }
      return data;
    } catch {
      return null;
    }
  });

  useAsyncData("eyefi-uploads", async () => {
    try {
      const { data } = await api.http.get<UploadEntry[]>({ url: route("/plugins/eyefi/uploads") });
      uploadHistory.value = data || [];
      return data;
    } catch {
      return [];
    }
  });

  useAsyncData("eyefi-server-status", async () => {
    try {
      const { data } = await api.http.get<{ running: boolean }>({ url: route("/plugins/eyefi/server/status") });
      if (data) serverRunning.value = data.running;
      return data;
    } catch {
      return null;
    }
  });

  // --- Methods ---

  async function saveSoapConfig() {
    loading.value = true;
    try {
      await api.plugins.saveConfig("eyefi", {
        soap_port: String(soapConfig.port),
        upload_key: soapConfig.uploadKey,
        upload_directory: soapConfig.uploadDirectory,
      });
    } finally {
      loading.value = false;
    }
  }

  async function saveWatcherConfig() {
    loading.value = true;
    try {
      await api.plugins.saveConfig("eyefi", {
        watch_directory: watcherConfig.watchDirectory,
        auto_process: String(watcherConfig.autoProcess),
        file_extensions: watcherConfig.extensions,
      });
    } finally {
      loading.value = false;
    }
  }

  async function savePipelineConfig() {
    loading.value = true;
    try {
      await api.plugins.saveConfig("eyefi", {
        pipeline_enabled: String(pipelineConfig.enabled),
        pipeline_model: pipelineConfig.model,
      });
    } finally {
      loading.value = false;
    }
  }

  async function testConnection() {
    testResult.value = "testing";
    testMessage.value = "";
    try {
      const { data, error } = await api.http.post<void, { success: boolean; message: string }>({
        url: route("/plugins/eyefi/test-connection"),
      });
      if (error || !data?.success) {
        testResult.value = "error";
        testMessage.value = data?.message || "SOAP server is not responding";
      } else {
        testResult.value = "success";
        testMessage.value = data.message || "SOAP server is running";
      }
    } catch {
      testResult.value = "error";
      testMessage.value = "Failed to reach SOAP server";
    }
  }

  function statusBadgeClass(status: string): string {
    if (status === "completed") return "badge-success";
    if (status === "processing") return "badge-warning";
    if (status === "error") return "badge-error";
    return "badge-ghost";
  }

  function formatFileSize(size: string): string {
    return size;
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Eye-Fi Camera Integration</h1>
          <p class="text-sm opacity-70">
            Configure Eye-Fi X2 wireless camera uploads and automatic photo processing.
          </p>
        </div>
        <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
          Back to Plugins
        </NuxtLink>
      </div>

      <!-- Camera Status -->
      <section>
        <Subtitle>Camera Status</Subtitle>
        <div class="stats shadow w-full">
          <div class="stat">
            <div class="stat-title">Connection</div>
            <div class="stat-value text-sm">
              <span class="badge" :class="cameraStatus.connected ? 'badge-success' : 'badge-ghost'">
                {{ cameraStatus.connected ? "Connected" : "Disconnected" }}
              </span>
            </div>
          </div>
          <div class="stat">
            <div class="stat-title">Last Upload</div>
            <div class="stat-value text-sm">{{ cameraStatus.lastUpload }}</div>
          </div>
          <div class="stat">
            <div class="stat-title">Photos Received</div>
            <div class="stat-value text-primary">{{ cameraStatus.photosReceived }}</div>
          </div>
          <div class="stat">
            <div class="stat-title">Battery</div>
            <div class="stat-value text-sm">{{ cameraStatus.battery || "N/A" }}</div>
          </div>
        </div>
      </section>

      <!-- SOAP Server Settings -->
      <section>
        <Subtitle>SOAP Server Settings</Subtitle>
        <BaseCard>
          <div class="p-4">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Port Number</span>
                </label>
                <input
                  v-model.number="soapConfig.port"
                  type="number"
                  min="1024"
                  max="65535"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label">
                  <span class="label-text-alt opacity-50">Default: 59278</span>
                </label>
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Upload Key</span>
                </label>
                <input
                  v-model="soapConfig.uploadKey"
                  type="password"
                  placeholder="Eye-Fi card upload key"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label">
                  <span class="label-text-alt opacity-50">Found on your Eye-Fi card</span>
                </label>
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Upload Directory</span>
                </label>
                <input
                  v-model="soapConfig.uploadDirectory"
                  type="text"
                  placeholder="/data/eyefi/uploads"
                  class="input input-bordered input-sm w-full"
                />
              </div>
            </div>
            <div class="flex items-center gap-3 mt-4">
              <button class="btn btn-sm btn-primary" :disabled="loading" @click="saveSoapConfig">
                Save Settings
              </button>
              <div class="flex items-center gap-2">
                <span class="badge badge-sm" :class="serverRunning ? 'badge-success' : 'badge-ghost'">
                  SOAP Server: {{ serverRunning ? "Running" : "Stopped" }}
                </span>
              </div>
            </div>
          </div>
        </BaseCard>
      </section>

      <!-- File Watcher Settings -->
      <section>
        <Subtitle>File Watcher Settings</Subtitle>
        <BaseCard>
          <div class="p-4">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Watch Directory</span>
                </label>
                <input
                  v-model="watcherConfig.watchDirectory"
                  type="text"
                  placeholder="/data/eyefi/watch"
                  class="input input-bordered input-sm w-full"
                />
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Auto-Process</span>
                </label>
                <div class="flex items-center gap-2 mt-1">
                  <input
                    v-model="watcherConfig.autoProcess"
                    type="checkbox"
                    class="toggle toggle-primary toggle-sm"
                  />
                  <span class="text-sm">{{ watcherConfig.autoProcess ? "Enabled" : "Disabled" }}</span>
                </div>
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text">File Extensions</span>
                </label>
                <input
                  v-model="watcherConfig.extensions"
                  type="text"
                  placeholder=".jpg,.jpeg,.png,.raw"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label">
                  <span class="label-text-alt opacity-50">Comma-separated</span>
                </label>
              </div>
            </div>
            <div class="flex justify-start mt-4">
              <button class="btn btn-sm btn-primary" :disabled="loading" @click="saveWatcherConfig">
                Save Watcher Settings
              </button>
            </div>
          </div>
        </BaseCard>
      </section>

      <!-- Auto-Process Pipeline -->
      <section>
        <Subtitle>Auto-Process Pipeline</Subtitle>
        <BaseCard>
          <div class="p-4">
            <p class="text-sm opacity-70 mb-4">
              When enabled, uploaded photos are automatically sent through AI Vision for item identification and tagging.
            </p>
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div class="form-control">
                <label class="label">
                  <span class="label-text">AI Vision Processing</span>
                </label>
                <div class="flex items-center gap-2">
                  <input
                    v-model="pipelineConfig.enabled"
                    type="checkbox"
                    class="toggle toggle-primary toggle-sm"
                  />
                  <span class="text-sm">{{ pipelineConfig.enabled ? "Enabled" : "Disabled" }}</span>
                </div>
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Vision Model</span>
                </label>
                <select
                  v-model="pipelineConfig.model"
                  class="select select-bordered select-sm w-full"
                  :disabled="!pipelineConfig.enabled"
                >
                  <option value="minicpm-v">minicpm-v (default)</option>
                  <option value="llava">LLaVA</option>
                  <option value="bakllava">BakLLaVA</option>
                </select>
              </div>
            </div>
            <div class="flex justify-start mt-4">
              <button class="btn btn-sm btn-primary" :disabled="loading" @click="savePipelineConfig">
                Save Pipeline Settings
              </button>
            </div>
          </div>
        </BaseCard>
      </section>

      <!-- Test Connection -->
      <section>
        <Subtitle>Server Test</Subtitle>
        <BaseCard>
          <div class="p-4 flex items-center gap-4">
            <button
              class="btn btn-sm btn-outline"
              :disabled="testResult === 'testing'"
              @click="testConnection"
            >
              {{ testResult === "testing" ? "Testing..." : "Test SOAP Server" }}
            </button>
            <span
              v-if="testResult !== 'idle'"
              class="badge"
              :class="{
                'badge-success': testResult === 'success',
                'badge-error': testResult === 'error',
                'badge-warning': testResult === 'testing',
              }"
            >
              {{ testResult === "success" ? "OK" : testResult === "error" ? "Failed" : "Testing" }}
            </span>
            <span v-if="testMessage" class="text-sm opacity-70">{{ testMessage }}</span>
          </div>
        </BaseCard>
      </section>

      <!-- Upload History -->
      <section>
        <Subtitle>Upload History</Subtitle>
        <BaseCard>
          <div v-if="uploadHistory.length === 0" class="p-6 text-center opacity-50">
            <p>No upload history yet. Photos will appear here when received from the Eye-Fi card.</p>
          </div>
          <div v-else class="overflow-x-auto">
            <table class="table table-sm w-full">
              <thead>
                <tr>
                  <th>Filename</th>
                  <th>Timestamp</th>
                  <th>Size</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="entry in uploadHistory" :key="entry.id">
                  <td class="font-medium font-mono text-sm">{{ entry.filename }}</td>
                  <td class="text-sm opacity-70">{{ entry.timestamp }}</td>
                  <td class="text-sm">{{ formatFileSize(entry.size) }}</td>
                  <td>
                    <span class="badge badge-sm" :class="statusBadgeClass(entry.status)">
                      {{ entry.status }}
                    </span>
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
