<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import { Button } from "@/components/ui/button";
  import { Input } from "@/components/ui/input";
  import { Badge } from "@/components/ui/badge";
  import type { CatalogPlugin } from "~/lib/api/classes/plugins";
  import MdiLoading from "~icons/mdi/loading";

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
    try {
      const { data } = await api.plugins.getCatalog();
      return data;
    } catch {
      return [];
    }
  });

  const categories = computed(() => {
    if (!catalog.value) return [];
    const cats = new Set((catalog.value as CatalogPlugin[]).map(p => p.category || "Other"));
    return ["all", ...Array.from(cats).sort()];
  });

  const filteredPlugins = computed(() => {
    if (!catalog.value) return [];

    return (catalog.value as CatalogPlugin[]).filter(p => {
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
          <p class="text-sm text-muted-foreground">
            Browse and install community plugins from GitHub repositories.
          </p>
        </div>
        <div class="flex gap-2">
          <NuxtLink to="/plugins">
            <Button variant="outline" size="sm">
              Back to Manager
            </Button>
          </NuxtLink>
          <Button size="sm" @click="doRefreshCatalog">
            Refresh Catalog
          </Button>
        </div>
      </div>

      <!-- Search & Filter -->
      <div class="flex flex-wrap gap-4">
        <Input
          v-model="searchQuery"
          type="text"
          placeholder="Search plugins..."
          class="min-w-[200px] flex-1"
        />
        <div class="flex gap-1">
          <Button
            v-for="cat in categories"
            :key="cat"
            :variant="selectedCategory === cat ? 'default' : 'ghost'"
            size="sm"
            @click="selectedCategory = cat"
          >
            {{ cat }}
          </Button>
        </div>
      </div>

      <!-- Plugin Grid -->
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        <BaseCard v-for="plugin in filteredPlugins" :key="plugin.name">
          <div class="p-4">
            <div class="flex items-start justify-between">
              <div>
                <h3 class="font-semibold">{{ plugin.name }}</h3>
                <p class="text-xs text-muted-foreground">v{{ plugin.version }} by {{ plugin.author }}</p>
              </div>
              <Badge v-if="plugin.category" variant="outline">{{ plugin.category }}</Badge>
            </div>
            <p class="mt-2 line-clamp-3 text-sm">{{ plugin.description }}</p>

            <div class="mt-4 flex items-center justify-between">
              <a
                v-if="plugin.repository"
                :href="plugin.repository"
                target="_blank"
                rel="noopener"
                class="text-xs text-primary hover:underline"
              >
                View on GitHub
              </a>

              <Button
                v-if="plugin.installed"
                variant="outline"
                size="sm"
                disabled
                class="text-green-500"
              >
                Installed
              </Button>
              <Button
                v-else
                size="sm"
                :disabled="installing !== null"
                @click="installPlugin(plugin)"
              >
                <MdiLoading v-if="installing === plugin.name" class="mr-1 size-3 animate-spin" />
                Install
              </Button>
            </div>
          </div>
        </BaseCard>
      </div>

      <!-- Empty State -->
      <div v-if="filteredPlugins.length === 0" class="py-12 text-center text-muted-foreground">
        <p class="text-lg">No plugins found</p>
        <p v-if="searchQuery" class="mt-2 text-sm">Try adjusting your search query.</p>
        <p v-else class="mt-2 text-sm">The catalog is empty or hasn't been loaded yet.</p>
      </div>
    </BaseContainer>
  </div>
</template>
