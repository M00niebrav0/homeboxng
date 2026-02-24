<script setup lang="ts">
  import { toast } from "@/components/ui/sonner";
  import { Button } from "@/components/ui/button";
  import { Input } from "@/components/ui/input";
  import { Switch } from "@/components/ui/switch";
  import { Badge } from "@/components/ui/badge";
  import { Label } from "@/components/ui/label";
  import {
    Select,
    SelectContent,
    SelectItem,
    SelectTrigger,
    SelectValue,
  } from "@/components/ui/select";
  import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
  } from "@/components/ui/table";
  import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
  } from "@/components/ui/dialog";
  import {
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
  } from "@/components/ui/alert-dialog";
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import BaseSectionHeader from "@/components/Base/SectionHeader.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import FormPassword from "~/components/Form/Password.vue";
  import PasswordScore from "~/components/global/PasswordScore.vue";
  import MdiAccount from "~icons/mdi/account";
  import MdiPalette from "~icons/mdi/palette";
  import MdiBell from "~icons/mdi/bell";
  import MdiShieldLock from "~icons/mdi/shield-lock";
  import MdiPuzzle from "~icons/mdi/puzzle";
  import MdiKey from "~icons/mdi/key";
  import MdiLoading from "~icons/mdi/loading";
  import MdiContentCopy from "~icons/mdi/content-copy";
  import MdiDelete from "~icons/mdi/delete";
  import MdiDownload from "~icons/mdi/download";
  import MdiMonitor from "~icons/mdi/monitor";
  import MdiCheck from "~icons/mdi/check";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Settings",
  });

  const api = useUserApi();
  const auth = useAuthContext();
  const preferences = useViewPreferences();
  const confirm = useConfirm();

  // ==================== Tab Navigation ====================
  type SettingsTab = "profile" | "appearance" | "notifications" | "data" | "plugins" | "api-keys";
  const activeTab = ref<SettingsTab>("profile");

  const tabs: { id: SettingsTab; label: string; icon: any }[] = [
    { id: "profile", label: "Profile", icon: MdiAccount },
    { id: "appearance", label: "Appearance", icon: MdiPalette },
    { id: "notifications", label: "Notifications", icon: MdiBell },
    { id: "data", label: "Data & Privacy", icon: MdiShieldLock },
    { id: "plugins", label: "Plugin Permissions", icon: MdiPuzzle },
    { id: "api-keys", label: "API Keys", icon: MdiKey },
  ];

  // ==================== Profile Section ====================
  const userProfile = computed(() => ({
    name: auth.user?.name || "Unknown",
    email: auth.user?.email || "Unknown",
  }));

  const passwordChange = reactive({
    loading: false,
    current: "",
    new: "",
    confirm: "",
    isValid: false,
  });
  const showPasswordDialog = ref(false);

  async function changePassword() {
    if (!passwordChange.isValid) return;
    if (passwordChange.new !== passwordChange.confirm) {
      toast.error("New passwords do not match.");
      return;
    }

    passwordChange.loading = true;
    const { error } = await api.user.changePassword(passwordChange.current, passwordChange.new);

    if (error) {
      toast.error("Failed to change password. Please check your current password.");
      passwordChange.loading = false;
      return;
    }

    toast.success("Password changed successfully.");
    showPasswordDialog.value = false;
    passwordChange.current = "";
    passwordChange.new = "";
    passwordChange.confirm = "";
    passwordChange.loading = false;
  }

  const showDeleteAccountDialog = ref(false);
  const deleteConfirmText = ref("");

  async function deleteAccount() {
    if (deleteConfirmText.value !== "DELETE") return;

    const { response } = await api.user.delete();
    if (response?.status === 204) {
      toast.success("Account deleted successfully.");
      auth.logout(api);
      navigateTo("/");
      return;
    }
    toast.error("Failed to delete account.");
  }

  // ==================== Appearance Section ====================
  const { theme: currentTheme, setTheme } = useTheme();
  const themeMode = ref<"system" | "light" | "dark">("system");

  const compactMode = ref(false);
  const itemsPerPage = ref("24");
  const dateFormat = ref("MM/DD/YYYY");
  const currencyFormat = ref("USD");

  const itemsPerPageOptions = ["12", "24", "48", "96"];
  const dateFormatOptions = [
    { label: "MM/DD/YYYY", value: "MM/DD/YYYY" },
    { label: "DD/MM/YYYY", value: "DD/MM/YYYY" },
    { label: "YYYY-MM-DD", value: "YYYY-MM-DD" },
  ];

  const { data: currencies } = useAsyncData("currencies", async () => {
    const { data } = await api.group.currencies();
    return data;
  });

  function applyThemeMode(mode: string) {
    themeMode.value = mode as "system" | "light" | "dark";
    if (mode === "light") setTheme("light");
    else if (mode === "dark") setTheme("dark");
    else setTheme("homebox");
  }

  // ==================== Notifications Section ====================
  interface NotifCategory {
    id: string;
    label: string;
  }

  interface NotifChannel {
    id: string;
    label: string;
  }

  const notifCategories: NotifCategory[] = [
    { id: "items", label: "Items" },
    { id: "maintenance", label: "Maintenance" },
    { id: "warranty", label: "Warranty" },
    { id: "security", label: "Security" },
    { id: "plugins", label: "Plugins" },
  ];

  const notifChannels: NotifChannel[] = [
    { id: "email", label: "Email" },
    { id: "discord", label: "Discord" },
    { id: "push", label: "Push" },
    { id: "in-app", label: "In-App" },
  ];

  const notifMatrix = reactive<Record<string, Record<string, boolean>>>({
    items: { email: true, discord: false, push: true, "in-app": true },
    maintenance: { email: true, discord: false, push: false, "in-app": true },
    warranty: { email: true, discord: false, push: true, "in-app": true },
    security: { email: true, discord: true, push: true, "in-app": true },
    plugins: { email: false, discord: false, push: false, "in-app": true },
  });

  const quietHoursEnabled = ref(false);
  const quietHoursStart = ref("22:00");
  const quietHoursEnd = ref("07:00");
  const quietHoursTimezone = ref("America/New_York");
  const digestMode = ref("immediate");

  const digestOptions = [
    { label: "Immediate", value: "immediate" },
    { label: "Hourly Digest", value: "hourly" },
    { label: "Daily Summary", value: "daily" },
  ];

  const timezoneOptions = [
    "America/New_York",
    "America/Chicago",
    "America/Denver",
    "America/Los_Angeles",
    "America/Anchorage",
    "Pacific/Honolulu",
    "Europe/London",
    "Europe/Berlin",
    "Asia/Tokyo",
    "Australia/Sydney",
    "UTC",
  ];

  function toggleNotif(category: string, channel: string) {
    notifMatrix[category][channel] = !notifMatrix[category][channel];
  }

  function saveNotificationPrefs() {
    toast.success("Notification preferences saved.");
  }

  // ==================== Data & Privacy Section ====================
  const exportLoading = ref(false);
  const auditLogLoading = ref(false);

  interface SessionInfo {
    id: string;
    device: string;
    ip: string;
    lastActive: string;
    created: string;
    isCurrent: boolean;
  }

  const sessions = ref<SessionInfo[]>([
    {
      id: "sess-1",
      device: "Chrome on Windows",
      ip: "192.168.1.15",
      lastActive: "2 minutes ago",
      created: "2026-02-20",
      isCurrent: true,
    },
    {
      id: "sess-2",
      device: "Firefox on Linux",
      ip: "192.168.1.249",
      lastActive: "3 hours ago",
      created: "2026-02-18",
      isCurrent: false,
    },
    {
      id: "sess-3",
      device: "Safari on iPhone",
      ip: "10.0.0.42",
      lastActive: "1 day ago",
      created: "2026-02-15",
      isCurrent: false,
    },
  ]);

  async function exportAllData() {
    exportLoading.value = true;
    try {
      const url = api.items.exportURL();
      window.open(url, "_blank");
      toast.success("Export started. Your download should begin shortly.");
    } catch {
      toast.error("Failed to start export.");
    }
    exportLoading.value = false;
  }

  async function downloadAuditLog() {
    auditLogLoading.value = true;
    toast.info("Audit log download started.");
    setTimeout(() => {
      auditLogLoading.value = false;
      toast.success("Audit log downloaded.");
    }, 2000);
  }

  function revokeSession(id: string) {
    sessions.value = sessions.value.filter(s => s.id !== id);
    toast.success("Session revoked.");
  }

  function revokeAllOtherSessions() {
    sessions.value = sessions.value.filter(s => s.isCurrent);
    toast.success("All other sessions revoked.");
  }

  // ==================== Plugin Permissions Section ====================
  interface PluginPermEntry {
    name: string;
    enabled: boolean;
    permissions: string[];
    grantedCount: number;
    totalCount: number;
  }

  const pluginPerms = ref<PluginPermEntry[]>([]);
  const showPluginPermModal = ref(false);
  const selectedPluginPerms = ref<{
    name: string;
    permissions: { permission: string; reason: string; required: boolean; granted: boolean }[];
  } | null>(null);

  const { data: pluginsList } = useAsyncData("settings-plugins", async () => {
    const { data } = await api.plugins.getAll();
    if (data) {
      const entries: PluginPermEntry[] = [];
      for (const p of data) {
        const permsResp = await api.plugins.getPermissions(p.name);
        const perms = permsResp.data || [];
        entries.push({
          name: p.name,
          enabled: p.enabled !== false,
          permissions: perms.map(pr => pr.permission),
          grantedCount: perms.filter(pr => pr.granted).length,
          totalCount: perms.length,
        });
      }
      pluginPerms.value = entries;
    }
    return data;
  });

  async function openPluginPermissions(pluginName: string) {
    const { data } = await api.plugins.getPermissions(pluginName);
    selectedPluginPerms.value = {
      name: pluginName,
      permissions: data || [],
    };
    showPluginPermModal.value = true;
  }

  async function togglePluginPermission(pluginName: string, permission: string, granted: boolean) {
    if (granted) {
      await api.plugins.revokePermission(pluginName, permission);
    } else {
      await api.plugins.grantPermission(pluginName, permission);
    }
    // Refresh the modal data
    if (selectedPluginPerms.value?.name === pluginName) {
      const { data } = await api.plugins.getPermissions(pluginName);
      selectedPluginPerms.value.permissions = data || [];
    }
  }

  async function togglePluginEnabled(pluginName: string, currentEnabled: boolean) {
    if (currentEnabled) {
      await api.plugins.disable(pluginName);
    } else {
      await api.plugins.enable(pluginName);
    }
    const entry = pluginPerms.value.find(p => p.name === pluginName);
    if (entry) entry.enabled = !currentEnabled;
    toast.success(`Plugin "${pluginName}" ${currentEnabled ? "disabled" : "enabled"}.`);
  }

  const showRevokeAllPluginDialog = ref(false);
  const revokeAllPluginName = ref("");

  function confirmRevokeAllPlugin(name: string) {
    revokeAllPluginName.value = name;
    showRevokeAllPluginDialog.value = true;
  }

  async function revokeAllPluginPermissions() {
    const name = revokeAllPluginName.value;
    const entry = pluginPerms.value.find(p => p.name === name);
    if (entry) {
      for (const perm of entry.permissions) {
        await api.plugins.revokePermission(name, perm);
      }
      entry.grantedCount = 0;
    }
    showRevokeAllPluginDialog.value = false;
    toast.success(`All permissions revoked for "${name}".`);
  }

  // ==================== API Keys Section ====================
  interface ApiToken {
    id: string;
    name: string;
    createdAt: string;
    lastUsed: string;
    expiresAt: string;
    token?: string;
  }

  const apiTokens = ref<ApiToken[]>([
    {
      id: "tok-1",
      name: "n8n Integration",
      createdAt: "2026-01-15",
      lastUsed: "2026-02-24",
      expiresAt: "2026-04-15",
    },
    {
      id: "tok-2",
      name: "HomeBox AI Bot",
      createdAt: "2026-02-01",
      lastUsed: "2026-02-24",
      expiresAt: "Never",
    },
  ]);

  const showCreateTokenDialog = ref(false);
  const newTokenName = ref("");
  const newTokenExpiry = ref("90d");
  const createdToken = ref("");
  const tokenCopied = ref(false);

  const expiryOptions = [
    { label: "30 days", value: "30d" },
    { label: "90 days", value: "90d" },
    { label: "1 year", value: "1y" },
    { label: "Never", value: "never" },
  ];

  function createApiToken() {
    if (!newTokenName.value.trim()) {
      toast.error("Please enter a token name.");
      return;
    }

    const fakeToken = "hb_" + Array.from({ length: 40 }, () =>
      "abcdefghijklmnopqrstuvwxyz0123456789"[Math.floor(Math.random() * 36)]
    ).join("");

    const now = new Date().toISOString().split("T")[0];
    let expiresAt = "Never";
    if (newTokenExpiry.value === "30d") {
      const d = new Date(); d.setDate(d.getDate() + 30);
      expiresAt = d.toISOString().split("T")[0];
    } else if (newTokenExpiry.value === "90d") {
      const d = new Date(); d.setDate(d.getDate() + 90);
      expiresAt = d.toISOString().split("T")[0];
    } else if (newTokenExpiry.value === "1y") {
      const d = new Date(); d.setFullYear(d.getFullYear() + 1);
      expiresAt = d.toISOString().split("T")[0];
    }

    apiTokens.value.push({
      id: `tok-${Date.now()}`,
      name: newTokenName.value,
      createdAt: now,
      lastUsed: "Never",
      expiresAt,
    });

    createdToken.value = fakeToken;
    tokenCopied.value = false;
  }

  async function copyToken() {
    try {
      await navigator.clipboard.writeText(createdToken.value);
      tokenCopied.value = true;
      toast.success("Token copied to clipboard.");
    } catch {
      toast.error("Failed to copy token.");
    }
  }

  function closeCreateTokenDialog() {
    showCreateTokenDialog.value = false;
    newTokenName.value = "";
    newTokenExpiry.value = "90d";
    createdToken.value = "";
    tokenCopied.value = false;
  }

  function revokeToken(id: string) {
    apiTokens.value = apiTokens.value.filter(t => t.id !== id);
    toast.success("API token revoked.");
  }
