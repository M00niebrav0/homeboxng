<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import type { CatalogPlugin } from "~/lib/api/classes/plugins";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Plugin Catalog",
  });

  const api = useUserApi();
  const searchQuery = ref("");
  const selectedCategory = ref("all");
  const installing = ref<string | null>(null);

  const { data: catalog, refresh: refreshCatalog } = useAsyncData("plugin-catalog", async () => {
    const { data } = await api.plugins.getCatalog();
    return data;
  });

  const categories = computed(() => {
    if (!catalog.value) return [];
    const cats = new Set(catalog.value.map(p => p.category || "Other"));
    return ["all", ...Array.from(cats).sort()];
  });

  const filteredPlugins = computed(() => {
    if (!catalog.value) return [];

    return catalog.value.filter(p => {
      const matchesSearch =
        !searchQuery.value ||
        p.name.toLowerCase().includes(searchQuery.value.toLowerCase()) ||
        p.description.toLowerCase().includes(searchQuery.value.toLowerCase());

      const matchesCategory =
        selectedCategory.value === "all" || (p.category || "Other") === selectedCategory.value;

      return matchesSearch && matchesCategory;
    });
  });

  async function installPlugin(plugin: CatalogPlugin) {
    installing.value = plugin.name;
    try {
      await api.plugins.installPlugin(plugin.name);
      refreshCatalog();
    } finally {
      installing.value = null;
    }
  }

  async function doRefreshCatalog() {
    await api.plugins.refreshCatalog();
    refreshCatalog();
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Plugin Catalog</h1>
          <p class="text-sm opacity-70">
            Browse and install community plugins from GitHub repositories.
          </p>
        </div>
        <div class="flex gap-2">
          <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
            Back to Manager
          </NuxtLink>
          <button class="btn btn-sm btn-primary" @click="doRefreshCatalog">
            Refresh Catalog
          </button>
        </div>
      </div>

      <!-- Search & Filter -->
      <div class="flex gap-4 flex-wrap">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search plugins..."
          class="input input-bordered input-sm flex-1 min-w-[200px]"
        />
        <div class="flex gap-1">
          <button
            v-for="cat in categories"
            :key="cat"
            class="btn btn-xs"
            :class="selectedCategory === cat ? 'btn-primary' : 'btn-ghost'"
            @click="selectedCategory = cat"
          >
            {{ cat }}
          </button>
        </div>
      </div>

      <!-- Plugin Grid -->
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        <BaseCard v-for="plugin in filteredPlugins" :key="plugin.name">
          <div class="p-4">
            <div class="flex items-start justify-between">
              <div>
                <h3 class="font-semibold">{{ plugin.name }}</h3>
                <p class="text-xs opacity-60">v{{ plugin.version }} by {{ plugin.author }}</p>
              </div>
              <span v-if="plugin.category" class="badge badge-xs badge-outline">{{ plugin.category }}</span>
            </div>
            <p class="text-sm mt-2 line-clamp-3">{{ plugin.description }}</p>

            <div class="flex items-center justify-between mt-4">
              <a
                v-if="plugin.repository"
                :href="plugin.repository"
                target="_blank"
                rel="noopener"
                class="link link-primary text-xs"
              >
                View on GitHub
              </a>

              <button
                v-if="plugin.installed"
                class="btn btn-xs btn-success btn-outline"
                disabled
              >
                Installed
              </button>
              <button
                v-else
                class="btn btn-xs btn-primary"
                :class="{ loading: installing === plugin.name }"
                :disabled="installing !== null"
                @click="installPlugin(plugin)"
              >
                Install
              </button>
            </div>
          </div>
        </BaseCard>
      </div>

      <!-- Empty State -->
      <div v-if="filteredPlugins.length === 0" class="text-center py-12 opacity-50">
        <p class="text-lg">No plugins found</p>
        <p v-if="searchQuery" class="text-sm mt-2">Try adjusting your search query.</p>
        <p v-else class="text-sm mt-2">The catalog is empty or hasn't been loaded yet.</p>
      </div>
    </BaseContainer>
  </div>
</template>
