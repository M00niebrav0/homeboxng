<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import type { NotificationPlatform, NotificationCategory, PluginConfig } from "~/lib/api/classes/plugins";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Notification Settings",
  });

  const api = useUserApi();

  // Fetch notification platforms
  const { data: platforms, refresh: refreshPlatforms } = useAsyncData("notification-platforms", async () => {
    const { data } = await api.plugins.getNotificationPlatforms();
    return data || [];
  });

  // Fetch notification categories
  const { data: categories } = useAsyncData("notification-categories", async () => {
    const { data } = await api.plugins.getNotificationCategories();
    return data || [];
  });

  // Category preferences: { [categoryId]: { [platform]: boolean } }
  const categoryPrefs = ref<Record<string, Record<string, boolean>>>({});

  // Load existing preferences
  const { data: savedPrefs } = useAsyncData("notification-prefs", async () => {
    const { data } = await api.http.get<Record<string, Record<string, boolean>>>({
      url: "/api/v1/plugins/notifications/preferences",
    });
    return data;
  });

  watch(savedPrefs, (prefs) => {
    if (prefs) {
      categoryPrefs.value = prefs;
    }
  }, { immediate: true });

  // Config modal state
  const showConfigModal = ref(false);
  const selectedPlatform = ref<NotificationPlatform | null>(null);
  const platformConfig = ref<PluginConfig[]>([]);
  const configValues = ref<Record<string, string>>({});
  const testingPlatform = ref<string | null>(null);

  async function openConfig(platform: NotificationPlatform) {
    selectedPlatform.value = platform;
    const { data } = await api.plugins.getConfig(`notify-${platform.platform}`);
    platformConfig.value = data || [];
    configValues.value = {};
    for (const field of platformConfig.value) {
      configValues.value[field.key] = field.value || field.default || "";
    }
    showConfigModal.value = true;
  }

  async function saveConfig() {
    if (!selectedPlatform.value) return;
    await api.plugins.saveConfig(`notify-${selectedPlatform.value.platform}`, configValues.value);
    showConfigModal.value = false;
    refreshPlatforms();
  }

  async function testNotification(platform: NotificationPlatform) {
    testingPlatform.value = platform.platform;
    try {
      await api.plugins.testNotification(platform.platform);
    } finally {
      testingPlatform.value = null;
    }
  }

  async function togglePlatform(platform: NotificationPlatform) {
    if (platform.enabled) {
      await api.plugins.disable(`notify-${platform.platform}`);
    } else {
      await api.plugins.enable(`notify-${platform.platform}`);
    }
    refreshPlatforms();
  }

  async function toggleCategoryPref(categoryId: string, platformName: string) {
    if (!categoryPrefs.value[categoryId]) {
      categoryPrefs.value[categoryId] = {};
    }
    categoryPrefs.value[categoryId][platformName] = !categoryPrefs.value[categoryId][platformName];

    await api.http.put<Record<string, Record<string, boolean>>, void>({
      url: "/api/v1/plugins/notifications/preferences",
      body: categoryPrefs.value,
    });
  }

  function isCategoryEnabled(categoryId: string, platformName: string): boolean {
    return categoryPrefs.value[categoryId]?.[platformName] ?? false;
  }

  function platformIcon(name: string): string {
    const icons: Record<string, string> = {
      email: "M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z",
      discord: "M20.317 4.37a19.791 19.791 0 00-4.885-1.515.074.074 0 00-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 00-5.487 0 12.64 12.64 0 00-.617-1.25.077.077 0 00-.079-.037A19.736 19.736 0 003.677 4.37a.07.07 0 00-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 00.031.057 19.9 19.9 0 005.993 3.03.078.078 0 00.084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 00-.041-.106 13.107 13.107 0 01-1.872-.892.077.077 0 01-.008-.128 10.2 10.2 0 00.372-.292.074.074 0 01.077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 01.078.01c.12.098.246.198.373.292a.077.077 0 01-.006.127 12.299 12.299 0 01-1.873.892.077.077 0 00-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 00.084.028 19.839 19.839 0 006.002-3.03.077.077 0 00.032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 00-.031-.03z",
      webpush: "M15 17h5l-1.405-1.405A2.032 2.032 0 0118 14.158V11a6.002 6.002 0 00-4-5.659V5a2 2 0 10-4 0v.341C7.67 6.165 6 8.388 6 11v3.159c0 .538-.214 1.055-.595 1.436L4 17h5m6 0v1a3 3 0 11-6 0v-1m6 0H9",
      gotify: "M13 10V3L4 14h7v7l9-11h-7z",
      ntfy: "M11 5.882V19.24a1.76 1.76 0 01-3.417.592l-2.147-6.15M18 13a3 3 0 100-6M5.436 13.683A4.001 4.001 0 017 6h1.832c4.1 0 7.625-1.234 9.168-3v14c-1.543-1.766-5.067-3-9.168-3H7a3.988 3.988 0 01-1.564-.317z",
    };
    return icons[name] || "M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z";
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Notification Settings</h1>
          <p class="text-sm opacity-70">
            Configure notification platforms and choose which events trigger alerts.
          </p>
        </div>
        <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
          Back to Plugins
        </NuxtLink>
      </div>

      <!-- Platforms Grid -->
      <section>
        <Subtitle>Notification Platforms</Subtitle>
        <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          <BaseCard v-for="platform in platforms" :key="platform.platform">
            <div class="p-4">
              <div class="flex items-start gap-3">
                <!-- Platform Icon -->
                <div class="flex-shrink-0 w-10 h-10 rounded-lg bg-base-200 flex items-center justify-center">
                  <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5 opacity-70" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" :d="platformIcon(platform.platform)" />
                  </svg>
                </div>

                <!-- Platform Info -->
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-2">
                    <h3 class="font-semibold">{{ platform.label }}</h3>
                  </div>
                  <div class="flex gap-1 mt-1">
                    <span
                      class="badge badge-xs"
                      :class="platform.configured ? 'badge-success' : 'badge-warning'"
                    >
                      {{ platform.configured ? "configured" : "unconfigured" }}
                    </span>
                    <span
                      class="badge badge-xs"
                      :class="platform.enabled ? 'badge-info' : 'badge-ghost'"
                    >
                      {{ platform.enabled ? "enabled" : "disabled" }}
                    </span>
                  </div>
                </div>

                <!-- Enable Toggle -->
                <input
                  type="checkbox"
                  class="toggle toggle-primary toggle-sm"
                  :checked="platform.enabled"
                  @change="togglePlatform(platform)"
                />
              </div>

              <!-- Platform Actions -->
              <div class="flex gap-2 mt-4 justify-end">
                <button
                  class="btn btn-xs btn-ghost"
                  :class="{ 'loading': testingPlatform === platform.platform }"
                  :disabled="!platform.configured || !platform.enabled || testingPlatform !== null"
                  @click="testNotification(platform)"
                >
                  {{ testingPlatform === platform.platform ? "" : "Test" }}
                </button>
                <button class="btn btn-xs btn-primary" @click="openConfig(platform)">
                  Configure
                </button>
              </div>
            </div>
          </BaseCard>
        </div>

        <div v-if="!platforms || platforms.length === 0" class="text-center py-8 opacity-50">
          <p>No notification platforms available.</p>
        </div>
      </section>

      <!-- Category Preferences -->
      <section v-if="categories && categories.length > 0 && platforms && platforms.length > 0">
        <Subtitle>Event Preferences</Subtitle>
        <p class="text-sm opacity-70 mb-4">
          Choose which notification events are sent to each platform.
        </p>
        <div class="overflow-x-auto">
          <table class="table table-sm w-full">
            <thead>
              <tr>
                <th class="min-w-[200px]">Event</th>
                <th
                  v-for="platform in platforms"
                  :key="platform.platform"
                  class="text-center min-w-[100px]"
                >
                  {{ platform.label }}
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="category in categories" :key="category.id">
                <td>
                  <div>
                    <p class="font-medium text-sm">{{ category.label }}</p>
                    <p class="text-xs opacity-50">{{ category.description }}</p>
                  </div>
                </td>
                <td
                  v-for="platform in platforms"
                  :key="platform.platform"
                  class="text-center"
                >
                  <input
                    type="checkbox"
                    class="toggle toggle-primary toggle-xs"
                    :checked="isCategoryEnabled(category.id, platform.platform)"
                    :disabled="!platform.enabled || !platform.configured"
                    @change="toggleCategoryPref(category.id, platform.platform)"
                  />
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      <!-- Empty States -->
      <div v-if="categories && categories.length === 0" class="text-center py-8 opacity-50">
        <p>No notification categories defined yet.</p>
      </div>
    </BaseContainer>

    <!-- Config Modal -->
    <dialog class="modal" :class="{ 'modal-open': showConfigModal }">
      <div class="modal-box max-w-lg">
        <h3 class="font-bold text-lg">{{ selectedPlatform?.label }} - Configuration</h3>
        <div class="py-4 space-y-4">
          <div v-for="field in platformConfig" :key="field.key" class="form-control">
            <label class="label">
              <span class="label-text font-medium">{{ field.label }}</span>
              <span v-if="field.required" class="label-text-alt text-error">Required</span>
            </label>
            <p class="text-xs opacity-60 mb-1">{{ field.description }}</p>

            <select
              v-if="field.type === 'select'"
              v-model="configValues[field.key]"
              class="select select-bordered select-sm w-full"
            >
              <option v-for="opt in field.options" :key="opt" :value="opt">{{ opt }}</option>
            </select>

            <input
              v-else-if="field.type === 'boolean'"
              v-model="configValues[field.key]"
              type="checkbox"
              class="toggle toggle-primary"
              true-value="true"
              false-value="false"
            />

            <input
              v-else
              v-model="configValues[field.key]"
              :type="field.type === 'secret' ? 'password' : field.type === 'number' ? 'number' : 'text'"
              :placeholder="field.default"
              class="input input-bordered input-sm w-full"
            />
          </div>

          <div v-if="platformConfig.length === 0" class="text-center py-4 opacity-50">
            <p>No configuration options available for this platform.</p>
          </div>
        </div>
        <div class="modal-action">
          <button class="btn btn-sm" @click="showConfigModal = false">Cancel</button>
          <button class="btn btn-sm btn-primary" @click="saveConfig">Save</button>
        </div>
      </div>
      <div class="modal-backdrop" @click="showConfigModal = false" />
    </dialog>
  </div>
</template>
