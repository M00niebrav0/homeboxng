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
    cardMac: string;
    importMode: string;
    pollInterval: number;
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
    uploadDirectory: "/tmp/homeboxng-eyefi",
    cardMac: "",
    importMode: "manual",
    pollInterval: 10,
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
  const quickStartDismissed = ref(false);
  const showSetupWizard = ref(false);

  // --- Computed ---

  const isConfigured = computed(() => !!soapConfig.uploadKey && !!soapConfig.uploadDirectory);
  const setupProgress = computed(() => {
    let steps = 0;
    if (soapConfig.uploadKey) steps++;
    if (soapConfig.uploadDirectory) steps++;
    if (serverRunning.value) steps++;
    if (cameraStatus.value.connected) steps++;
    return steps;
  });

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
          if (field.key === "card_mac") soapConfig.cardMac = field.value || "";
          if (field.key === "upload_dir") soapConfig.uploadDirectory = field.value || field.default || "/tmp/homeboxng-eyefi";
          if (field.key === "import_mode") soapConfig.importMode = field.value || "manual";
          if (field.key === "poll_interval") soapConfig.pollInterval = parseInt(field.value || "10");
          if (field.key === "watch_directory") watcherConfig.watchDirectory = field.value || "";
          if (field.key === "auto_process") watcherConfig.autoProcess = field.value === "true";
          if (field.key === "file_extensions") watcherConfig.extensions = field.value || ".jpg,.jpeg,.png,.raw,.cr2,.nef";
          if (field.key === "pipeline_enabled") pipelineConfig.enabled = field.value === "true";
          if (field.key === "pipeline_model") pipelineConfig.model = field.value || "minicpm-v";
        }
        // Show wizard if not yet configured
        if (!soapConfig.uploadKey) showSetupWizard.value = true;
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
        upload_dir: soapConfig.uploadDirectory,
        card_mac: soapConfig.cardMac,
        import_mode: soapConfig.importMode,
        poll_interval: String(soapConfig.pollInterval),
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
            Wirelessly upload photos from an Eye-Fi X2 card directly into your HomeBoxNG inventory.
          </p>
        </div>
        <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
          Back to Plugins
        </NuxtLink>
      </div>

      <!-- Status Overview Bar -->
      <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
        <div class="rounded-lg border p-3">
          <p class="text-xs opacity-50">Camera</p>
          <div class="flex items-center gap-2 mt-1">
            <span
              class="w-2.5 h-2.5 rounded-full"
              :class="cameraStatus.connected ? 'bg-success' : 'bg-base-300'"
            />
            <span class="text-sm font-semibold">{{ cameraStatus.connected ? "Connected" : "Disconnected" }}</span>
          </div>
        </div>
        <div class="rounded-lg border p-3">
          <p class="text-xs opacity-50">SOAP Server</p>
          <div class="flex items-center gap-2 mt-1">
            <span
              class="w-2.5 h-2.5 rounded-full"
              :class="serverRunning ? 'bg-success' : 'bg-base-300'"
            />
            <span class="text-sm font-semibold">{{ serverRunning ? "Running" : "Stopped" }}</span>
          </div>
        </div>
        <div class="rounded-lg border p-3">
          <p class="text-xs opacity-50">Photos Received</p>
          <p class="text-2xl font-bold text-primary mt-0.5">{{ cameraStatus.photosReceived }}</p>
        </div>
        <div class="rounded-lg border p-3">
          <p class="text-xs opacity-50">Last Upload</p>
          <p class="text-sm font-semibold mt-1">{{ cameraStatus.lastUpload }}</p>
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
              <p class="text-sm opacity-70 mt-1">Get your Eye-Fi card uploading photos to HomeBoxNG in minutes.</p>
            </div>
            <button class="btn btn-ghost btn-xs" @click="quickStartDismissed = true">Dismiss</button>
          </div>
          <div class="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <!-- Step 1 -->
            <div class="rounded-lg border bg-base-100 p-3" :class="soapConfig.uploadKey ? 'border-success/30' : ''">
              <div class="flex items-center gap-2 mb-2">
                <span
                  class="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                  :class="soapConfig.uploadKey ? 'bg-success text-success-content' : 'bg-base-200'"
                >1</span>
                <span class="font-medium text-sm">Get Upload Key</span>
              </div>
              <p class="text-xs opacity-60">
                Find your Eye-Fi card's upload key in
                <code class="bg-base-200 px-1 rounded text-[10px]">Settings.xml</code>
                on the card itself, or from the Eye-Fi Center app's export.
              </p>
            </div>
            <!-- Step 2 -->
            <div class="rounded-lg border bg-base-100 p-3" :class="soapConfig.uploadDirectory ? 'border-success/30' : ''">
              <div class="flex items-center gap-2 mb-2">
                <span
                  class="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                  :class="soapConfig.uploadDirectory ? 'bg-success text-success-content' : 'bg-base-200'"
                >2</span>
                <span class="font-medium text-sm">Configure Settings</span>
              </div>
              <p class="text-xs opacity-60">
                Enter the upload key and choose a directory for incoming photos. You can also set these as environment variables.
              </p>
            </div>
            <!-- Step 3 -->
            <div class="rounded-lg border bg-base-100 p-3" :class="serverRunning ? 'border-success/30' : ''">
              <div class="flex items-center gap-2 mb-2">
                <span
                  class="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                  :class="serverRunning ? 'bg-success text-success-content' : 'bg-base-200'"
                >3</span>
                <span class="font-medium text-sm">Start SOAP Server</span>
              </div>
              <p class="text-xs opacity-60">
                Save your config and the SOAP server will start listening on port {{ soapConfig.port }}. Use "Test Connection" to verify.
              </p>
            </div>
            <!-- Step 4 -->
            <div class="rounded-lg border bg-base-100 p-3" :class="cameraStatus.connected ? 'border-success/30' : ''">
              <div class="flex items-center gap-2 mb-2">
                <span
                  class="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold"
                  :class="cameraStatus.connected ? 'bg-success text-success-content' : 'bg-base-200'"
                >4</span>
                <span class="font-medium text-sm">Connect Camera</span>
              </div>
              <p class="text-xs opacity-60">
                Power on your camera with the Eye-Fi card. It will auto-discover the SOAP server on your network and begin uploading.
              </p>
            </div>
          </div>
        </div>
      </section>

      <!-- Setup Wizard (inline expandable) -->
      <section v-if="showSetupWizard && !isConfigured">
        <div class="rounded-lg border border-warning/30 bg-warning/5 p-4">
          <div class="flex items-start gap-3">
            <span class="text-warning text-lg mt-0.5">!</span>
            <div class="flex-1">
              <h3 class="font-semibold text-sm">Where to find your Upload Key</h3>
              <div class="mt-2 text-sm opacity-70 space-y-2">
                <p>The Eye-Fi upload key is a 32-character hexadecimal string unique to your card. Here is how to find it:</p>
                <ol class="list-decimal list-inside space-y-1 ml-1">
                  <li>Insert your Eye-Fi card into a card reader on your computer.</li>
                  <li>Open the card in a file manager and look for <code class="bg-base-200 px-1 rounded text-xs">Settings.xml</code>.</li>
                  <li>Open the file in a text editor and find the <code class="bg-base-200 px-1 rounded text-xs">&lt;UploadKey&gt;</code> element.</li>
                  <li>Copy the 32-character hex value and paste it below.</li>
                </ol>
                <p class="mt-2">
                  Alternatively, if you used Eye-Fi Center software, the key is stored in:
                </p>
                <ul class="list-disc list-inside space-y-1 ml-1">
                  <li>Windows: <code class="bg-base-200 px-1 rounded text-xs">%APPDATA%\Eye-Fi\Settings.xml</code></li>
                  <li>macOS: <code class="bg-base-200 px-1 rounded text-xs">~/Library/Eye-Fi/Settings.xml</code></li>
                </ul>
              </div>
              <button class="btn btn-xs btn-ghost mt-3" @click="showSetupWizard = false">Got it, close</button>
            </div>
          </div>
        </div>
      </section>

      <!-- SOAP Server & Core Settings -->
      <section>
        <Subtitle>SOAP Server Settings</Subtitle>
        <BaseCard>
          <div class="p-4">
            <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
              <!-- Upload Key -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">
                    Upload Key
                    <span class="text-error ml-0.5">*</span>
                  </span>
                </label>
                <input
                  v-model="soapConfig.uploadKey"
                  type="password"
                  placeholder="32-character hex upload key"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label pb-0">
                  <span class="label-text-alt opacity-50">Found in Settings.xml on your Eye-Fi card</span>
                </label>
                <label class="label pt-0">
                  <span class="label-text-alt opacity-40 font-mono text-[11px]">
                    or set <code class="bg-base-200 px-1 py-0.5 rounded text-[10px]">HBOX_EYEFI_UPLOAD_KEY</code>
                  </span>
                </label>
              </div>

              <!-- Card MAC -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Card MAC Address</span>
                </label>
                <input
                  v-model="soapConfig.cardMac"
                  type="text"
                  placeholder="00:1A:7D:DA:71:XX"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label pb-0">
                  <span class="label-text-alt opacity-50">Optional. Restricts uploads to a specific card.</span>
                </label>
                <label class="label pt-0">
                  <span class="label-text-alt opacity-40 font-mono text-[11px]">
                    or set <code class="bg-base-200 px-1 py-0.5 rounded text-[10px]">HBOX_EYEFI_CARD_MAC</code>
                  </span>
                </label>
              </div>

              <!-- Upload Directory -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">
                    Upload Directory
                    <span class="text-error ml-0.5">*</span>
                  </span>
                </label>
                <input
                  v-model="soapConfig.uploadDirectory"
                  type="text"
                  placeholder="/tmp/homeboxng-eyefi"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label pb-0">
                  <span class="label-text-alt opacity-50">Directory for incoming photo uploads</span>
                </label>
                <label class="label pt-0">
                  <span class="label-text-alt opacity-40 font-mono text-[11px]">
                    or set <code class="bg-base-200 px-1 py-0.5 rounded text-[10px]">HBOX_EYEFI_UPLOAD_DIR</code>
                  </span>
                </label>
              </div>

              <!-- Port -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">SOAP Port</span>
                </label>
                <input
                  v-model.number="soapConfig.port"
                  type="number"
                  min="1024"
                  max="65535"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label">
                  <span class="label-text-alt opacity-50">Default: 59278. The Eye-Fi card auto-discovers this port.</span>
                </label>
              </div>

              <!-- Import Mode -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Import Mode</span>
                </label>
                <select
                  v-model="soapConfig.importMode"
                  class="select select-bordered select-sm w-full"
                >
                  <option value="auto">Auto (immediate processing)</option>
                  <option value="manual">Manual (on demand)</option>
                  <option value="off">Off (store only)</option>
                </select>
                <label class="label">
                  <span class="label-text-alt opacity-50">How uploaded photos are handled after receipt</span>
                </label>
              </div>

              <!-- Poll Interval -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Poll Interval (seconds)</span>
                </label>
                <input
                  v-model.number="soapConfig.pollInterval"
                  type="number"
                  min="1"
                  max="3600"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label">
                  <span class="label-text-alt opacity-50">How often to check for new files. Default: 10</span>
                </label>
              </div>
            </div>

            <div class="flex items-center gap-3 mt-4">
              <button class="btn btn-sm btn-primary" :disabled="loading" @click="saveSoapConfig">
                Save Settings
              </button>
              <button
                class="btn btn-sm btn-outline"
                :disabled="testResult === 'testing' || !soapConfig.uploadKey"
                @click="testConnection"
              >
                {{ testResult === "testing" ? "Testing..." : "Test Connection" }}
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
                  <span class="label-text font-medium">Watch Directory</span>
                </label>
                <input
                  v-model="watcherConfig.watchDirectory"
                  type="text"
                  placeholder="/data/eyefi/watch"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label">
                  <span class="label-text-alt opacity-50">Additional directory to monitor for new files</span>
                </label>
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Auto-Process</span>
                </label>
                <div class="flex items-center gap-2 mt-1">
                  <input
                    v-model="watcherConfig.autoProcess"
                    type="checkbox"
                    class="toggle toggle-primary toggle-sm"
                  />
                  <span class="text-sm">{{ watcherConfig.autoProcess ? "Enabled" : "Disabled" }}</span>
                </div>
                <label class="label">
                  <span class="label-text-alt opacity-50">Automatically process files as they appear</span>
                </label>
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">File Extensions</span>
                </label>
                <input
                  v-model="watcherConfig.extensions"
                  type="text"
                  placeholder=".jpg,.jpeg,.png,.raw"
                  class="input input-bordered input-sm w-full"
                />
                <label class="label">
                  <span class="label-text-alt opacity-50">Comma-separated list of accepted extensions</span>
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
                  <span class="label-text font-medium">AI Vision Processing</span>
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
                  <span class="label-text font-medium">Vision Model</span>
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

      <!-- Upload History -->
      <section>
        <Subtitle>Upload History</Subtitle>
        <BaseCard>
          <div v-if="uploadHistory.length === 0" class="p-6 text-center opacity-50">
            <p class="text-lg mb-1">No uploads yet</p>
            <p class="text-sm">Photos will appear here when received from the Eye-Fi card.</p>
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

      <!-- Environment Variables Reference -->
      <section>
        <Subtitle>Environment Variables</Subtitle>
        <BaseCard>
          <div class="p-4">
            <p class="text-sm opacity-70 mb-3">
              All Eye-Fi settings can be configured via environment variables. These take priority over default values
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
                    <td class="font-mono text-xs"><code class="bg-base-200 px-1.5 py-0.5 rounded">HBOX_EYEFI_UPLOAD_KEY</code></td>
                    <td class="text-sm">32-character hex upload key from the Eye-Fi card</td>
                    <td><span class="badge badge-xs badge-error">Required</span></td>
                  </tr>
                  <tr>
                    <td class="font-mono text-xs"><code class="bg-base-200 px-1.5 py-0.5 rounded">HBOX_EYEFI_CARD_MAC</code></td>
                    <td class="text-sm">Card MAC address to restrict which card can upload</td>
                    <td><span class="badge badge-xs badge-ghost">Optional</span></td>
                  </tr>
                  <tr>
                    <td class="font-mono text-xs"><code class="bg-base-200 px-1.5 py-0.5 rounded">HBOX_EYEFI_UPLOAD_DIR</code></td>
                    <td class="text-sm">Directory path for incoming photo uploads</td>
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
