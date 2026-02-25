<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import { Input } from "~/components/ui/input";
  import { Button } from "~/components/ui/button";
  import { Badge } from "~/components/ui/badge";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Batteries & Storage",
  });

  const api = useUserApi();

  // Interfaces
  interface BatteryModel {
    sku: string;
    label: string;
    ah: number;
    highOutput: boolean;
    cells?: string;
    weightLbs?: number;
    notes?: string;
  }

  interface ChargerModel {
    sku: string;
    label: string;
    ports: number;
    rapid: boolean;
    platforms?: string[];
    notes?: string;
  }

  interface BatteryPlatform {
    slug: string;
    brand: string;
    name: string;
    nominalVolts: number;
    description: string;
    batteries: BatteryModel[];
    chargers: ChargerModel[];
    color: string;
  }

  interface StorageItem {
    sku: string;
    label: string;
    category: string;
    description?: string;
  }

  interface StorageSystem {
    slug: string;
    brand: string;
    name: string;
    description: string;
    color: string;
    items: StorageItem[];
  }

  // Fetch data
  const { data: platforms } = useAsyncData("battery-platforms", async () => {
    const { data } = await api.http.get<BatteryPlatform[]>({
      url: "/api/v1/plugins/batteries/platforms",
    });
    return data || [];
  });

  const { data: storageSystems } = useAsyncData("storage-systems", async () => {
    const { data } = await api.http.get<StorageSystem[]>({
      url: "/api/v1/plugins/batteries/storage",
    });
    return data || [];
  });

  // State
  const activeTab = ref<"batteries" | "storage">("batteries");
  const selectedPlatform = ref<BatteryPlatform | null>(null);
  const selectedStorage = ref<StorageSystem | null>(null);
  const searchQuery = ref("");

  // Computed
  const filteredPlatforms = computed(() => {
    if (!platforms.value) return [];
    if (!searchQuery.value) return platforms.value;
    const q = searchQuery.value.toLowerCase();
    return platforms.value.filter(
      p => p.brand.toLowerCase().includes(q) ||
           p.name.toLowerCase().includes(q) ||
           p.description.toLowerCase().includes(q)
    );
  });

  const filteredStorage = computed(() => {
    if (!storageSystems.value) return [];
    if (!searchQuery.value) return storageSystems.value;
    const q = searchQuery.value.toLowerCase();
    return storageSystems.value.filter(
      s => s.brand.toLowerCase().includes(q) ||
           s.name.toLowerCase().includes(q) ||
           s.description.toLowerCase().includes(q)
    );
  });

  // Unique brands for batteries
  const brands = computed(() => {
    if (!platforms.value) return [];
    const seen = new Set<string>();
    return platforms.value
      .filter(p => { if (seen.has(p.brand)) return false; seen.add(p.brand); return true; })
      .map(p => ({ brand: p.brand, color: p.color }));
  });

  // Stats
  const totalBatteries = computed(() =>
    platforms.value?.reduce((sum, p) => sum + p.batteries.length, 0) || 0
  );
  const totalChargers = computed(() =>
    platforms.value?.reduce((sum, p) => sum + p.chargers.length, 0) || 0
  );
  const totalStorageItems = computed(() =>
    storageSystems.value?.reduce((sum, s) => sum + s.items.length, 0) || 0
  );

  // Category label for storage
  function storageCategoryLabel(category: string): string {
    const map: Record<string, string> = {
      toolbox: "Tool Box",
      organizer: "Organizer",
      bag: "Bag/Tote",
      "wall-mount": "Wall Mount",
      accessory: "Accessory",
    };
    return map[category] || category;
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Batteries & Storage</h1>
          <p class="text-sm text-muted-foreground">
            Track power tool batteries, chargers, and modular storage systems.
          </p>
        </div>
        <NuxtLink to="/plugins">
          <Button variant="outline" size="sm">Back to Plugins</Button>
        </NuxtLink>
      </div>

      <!-- Stats -->
      <div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <BaseCard>
          <div class="p-4 text-center">
            <p class="text-2xl font-bold text-primary">{{ platforms?.length || 0 }}</p>
            <p class="text-xs text-muted-foreground">Platforms</p>
          </div>
        </BaseCard>
        <BaseCard>
          <div class="p-4 text-center">
            <p class="text-2xl font-bold text-primary">{{ totalBatteries }}</p>
            <p class="text-xs text-muted-foreground">Battery Models</p>
          </div>
        </BaseCard>
        <BaseCard>
          <div class="p-4 text-center">
            <p class="text-2xl font-bold text-primary">{{ totalChargers }}</p>
            <p class="text-xs text-muted-foreground">Charger Models</p>
          </div>
        </BaseCard>
        <BaseCard>
          <div class="p-4 text-center">
            <p class="text-2xl font-bold text-primary">{{ totalStorageItems }}</p>
            <p class="text-xs text-muted-foreground">Storage Products</p>
          </div>
        </BaseCard>
      </div>

      <!-- Brand Chips -->
      <div class="flex flex-wrap gap-2">
        <div
          v-for="b in brands"
          :key="b.brand"
          class="flex items-center gap-2 rounded-full border border-border px-3 py-1.5"
        >
          <span class="size-3 rounded-full" :style="{ backgroundColor: b.color }" />
          <span class="text-xs font-medium">{{ b.brand }}</span>
        </div>
      </div>

      <!-- Tab Bar -->
      <div class="flex items-center gap-1 overflow-x-auto border-b border-border pb-0">
        <button
          v-for="tab in [
            { key: 'batteries' as const, label: 'Battery Platforms', count: platforms?.length || 0 },
            { key: 'storage' as const, label: 'Storage Systems', count: storageSystems?.length || 0 },
          ]"
          :key="tab.key"
          class="flex shrink-0 items-center gap-1.5 border-b-2 px-4 py-2.5 text-sm font-medium transition-colors"
          :class="
            activeTab === tab.key
              ? 'border-primary text-primary'
              : 'border-transparent text-muted-foreground hover:border-muted hover:text-foreground'
          "
          @click="activeTab = tab.key; selectedPlatform = null; selectedStorage = null"
        >
          {{ tab.label }}
          <Badge variant="secondary" class="ml-0.5 h-5 min-w-[20px] px-1.5 text-[10px]">
            {{ tab.count }}
          </Badge>
        </button>
      </div>

      <!-- Search -->
      <Input
        v-model="searchQuery"
        class="h-10"
        :placeholder="activeTab === 'batteries' ? 'Search battery platforms...' : 'Search storage systems...'"
        type="search"
      />

      <!-- BATTERIES TAB -->
      <div v-if="activeTab === 'batteries'">
        <!-- Platform Detail -->
        <div v-if="selectedPlatform" class="space-y-6">
          <div class="flex items-center gap-3">
            <Button variant="ghost" size="sm" @click="selectedPlatform = null">Back</Button>
            <span class="size-4 rounded-full" :style="{ backgroundColor: selectedPlatform.color }" />
            <h2 class="text-lg font-semibold">{{ selectedPlatform.brand }} {{ selectedPlatform.name }}</h2>
            <Badge variant="secondary">{{ selectedPlatform.nominalVolts }}V</Badge>
          </div>

          <p class="text-sm text-muted-foreground">{{ selectedPlatform.description }}</p>

          <!-- Batteries Table -->
          <section>
            <h3 class="mb-3 text-sm font-semibold">Batteries ({{ selectedPlatform.batteries.length }})</h3>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="border-b text-left">
                    <th class="pb-2 font-medium">SKU</th>
                    <th class="pb-2 font-medium">Model</th>
                    <th class="pb-2 font-medium text-right">Ah</th>
                    <th class="pb-2 font-medium text-center">High Output</th>
                    <th class="pb-2 font-medium hidden sm:table-cell">Notes</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="bat in selectedPlatform.batteries"
                    :key="bat.sku"
                    class="border-b border-border/50"
                  >
                    <td class="py-2">
                      <Badge variant="secondary" class="font-mono text-[10px]">{{ bat.sku }}</Badge>
                    </td>
                    <td class="py-2 font-medium">{{ bat.label }}</td>
                    <td class="py-2 text-right font-semibold">{{ bat.ah }}</td>
                    <td class="py-2 text-center">
                      <Badge v-if="bat.highOutput" class="bg-orange-500/10 text-orange-500 text-[10px]">HO</Badge>
                      <span v-else class="text-muted-foreground">-</span>
                    </td>
                    <td class="py-2 text-xs text-muted-foreground hidden sm:table-cell">
                      {{ bat.notes || '-' }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <!-- Chargers Table -->
          <section>
            <h3 class="mb-3 text-sm font-semibold">Chargers ({{ selectedPlatform.chargers.length }})</h3>
            <div class="overflow-x-auto">
              <table class="w-full text-sm">
                <thead>
                  <tr class="border-b text-left">
                    <th class="pb-2 font-medium">SKU</th>
                    <th class="pb-2 font-medium">Model</th>
                    <th class="pb-2 font-medium text-center">Ports</th>
                    <th class="pb-2 font-medium text-center">Rapid</th>
                    <th class="pb-2 font-medium hidden sm:table-cell">Notes</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="chg in selectedPlatform.chargers"
                    :key="chg.sku"
                    class="border-b border-border/50"
                  >
                    <td class="py-2">
                      <Badge variant="secondary" class="font-mono text-[10px]">{{ chg.sku }}</Badge>
                    </td>
                    <td class="py-2 font-medium">{{ chg.label }}</td>
                    <td class="py-2 text-center font-semibold">{{ chg.ports }}</td>
                    <td class="py-2 text-center">
                      <Badge v-if="chg.rapid" class="bg-green-500/10 text-green-500 text-[10px]">Rapid</Badge>
                      <span v-else class="text-muted-foreground">-</span>
                    </td>
                    <td class="py-2 text-xs text-muted-foreground hidden sm:table-cell">
                      {{ chg.notes || (chg.platforms && chg.platforms.length > 1 ? `Cross-platform: ${chg.platforms.join(', ')}` : '-') }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>
        </div>

        <!-- Platform Grid -->
        <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="platform in filteredPlatforms"
            :key="platform.slug"
            class="group cursor-pointer rounded-xl border border-border bg-card p-4 shadow-sm transition-all hover:border-primary/40 hover:shadow-md"
            @click="selectedPlatform = platform"
          >
            <div class="mb-3 flex items-center justify-between">
              <div class="flex items-center gap-2">
                <span class="size-4 rounded-full" :style="{ backgroundColor: platform.color }" />
                <h3 class="text-sm font-semibold group-hover:text-primary">
                  {{ platform.brand }} {{ platform.name }}
                </h3>
              </div>
              <Badge variant="secondary">{{ platform.nominalVolts }}V</Badge>
            </div>
            <p class="mb-3 text-xs text-muted-foreground line-clamp-2">{{ platform.description }}</p>
            <div class="flex items-center gap-4 text-xs text-muted-foreground">
              <span>{{ platform.batteries.length }} batteries</span>
              <span>{{ platform.chargers.length }} chargers</span>
            </div>
            <div class="mt-2 flex flex-wrap gap-1">
              <Badge
                v-for="bat in platform.batteries.filter(b => b.highOutput).slice(0, 2)"
                :key="bat.sku"
                class="bg-orange-500/10 text-orange-500 text-[10px]"
              >
                {{ bat.ah }}Ah HO
              </Badge>
              <Badge
                v-for="bat in platform.batteries.filter(b => !b.highOutput).slice(0, 3)"
                :key="bat.sku"
                variant="secondary"
                class="text-[10px]"
              >
                {{ bat.ah }}Ah
              </Badge>
            </div>
          </div>
        </div>
      </div>

      <!-- STORAGE TAB -->
      <div v-if="activeTab === 'storage'">
        <!-- Storage Detail -->
        <div v-if="selectedStorage" class="space-y-6">
          <div class="flex items-center gap-3">
            <Button variant="ghost" size="sm" @click="selectedStorage = null">Back</Button>
            <span class="size-4 rounded-full" :style="{ backgroundColor: selectedStorage.color }" />
            <h2 class="text-lg font-semibold">{{ selectedStorage.brand }} {{ selectedStorage.name }}</h2>
          </div>

          <p class="text-sm text-muted-foreground">{{ selectedStorage.description }}</p>

          <!-- Group by category -->
          <div
            v-for="category in [...new Set(selectedStorage.items.map(i => i.category))]"
            :key="category"
            class="space-y-3"
          >
            <h3 class="text-sm font-semibold">{{ storageCategoryLabel(category) }}</h3>
            <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
              <div
                v-for="item in selectedStorage.items.filter(i => i.category === category)"
                :key="item.sku"
                class="rounded-lg border border-border p-3"
              >
                <div class="flex items-center justify-between">
                  <p class="text-sm font-medium">{{ item.label }}</p>
                  <Badge variant="secondary" class="font-mono text-[10px]">{{ item.sku }}</Badge>
                </div>
                <p v-if="item.description" class="mt-1 text-xs text-muted-foreground">{{ item.description }}</p>
              </div>
            </div>
          </div>
        </div>

        <!-- Storage Grid -->
        <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="system in filteredStorage"
            :key="system.slug"
            class="group cursor-pointer rounded-xl border border-border bg-card p-4 shadow-sm transition-all hover:border-primary/40 hover:shadow-md"
            @click="selectedStorage = system"
          >
            <div class="mb-3 flex items-center gap-2">
              <span class="size-4 rounded-full" :style="{ backgroundColor: system.color }" />
              <h3 class="text-sm font-semibold group-hover:text-primary">
                {{ system.brand }} {{ system.name }}
              </h3>
            </div>
            <p class="mb-3 text-xs text-muted-foreground line-clamp-2">{{ system.description }}</p>
            <div class="flex items-center gap-2 text-xs text-muted-foreground">
              <span>{{ system.items.length }} products</span>
            </div>
            <div class="mt-2 flex flex-wrap gap-1">
              <Badge
                v-for="cat in [...new Set(system.items.map(i => i.category))]"
                :key="cat"
                variant="secondary"
                class="text-[10px]"
              >
                {{ storageCategoryLabel(cat) }}
              </Badge>
            </div>
          </div>
        </div>
      </div>
    </BaseContainer>
  </div>
</template>
