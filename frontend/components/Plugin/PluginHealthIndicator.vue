<script setup lang="ts">
  const props = defineProps<{
    status: "healthy" | "degraded" | "error" | "disabled" | "unknown";
    lastCheck: string;
    message?: string;
  }>();

  const statusConfig = computed(() => {
    const map: Record<string, { dot: string; text: string; label: string }> = {
      healthy: { dot: "bg-success", text: "text-success", label: "Healthy" },
      degraded: { dot: "bg-warning", text: "text-warning", label: "Degraded" },
      error: { dot: "bg-error", text: "text-error", label: "Error" },
      disabled: { dot: "bg-base-300", text: "text-base-content/50", label: "Disabled" },
      unknown: { dot: "bg-base-300", text: "text-base-content/50", label: "Unknown" },
    };
    return map[props.status] || map.unknown;
  });

  const formattedLastCheck = computed(() => {
    if (!props.lastCheck) return "Never";
    try {
      const date = new Date(props.lastCheck);
      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const diffMins = Math.floor(diffMs / 60000);

      if (diffMins < 1) return "Just now";
      if (diffMins < 60) return `${diffMins}m ago`;
      const diffHours = Math.floor(diffMins / 60);
      if (diffHours < 24) return `${diffHours}h ago`;
      const diffDays = Math.floor(diffHours / 24);
      return `${diffDays}d ago`;
    } catch {
      return props.lastCheck;
    }
  });

  const tooltipText = computed(() => {
    let tip = `Status: ${statusConfig.value.label}`;
    tip += `\nLast check: ${props.lastCheck || "Never"}`;
    if (props.message) tip += `\n${props.message}`;
    return tip;
  });
</script>

<template>
  <div class="inline-flex items-center gap-2" :title="tooltipText">
    <!-- Animated pulse dot for healthy, static for others -->
    <span class="relative flex h-2.5 w-2.5">
      <span
        v-if="status === 'healthy'"
        class="animate-ping absolute inline-flex h-full w-full rounded-full opacity-75"
        :class="statusConfig.dot"
      />
      <span
        class="relative inline-flex rounded-full h-2.5 w-2.5"
        :class="statusConfig.dot"
      />
    </span>

    <!-- Label -->
    <span class="text-xs font-medium" :class="statusConfig.text">
      {{ statusConfig.label }}
    </span>

    <!-- Last check time -->
    <span class="text-[10px] opacity-40">
      {{ formattedLastCheck }}
    </span>

    <!-- Message indicator -->
    <span
      v-if="message"
      class="text-[10px] opacity-50 truncate max-w-[150px]"
      :title="message"
    >
      - {{ message }}
    </span>
  </div>
</template>
