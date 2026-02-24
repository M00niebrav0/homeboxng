<script setup lang="ts">
  const props = defineProps<{
    name: string;
    slug: string;
    description: string;
    icon: string;
    version: string;
    status: "enabled" | "disabled" | "error";
    permissionCount: number;
    builtIn: boolean;
    category: string;
  }>();

  const router = useRouter();

  function navigateToPlugin() {
    router.push(`/plugins/${props.slug}`);
  }

  const statusDotClass = computed(() => {
    switch (props.status) {
      case "enabled":
        return "bg-success";
      case "disabled":
        return "bg-base-300";
      case "error":
        return "bg-error";
      default:
        return "bg-base-300";
    }
  });

  const statusLabel = computed(() => {
    switch (props.status) {
      case "enabled":
        return "Enabled";
      case "disabled":
        return "Disabled";
      case "error":
        return "Error";
      default:
        return "Unknown";
    }
  });

  const categoryBadgeClass = computed(() => {
    const map: Record<string, string> = {
      Data: "badge-info",
      Analytics: "badge-secondary",
      Integration: "badge-accent",
      Utility: "badge-warning",
      System: "badge-neutral",
    };
    return map[props.category] || "badge-ghost";
  });
</script>

<template>
  <div
    class="card bg-base-100 shadow-md border border-base-200 cursor-pointer transition-all duration-200 hover:shadow-lg hover:scale-[1.02] hover:border-primary/30 relative overflow-hidden"
    @click="navigateToPlugin"
  >
    <div class="card-body p-4 gap-2">
      <!-- Top row: icon + name + status dot -->
      <div class="flex items-start gap-3">
        <div class="text-2xl flex-shrink-0 w-10 h-10 rounded-lg bg-base-200 flex items-center justify-center">
          {{ icon }}
        </div>
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2">
            <h3 class="font-semibold text-sm truncate">{{ name }}</h3>
            <span
              class="w-2 h-2 rounded-full flex-shrink-0"
              :class="statusDotClass"
              :title="statusLabel"
            />
          </div>
          <p class="text-xs opacity-50 mt-0.5 line-clamp-2">{{ description }}</p>
        </div>
      </div>

      <!-- Bottom row: badges -->
      <div class="flex items-center gap-1.5 flex-wrap mt-1">
        <span class="badge badge-xs" :class="categoryBadgeClass">{{ category }}</span>
        <span v-if="builtIn" class="badge badge-xs badge-outline">Built-in</span>
        <span v-if="permissionCount > 0" class="badge badge-xs badge-ghost">
          {{ permissionCount }} permission{{ permissionCount !== 1 ? "s" : "" }}
        </span>
      </div>
    </div>

    <!-- Version corner -->
    <div class="absolute top-2 right-2">
      <span class="text-[10px] opacity-40 font-mono">v{{ version }}</span>
    </div>
  </div>
</template>
