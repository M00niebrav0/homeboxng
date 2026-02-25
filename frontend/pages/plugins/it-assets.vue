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
    title: "HomeBoxNG | IT Assets",
  });

  const api = useUserApi();

  // Interfaces matching backend types
  interface AssetTemplate {
    slug: string;
    label: string;
    description: string;
    formFactors: string[];
    components: {
      cpu: boolean;
      memory: boolean;
      storage: boolean;
      pcie: boolean;
      gpu: boolean;
      nic: boolean;
      psu: boolean;
      display: boolean;
    };
    suggestedLabels: string[];
  }

  interface HardwareLabel {
    key: string;
    label: string;
    description: string;
    icon: string;
  }

  interface ConnectionProtocol {
    id: string;
    label: string;
    defaultPort: number;
    tools: string[];
  }

  interface ComponentFieldSchema {
    componentType: string;
    label: string;
    fields: {
      key: string;
      label: string;
      type: string;
      options?: string[];
      required: boolean;
      description?: string;
    }[];
  }

  // Fetch data from backend
  const { data: templates } = useAsyncData("it-templates", async () => {
    const { data } = await api.http.get<AssetTemplate[]>({
      url: "/api/v1/plugins/it-assets/templates",
    });
    return data || [];
  });

  const { data: labels } = useAsyncData("it-labels", async () => {
    const { data } = await api.http.get<HardwareLabel[]>({
      url: "/api/v1/plugins/it-assets/labels",
    });
    return data || [];
  });

  const { data: protocols } = useAsyncData("it-protocols", async () => {
    const { data } = await api.http.get<ConnectionProtocol[]>({
      url: "/api/v1/plugins/it-assets/protocols",
    });
    return data || [];
  });

  const { data: schemas } = useAsyncData("it-schemas", async () => {
    const { data } = await api.http.get<ComponentFieldSchema[]>({
      url: "/api/v1/plugins/it-assets/component-schemas",
    });
    return data || [];
  });

  // State
  const activeTab = ref<"templates" | "components" | "protocols" | "labels">("templates");
  const searchQuery = ref("");
  const selectedTemplate = ref<AssetTemplate | null>(null);
  const selectedSchema = ref<ComponentFieldSchema | null>(null);

  // Filtered templates
  const filteredTemplates = computed(() => {
    if (!templates.value) return [];
    if (!searchQuery.value) return templates.value;
    const q = searchQuery.value.toLowerCase();
    return templates.value.filter(
      t => t.label.toLowerCase().includes(q) || t.description.toLowerCase().includes(q)
    );
  });

  // Component count for a template
  function componentCount(t: AssetTemplate): number {
    let count = 0;
    if (t.components.cpu) count++;
    if (t.components.memory) count++;
    if (t.components.storage) count++;
    if (t.components.pcie) count++;
    if (t.components.gpu) count++;
    if (t.components.nic) count++;
    if (t.components.psu) count++;
    if (t.components.display) count++;
    return count;
  }

  // Icon map for templates
  function templateIcon(slug: string): string {
    const icons: Record<string, string> = {
      "server-rackmount": "mdi-server",
      "server-tower": "mdi-server",
      "server-blade": "mdi-server-network",
      "desktop-pc": "mdi-desktop-tower-monitor",
      "laptop": "mdi-laptop",
      "nas": "mdi-nas",
      "network-switch": "mdi-switch",
      "router-firewall": "mdi-router-wireless",
      "ups": "mdi-battery-charging",
      "kvm": "mdi-monitor-multiple",
      "monitor": "mdi-monitor",
      "printer": "mdi-printer",
    };
    return icons[slug] || "mdi-server";
  }

  const tabs = [
    { key: "templates" as const, label: "Asset Templates", count: () => templates.value?.length || 0 },
    { key: "components" as const, label: "Component Schemas", count: () => schemas.value?.length || 0 },
    { key: "protocols" as const, label: "Connection Protocols", count: () => protocols.value?.length || 0 },
    { key: "labels" as const, label: "Hardware Labels", count: () => labels.value?.length || 0 },
  ];
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">IT Asset Manager</h1>
          <p class="text-sm text-muted-foreground">
            Track servers, desktops, laptops, and networking equipment with detailed hardware components.
          </p>
        </div>
        <NuxtLink to="/plugins">
          <Button variant="outline" size="sm">Back to Plugins</Button>
        </NuxtLink>
      </div>

      <!-- Stats Cards -->
      <div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <BaseCard>
          <div class="p-4 text-center">
            <p class="text-2xl font-bold text-primary">{{ templates?.length || 0 }}</p>
            <p class="text-xs text-muted-foreground">Asset Types</p>
          </div>
        </BaseCard>
        <BaseCard>
          <div class="p-4 text-center">
            <p class="text-2xl font-bold text-primary">{{ schemas?.length || 0 }}</p>
            <p class="text-xs text-muted-foreground">Component Types</p>
          </div>
        </BaseCard>
        <BaseCard>
          <div class="p-4 text-center">
            <p class="text-2xl font-bold text-primary">{{ protocols?.length || 0 }}</p>
            <p class="text-xs text-muted-foreground">Protocols</p>
          </div>
        </BaseCard>
        <BaseCard>
          <div class="p-4 text-center">
            <p class="text-2xl font-bold text-primary">{{ labels?.length || 0 }}</p>
            <p class="text-xs text-muted-foreground">Labels</p>
          </div>
        </BaseCard>
      </div>

      <!-- Tab Bar -->
      <div class="flex items-center gap-1 overflow-x-auto border-b border-border pb-0">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="flex shrink-0 items-center gap-1.5 border-b-2 px-4 py-2.5 text-sm font-medium transition-colors"
          :class="
            activeTab === tab.key
              ? 'border-primary text-primary'
              : 'border-transparent text-muted-foreground hover:border-muted hover:text-foreground'
          "
          @click="activeTab = tab.key; selectedTemplate = null; selectedSchema = null"
        >
          {{ tab.label }}
          <Badge variant="secondary" class="ml-0.5 h-5 min-w-[20px] px-1.5 text-[10px]">
            {{ tab.count() }}
          </Badge>
        </button>
      </div>

      <!-- Search (for templates tab) -->
      <div v-if="activeTab === 'templates'" class="relative">
        <Input
          v-model="searchQuery"
          class="h-10 pl-4"
          placeholder="Search asset templates..."
          type="search"
        />
      </div>

      <!-- TEMPLATES TAB -->
      <div v-if="activeTab === 'templates'">
        <!-- Template Detail -->
        <div v-if="selectedTemplate" class="space-y-4">
          <div class="flex items-center gap-2">
            <Button variant="ghost" size="sm" @click="selectedTemplate = null">Back</Button>
            <h2 class="text-lg font-semibold">{{ selectedTemplate.label }}</h2>
          </div>

          <BaseCard>
            <div class="p-6 space-y-4">
              <p class="text-sm text-muted-foreground">{{ selectedTemplate.description }}</p>

              <div>
                <h3 class="text-sm font-medium mb-2">Form Factors</h3>
                <div class="flex flex-wrap gap-2">
                  <Badge v-for="ff in selectedTemplate.formFactors" :key="ff" variant="secondary">
                    {{ ff }}
                  </Badge>
                </div>
              </div>

              <div>
                <h3 class="text-sm font-medium mb-2">Hardware Components</h3>
                <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
                  <div
                    v-for="(enabled, name) in selectedTemplate.components"
                    :key="name"
                    class="flex items-center gap-2 rounded-lg border p-2 text-xs"
                    :class="enabled ? 'border-primary/30 bg-primary/5 text-foreground' : 'border-border bg-muted/50 text-muted-foreground line-through'"
                  >
                    <span class="size-2 rounded-full" :class="enabled ? 'bg-green-500' : 'bg-gray-300'" />
                    {{ String(name).toUpperCase() }}
                  </div>
                </div>
              </div>

              <div v-if="selectedTemplate.suggestedLabels.length > 0">
                <h3 class="text-sm font-medium mb-2">Suggested Labels</h3>
                <div class="flex flex-wrap gap-2">
                  <Badge v-for="label in selectedTemplate.suggestedLabels" :key="label">
                    {{ label }}
                  </Badge>
                </div>
              </div>
            </div>
          </BaseCard>
        </div>

        <!-- Template Grid -->
        <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="template in filteredTemplates"
            :key="template.slug"
            class="group cursor-pointer rounded-xl border border-border bg-card p-4 shadow-sm transition-all hover:border-primary/40 hover:shadow-md"
            @click="selectedTemplate = template"
          >
            <div class="mb-3 flex items-center gap-3">
              <div class="flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary">
                <span class="text-lg">{{ template.slug.includes('server') ? '&#x1F5A5;' : template.slug === 'laptop' ? '&#x1F4BB;' : template.slug === 'nas' ? '&#x1F4BE;' : template.slug.includes('switch') || template.slug.includes('router') ? '&#x1F310;' : template.slug === 'ups' ? '&#x1F50C;' : template.slug === 'printer' ? '&#x1F5A8;' : template.slug === 'monitor' ? '&#x1F5B5;' : '&#x2699;' }}</span>
              </div>
              <div class="min-w-0 flex-1">
                <h3 class="text-sm font-semibold group-hover:text-primary">{{ template.label }}</h3>
                <p class="text-xs text-muted-foreground">{{ componentCount(template) }} component types</p>
              </div>
            </div>
            <p class="mb-3 text-xs text-muted-foreground">{{ template.description }}</p>
            <div class="flex flex-wrap gap-1">
              <Badge
                v-for="ff in template.formFactors.slice(0, 3)"
                :key="ff"
                variant="secondary"
                class="text-[10px]"
              >
                {{ ff }}
              </Badge>
              <Badge
                v-if="template.formFactors.length > 3"
                variant="secondary"
                class="text-[10px]"
              >
                +{{ template.formFactors.length - 3 }}
              </Badge>
            </div>
          </div>
        </div>
      </div>

      <!-- COMPONENT SCHEMAS TAB -->
      <div v-if="activeTab === 'components'">
        <div v-if="selectedSchema" class="space-y-4">
          <div class="flex items-center gap-2">
            <Button variant="ghost" size="sm" @click="selectedSchema = null">Back</Button>
            <h2 class="text-lg font-semibold">{{ selectedSchema.label }} Fields</h2>
          </div>

          <BaseCard>
            <div class="p-4">
              <table class="w-full text-sm">
                <thead>
                  <tr class="border-b text-left">
                    <th class="pb-2 font-medium">Field</th>
                    <th class="pb-2 font-medium">Type</th>
                    <th class="pb-2 font-medium">Required</th>
                    <th class="pb-2 font-medium hidden sm:table-cell">Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr
                    v-for="field in selectedSchema.fields"
                    :key="field.key"
                    class="border-b border-border/50"
                  >
                    <td class="py-2 font-medium">{{ field.label }}</td>
                    <td class="py-2">
                      <Badge variant="secondary" class="text-[10px]">{{ field.type }}</Badge>
                    </td>
                    <td class="py-2">
                      <span v-if="field.required" class="text-green-500">Yes</span>
                      <span v-else class="text-muted-foreground">No</span>
                    </td>
                    <td class="py-2 text-xs text-muted-foreground hidden sm:table-cell">
                      {{ field.description || (field.options ? `Options: ${field.options.join(', ')}` : '-') }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </BaseCard>
        </div>

        <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <div
            v-for="schema in schemas"
            :key="schema.componentType"
            class="group cursor-pointer rounded-xl border border-border bg-card p-4 shadow-sm transition-all hover:border-primary/40 hover:shadow-md"
            @click="selectedSchema = schema"
          >
            <h3 class="text-sm font-semibold group-hover:text-primary">{{ schema.label }}</h3>
            <p class="mt-1 text-xs text-muted-foreground">{{ schema.fields.length }} fields</p>
            <div class="mt-2 flex flex-wrap gap-1">
              <Badge
                v-for="field in schema.fields.filter(f => f.required).slice(0, 3)"
                :key="field.key"
                class="text-[10px]"
              >
                {{ field.label }}
              </Badge>
              <Badge
                v-for="field in schema.fields.filter(f => !f.required).slice(0, 2)"
                :key="field.key"
                variant="secondary"
                class="text-[10px]"
              >
                {{ field.label }}
              </Badge>
            </div>
          </div>
        </div>
      </div>

      <!-- PROTOCOLS TAB -->
      <div v-if="activeTab === 'protocols'">
        <div class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b text-left">
                <th class="pb-2 font-medium">Protocol</th>
                <th class="pb-2 font-medium">Default Port</th>
                <th class="pb-2 font-medium">Compatible Tools</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="proto in protocols"
                :key="proto.id"
                class="border-b border-border/50"
              >
                <td class="py-2.5 font-medium">{{ proto.label }}</td>
                <td class="py-2.5">
                  <Badge variant="secondary">{{ proto.defaultPort || 'N/A' }}</Badge>
                </td>
                <td class="py-2.5">
                  <div class="flex flex-wrap gap-1">
                    <Badge
                      v-for="tool in proto.tools"
                      :key="tool"
                      variant="outline"
                      class="text-[10px]"
                    >
                      {{ tool }}
                    </Badge>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- LABELS TAB -->
      <div v-if="activeTab === 'labels'">
        <div class="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <BaseCard v-for="label in labels" :key="label.key">
            <div class="flex items-center gap-3 p-4">
              <div class="flex size-10 items-center justify-center rounded-lg bg-primary/10 text-primary text-lg">
                {{ label.icon === 'mdi-chip' ? '&#x1F4BB;' : label.icon === 'mdi-memory' ? '&#x1F4AC;' : label.icon === 'mdi-harddisk' ? '&#x1F4BE;' : label.icon === 'mdi-gpu' ? '&#x1F3AE;' : label.icon === 'mdi-ethernet' ? '&#x1F310;' : label.icon === 'mdi-flash' ? '&#x26A1;' : '&#x2699;' }}
              </div>
              <div class="min-w-0 flex-1">
                <h3 class="text-sm font-semibold">{{ label.label }}</h3>
                <p class="text-xs text-muted-foreground">{{ label.description }}</p>
                <Badge variant="secondary" class="mt-1 text-[10px]">{{ label.key }}</Badge>
              </div>
            </div>
          </BaseCard>
        </div>
      </div>
    </BaseContainer>
  </div>
</template>