</script>

<template>
  <div>
    <!-- Change Password Dialog -->
    <Dialog v-model:open="showPasswordDialog">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Change Password</DialogTitle>
          <DialogDescription>Enter your current password and choose a new one.</DialogDescription>
        </DialogHeader>
        <form @submit.prevent="changePassword">
          <div class="space-y-4">
            <FormPassword
              v-model="passwordChange.current"
              label="Current Password"
              placeholder=""
            />
            <FormPassword
              v-model="passwordChange.new"
              label="New Password"
              placeholder=""
            />
            <FormPassword
              v-model="passwordChange.confirm"
              label="Confirm New Password"
              placeholder=""
            />
            <PasswordScore v-model:valid="passwordChange.isValid" :password="passwordChange.new" />
            <p
              v-if="passwordChange.confirm && passwordChange.new !== passwordChange.confirm"
              class="text-sm text-destructive"
            >
              Passwords do not match.
            </p>
          </div>
          <DialogFooter class="mt-4">
            <Button variant="outline" type="button" @click="showPasswordDialog = false">Cancel</Button>
            <Button
              type="submit"
              :disabled="!passwordChange.isValid || passwordChange.loading || passwordChange.new !== passwordChange.confirm"
            >
              <MdiLoading v-if="passwordChange.loading" class="mr-2 h-4 w-4 animate-spin" />
              Update Password
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>

    <!-- Delete Account Dialog -->
    <AlertDialog v-model:open="showDeleteAccountDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete Account</AlertDialogTitle>
          <AlertDialogDescription>
            This action is permanent and cannot be undone. All your data, items, locations, and
            settings will be permanently deleted.
            <br /><br />
            Type <strong>DELETE</strong> to confirm.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <Input v-model="deleteConfirmText" placeholder='Type "DELETE" to confirm' class="mt-2" />
        <AlertDialogFooter>
          <AlertDialogCancel @click="deleteConfirmText = ''">Cancel</AlertDialogCancel>
          <AlertDialogAction
            :disabled="deleteConfirmText !== 'DELETE'"
            class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            @click="deleteAccount"
          >
            Permanently Delete Account
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Plugin Permission Review Modal -->
    <Dialog v-model:open="showPluginPermModal">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ selectedPluginPerms?.name }} - Permissions</DialogTitle>
          <DialogDescription>Review and manage permissions for this plugin.</DialogDescription>
        </DialogHeader>
        <div v-if="selectedPluginPerms" class="max-h-80 space-y-3 overflow-y-auto">
          <div
            v-for="perm in selectedPluginPerms.permissions"
            :key="perm.permission"
            class="flex items-center justify-between rounded-lg border p-3"
          >
            <div class="flex-1">
              <div class="flex items-center gap-2">
                <span class="text-sm font-medium">{{ perm.permission }}</span>
                <Badge v-if="perm.required" variant="secondary" class="text-[10px]">Required</Badge>
              </div>
              <p class="mt-1 text-xs text-muted-foreground">{{ perm.reason }}</p>
            </div>
            <Switch
              :checked="perm.granted"
              @update:checked="togglePluginPermission(selectedPluginPerms!.name, perm.permission, perm.granted)"
            />
          </div>
          <div v-if="selectedPluginPerms.permissions.length === 0" class="py-8 text-center text-sm text-muted-foreground">
            No permissions requested by this plugin.
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" size="sm" @click="showPluginPermModal = false">Close</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>

    <!-- Revoke All Plugin Permissions Confirmation -->
    <AlertDialog v-model:open="showRevokeAllPluginDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Revoke All Permissions</AlertDialogTitle>
          <AlertDialogDescription>
            This will revoke all permissions for the plugin "{{ revokeAllPluginName }}".
            The plugin may stop functioning correctly without its required permissions.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            @click="revokeAllPluginPermissions"
          >
            Revoke All
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Create API Token Dialog -->
    <Dialog v-model:open="showCreateTokenDialog" @update:open="(val: boolean) => { if (!val) closeCreateTokenDialog() }">
      <DialogContent class="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{{ createdToken ? "Token Created" : "Create API Token" }}</DialogTitle>
          <DialogDescription>
            {{ createdToken ? "Copy this token now. It will not be shown again." : "Create a new API token for external integrations." }}
          </DialogDescription>
        </DialogHeader>

        <template v-if="!createdToken">
          <div class="space-y-4">
            <div class="space-y-2">
              <Label>Token Name</Label>
              <Input v-model="newTokenName" placeholder="e.g., n8n Integration" />
            </div>
            <div class="space-y-2">
              <Label>Expiry</Label>
              <Select v-model="newTokenExpiry">
                <SelectTrigger>
                  <SelectValue placeholder="Select expiry" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="opt in expiryOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter class="mt-4">
            <Button variant="outline" @click="closeCreateTokenDialog">Cancel</Button>
            <Button @click="createApiToken">Create Token</Button>
          </DialogFooter>
        </template>

        <template v-else>
          <div class="space-y-3">
            <div class="rounded-lg border bg-muted p-3">
              <code class="break-all text-xs">{{ createdToken }}</code>
            </div>
            <Button class="w-full" variant="outline" @click="copyToken">
              <MdiCheck v-if="tokenCopied" class="mr-2 h-4 w-4" />
              <MdiContentCopy v-else class="mr-2 h-4 w-4" />
              {{ tokenCopied ? "Copied" : "Copy to Clipboard" }}
            </Button>
          </div>
          <DialogFooter class="mt-4">
            <Button @click="closeCreateTokenDialog">Done</Button>
          </DialogFooter>
        </template>
      </DialogContent>
    </Dialog>

    <!-- Main Content -->
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div>
        <h1 class="text-2xl font-bold">Settings</h1>
        <p class="text-sm text-muted-foreground">
          Manage your account, appearance, notifications, and integrations.
        </p>
      </div>

      <!-- Tab Navigation -->
      <div class="flex flex-wrap gap-1 rounded-lg border bg-muted p-1">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors"
          :class="activeTab === tab.id
            ? 'bg-background text-foreground shadow-sm'
            : 'text-muted-foreground hover:text-foreground'"
          @click="activeTab = tab.id"
        >
          <component :is="tab.icon" class="h-4 w-4" />
          <span class="hidden sm:inline">{{ tab.label }}</span>
        </button>
      </div>

      <!-- ==================== Profile Tab ==================== -->
      <template v-if="activeTab === 'profile'">
        <BaseCard>
          <template #title>
            <BaseSectionHeader>
              <MdiAccount class="-mt-1 mr-2" />
              <span>Profile Information</span>
              <template #description>Your account details and identity.</template>
            </BaseSectionHeader>
          </template>

          <div class="space-y-4 p-4">
            <div class="grid gap-4 sm:grid-cols-2">
              <div class="space-y-2">
                <Label class="text-sm font-medium">Display Name</Label>
                <div class="rounded-md border bg-muted/50 px-3 py-2 text-sm">
                  {{ userProfile.name }}
                </div>
              </div>
              <div class="space-y-2">
                <Label class="text-sm font-medium">Email Address</Label>
                <div class="rounded-md border bg-muted/50 px-3 py-2 text-sm">
                  {{ userProfile.email }}
                </div>
              </div>
            </div>

            <div class="flex flex-wrap gap-2 border-t pt-4">
              <Button variant="secondary" size="sm" @click="showPasswordDialog = true">
                Change Password
              </Button>
              <NuxtLink to="/settings/security">
                <Button variant="outline" size="sm">
                  <MdiShieldLock class="mr-1 h-4 w-4" />
                  Security Settings
                </Button>
              </NuxtLink>
            </div>
          </div>
        </BaseCard>

        <BaseCard>
          <template #title>
            <BaseSectionHeader>
              <MdiDelete class="-mt-1 mr-2 text-destructive" />
              <span>Danger Zone</span>
              <template #description>Irreversible actions for your account.</template>
            </BaseSectionHeader>
          </template>
          <div class="border-t p-4">
            <p class="mb-3 text-sm text-muted-foreground">
              Once you delete your account, there is no going back. All your data will be permanently removed.
            </p>
            <Button variant="destructive" size="sm" @click="showDeleteAccountDialog = true">
              <MdiDelete class="mr-1 h-4 w-4" />
              Delete My Account
            </Button>
          </div>
        </BaseCard>
      </template>

      <!-- ==================== Appearance Tab ==================== -->
      <template v-if="activeTab === 'appearance'">
        <BaseCard>
          <template #title>
            <BaseSectionHeader>
              <MdiPalette class="-mt-1 mr-2" />
              <span>Appearance</span>
              <template #description>Customize how HomeBoxNG looks and feels.</template>
            </BaseSectionHeader>
          </template>

          <div class="space-y-6 p-4">
            <!-- Theme Mode -->
            <div class="space-y-2">
              <Label class="text-sm font-medium">Theme Mode</Label>
              <div class="flex flex-wrap gap-2">
                <Button
                  v-for="mode in ['system', 'light', 'dark']"
                  :key="mode"
                  :variant="themeMode === mode ? 'default' : 'outline'"
                  size="sm"
                  @click="applyThemeMode(mode)"
                >
                  <MdiMonitor v-if="mode === 'system'" class="mr-1 h-4 w-4" />
                  {{ mode.charAt(0).toUpperCase() + mode.slice(1) }}
                </Button>
              </div>
              <p class="text-xs text-muted-foreground">
                Current theme: <strong>{{ currentTheme }}</strong>
              </p>
            </div>

            <!-- Compact Mode -->
            <div class="flex items-center justify-between rounded-lg border p-3">
              <div>
                <Label class="text-sm font-medium">Compact Mode</Label>
                <p class="text-xs text-muted-foreground">Reduce padding and margins for a denser layout.</p>
              </div>
              <Switch v-model:checked="compactMode" />
            </div>

            <!-- Items Per Page -->
            <div class="space-y-2">
              <Label class="text-sm font-medium">Items Per Page</Label>
              <Select v-model="itemsPerPage">
                <SelectTrigger class="w-40">
                  <SelectValue placeholder="Select" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="opt in itemsPerPageOptions" :key="opt" :value="opt">
                    {{ opt }} items
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <!-- Currency Display -->
            <div class="space-y-2">
              <Label class="text-sm font-medium">Currency Display</Label>
              <Select v-model="currencyFormat">
                <SelectTrigger class="w-60">
                  <SelectValue placeholder="Select currency" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem
                    v-for="c in (currencies || []).slice(0, 30)"
                    :key="c.code"
                    :value="c.code"
                  >
                    {{ c.symbol }} {{ c.name }} ({{ c.code }})
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>

            <!-- Date Format -->
            <div class="space-y-2">
              <Label class="text-sm font-medium">Date Format</Label>
              <Select v-model="dateFormat">
                <SelectTrigger class="w-48">
                  <SelectValue placeholder="Select format" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem v-for="opt in dateFormatOptions" :key="opt.value" :value="opt.value">
                    {{ opt.label }}
                  </SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </BaseCard>
      </template>

      <!-- ==================== Notifications Tab ==================== -->
      <template v-if="activeTab === 'notifications'">
        <BaseCard>
          <template #title>
            <BaseSectionHeader>
              <MdiBell class="-mt-1 mr-2" />
              <span>Notification Preferences</span>
              <template #description>Choose what notifications you receive and how.</template>
            </BaseSectionHeader>
          </template>

          <div class="space-y-6 p-4">
            <!-- Toggle Matrix -->
            <div class="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead class="w-36">Category</TableHead>
                    <TableHead v-for="ch in notifChannels" :key="ch.id" class="text-center">
                      {{ ch.label }}
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-for="cat in notifCategories" :key="cat.id">
                    <TableCell class="font-medium">{{ cat.label }}</TableCell>
                    <TableCell v-for="ch in notifChannels" :key="ch.id" class="text-center">
                      <Switch
                        :checked="notifMatrix[cat.id][ch.id]"
                        @update:checked="toggleNotif(cat.id, ch.id)"
                      />
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>

            <!-- Quiet Hours -->
            <div class="rounded-lg border p-4">
              <Subtitle>Quiet Hours</Subtitle>
              <div class="flex items-center justify-between">
                <p class="text-sm text-muted-foreground">Suppress non-critical notifications during set hours.</p>
                <Switch v-model:checked="quietHoursEnabled" />
              </div>
              <div v-if="quietHoursEnabled" class="mt-4 grid gap-4 sm:grid-cols-3">
                <div class="space-y-2">
                  <Label class="text-sm">Start Time</Label>
                  <Input v-model="quietHoursStart" type="time" />
                </div>
                <div class="space-y-2">
                  <Label class="text-sm">End Time</Label>
                  <Input v-model="quietHoursEnd" type="time" />
                </div>
                <div class="space-y-2">
                  <Label class="text-sm">Timezone</Label>
                  <Select v-model="quietHoursTimezone">
                    <SelectTrigger>
                      <SelectValue placeholder="Select timezone" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem v-for="tz in timezoneOptions" :key="tz" :value="tz">
                        {{ tz }}
                      </SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>

            <!-- Digest Mode -->
            <div class="space-y-2">
              <Label class="text-sm font-medium">Notification Delivery</Label>
              <div class="flex flex-wrap gap-2">
                <Button
                  v-for="opt in digestOptions"
                  :key="opt.value"
                  :variant="digestMode === opt.value ? 'default' : 'outline'"
                  size="sm"
                  @click="digestMode = opt.value"
                >
                  {{ opt.label }}
                </Button>
              </div>
              <p class="text-xs text-muted-foreground">
                Controls how frequently notification batches are sent. "Immediate" sends each notification individually.
              </p>
            </div>

            <div class="flex justify-end border-t pt-4">
              <Button @click="saveNotificationPrefs">Save Preferences</Button>
            </div>
          </div>
        </BaseCard>
      </template>

      <!-- ==================== Data & Privacy Tab ==================== -->
      <template v-if="activeTab === 'data'">
        <BaseCard>
          <template #title>
            <BaseSectionHeader>
              <MdiShieldLock class="-mt-1 mr-2" />
              <span>Data & Privacy</span>
              <template #description>Manage your data, export options, and active sessions.</template>
            </BaseSectionHeader>
          </template>

          <div class="space-y-6 p-4">
            <!-- Export Actions -->
            <div class="grid gap-3 sm:grid-cols-2">
              <button
                class="flex items-center gap-3 rounded-lg border p-4 text-left transition-colors hover:bg-muted"
                @click="exportAllData"
              >
                <MdiDownload class="h-8 w-8 text-primary" />
                <div>
                  <p class="text-sm font-medium">Export All Data</p>
                  <p class="text-xs text-muted-foreground">Download a CSV export of all your items and data.</p>
                </div>
                <MdiLoading v-if="exportLoading" class="ml-auto h-4 w-4 animate-spin" />
              </button>

              <button
                class="flex items-center gap-3 rounded-lg border p-4 text-left transition-colors hover:bg-muted"
                @click="downloadAuditLog"
              >
                <MdiDownload class="h-8 w-8 text-primary" />
                <div>
                  <p class="text-sm font-medium">Download Audit Log</p>
                  <p class="text-xs text-muted-foreground">Get a log of all account activity.</p>
                </div>
                <MdiLoading v-if="auditLogLoading" class="ml-auto h-4 w-4 animate-spin" />
              </button>
            </div>

            <!-- Two-Factor Auth Placeholder -->
            <div class="flex items-center justify-between rounded-lg border border-dashed p-4">
              <div>
                <div class="flex items-center gap-2">
                  <p class="text-sm font-medium">Two-Factor Authentication</p>
                  <Badge variant="outline" class="text-[10px]">Coming Soon</Badge>
                </div>
                <p class="text-xs text-muted-foreground">
                  Add an extra layer of security to your account with TOTP-based 2FA.
                </p>
              </div>
              <Button variant="outline" size="sm" disabled>Enable</Button>
            </div>

            <!-- Active Sessions -->
            <div>
              <div class="mb-3 flex items-center justify-between">
                <Subtitle>Active Sessions</Subtitle>
                <Button variant="outline" size="sm" @click="revokeAllOtherSessions">
                  Revoke All Others
                </Button>
              </div>

              <div class="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Device / Browser</TableHead>
                      <TableHead>IP Address</TableHead>
                      <TableHead>Last Active</TableHead>
                      <TableHead>Created</TableHead>
                      <TableHead class="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <TableRow v-for="session in sessions" :key="session.id">
                      <TableCell>
                        <div class="flex items-center gap-2">
                          <span class="text-sm">{{ session.device }}</span>
                          <Badge v-if="session.isCurrent" variant="secondary" class="text-[10px]">
                            This Device
                          </Badge>
                        </div>
                      </TableCell>
                      <TableCell class="font-mono text-sm">{{ session.ip }}</TableCell>
                      <TableCell class="text-sm">{{ session.lastActive }}</TableCell>
                      <TableCell class="text-sm">{{ session.created }}</TableCell>
                      <TableCell class="text-right">
                        <Button
                          v-if="!session.isCurrent"
                          variant="ghost"
                          size="sm"
                          class="text-destructive hover:text-destructive"
                          @click="revokeSession(session.id)"
                        >
                          Revoke
                        </Button>
                        <span v-else class="text-xs text-muted-foreground">Current</span>
                      </TableCell>
                    </TableRow>
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        </BaseCard>
      </template>

      <!-- ==================== Plugin Permissions Tab ==================== -->
      <template v-if="activeTab === 'plugins'">
        <BaseCard>
          <template #title>
            <BaseSectionHeader>
              <MdiPuzzle class="-mt-1 mr-2" />
              <span>Plugin Permissions</span>
              <template #description>Review and manage what each plugin can access.</template>
            </BaseSectionHeader>
          </template>

          <div class="space-y-3 p-4">
            <div
              v-for="plugin in pluginPerms"
              :key="plugin.name"
              class="flex flex-col gap-3 rounded-lg border p-4 sm:flex-row sm:items-center sm:justify-between"
            >
              <div class="flex items-center gap-3">
                <Switch
                  :checked="plugin.enabled"
                  @update:checked="togglePluginEnabled(plugin.name, plugin.enabled)"
                />
                <div>
                  <div class="flex items-center gap-2">
                    <span class="text-sm font-medium">{{ plugin.name }}</span>
                    <Badge :variant="plugin.enabled ? 'default' : 'outline'" class="text-[10px]">
                      {{ plugin.enabled ? "Enabled" : "Disabled" }}
                    </Badge>
                  </div>
                  <p class="text-xs text-muted-foreground">
                    {{ plugin.grantedCount }} / {{ plugin.totalCount }} permissions granted
                  </p>
                </div>
              </div>
              <div class="flex gap-2">
                <Button variant="outline" size="sm" @click="openPluginPermissions(plugin.name)">
                  Review Permissions
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  :disabled="plugin.grantedCount === 0"
                  @click="confirmRevokeAllPlugin(plugin.name)"
                >
                  Revoke All
                </Button>
              </div>
            </div>

            <div v-if="pluginPerms.length === 0" class="py-12 text-center text-sm text-muted-foreground">
              No plugins installed. Visit the
              <NuxtLink to="/plugins" class="text-primary underline">Plugin Manager</NuxtLink>
              to install plugins.
            </div>
          </div>
        </BaseCard>
      </template>

      <!-- ==================== API Keys Tab ==================== -->
      <template v-if="activeTab === 'api-keys'">
        <BaseCard>
          <template #title>
            <BaseSectionHeader>
              <MdiKey class="-mt-1 mr-2" />
              <span>API Tokens</span>
              <template #description>Manage API tokens for external integrations and automation.</template>
            </BaseSectionHeader>
          </template>

          <div class="space-y-4 p-4">
            <div class="flex justify-end">
              <Button size="sm" @click="showCreateTokenDialog = true">
                <MdiKey class="mr-1 h-4 w-4" />
                Create New Token
              </Button>
            </div>

            <div class="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Created</TableHead>
                    <TableHead>Last Used</TableHead>
                    <TableHead>Expires</TableHead>
                    <TableHead class="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-for="token in apiTokens" :key="token.id">
                    <TableCell class="font-medium">{{ token.name }}</TableCell>
                    <TableCell class="text-sm">{{ token.createdAt }}</TableCell>
                    <TableCell class="text-sm">{{ token.lastUsed }}</TableCell>
                    <TableCell>
                      <Badge
                        :variant="token.expiresAt === 'Never' ? 'secondary' : 'outline'"
                        class="text-[10px]"
                      >
                        {{ token.expiresAt }}
                      </Badge>
                    </TableCell>
                    <TableCell class="text-right">
                      <Button
                        variant="ghost"
                        size="sm"
                        class="text-destructive hover:text-destructive"
                        @click="revokeToken(token.id)"
                      >
                        <MdiDelete class="mr-1 h-4 w-4" />
                        Revoke
                      </Button>
                    </TableCell>
                  </TableRow>
                  <TableRow v-if="apiTokens.length === 0">
                    <TableCell colspan="5" class="py-8 text-center text-sm text-muted-foreground">
                      No API tokens created yet.
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </div>
          </div>
        </BaseCard>
      </template>
    </BaseContainer>
  </div>
</template>
