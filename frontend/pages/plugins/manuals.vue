<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import { Input } from "~/components/ui/input";
  import { Button } from "~/components/ui/button";
  import { Badge } from "~/components/ui/badge";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Manuals",
  });

  const api = useUserApi();

  // Interfaces
  interface ManualCategory {
    slug: string;
    label: string;
    description: string;
  }

  interface ManualLink {
    id: string;
    itemId: string;
    title: string;
    category: string;
    source: string;
    url: string;
    attachmentId?: string;
    manufacturer: string;
    modelNumber: string;
    confidence: number;
    pages: number;
    fileSize: number;
    mimeType: string;
    createdAt: string;
  }

  interface FuzzyQuery {
    query: string;
    weight: number;
    searchUrl: string;
  }

  interface SuggestResponse {
    itemId: string;
    itemName: string;
    queries: FuzzyQuery[];
    results: any[];
    totalFound: number;
  }

  // Fetch categories
  const { data: categories } = useAsyncData("manual-categories", async () => {
    const { data } = await api.http.get<ManualCategory[]>({
      url: "/api/v1/plugins/manuals/categories",
    });
    return data || [];
  });

  // State
  const activeTab = ref<"search" | "link" | "categories">("search");

  // Search state
  const searchQuery = ref("");
  const searchBrand = ref("");
  const searchLoading = ref(false);
  const searchResults = ref<any>(null);

  // Link manual state
  const linkForm = reactive({
    itemId: "",
    itemSearch: "",
    itemResults: [] as Array<{ id: string; name: string; location: string }>,
    selectedItem: null as { id: string; name: string; location: string } | null,
    title: "",
    category: "user_manual",
    source: "manualslib",
    url: "",
    manufacturer: "",
    modelNumber: "",
    pages: 0,
    submitting: false,
    searching: false,
  });

  // Suggest state
  const suggestForm = reactive({
    itemName: "",
    manufacturer: "",
    modelNumber: "",
    loading: false,
    results: null as SuggestResponse | null,
  });

  // Item manuals view state
  const viewItemId = ref("");
  const viewItemManuals = ref<ManualLink[]>([]);
  const viewLoading = ref(false);

  // Search ManualsLib
  async function handleSearch() {
    if (!searchQuery.value) return;
    searchLoading.value = true;
    try {
      const params = new URLSearchParams({ q: searchQuery.value });
      if (searchBrand.value) params.set("brand", searchBrand.value);
      const { data } = await api.http.get<any>({
        url: `/api/v1/plugins/manuals/search?${params}`,
      });
      searchResults.value = data;
    } finally {
      searchLoading.value = false;
    }
  }

  // Search items for linking
  let itemSearchTimeout: ReturnType<typeof setTimeout>;
  watch(() => linkForm.itemSearch, (query) => {
    clearTimeout(itemSearchTimeout);
    if (query.length < 2) {
      linkForm.itemResults = [];
      return;
    }
    itemSearchTimeout = setTimeout(async () => {
      linkForm.searching = true;
      try {
        const { data } = await api.items.getAll({ q: query, page: 1, pageSize: 10 });
        linkForm.itemResults = (data?.items || []).map((item: any) => ({
          id: item.id,
          name: item.name,
          location: item.location?.name || "No location",
        }));
      } finally {
        linkForm.searching = false;
      }
    }, 300);
  });

  function selectItem(item: { id: string; name: string; location: string }) {
    linkForm.selectedItem = item;
    linkForm.itemSearch = item.name;
    linkForm.itemResults = [];
    linkForm.itemId = item.id;
  }

  function clearItem() {
    linkForm.selectedItem = null;
    linkForm.itemSearch = "";
    linkForm.itemId = "";
  }

  // Link manual to item
  async function submitLink() {
    if (!linkForm.itemId || !linkForm.url) return;
    linkForm.submitting = true;
    try {
      await api.http.post<any, void>({
        url: `/api/v1/plugins/manuals/item/${linkForm.itemId}/link`,
        body: {
          title: linkForm.title || "Manual",
          category: linkForm.category,
          source: linkForm.source,
          url: linkForm.url,
          manufacturer: linkForm.manufacturer,
          modelNumber: linkForm.modelNumber,
          pages: linkForm.pages,
        },
      });

      // Reset form
      linkForm.url = "";
      linkForm.title = "";
      linkForm.manufacturer = "";
      linkForm.modelNumber = "";
      linkForm.pages = 0;

      // Refresh manuals for this item if viewing
      if (viewItemId.value === linkForm.itemId) {
        await loadItemManuals(linkForm.itemId);
      }
    } finally {
      linkForm.submitting = false;
    }
  }

  // Suggest manuals for an item
  async function handleSuggest() {
    if (!suggestForm.itemName) return;
    suggestForm.loading = true;
    try {
      const { data } = await api.http.post<any, SuggestResponse>({
        url: `/api/v1/plugins/manuals/suggest/preview`,
        body: {
          itemName: suggestForm.itemName,
          manufacturer: suggestForm.manufacturer,
          modelNumber: suggestForm.modelNumber,
        },
      });
      suggestForm.results = data;
    } finally {
      suggestForm.loading = false;
    }
  }

  // Load manuals for an item
  async function loadItemManuals(itemId: string) {
    viewLoading.value = true;
    viewItemId.value = itemId;
    try {
      const { data } = await api.http.get<ManualLink[]>({
        url: `/api/v1/plugins/manuals/item/${itemId}`,
      });
      viewItemManuals.value = data || [];
    } finally {
      viewLoading.value = false;
    }
  }

  // Delete manual
  async function deleteManual(itemId: string, manualId: string) {
    await api.http.delete<void>({
      url: `/api/v1/plugins/manuals/item/${itemId}/manual/${manualId}`,
    });
    await loadItemManuals(itemId);
  }

  // Source display
  function sourceLabel(source: string): string {
    const map: Record<string, string> = {
      manualslib: "ManualsLib",
      manufacturer: "Manufacturer",
      user_upload: "Uploaded",
      ifixit: "iFixit",
    };
    return map[source] || source;
  }

  function sourceColor(source: string): string {
    const map: Record<string, string> = {
      manualslib: "bg-blue-500/10 text-blue-500",
      manufacturer: "bg-green-500/10 text-green-500",
      user_upload: "bg-purple-500/10 text-purple-500",
      ifixit: "bg-orange-500/10 text-orange-500",
    };
    return map[source] || "";
  }

  function formatSize(bytes: number): string {
    if (!bytes) return "";
    if (bytes < 1024) return `${bytes} B`;
    if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`;
    return `${(bytes / 1048576).toFixed(1)} MB`;
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Manuals</h1>
          <p class="text-sm text-muted-foreground">
            Find, link, and manage product manuals for your inventory items.
          </p>
        </div>
        <NuxtLink to="/plugins">
          <Button variant="outline" size="sm">Back to Plugins</Button>
        </NuxtLink>
      </div>

      <!-- Tab Bar -->
      <div class="flex items-center gap-1 overflow-x-auto border-b border-border pb-0">
        <button
          v-for="tab in [
            { key: 'search' as const, label: 'Search Manuals' },
            { key: 'link' as const, label: 'Link Manual' },
            { key: 'categories' as const, label: 'Categories' },
          ]"
          :key="tab.key"
          class="flex shrink-0 items-center gap-1.5 border-b-2 px-4 py-2.5 text-sm font-medium transition-colors"
          :class="
            activeTab === tab.key
              ? 'border-primary text-primary'
              : 'border-transparent text-muted-foreground hover:border-muted hover:text-foreground'
          "
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </div>

      <!-- SEARCH TAB -->
      <div v-if="activeTab === 'search'" class="space-y-6">
        <!-- Search Form -->
        <BaseCard>
          <div class="p-4 space-y-4">
            <h3 class="text-sm font-semibold">Search for Manuals</h3>
            <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <div class="sm:col-span-2">
                <label class="mb-1 block text-xs font-medium text-muted-foreground">Product Name or Model</label>
                <Input
                  v-model="searchQuery"
                  placeholder="e.g. DeWalt DCD771C2, Brother DCP-L2640DW..."
                  @keyup.enter="handleSearch"
                />
              </div>
              <div>
                <label class="mb-1 block text-xs font-medium text-muted-foreground">Brand (optional)</label>
                <Input v-model="searchBrand" placeholder="e.g. DeWalt, Brother..." />
              </div>
            </div>
            <Button
              size="sm"
              :disabled="!searchQuery || searchLoading"
              @click="handleSearch"
            >
              {{ searchLoading ? "Searching..." : "Search ManualsLib" }}
            </Button>
          </div>
        </BaseCard>

        <!-- Search Results -->
        <div v-if="searchResults">
          <BaseCard>
            <div class="p-4">
              <div class="flex items-center justify-between mb-3">
                <h3 class="text-sm font-semibold">Search Results</h3>
                <Badge variant="secondary">{{ searchResults.totalFound || 0 }} found</Badge>
              </div>

              <div v-if="searchResults.results && searchResults.results.length > 0" class="space-y-2">
                <div
                  v-for="(result, i) in searchResults.results"
                  :key="i"
                  class="rounded-lg border border-border p-3"
                >
                  <p class="text-sm font-medium">{{ result.title }}</p>
                  <p class="text-xs text-muted-foreground">{{ result.brand }} - {{ result.category }}</p>
                </div>
              </div>

              <div v-else class="py-6 text-center">
                <p class="text-sm text-muted-foreground mb-2">
                  {{ searchResults.message || "No results found." }}
                </p>
                <a
                  v-if="searchResults.searchUrl"
                  :href="searchResults.searchUrl"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="inline-flex items-center gap-1 text-sm text-primary hover:underline"
                >
                  Search on ManualsLib directly
                  <span class="text-xs">&#x2197;</span>
                </a>
              </div>
            </div>
          </BaseCard>
        </div>

        <!-- Fuzzy Suggest Tool -->
        <BaseCard>
          <div class="p-4 space-y-4">
            <h3 class="text-sm font-semibold">AI-Powered Manual Suggestions</h3>
            <p class="text-xs text-muted-foreground">
              Enter item details and the system will generate fuzzy search queries to find matching manuals.
            </p>
            <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
              <Input v-model="suggestForm.itemName" placeholder="Item name (e.g. 2018 Toyota Tacoma)" />
              <Input v-model="suggestForm.manufacturer" placeholder="Manufacturer (optional)" />
              <Input v-model="suggestForm.modelNumber" placeholder="Model number (optional)" />
            </div>
            <Button
              size="sm"
              variant="secondary"
              :disabled="!suggestForm.itemName || suggestForm.loading"
              @click="handleSuggest"
            >
              {{ suggestForm.loading ? "Generating..." : "Generate Search Queries" }}
            </Button>

            <div v-if="suggestForm.results" class="mt-4 space-y-2">
              <p class="text-xs font-medium text-muted-foreground">Generated Queries (sorted by relevance):</p>
              <div
                v-for="(query, i) in suggestForm.results.queries"
                :key="i"
                class="flex items-center justify-between rounded-lg border border-border p-3"
              >
                <div class="flex items-center gap-3">
                  <div
                    class="flex size-8 items-center justify-center rounded-full text-xs font-bold"
                    :class="query.weight >= 0.9 ? 'bg-green-500/10 text-green-500' : query.weight >= 0.7 ? 'bg-yellow-500/10 text-yellow-500' : 'bg-gray-500/10 text-gray-500'"
                  >
                    {{ Math.round(query.weight * 100) }}
                  </div>
                  <span class="text-sm font-medium">{{ query.query }}</span>
                </div>
                <a
                  :href="query.searchUrl"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-xs text-primary hover:underline"
                >
                  Search
                </a>
              </div>
            </div>
          </div>
        </BaseCard>
      </div>

      <!-- LINK TAB -->
      <div v-if="activeTab === 'link'" class="space-y-6">
        <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
          <!-- Link Manual Form -->
          <BaseCard>
            <div class="p-4 space-y-4">
              <h3 class="text-sm font-semibold">Link Manual to Item</h3>

              <!-- Item Search -->
              <div>
                <label class="mb-1 block text-xs font-medium text-muted-foreground">Item</label>
                <div v-if="linkForm.selectedItem" class="flex items-center gap-2">
                  <div class="flex-1 rounded-lg border border-border bg-muted/50 p-2">
                    <p class="text-sm font-medium">{{ linkForm.selectedItem.name }}</p>
                    <p class="text-xs text-muted-foreground">{{ linkForm.selectedItem.location }}</p>
                  </div>
                  <Button variant="ghost" size="sm" @click="clearItem">Clear</Button>
                </div>
                <div v-else class="relative">
                  <Input
                    v-model="linkForm.itemSearch"
                    placeholder="Search for an item..."
                  />
                  <div
                    v-if="linkForm.itemResults.length > 0"
                    class="absolute z-10 mt-1 w-full rounded-lg border border-border bg-card shadow-lg max-h-48 overflow-y-auto"
                  >
                    <button
                      v-for="item in linkForm.itemResults"
                      :key="item.id"
                      class="w-full text-left px-3 py-2 hover:bg-muted/50 transition-colors"
                      @click="selectItem(item)"
                    >
                      <p class="text-sm font-medium">{{ item.name }}</p>
                      <p class="text-xs text-muted-foreground">{{ item.location }}</p>
                    </button>
                  </div>
                </div>
              </div>

              <!-- URL -->
              <div>
                <label class="mb-1 block text-xs font-medium text-muted-foreground">Manual URL</label>
                <Input v-model="linkForm.url" placeholder="https://www.manualslib.com/manual/..." />
              </div>

              <!-- Title -->
              <div>
                <label class="mb-1 block text-xs font-medium text-muted-foreground">Title</label>
                <Input v-model="linkForm.title" placeholder="Owner's Manual, Service Manual, etc." />
              </div>

              <!-- Category + Source row -->
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="mb-1 block text-xs font-medium text-muted-foreground">Category</label>
                  <select v-model="linkForm.category" class="w-full rounded-md border border-border bg-background px-3 py-2 text-sm">
                    <option v-for="cat in categories" :key="cat.slug" :value="cat.slug">
                      {{ cat.label }}
                    </option>
                  </select>
                </div>
                <div>
                  <label class="mb-1 block text-xs font-medium text-muted-foreground">Source</label>
                  <select v-model="linkForm.source" class="w-full rounded-md border border-border bg-background px-3 py-2 text-sm">
                    <option value="manualslib">ManualsLib</option>
                    <option value="manufacturer">Manufacturer</option>
                    <option value="ifixit">iFixit</option>
                    <option value="user_upload">Uploaded</option>
                  </select>
                </div>
              </div>

              <!-- Manufacturer + Model row -->
              <div class="grid grid-cols-2 gap-4">
                <div>
                  <label class="mb-1 block text-xs font-medium text-muted-foreground">Manufacturer</label>
                  <Input v-model="linkForm.manufacturer" placeholder="Brand name" />
                </div>
                <div>
                  <label class="mb-1 block text-xs font-medium text-muted-foreground">Model Number</label>
                  <Input v-model="linkForm.modelNumber" placeholder="e.g. DCD771C2" />
                </div>
              </div>

              <Button
                size="sm"
                class="w-full"
                :disabled="!linkForm.itemId || !linkForm.url || linkForm.submitting"
                @click="submitLink"
              >
                {{ linkForm.submitting ? "Linking..." : "Link Manual" }}
              </Button>
            </div>
          </BaseCard>

          <!-- Item Manuals Viewer -->
          <BaseCard>
            <div class="p-4 space-y-4">
              <h3 class="text-sm font-semibold">Item Manuals</h3>
              <p class="text-xs text-muted-foreground">
                {{ linkForm.selectedItem ? `Manuals for: ${linkForm.selectedItem.name}` : 'Select an item on the left to view its linked manuals.' }}
              </p>

              <div v-if="linkForm.selectedItem">
                <Button
                  v-if="viewItemId !== linkForm.itemId"
                  size="sm"
                  variant="secondary"
                  :disabled="viewLoading"
                  @click="loadItemManuals(linkForm.itemId)"
                >
                  Load Manuals
                </Button>
              </div>

              <div v-if="viewItemManuals.length > 0" class="space-y-2">
                <div
                  v-for="manual in viewItemManuals"
                  :key="manual.id"
                  class="rounded-lg border border-border p-3"
                >
                  <div class="flex items-start justify-between">
                    <div class="min-w-0 flex-1">
                      <a
                        :href="manual.url"
                        target="_blank"
                        rel="noopener noreferrer"
                        class="text-sm font-medium text-primary hover:underline"
                      >
                        {{ manual.title }}
                      </a>
                      <div class="mt-1 flex flex-wrap items-center gap-2">
                        <Badge :class="sourceColor(manual.source)" class="text-[10px]">
                          {{ sourceLabel(manual.source) }}
                        </Badge>
                        <span v-if="manual.pages" class="text-[10px] text-muted-foreground">
                          {{ manual.pages }} pages
                        </span>
                        <span v-if="manual.fileSize" class="text-[10px] text-muted-foreground">
                          {{ formatSize(manual.fileSize) }}
                        </span>
                      </div>
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      class="text-xs text-destructive"
                      @click="deleteManual(manual.itemId, manual.id)"
                    >
                      Remove
                    </Button>
                  </div>
                </div>
              </div>

              <div v-else-if="viewItemId && !viewLoading" class="py-4 text-center">
                <p class="text-xs text-muted-foreground">No manuals linked to this item yet.</p>
              </div>
            </div>
          </BaseCard>
        </div>
      </div>

      <!-- CATEGORIES TAB -->
      <div v-if="activeTab === 'categories'">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <BaseCard v-for="cat in categories" :key="cat.slug">
            <div class="p-4">
              <h3 class="text-sm font-semibold">{{ cat.label }}</h3>
              <p class="mt-1 text-xs text-muted-foreground">{{ cat.description }}</p>
              <Badge variant="secondary" class="mt-2 text-[10px]">{{ cat.slug }}</Badge>
            </div>
          </BaseCard>
        </div>
      </div>
    </BaseContainer>
  </div>
</template>
