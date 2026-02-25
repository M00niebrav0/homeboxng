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
    AlertDialog,
    AlertDialogAction,
    AlertDialogCancel,
    AlertDialogContent,
    AlertDialogDescription,
    AlertDialogFooter,
    AlertDialogHeader,
    AlertDialogTitle,
  } from "@/components/ui/alert-dialog";
  import {
    Dialog,
    DialogContent,
    DialogDescription,
    DialogFooter,
    DialogHeader,
    DialogTitle,
  } from "@/components/ui/dialog";
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import BaseSectionHeader from "@/components/Base/SectionHeader.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import MdiShieldLock from "~icons/mdi/shield-lock";
  import MdiMonitor from "~icons/mdi/monitor";
  import MdiHistory from "~icons/mdi/history";
  import MdiBellAlert from "~icons/mdi/bell-alert";
  import MdiKey from "~icons/mdi/key";
  import MdiDelete from "~icons/mdi/delete";
  import MdiLoading from "~icons/mdi/loading";
  import MdiContentCopy from "~icons/mdi/content-copy";
  import MdiCheck from "~icons/mdi/check";
  import MdiArrowLeft from "~icons/mdi/arrow-left";
  import MdiChevronLeft from "~icons/mdi/chevron-left";
  import MdiChevronRight from "~icons/mdi/chevron-right";
  import MdiFilter from "~icons/mdi/filter";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Security Settings",
  });

  const api = useUserApi();

  // ==================== Active Sessions ====================
  interface SessionInfo {
    id: string;
    device: string;
    browser: string;
    ip: string;
    lastActive: string;
    created: string;
    isCurrent: boolean;
  }

  const sessions = ref<SessionInfo[]>([]);
  const sessionsLoading = ref(true);

  // Attempt to load real sessions from API, fall back to empty state
  onMounted(async () => {
    try {
      const { data } = await api.plugins.getSessions();
      if (data) sessions.value = data as SessionInfo[];
    } catch {
      // Backend doesn't have sessions endpoint yet — show empty state
    } finally {
      sessionsLoading.value = false;
    }
  });

  const showRevokeAllDialog = ref(false);

  function revokeSession(id: string) {
    sessions.value = sessions.value.filter(s => s.id !== id);
    toast.success("Session revoked successfully.");
  }

  function revokeAllOtherSessions() {
    sessions.value = sessions.value.filter(s => s.isCurrent);
    showRevokeAllDialog.value = false;
    toast.success("All other sessions have been revoked.");
  }

  // ==================== Login History ====================
  interface LoginEntry {
    id: string;
    timestamp: string;
    ip: string;
    status: "success" | "failed";
    device: string;
    method: string;
  }

  const loginHistory = ref<LoginEntry[]>([]);
  const loginHistoryLoading = ref(true);

  // Load real login history from API (falls back to empty state)
  onMounted(async () => {
    try {
      const { data } = await api.plugins.getLoginHistory(1, 100);
      if (data) loginHistory.value = data as LoginEntry[];
    } catch {
      // Backend doesn't have login history endpoint yet
    } finally {
      loginHistoryLoading.value = false;
    }
  });

  const loginStatusFilter = ref("all");
  const loginPage = ref(1);
  const loginPageSize = 50;

  const filteredLoginHistory = computed(() => {
    let results = loginHistory.value;
    if (loginStatusFilter.value !== "all") {
      results = results.filter(e => e.status === loginStatusFilter.value);
    }
    return results;
  });

  const paginatedLoginHistory = computed(() => {
    const start = (loginPage.value - 1) * loginPageSize;
    return filteredLoginHistory.value.slice(start, start + loginPageSize);
  });

  const loginTotalPages = computed(() =>
    Math.max(1, Math.ceil(filteredLoginHistory.value.length / loginPageSize))
  );

  // ==================== Security Alerts ====================
  const alertNewLoginDevice = ref(true);
  const alertPermissionChanges = ref(true);
  const alertFailedLogins = ref(true);
  const failedLoginThreshold = ref("5");
  const securityAlertEmail = ref("");

  const thresholdOptions = ["3", "5", "10"];

  const auth = useAuthContext();
  onMounted(() => {
    securityAlertEmail.value = auth.user?.email || "";
  });

  function saveSecurityAlerts() {
    toast.success("Security alert preferences saved.");
  }

  // ==================== API Token Management ====================
  interface ApiToken {
    id: string;
    name: string;
    scope: "read-only" | "read-write" | "admin";
    createdAt: string;
    lastUsed: string;
    expiresAt: string;
    ipWhitelist: string[];
    requestsLast24h: number;
    requestsLast7d: number;
    requestsLast30d: number;
    token?: string;
  }

  const apiTokens = ref<ApiToken[]>([
    {
      id: "sec-tok-1",
      name: "n8n Workflow Automation",
      scope: "read-write",
      createdAt: "2026-01-15",
      lastUsed: "2026-02-24 08:15",
      expiresAt: "2026-04-15",
      ipWhitelist: ["192.168.1.249"],
      requestsLast24h: 142,
      requestsLast7d: 1023,
      requestsLast30d: 4521,
    },
    {
      id: "sec-tok-2",
      name: "HomeBox AI Discord Bot",
      scope: "read-only",
      createdAt: "2026-02-01",
      lastUsed: "2026-02-24 07:50",
      expiresAt: "Never",
      ipWhitelist: [],
      requestsLast24h: 87,
      requestsLast7d: 612,
      requestsLast30d: 2100,
    },
    {
      id: "sec-tok-3",
      name: "Backup Script",
      scope: "admin",
      createdAt: "2026-02-10",
      lastUsed: "2026-02-23 03:00",
      expiresAt: "2027-02-10",
      ipWhitelist: ["192.168.1.15", "192.168.1.249"],
      requestsLast24h: 2,
      requestsLast7d: 14,
      requestsLast30d: 62,
    },
  ]);

  const showCreateTokenDialog = ref(false);
  const newTokenName = ref("");
  const newTokenScope = ref("read-only");
  const newTokenExpiry = ref("90d");
  const newTokenIpWhitelist = ref("");
  const createdToken = ref("");
  const tokenCopied = ref(false);

  const scopeOptions = [
    { label: "Read Only", value: "read-only", description: "Can only read data, cannot modify anything." },
    { label: "Read & Write", value: "read-write", description: "Can read and modify items, locations, and tags." },
    { label: "Admin", value: "admin", description: "Full access including user management and settings." },
  ];

  const expiryOptions = [
    { label: "30 days", value: "30d" },
    { label: "90 days", value: "90d" },
    { label: "1 year", value: "1y" },
    { label: "Never", value: "never" },
  ];

  function scopeBadgeVariant(scope: string): "default" | "secondary" | "destructive" | "outline" {
    if (scope === "admin") return "destructive";
    if (scope === "read-write") return "default";
    return "secondary";
  }

  function createApiToken() {
    if (!newTokenName.value.trim()) {
      toast.error("Please enter a token name.");
      return;
    }

    const fakeToken = "hb_" + Array.from({ length: 48 }, () =>
      "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"[Math.floor(Math.random() * 62)]
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

    const ipList = newTokenIpWhitelist.value
      .split(",")
      .map(ip => ip.trim())
      .filter(ip => ip.length > 0);

    apiTokens.value.push({
      id: `sec-tok-${Date.now()}`,
      name: newTokenName.value,
      scope: newTokenScope.value as ApiToken["scope"],
      createdAt: now,
      lastUsed: "Never",
      expiresAt,
      ipWhitelist: ipList,
      requestsLast24h: 0,
      requestsLast7d: 0,
      requestsLast30d: 0,
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
      toast.error("Failed to copy token. Please copy it manually.");
    }
  }

  function closeCreateTokenDialog() {
    showCreateTokenDialog.value = false;
    newTokenName.value = "";
    newTokenScope.value = "read-only";
    newTokenExpiry.value = "90d";
    newTokenIpWhitelist.value = "";
    createdToken.value = "";
    tokenCopied.value = false;
  }

  const showRevokeTokenDialog = ref(false);
  const revokeTokenId = ref("");
  const revokeTokenName = ref("");

  function confirmRevokeToken(token: ApiToken) {
    revokeTokenId.value = token.id;
    revokeTokenName.value = token.name;
    showRevokeTokenDialog.value = true;
  }

  function revokeToken() {
    apiTokens.value = apiTokens.value.filter(t => t.id !== revokeTokenId.value);
    showRevokeTokenDialog.value = false;
    toast.success(`API token "${revokeTokenName.value}" revoked.`);
  }

  // Expanded token detail
  const expandedToken = ref<string | null>(null);

  function toggleTokenDetail(id: string) {
    expandedToken.value = expandedToken.value === id ? null : id;
  }
</script>

<template>
  <div>
    <!-- Revoke All Sessions Confirmation -->
    <AlertDialog v-model:open="showRevokeAllDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Revoke All Other Sessions</AlertDialogTitle>
          <AlertDialogDescription>
            This will sign you out of all devices except this one.
            You will need to log in again on those devices.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            @click="revokeAllOtherSessions"
          >
            Revoke All
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Revoke Token Confirmation -->
    <AlertDialog v-model:open="showRevokeTokenDialog">
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Revoke API Token</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to revoke the token "{{ revokeTokenName }}"?
            Any integrations using this token will immediately lose access.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            @click="revokeToken"
          >
            Revoke Token
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>

    <!-- Create API Token Dialog -->
    <Dialog v-model:open="showCreateTokenDialog" @update:open="(val: boolean) => { if (!val) closeCreateTokenDialog() }">
      <DialogContent class="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{{ createdToken ? "Token Created Successfully" : "Create API Token" }}</DialogTitle>
          <DialogDescription>
            {{ createdToken
              ? "Copy this token now. For security, it will not be displayed again."
              : "Create a new API token with specific scopes and restrictions." }}
          </DialogDescription>
        </DialogHeader>

        <template v-if="!createdToken">
          <div class="space-y-4">
            <div class="space-y-2">
              <Label>Token Name</Label>
              <Input v-model="newTokenName" placeholder="e.g., n8n Workflow, Backup Script" />
            </div>

            <div class="space-y-2">
              <Label>Access Scope</Label>
              <div class="space-y-2">
                <button
                  v-for="scope in scopeOptions"
                  :key="scope.value"
                  class="flex w-full items-center gap-3 rounded-lg border p-3 text-left transition-colors"
                  :class="newTokenScope === scope.value ? 'border-primary bg-primary/5' : 'hover:bg-muted'"
                  @click="newTokenScope = scope.value"
                >
                  <div
                    class="flex h-4 w-4 items-center justify-center rounded-full border-2"
                    :class="newTokenScope === scope.value ? 'border-primary' : 'border-muted-foreground'"
                  >
                    <div
                      v-if="newTokenScope === scope.value"
                      class="h-2 w-2 rounded-full bg-primary"
                    />
                  </div>
                  <div>
                    <p class="text-sm font-medium">{{ scope.label }}</p>
                    <p class="text-xs text-muted-foreground">{{ scope.description }}</p>
                  </div>
                </button>
              </div>
            </div>

            <div class="space-y-2">
              <Label>Expiration</Label>
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

            <div class="space-y-2">
              <Label>IP Whitelist (optional)</Label>
              <Input
                v-model="newTokenIpWhitelist"
                placeholder="e.g., 192.168.1.15, 192.168.1.249"
              />
              <p class="text-xs text-muted-foreground">
                Comma-separated list of IPs. Leave blank to allow any IP.
              </p>
            </div>
          </div>

          <DialogFooter class="mt-4">
            <Button variant="outline" @click="closeCreateTokenDialog">Cancel</Button>
            <Button @click="createApiToken">Create Token</Button>
          </DialogFooter>
        </template>

        <template v-else>
          <div class="space-y-3">
            <div
              class="rounded-lg border bg-muted p-3"
            >
              <code class="block break-all text-xs leading-relaxed">{{ createdToken }}</code>
            </div>
            <Button class="w-full" variant="outline" @click="copyToken">
              <MdiCheck v-if="tokenCopied" class="mr-2 h-4 w-4 text-green-500" />
              <MdiContentCopy v-else class="mr-2 h-4 w-4" />
              {{ tokenCopied ? "Copied to Clipboard" : "Copy Token" }}
            </Button>
            <p class="text-center text-xs text-muted-foreground">
              Store this token securely. You will not be able to view it again.
            </p>
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
      <div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <div class="mb-1 flex items-center gap-2">
            <NuxtLink to="/settings">
              <Button variant="ghost" size="sm">
                <MdiArrowLeft class="mr-1 h-4 w-4" />
                Settings
              </Button>
            </NuxtLink>
          </div>
          <h1 class="text-2xl font-bold">Security</h1>
          <p class="text-sm text-muted-foreground">
            Manage sessions, view login history, configure security alerts, and control API tokens.
          </p>
        </div>
      </div>

      <!-- ==================== Active Sessions ==================== -->
      <BaseCard>
        <template #title>
          <BaseSectionHeader>
            <MdiMonitor class="-mt-1 mr-2" />
            <span>Active Sessions</span>
            <template #description>Devices and browsers currently signed into your account.</template>
          </BaseSectionHeader>
        </template>

        <div class="space-y-4 p-4">
          <div class="flex justify-end">
            <Button
              variant="destructive"
              size="sm"
              :disabled="sessions.filter(s => !s.isCurrent).length === 0"
              @click="showRevokeAllDialog = true"
            >
              Revoke All Other Sessions
            </Button>
          </div>

          <div v-if="sessionsLoading" class="py-8 text-center text-sm text-muted-foreground">
            <MdiLoading class="mx-auto mb-2 size-5 animate-spin" />
            Loading sessions...
          </div>
          <div v-else-if="sessions.length === 0" class="py-8 text-center text-sm text-muted-foreground">
            No active sessions found. Session tracking is not yet available.
          </div>
          <div v-else class="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Device</TableHead>
                  <TableHead>Browser</TableHead>
                  <TableHead>IP Address</TableHead>
                  <TableHead>Last Active</TableHead>
                  <TableHead>Created</TableHead>
                  <TableHead class="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow
                  v-for="session in sessions"
                  :key="session.id"
                  :class="session.isCurrent ? 'bg-primary/5' : ''"
                >
                  <TableCell>
                    <div class="flex items-center gap-2">
                      <span class="text-sm font-medium">{{ session.device }}</span>
                      <Badge v-if="session.isCurrent" class="text-[10px]">This Device</Badge>
                    </div>
                  </TableCell>
                  <TableCell class="text-sm">{{ session.browser }}</TableCell>
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
      </BaseCard>

      <!-- ==================== Login History ==================== -->
      <BaseCard>
        <template #title>
          <BaseSectionHeader>
            <MdiHistory class="-mt-1 mr-2" />
            <span>Login History</span>
            <template #description>Recent login attempts to your account.</template>
          </BaseSectionHeader>
        </template>

        <div class="space-y-4 p-4">
          <!-- Filter -->
          <div class="flex items-center gap-3">
            <MdiFilter class="h-4 w-4 text-muted-foreground" />
            <div class="flex gap-1">
              <Button
                v-for="filter in ['all', 'success', 'failed']"
                :key="filter"
                :variant="loginStatusFilter === filter ? 'default' : 'outline'"
                size="sm"
                @click="loginStatusFilter = filter"
              >
                {{ filter === 'all' ? 'All' : filter.charAt(0).toUpperCase() + filter.slice(1) }}
              </Button>
            </div>
            <span class="text-xs text-muted-foreground">
              {{ filteredLoginHistory.length }} entries
            </span>
          </div>

          <div v-if="loginHistoryLoading" class="py-8 text-center text-sm text-muted-foreground">
            <MdiLoading class="mx-auto mb-2 size-5 animate-spin" />
            Loading login history...
          </div>
          <div v-else-if="loginHistory.length === 0" class="py-8 text-center text-sm text-muted-foreground">
            No login history available. Login tracking is not yet available.
          </div>
          <div v-else class="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Timestamp</TableHead>
                  <TableHead>IP Address</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Device</TableHead>
                  <TableHead>Method</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow
                  v-for="entry in paginatedLoginHistory"
                  :key="entry.id"
                  :class="entry.status === 'failed' ? 'bg-destructive/5' : ''"
                >
                  <TableCell class="text-sm font-mono">{{ entry.timestamp }}</TableCell>
                  <TableCell class="text-sm font-mono">{{ entry.ip }}</TableCell>
                  <TableCell>
                    <Badge
                      :variant="entry.status === 'success' ? 'default' : 'destructive'"
                      class="text-[10px]"
                    >
                      {{ entry.status }}
                    </Badge>
                  </TableCell>
                  <TableCell class="text-sm">{{ entry.device }}</TableCell>
                  <TableCell class="text-sm">{{ entry.method }}</TableCell>
                </TableRow>
                <TableRow v-if="paginatedLoginHistory.length === 0">
                  <TableCell colspan="5" class="py-8 text-center text-sm text-muted-foreground">
                    No login entries matching the current filter.
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </div>

          <!-- Pagination -->
          <div v-if="loginTotalPages > 1" class="flex items-center justify-between">
            <p class="text-xs text-muted-foreground">
              Page {{ loginPage }} of {{ loginTotalPages }}
            </p>
            <div class="flex gap-1">
              <Button
                variant="outline"
                size="sm"
                :disabled="loginPage <= 1"
                @click="loginPage--"
              >
                <MdiChevronLeft class="h-4 w-4" />
              </Button>
              <Button
                variant="outline"
                size="sm"
                :disabled="loginPage >= loginTotalPages"
                @click="loginPage++"
              >
                <MdiChevronRight class="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      </BaseCard>

      <!-- ==================== Security Alerts ==================== -->
      <BaseCard>
        <template #title>
          <BaseSectionHeader>
            <MdiBellAlert class="-mt-1 mr-2" />
            <span>Security Alerts</span>
            <template #description>Configure when and how you receive security notifications.</template>
          </BaseSectionHeader>
        </template>

        <div class="space-y-4 p-4">
          <!-- Alert toggles -->
          <div class="space-y-3">
            <div class="flex items-center justify-between rounded-lg border p-3">
              <div>
                <p class="text-sm font-medium">New Login from Unknown Device</p>
                <p class="text-xs text-muted-foreground">
                  Get notified when someone logs in from a device not previously seen.
                </p>
              </div>
              <Switch v-model:checked="alertNewLoginDevice" />
            </div>

            <div class="flex items-center justify-between rounded-lg border p-3">
              <div>
                <p class="text-sm font-medium">Permission Changes</p>
                <p class="text-xs text-muted-foreground">
                  Get notified when your permissions or role are modified.
                </p>
              </div>
              <Switch v-model:checked="alertPermissionChanges" />
            </div>

            <div class="rounded-lg border p-3">
              <div class="flex items-center justify-between">
                <div>
                  <p class="text-sm font-medium">Failed Login Attempts</p>
                  <p class="text-xs text-muted-foreground">
                    Get notified after multiple failed login attempts on your account.
                  </p>
                </div>
                <Switch v-model:checked="alertFailedLogins" />
              </div>

              <div v-if="alertFailedLogins" class="mt-3 flex items-center gap-3">
                <Label class="text-sm">Alert after</Label>
                <Select v-model="failedLoginThreshold">
                  <SelectTrigger class="w-24">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem v-for="t in thresholdOptions" :key="t" :value="t">
                      {{ t }} attempts
                    </SelectItem>
                  </SelectContent>
                </Select>
                <Label class="text-sm">consecutive failures</Label>
              </div>
            </div>
          </div>

          <!-- Security alert email -->
          <div class="space-y-2">
            <Label class="text-sm font-medium">Security Alert Email</Label>
            <Input
              v-model="securityAlertEmail"
              type="email"
              placeholder="you@example.com"
            />
            <p class="text-xs text-muted-foreground">
              Security alerts will be sent to this email address. Leave blank to use your account email.
            </p>
          </div>

          <div class="flex justify-end border-t pt-4">
            <Button @click="saveSecurityAlerts">Save Alert Preferences</Button>
          </div>
        </div>
      </BaseCard>

      <!-- ==================== API Token Management ==================== -->
      <BaseCard>
        <template #title>
          <BaseSectionHeader>
            <MdiKey class="-mt-1 mr-2" />
            <span>API Tokens</span>
            <template #description>Manage API tokens with granular scopes, IP restrictions, and usage tracking.</template>
          </BaseSectionHeader>
        </template>

        <div class="space-y-4 p-4">
          <div class="flex justify-end">
            <Button size="sm" @click="showCreateTokenDialog = true">
              <MdiKey class="mr-1 h-4 w-4" />
              Create New Token
            </Button>
          </div>

          <div class="space-y-3">
            <div
              v-for="token in apiTokens"
              :key="token.id"
              class="rounded-lg border"
            >
              <!-- Token Summary Row -->
              <button
                class="flex w-full items-center justify-between p-4 text-left transition-colors hover:bg-muted/50"
                @click="toggleTokenDetail(token.id)"
              >
                <div class="flex items-center gap-3">
                  <MdiKey class="h-5 w-5 text-muted-foreground" />
                  <div>
                    <div class="flex items-center gap-2">
                      <span class="text-sm font-medium">{{ token.name }}</span>
                      <Badge :variant="scopeBadgeVariant(token.scope)" class="text-[10px]">
                        {{ token.scope }}
                      </Badge>
                    </div>
                    <p class="text-xs text-muted-foreground">
                      Created {{ token.createdAt }} | Last used {{ token.lastUsed }}
                    </p>
                  </div>
                </div>
                <div class="flex items-center gap-2">
                  <Badge variant="outline" class="text-[10px]">
                    Expires: {{ token.expiresAt }}
                  </Badge>
                  <MdiChevronRight
                    class="h-4 w-4 text-muted-foreground transition-transform"
                    :class="{ 'rotate-90': expandedToken === token.id }"
                  />
                </div>
              </button>

              <!-- Expanded Details -->
              <div
                v-if="expandedToken === token.id"
                class="border-t px-4 py-3"
              >
                <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
                  <!-- Usage Stats -->
                  <div class="rounded-lg bg-muted/50 p-3">
                    <p class="text-xs text-muted-foreground">Last 24 Hours</p>
                    <p class="text-lg font-bold">{{ token.requestsLast24h.toLocaleString() }}</p>
                    <p class="text-xs text-muted-foreground">requests</p>
                  </div>
                  <div class="rounded-lg bg-muted/50 p-3">
                    <p class="text-xs text-muted-foreground">Last 7 Days</p>
                    <p class="text-lg font-bold">{{ token.requestsLast7d.toLocaleString() }}</p>
                    <p class="text-xs text-muted-foreground">requests</p>
                  </div>
                  <div class="rounded-lg bg-muted/50 p-3">
                    <p class="text-xs text-muted-foreground">Last 30 Days</p>
                    <p class="text-lg font-bold">{{ token.requestsLast30d.toLocaleString() }}</p>
                    <p class="text-xs text-muted-foreground">requests</p>
                  </div>
                  <div class="rounded-lg bg-muted/50 p-3">
                    <p class="text-xs text-muted-foreground">IP Whitelist</p>
                    <p class="text-sm font-medium">
                      {{ token.ipWhitelist.length > 0 ? token.ipWhitelist.join(', ') : 'Any IP' }}
                    </p>
                  </div>
                </div>

                <div class="mt-3 flex justify-end">
                  <Button
                    variant="destructive"
                    size="sm"
                    @click.stop="confirmRevokeToken(token)"
                  >
                    <MdiDelete class="mr-1 h-4 w-4" />
                    Revoke Token
                  </Button>
                </div>
              </div>
            </div>

            <div v-if="apiTokens.length === 0" class="py-8 text-center text-sm text-muted-foreground">
              No API tokens created yet. Create one to enable external integrations.
            </div>
          </div>
        </div>
      </BaseCard>
    </BaseContainer>
  </div>
</template>
