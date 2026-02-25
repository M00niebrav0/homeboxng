<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import { Button } from "~/components/ui/button";
  import { Badge } from "~/components/ui/badge";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Updates",
  });

  const api = useUserApi();

  // Interfaces
  interface VersionInfo {
    currentVersion: string;
    currentCommit: string;
    latestVersion: string;
    latestCommit: string;
    updateAvailable: boolean;
    lastCheck: string;
    autoUpdate: boolean;
    backupEnabled: boolean;
    branch: string;
  }

  interface UpdateLogEntry {
    timestamp: string;
    fromVersion: string;
    toVersion: string;
    status: string;
    backupPath?: string;
    error?: string;
    duration?: string;
  }

  interface ChangelogResponse {
    commits: string[];
    count: number;
    fromCommit: string;
  }

  // Fetch status
  const { data: status, refresh: refreshStatus } = useAsyncData("updater-status", async () => {
    try {
      const { data } = await api.http.get<VersionInfo>({
        url: "/api/v1/plugins/updater/status",
      });
      return data;
    } catch {
      return null;
    }
  });

  // Fetch update log
  const { data: updateLog, refresh: refreshLog } = useAsyncData("updater-log", async () => {
    try {
      const { data } = await api.http.get<UpdateLogEntry[]>({
        url: "/api/v1/plugins/updater/log",
      });
      return data || [];
    } catch {
      return [];
    }
  });

  // State
  const checking = ref(false);
  const updating = ref(false);
  const backingUp = ref(false);
  const backupBeforeUpdate = ref(true);
  const changelog = ref<ChangelogResponse | null>(null);
  const changelogLoading = ref(false);
  const updateResult = ref<UpdateLogEntry | null>(null);
  const backupResult = ref<string | null>(null);

  // Check for updates
  async function checkForUpdates() {
    checking.value = true;
    changelog.value = null;
    try {
      await api.http.post<void, void>({
        url: "/api/v1/plugins/updater/check",
        body: {},
      });
      await refreshStatus();

      // If update available, fetch changelog
      if (status.value?.updateAvailable) {
        await fetchChangelog();
      }
    } finally {
      checking.value = false;
    }
  }

  // Fetch changelog
  async function fetchChangelog() {
    changelogLoading.value = true;
    try {
      const { data } = await api.http.get<ChangelogResponse>({
        url: "/api/v1/plugins/updater/changelog",
      });
      changelog.value = data;
    } finally {
      changelogLoading.value = false;
    }
  }

  // Trigger update
  async function triggerUpdate() {
    updating.value = true;
    updateResult.value = null;
    try {
      const { data } = await api.http.post<{ backupFirst: boolean }, UpdateLogEntry>({
        url: "/api/v1/plugins/updater/update",
        body: { backupFirst: backupBeforeUpdate.value },
      });
      updateResult.value = data;
      await refreshStatus();
      await refreshLog();
    } finally {
      updating.value = false;
    }
  }

  // Create manual backup
  async function createBackup() {
    backingUp.value = true;
    backupResult.value = null;
    try {
      const { data } = await api.http.post<void, { status: string; backupPath: string }>({
        url: "/api/v1/plugins/updater/backup",
        body: {},
      });
      backupResult.value = data?.backupPath || "Backup created";
    } finally {
      backingUp.value = false;
    }
  }

  // Config save
  async function saveAutoUpdateSetting(enabled: boolean) {
    await api.http.post<any, void>({
      url: "/api/v1/plugins/updater/config",
      body: {
        auto_update: enabled ? "true" : "false",
        backup_before_update: backupBeforeUpdate.value ? "true" : "false",
      },
    });
    await refreshStatus();
  }

  function formatDate(dateStr: string): string {
    if (!dateStr || dateStr === "0001-01-01T00:00:00Z") return "Never";
    return new Date(dateStr).toLocaleString();
  }

  function statusBadgeClass(logStatus: string): string {
    switch (logStatus) {
      case "success": return "bg-green-500/10 text-green-500";
      case "failed": return "bg-red-500/10 text-red-500";
      case "skipped": return "bg-yellow-500/10 text-yellow-500";
      default: return "";
    }
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">System Updates</h1>
          <p class="text-sm text-muted-foreground">
            Manage HomeBoxNG updates, backups, and version control.
          </p>
        </div>
        <NuxtLink to="/plugins">
          <Button variant="outline" size="sm">Back to Plugins</Button>
        </NuxtLink>
      </div>

      <!-- Current Version Card -->
      <BaseCard>
        <div class="p-6">
          <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div>
              <h2 class="text-sm font-medium text-muted-foreground">Current Version</h2>
              <div class="mt-1 flex items-center gap-3">
                <span class="text-2xl font-bold text-primary">{{ status?.currentVersion || 'Loading...' }}</span>
                <Badge v-if="status?.currentCommit" variant="secondary" class="font-mono text-xs">
                  {{ status.currentCommit }}
                </Badge>
              </div>
              <p class="mt-1 text-xs text-muted-foreground">
                Branch: <span class="font-mono">{{ status?.branch || '...' }}</span>
                <span class="mx-2">|</span>
                Last checked: {{ formatDate(status?.lastCheck || '') }}
              </p>
            </div>

            <div class="flex flex-col gap-2 sm:items-end">
              <div v-if="status?.updateAvailable" class="flex items-center gap-2">
                <span class="size-3 animate-pulse rounded-full bg-green-500" />
                <span class="text-sm font-semibold text-green-500">Update Available</span>
              </div>
              <div v-else-if="status" class="flex items-center gap-2">
                <span class="size-3 rounded-full bg-gray-400" />
                <span class="text-sm text-muted-foreground">Up to date</span>
              </div>
              <Button
                size="sm"
                variant="secondary"
                :disabled="checking"
                @click="checkForUpdates"
              >
                {{ checking ? "Checking..." : "Check for Updates" }}
              </Button>
            </div>
          </div>
        </div>
      </BaseCard>

      <!-- Update Available Section -->
      <div v-if="status?.updateAvailable" class="space-y-4">
        <!-- Changelog -->
        <BaseCard>
          <div class="p-6">
            <h3 class="text-sm font-semibold mb-3">What's New</h3>

            <div v-if="changelogLoading" class="flex items-center gap-2 py-4">
              <div class="size-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
              <span class="text-sm text-muted-foreground">Loading changelog...</span>
            </div>

            <div v-else-if="changelog && changelog.commits.length > 0" class="space-y-1.5">
              <div
                v-for="(commit, i) in changelog.commits"
                :key="i"
                class="flex items-start gap-2 rounded-lg border border-border/50 px-3 py-2"
              >
                <span class="mt-0.5 size-2 shrink-0 rounded-full bg-primary" />
                <span class="text-sm font-mono">{{ commit }}</span>
              </div>
            </div>

            <p v-else class="text-sm text-muted-foreground">
              {{ changelog?.commits?.length === 0 ? 'No new commits found.' : 'Click "Check for Updates" to load changelog.' }}
            </p>
          </div>
        </BaseCard>

        <!-- Update Controls -->
        <BaseCard>
          <div class="p-6 space-y-4">
            <h3 class="text-sm font-semibold">Apply Update</h3>

            <!-- Backup checkbox -->
            <label class="flex items-center gap-3 cursor-pointer">
              <input
                v-model="backupBeforeUpdate"
                type="checkbox"
                class="size-4 rounded border-border"
              />
              <div>
                <span class="text-sm font-medium">Backup before update</span>
                <p class="text-xs text-muted-foreground">Creates a tarball of your data directory before pulling changes</p>
              </div>
            </label>

            <!-- Update button -->
            <div class="flex items-center gap-3">
              <Button
                :disabled="updating"
                @click="triggerUpdate"
              >
                {{ updating ? "Updating... (this may take a few minutes)" : "Update Now" }}
              </Button>
              <Button
                variant="outline"
                size="sm"
                :disabled="backingUp"
                @click="createBackup"
              >
                {{ backingUp ? "Creating..." : "Backup Only" }}
              </Button>
            </div>

            <!-- Backup result -->
            <div v-if="backupResult" class="rounded-lg border border-green-500/30 bg-green-500/5 p-3">
              <p class="text-sm text-green-500">Backup created: <span class="font-mono text-xs">{{ backupResult }}</span></p>
            </div>

            <!-- Update result -->
            <div v-if="updateResult" class="rounded-lg border p-3" :class="updateResult.status === 'success' ? 'border-green-500/30 bg-green-500/5' : 'border-red-500/30 bg-red-500/5'">
              <div class="flex items-center gap-2 mb-1">
                <Badge :class="statusBadgeClass(updateResult.status)" class="text-xs">
                  {{ updateResult.status }}
                </Badge>
                <span v-if="updateResult.duration" class="text-xs text-muted-foreground">
                  {{ updateResult.duration }}
                </span>
              </div>
              <p v-if="updateResult.status === 'success'" class="text-sm text-green-500">
                Updated from {{ updateResult.fromVersion }} to {{ updateResult.toVersion }}.
                Container will restart shortly.
              </p>
              <p v-else class="text-sm text-red-500">{{ updateResult.error }}</p>
              <p v-if="updateResult.backupPath" class="mt-1 text-xs text-muted-foreground">
                Backup: <span class="font-mono">{{ updateResult.backupPath }}</span>
              </p>
            </div>
          </div>
        </BaseCard>
      </div>

      <!-- Auto-Update Settings -->
      <BaseCard>
        <div class="p-6 space-y-4">
          <h3 class="text-sm font-semibold">Auto-Update Settings</h3>

          <label class="flex items-center gap-3 cursor-pointer">
            <div class="relative">
              <input
                type="checkbox"
                :checked="status?.autoUpdate"
                class="sr-only"
                @change="(e: Event) => saveAutoUpdateSetting((e.target as HTMLInputElement).checked)"
              />
              <div
                class="h-6 w-11 rounded-full transition-colors"
                :class="status?.autoUpdate ? 'bg-primary' : 'bg-muted'"
              >
                <div
                  class="mt-0.5 size-5 rounded-full bg-white shadow transition-transform"
                  :class="status?.autoUpdate ? 'translate-x-[22px]' : 'translate-x-0.5'"
                />
              </div>
            </div>
            <div>
              <span class="text-sm font-medium">Enable Automatic Updates</span>
              <p class="text-xs text-muted-foreground">
                Automatically check and install updates every {{ status?.autoUpdate ? '24' : '—' }} hours.
                Off by default — enable this during initial setup if desired.
              </p>
            </div>
          </label>

          <label class="flex items-center gap-3 cursor-pointer">
            <div class="relative">
              <input
                type="checkbox"
                :checked="status?.backupEnabled"
                class="sr-only"
                @change="() => {}"
              />
              <div
                class="h-6 w-11 rounded-full transition-colors"
                :class="status?.backupEnabled ? 'bg-primary' : 'bg-muted'"
              >
                <div
                  class="mt-0.5 size-5 rounded-full bg-white shadow transition-transform"
                  :class="status?.backupEnabled ? 'translate-x-[22px]' : 'translate-x-0.5'"
                />
              </div>
            </div>
            <div>
              <span class="text-sm font-medium">Always Backup Before Update</span>
              <p class="text-xs text-muted-foreground">
                Creates a data backup before every automatic or manual update.
              </p>
            </div>
          </label>
        </div>
      </BaseCard>

      <!-- Update History -->
      <section>
        <h3 class="mb-3 text-sm font-semibold">Update History</h3>
        <div v-if="!updateLog || updateLog.length === 0" class="py-8 text-center">
          <p class="text-sm text-muted-foreground">No update history yet.</p>
        </div>
        <div v-else class="overflow-x-auto">
          <table class="w-full text-sm">
            <thead>
              <tr class="border-b text-left">
                <th class="pb-2 font-medium">Date</th>
                <th class="pb-2 font-medium">From</th>
                <th class="pb-2 font-medium">To</th>
                <th class="pb-2 font-medium">Status</th>
                <th class="pb-2 font-medium">Duration</th>
                <th class="pb-2 font-medium hidden sm:table-cell">Backup</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="entry in updateLog"
                :key="entry.timestamp"
                class="border-b border-border/50"
              >
                <td class="py-2 text-xs">{{ formatDate(entry.timestamp) }}</td>
                <td class="py-2 font-mono text-xs">{{ entry.fromVersion }}</td>
                <td class="py-2 font-mono text-xs">{{ entry.toVersion || '-' }}</td>
                <td class="py-2">
                  <Badge :class="statusBadgeClass(entry.status)" class="text-[10px]">
                    {{ entry.status }}
                  </Badge>
                </td>
                <td class="py-2 text-xs text-muted-foreground">{{ entry.duration || '-' }}</td>
                <td class="py-2 text-xs text-muted-foreground hidden sm:table-cell font-mono">
                  {{ entry.backupPath || '-' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </BaseContainer>
  </div>
</template>
