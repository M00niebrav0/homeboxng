<script setup lang="ts">
  interface ConfigField {
    key: string;
    label: string;
    description: string;
    type: string;
    default: string;
    options?: string[];
    required: boolean;
    value: string;
    envVar?: string;
  }

  const props = defineProps<{
    field: ConfigField;
  }>();

  const emit = defineEmits<{
    "update:value": [newValue: string];
  }>();

  const localValue = computed({
    get: () => props.field.value ?? props.field.default ?? "",
    set: (val: string) => emit("update:value", val),
  });

  const showSecret = ref(false);

  function toggleSecret() {
    showSecret.value = !showSecret.value;
  }

  const isValidUrl = computed(() => {
    if (props.field.type !== "url" || !localValue.value) return null;
    try {
      new URL(localValue.value);
      return true;
    } catch {
      return false;
    }
  });

  const urlInputClass = computed(() => {
    if (isValidUrl.value === null) return "";
    return isValidUrl.value ? "input-success" : "input-error";
  });

  const booleanValue = computed({
    get: () => localValue.value === "true",
    set: (val: boolean) => emit("update:value", val ? "true" : "false"),
  });
</script>

<template>
  <div class="form-control w-full">
    <!-- Label -->
    <label class="label">
      <span class="label-text font-medium">
        {{ field.label }}
        <span v-if="field.required" class="text-error ml-0.5">*</span>
      </span>
    </label>

    <!-- String input -->
    <input
      v-if="field.type === 'string'"
      v-model="localValue"
      type="text"
      :placeholder="field.default || ''"
      class="input input-bordered input-sm w-full"
    />

    <!-- Number input -->
    <input
      v-else-if="field.type === 'number'"
      v-model="localValue"
      type="number"
      :placeholder="field.default || '0'"
      class="input input-bordered input-sm w-full"
    />

    <!-- Boolean toggle -->
    <div v-else-if="field.type === 'boolean'" class="flex items-center gap-2 py-1">
      <input
        v-model="booleanValue"
        type="checkbox"
        class="toggle toggle-primary toggle-sm"
      />
      <span class="text-sm opacity-70">{{ booleanValue ? "Enabled" : "Disabled" }}</span>
    </div>

    <!-- Secret / password input -->
    <div v-else-if="field.type === 'secret'" class="join w-full">
      <input
        v-model="localValue"
        :type="showSecret ? 'text' : 'password'"
        :placeholder="field.default || 'Enter secret...'"
        class="input input-bordered input-sm join-item flex-1"
      />
      <button
        type="button"
        class="btn btn-sm btn-outline join-item"
        :title="showSecret ? 'Hide' : 'Show'"
        @click="toggleSecret"
      >
        {{ showSecret ? "Hide" : "Show" }}
      </button>
    </div>

    <!-- Select dropdown -->
    <select
      v-else-if="field.type === 'select'"
      v-model="localValue"
      class="select select-bordered select-sm w-full"
    >
      <option value="" disabled>Select an option</option>
      <option v-for="opt in field.options" :key="opt" :value="opt">{{ opt }}</option>
    </select>

    <!-- URL input with validation -->
    <div v-else-if="field.type === 'url'" class="relative">
      <input
        v-model="localValue"
        type="url"
        :placeholder="field.default || 'https://...'"
        class="input input-bordered input-sm w-full pr-8"
        :class="urlInputClass"
      />
      <span
        v-if="isValidUrl !== null"
        class="absolute right-2 top-1/2 -translate-y-1/2 text-xs"
        :class="isValidUrl ? 'text-success' : 'text-error'"
      >
        {{ isValidUrl ? "\u2713" : "\u2717" }}
      </span>
    </div>

    <!-- Textarea -->
    <textarea
      v-else-if="field.type === 'textarea'"
      v-model="localValue"
      :placeholder="field.default || ''"
      class="textarea textarea-bordered textarea-sm w-full"
      rows="3"
    />

    <!-- Fallback: plain text input -->
    <input
      v-else
      v-model="localValue"
      type="text"
      :placeholder="field.default || ''"
      class="input input-bordered input-sm w-full"
    />

    <!-- Description helper text -->
    <label v-if="field.description" class="label pb-0">
      <span class="label-text-alt opacity-60">{{ field.description }}</span>
    </label>

    <!-- Environment variable hint -->
    <label v-if="field.envVar" class="label pt-0">
      <span class="label-text-alt opacity-40 font-mono text-[11px]">
        or set <code class="bg-base-200 px-1 py-0.5 rounded text-[10px]">{{ field.envVar }}</code>
      </span>
    </label>
  </div>
</template>
