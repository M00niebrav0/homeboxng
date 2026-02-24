<script setup lang="ts">
  defineProps<{
    permission: string;
    risk: "low" | "medium" | "high";
    granted: boolean;
  }>();

  const riskColors: Record<string, string> = {
    low: "badge-info",
    medium: "badge-warning",
    high: "badge-error",
  };

  const riskIcons: Record<string, string> = {
    low: "~",
    medium: "!",
    high: "!!",
  };
</script>

<template>
  <div class="inline-flex items-center gap-1.5">
    <span class="badge badge-sm gap-1" :class="riskColors[risk]">
      <span class="font-mono text-[10px]">{{ riskIcons[risk] }}</span>
      {{ permission }}
    </span>
    <span
      class="w-4 h-4 rounded-full flex items-center justify-center text-[10px] font-bold"
      :class="granted ? 'bg-success text-success-content' : 'bg-error/20 text-error'"
      :title="granted ? 'Permission granted' : 'Permission denied'"
    >
      {{ granted ? "\u2713" : "\u2717" }}
    </span>
  </div>
</template>
