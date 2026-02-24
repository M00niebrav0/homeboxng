<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import { route } from "~/lib/api/base";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Excel Export & Import",
  });

  const api = useUserApi();

  // --- Types ---

  interface ExportField {
    key: string;
    label: string;
    selected: boolean;
  }

  interface ImportHistory {
    id: string;
    date: string;
    filename: string;
    recordsImported: number;
    errors: number;
    status: string;
  }

  interface PreviewRow {
    [key: string]: string;
  }

  interface ColumnMapping {
    sourceColumn: string;
    targetField: string;
  }

  // --- State ---

  // Export
  const exportFormat = ref<"csv" | "tsv">("csv");
  const exportDataTypes = reactive({
    items: true,
    locations: false,
    labels: false,
  });
  const exportDateRange = reactive({
    enabled: false,
    start: "",
    end: "",
  });
  const exporting = ref(false);

  const exportFields = ref<ExportField[]>([
    { key: "name", label: "Name", selected: true },
    { key: "description", label: "Description", selected: true },
    { key: "location", label: "Location", selected: true },
    { key: "labels", label: "Labels", selected: true },
    { key: "purchasePrice", label: "Purchase Price", selected: true },
    { key: "quantity", label: "Quantity", selected: true },
    { key: "serialNumber", label: "Serial Number", selected: false },
    { key: "modelNumber", label: "Model Number", selected: false },
    { key: "manufacturer", label: "Manufacturer", selected: false },
    { key: "purchaseDate", label: "Purchase Date", selected: false },
    { key: "warrantyExpires", label: "Warranty Expires", selected: false },
    { key: "notes", label: "Notes", selected: false },
    { key: "assetId", label: "Asset ID", selected: false },
    { key: "insured", label: "Insured", selected: false },
    { key: "archived", label: "Archived", selected: false },
    { key: "createdAt", label: "Created At", selected: false },
    { key: "updatedAt", label: "Updated At", selected: false },
  ]);

  const showFieldSelector = ref(false);

  // Import
  const importFile = ref<File | null>(null);
  const importDragOver = ref(false);
  const previewData = ref<PreviewRow[]>([]);
  const previewColumns = ref<string[]>([]);
  const columnMappings = ref<ColumnMapping[]>([]);
  const conflictResolution = ref<"skip" | "update" | "duplicate">("skip");
  const importing = ref(false);
  const showConfirmImport = ref(false);

  const homeboxFields = [
    { value: "", label: "-- Skip --" },
    { value: "name", label: "Name" },
    { value: "description", label: "Description" },
    { value: "location", label: "Location" },
    { value: "labels", label: "Labels" },
    { value: "purchasePrice", label: "Purchase Price" },
    { value: "quantity", label: "Quantity" },
    { value: "serialNumber", label: "Serial Number" },
    { value: "modelNumber", label: "Model Number" },
    { value: "manufacturer", label: "Manufacturer" },
    { value: "purchaseDate", label: "Purchase Date" },
    { value: "warrantyExpires", label: "Warranty Expires" },
    { value: "notes", label: "Notes" },
    { value: "assetId", label: "Asset ID" },
    { value: "insured", label: "Insured" },
  ];

  // Import History
  const importHistory = ref<ImportHistory[]>([]);

  // --- Data Loading ---

  useAsyncData("import-history", async () => {
    try {
      const { data } = await api.http.get<ImportHistory[]>({ url: route("/plugins/excel-export/import-history") });
      importHistory.value = data || [];
      return data;
    } catch {
      return [];
    }
  });

  // --- Computed ---

  const selectedFieldCount = computed(() => exportFields.value.filter(f => f.selected).length);

  const selectedDataTypes = computed(() => {
    const types: string[] = [];
    if (exportDataTypes.items) types.push("items");
    if (exportDataTypes.locations) types.push("locations");
    if (exportDataTypes.labels) types.push("labels");
    return types;
  });

  const canExport = computed(() => selectedDataTypes.value.length > 0 && selectedFieldCount.value > 0);

  const canImport = computed(() => {
    return importFile.value !== null && columnMappings.value.some(m => m.targetField !== "");
  });

  // --- Export Methods ---

  function selectAllFields() {
    exportFields.value.forEach(f => { f.selected = true; });
  }

  function deselectAllFields() {
    exportFields.value.forEach(f => { f.selected = false; });
  }

  async function doExport() {
    if (!canExport.value) return;

    exporting.value = true;
    try {
      const selectedFields = exportFields.value.filter(f => f.selected).map(f => f.key);
      const params: Record<string, string> = {
        format: exportFormat.value,
        types: selectedDataTypes.value.join(","),
        fields: selectedFields.join(","),
      };
      if (exportDateRange.enabled && exportDateRange.start) {
        params.dateFrom = exportDateRange.start;
      }
      if (exportDateRange.enabled && exportDateRange.end) {
        params.dateTo = exportDateRange.end;
      }

      const response = await fetch(route("/plugins/excel-export/export", params), {
        method: "GET",
        credentials: "include",
      });

      if (!response.ok) {
        return;
      }

      const blob = await response.blob();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `homebox-export-${new Date().toISOString().slice(0, 10)}.${exportFormat.value}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
    } finally {
      exporting.value = false;
    }
  }

  // --- Import Methods ---

  function handleFileDrop(event: DragEvent) {
    importDragOver.value = false;
    const files = event.dataTransfer?.files;
    if (files && files.length > 0) {
      handleFileSelect(files[0]);
    }
  }

  function handleFileInput(event: Event) {
    const target = event.target as HTMLInputElement;
    if (target.files && target.files.length > 0) {
      handleFileSelect(target.files[0]);
    }
  }

  function handleFileSelect(file: File) {
    const validExtensions = [".csv", ".tsv"];
    const ext = file.name.substring(file.name.lastIndexOf(".")).toLowerCase();
    if (!validExtensions.includes(ext)) {
      return;
    }
    importFile.value = file;
    parsePreview(file);
  }

  async function parsePreview(file: File) {
    const text = await file.text();
    const delimiter = file.name.endsWith(".tsv") ? "\t" : ",";
    const lines = text.split("\n").filter(l => l.trim() !== "");

    if (lines.length === 0) return;

    const headers = lines[0].split(delimiter).map(h => h.trim().replace(/^"|"$/g, ""));
    previewColumns.value = headers;

    columnMappings.value = headers.map(col => ({
      sourceColumn: col,
      targetField: guessFieldMapping(col),
    }));

    const dataLines = lines.slice(1, 6);
    previewData.value = dataLines.map(line => {
      const values = line.split(delimiter).map(v => v.trim().replace(/^"|"$/g, ""));
      const row: PreviewRow = {};
      headers.forEach((h, i) => {
        row[h] = values[i] || "";
      });
      return row;
    });
  }

  function guessFieldMapping(columnName: string): string {
    const lower = columnName.toLowerCase().replace(/[^a-z]/g, "");
    const mappings: Record<string, string> = {
      name: "name",
      itemname: "name",
      title: "name",
      description: "description",
      desc: "description",
      location: "location",
      labels: "labels",
      tags: "labels",
      price: "purchasePrice",
      purchaseprice: "purchasePrice",
      cost: "purchasePrice",
      quantity: "quantity",
      qty: "quantity",
      serial: "serialNumber",
      serialnumber: "serialNumber",
      model: "modelNumber",
      modelnumber: "modelNumber",
      manufacturer: "manufacturer",
      brand: "manufacturer",
      notes: "notes",
      assetid: "assetId",
    };
    return mappings[lower] || "";
  }

  function clearImport() {
    importFile.value = null;
    previewData.value = [];
    previewColumns.value = [];
    columnMappings.value = [];
  }

  async function doImport() {
    if (!canImport.value || !importFile.value) return;

    showConfirmImport.value = false;
    importing.value = true;
    try {
      const formData = new FormData();
      formData.append("file", importFile.value);
      formData.append("mappings", JSON.stringify(columnMappings.value));
      formData.append("conflictResolution", conflictResolution.value);

      await fetch(route("/plugins/excel-export/import"), {
        method: "POST",
        credentials: "include",
        body: formData,
      });

      clearImport();
      const { data } = await api.http.get<ImportHistory[]>({ url: route("/plugins/excel-export/import-history") });
      importHistory.value = data || [];
    } finally {
      importing.value = false;
    }
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Excel Export & Import</h1>
          <p class="text-sm opacity-70">
            Export your inventory data to CSV/TSV or import from spreadsheets.
          </p>
        </div>
        <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
          Back to Plugins
        </NuxtLink>
      </div>

      <!-- Export Section -->
      <section>
        <Subtitle>Export Data</Subtitle>
        <BaseCard>
          <div class="p-4 flex flex-col gap-4">
            <!-- Format Selection -->
            <div>
              <label class="label">
                <span class="label-text font-medium">Export Format</span>
              </label>
              <div class="flex gap-4">
                <label class="flex items-center gap-2 cursor-pointer">
                  <input v-model="exportFormat" type="radio" value="csv" class="radio radio-primary radio-sm" />
                  <span class="text-sm">CSV (Comma-Separated)</span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer">
                  <input v-model="exportFormat" type="radio" value="tsv" class="radio radio-primary radio-sm" />
                  <span class="text-sm">TSV (Tab-Separated)</span>
                </label>
              </div>
            </div>

            <!-- Data Types -->
            <div>
              <label class="label">
                <span class="label-text font-medium">Data to Export</span>
              </label>
              <div class="flex gap-4">
                <label class="flex items-center gap-2 cursor-pointer">
                  <input v-model="exportDataTypes.items" type="checkbox" class="checkbox checkbox-primary checkbox-sm" />
                  <span class="text-sm">Items</span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer">
                  <input v-model="exportDataTypes.locations" type="checkbox" class="checkbox checkbox-primary checkbox-sm" />
                  <span class="text-sm">Locations</span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer">
                  <input v-model="exportDataTypes.labels" type="checkbox" class="checkbox checkbox-primary checkbox-sm" />
                  <span class="text-sm">Labels</span>
                </label>
              </div>
            </div>

            <!-- Field Selector -->
            <div>
              <div class="flex items-center justify-between">
                <label class="label">
                  <span class="label-text font-medium">Fields ({{ selectedFieldCount }} selected)</span>
                </label>
                <button class="btn btn-xs btn-ghost" @click="showFieldSelector = !showFieldSelector">
                  {{ showFieldSelector ? "Collapse" : "Expand" }}
                </button>
              </div>
              <div v-if="showFieldSelector" class="border rounded-lg p-3">
                <div class="flex gap-2 mb-3">
                  <button class="btn btn-xs btn-outline" @click="selectAllFields">Select All</button>
                  <button class="btn btn-xs btn-outline" @click="deselectAllFields">Deselect All</button>
                </div>
                <div class="grid grid-cols-2 gap-1 md:grid-cols-3 lg:grid-cols-4">
                  <label
                    v-for="field in exportFields"
                    :key="field.key"
                    class="flex items-center gap-2 cursor-pointer p-1 rounded hover:bg-base-200"
                  >
                    <input v-model="field.selected" type="checkbox" class="checkbox checkbox-xs checkbox-primary" />
                    <span class="text-sm">{{ field.label }}</span>
                  </label>
                </div>
              </div>
            </div>

            <!-- Date Range Filter -->
            <div>
              <label class="flex items-center gap-2 cursor-pointer mb-2">
                <input v-model="exportDateRange.enabled" type="checkbox" class="checkbox checkbox-primary checkbox-sm" />
                <span class="label-text font-medium">Filter by Date Range</span>
              </label>
              <div v-if="exportDateRange.enabled" class="grid grid-cols-1 gap-3 md:grid-cols-2">
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Start Date</span>
                  </label>
                  <input v-model="exportDateRange.start" type="date" class="input input-bordered input-sm w-full" />
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">End Date</span>
                  </label>
                  <input v-model="exportDateRange.end" type="date" class="input input-bordered input-sm w-full" />
                </div>
              </div>
            </div>

            <!-- Export Button -->
            <div class="flex justify-end">
              <button class="btn btn-sm btn-primary" :disabled="exporting || !canExport" @click="doExport">
                {{ exporting ? "Exporting..." : "Export" }}
              </button>
            </div>
          </div>
        </BaseCard>
      </section>

      <!-- Import Section -->
      <section>
        <Subtitle>Import Data</Subtitle>
        <BaseCard>
          <div class="p-4 flex flex-col gap-4">
            <!-- File Upload Dropzone -->
            <div
              class="border-2 border-dashed rounded-lg p-8 text-center transition-colors cursor-pointer"
              :class="importDragOver ? 'border-primary bg-primary/5' : 'border-base-300'"
              @dragover.prevent="importDragOver = true"
              @dragleave="importDragOver = false"
              @drop.prevent="handleFileDrop"
              @click="($refs.fileInput as HTMLInputElement)?.click()"
            >
              <input
                ref="fileInput"
                type="file"
                accept=".csv,.tsv"
                class="hidden"
                @change="handleFileInput"
              />
              <div v-if="!importFile">
                <p class="text-lg opacity-50">Drop a CSV or TSV file here</p>
                <p class="text-sm opacity-40 mt-1">or click to browse</p>
              </div>
              <div v-else class="flex items-center justify-center gap-3">
                <span class="font-medium">{{ importFile.name }}</span>
                <span class="badge badge-sm badge-outline">{{ (importFile.size / 1024).toFixed(1) }} KB</span>
                <button class="btn btn-xs btn-ghost text-error" @click.stop="clearImport">Remove</button>
              </div>
            </div>

            <!-- Preview Table -->
            <div v-if="previewData.length > 0">
              <label class="label">
                <span class="label-text font-medium">Preview (first {{ previewData.length }} rows)</span>
              </label>
              <div class="overflow-x-auto border rounded-lg">
                <table class="table table-xs w-full">
                  <thead>
                    <tr>
                      <th v-for="col in previewColumns" :key="col" class="text-xs">{{ col }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="(row, idx) in previewData" :key="idx">
                      <td v-for="col in previewColumns" :key="col" class="text-xs max-w-[150px] truncate">
                        {{ row[col] }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            <!-- Column Mapping -->
            <div v-if="columnMappings.length > 0">
              <label class="label">
                <span class="label-text font-medium">Column Mapping</span>
              </label>
              <div class="grid grid-cols-1 gap-2 md:grid-cols-2">
                <div
                  v-for="mapping in columnMappings"
                  :key="mapping.sourceColumn"
                  class="flex items-center gap-2 p-2 border rounded"
                >
                  <span class="text-sm font-mono flex-1 truncate">{{ mapping.sourceColumn }}</span>
                  <span class="text-xs opacity-50">-></span>
                  <select v-model="mapping.targetField" class="select select-bordered select-xs flex-1">
                    <option v-for="field in homeboxFields" :key="field.value" :value="field.value">
                      {{ field.label }}
                    </option>
                  </select>
                </div>
              </div>
            </div>

            <!-- Conflict Resolution -->
            <div v-if="columnMappings.length > 0">
              <label class="label">
                <span class="label-text font-medium">Conflict Resolution</span>
              </label>
              <div class="flex flex-col gap-2">
                <label class="flex items-center gap-2 cursor-pointer">
                  <input v-model="conflictResolution" type="radio" value="skip" class="radio radio-primary radio-sm" />
                  <span class="text-sm">Skip existing items</span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer">
                  <input v-model="conflictResolution" type="radio" value="update" class="radio radio-primary radio-sm" />
                  <span class="text-sm">Update existing items</span>
                </label>
                <label class="flex items-center gap-2 cursor-pointer">
                  <input v-model="conflictResolution" type="radio" value="duplicate" class="radio radio-primary radio-sm" />
                  <span class="text-sm">Create duplicates</span>
                </label>
              </div>
            </div>

            <!-- Import Button -->
            <div v-if="columnMappings.length > 0" class="flex justify-end">
              <button class="btn btn-sm btn-primary" :disabled="importing || !canImport" @click="showConfirmImport = true">
                {{ importing ? "Importing..." : "Import" }}
              </button>
            </div>
          </div>
        </BaseCard>
      </section>

      <!-- Import Confirmation Modal -->
      <dialog class="modal" :class="{ 'modal-open': showConfirmImport }">
        <div class="modal-box">
          <h3 class="font-bold text-lg">Confirm Import</h3>
          <div class="py-4">
            <p>You are about to import data from <strong>{{ importFile?.name }}</strong>.</p>
            <p class="mt-2 text-sm opacity-70">
              Mapped {{ columnMappings.filter(m => m.targetField !== "").length }} columns.
              Conflict resolution: <strong>{{ conflictResolution }}</strong>.
            </p>
            <p class="mt-2 text-sm text-warning">This action cannot be easily undone.</p>
          </div>
          <div class="modal-action">
            <button class="btn btn-sm" @click="showConfirmImport = false">Cancel</button>
            <button class="btn btn-sm btn-primary" @click="doImport">Confirm Import</button>
          </div>
        </div>
        <div class="modal-backdrop" @click="showConfirmImport = false" />
      </dialog>

      <!-- Import History -->
      <section>
        <Subtitle>Import History</Subtitle>
        <BaseCard>
          <div v-if="importHistory.length === 0" class="p-6 text-center opacity-50">
            <p>No import history. Upload and import a file to get started.</p>
          </div>
          <div v-else class="overflow-x-auto">
            <table class="table table-sm w-full">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Filename</th>
                  <th>Records Imported</th>
                  <th>Errors</th>
                  <th>Status</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="entry in importHistory" :key="entry.id">
                  <td class="text-sm">{{ entry.date }}</td>
                  <td class="font-mono text-sm">{{ entry.filename }}</td>
                  <td>{{ entry.recordsImported }}</td>
                  <td :class="entry.errors > 0 ? 'text-error' : ''">{{ entry.errors }}</td>
                  <td>
                    <span
                      class="badge badge-sm"
                      :class="{
                        'badge-success': entry.status === 'completed',
                        'badge-error': entry.status === 'failed',
                        'badge-warning': entry.status === 'partial',
                      }"
                    >
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
