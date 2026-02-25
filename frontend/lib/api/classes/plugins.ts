import { BaseAPI, route } from "../base";

// ========== Plugin Management Types ==========

export interface PluginInfo {
  name: string;
  version: string;
  description: string;
  author: string;
  builtIn: boolean;
  enabled?: boolean;
  status?: string;
}

export interface PluginConfig {
  key: string;
  label: string;
  description: string;
  type: string;
  default: string;
  options?: string[];
  required: boolean;
  value?: string;
  envVar?: string;
}

export interface PluginPermission {
  permission: string;
  reason: string;
  required: boolean;
  granted: boolean;
}

export interface PluginLogEntry {
  level: string;
  message: string;
  timestamp: string;
  fields?: Record<string, unknown>;
}

export interface CatalogPlugin {
  name: string;
  version: string;
  description: string;
  author: string;
  repository: string;
  homepage?: string;
  category?: string;
  installed: boolean;
}

export interface NotificationPlatform {
  platform: string;
  label: string;
  icon: string;
  enabled: boolean;
  configured: boolean;
}

export interface NotificationCategory {
  id: string;
  label: string;
  description: string;
}

// ========== AI Vision Types ==========

export interface VisionScanResult {
  id: string;
  imageUrl: string;
  identifiedItem: string;
  confidence: number;
  suggestedLocation: string;
  suggestedLabels: string[];
  timestamp: string;
  matched: boolean;
  itemId?: string;
}

export interface VisionScanRequest {
  images: string[]; // base64 or URLs
  groupSimilar: boolean;
}

// ========== Label Printer Types ==========

export interface PrintJob {
  id: string;
  itemId: string;
  itemName: string;
  labelSize: string;
  orientation: "horizontal" | "vertical";
  halfLabel: boolean;
  status: "pending" | "printing" | "done" | "error";
  timestamp: string;
}

export interface PrintRequest {
  itemId: string;
  labelSize: string;
  orientation: "horizontal" | "vertical";
  halfLabel: boolean;
  copies: number;
}

export interface PrintPreview {
  previewUrl: string;
  width: number;
  height: number;
}

// ========== Lending Types ==========

export interface Loan {
  id: string;
  itemId: string;
  itemName: string;
  borrowerName: string;
  borrowerEmail?: string;
  checkoutDate: string;
  dueDate: string;
  returnDate?: string;
  notes: string;
  overdue: boolean;
}

export interface CheckoutRequest {
  itemId: string;
  borrowerName: string;
  borrowerEmail?: string;
  dueDate: string;
  notes?: string;
}

// ========== Shopping Types ==========

export interface ShoppingItem {
  id: string;
  name: string;
  quantity: number;
  store: string;
  priority: "high" | "medium" | "low";
  notes: string;
  purchased: boolean;
  addedDate: string;
}

export interface ReorderRule {
  id: string;
  itemId: string;
  itemName: string;
  minQuantity: number;
  reorderQuantity: number;
  preferredStore: string;
}

// ========== Analytics Types ==========

export interface AnalyticsOverview {
  totalItems: number;
  totalValue: number;
  itemsThisMonth: number;
  locationsCount: number;
  tagsCount: number;
}

export interface ValueByLocation {
  locationId: string;
  locationName: string;
  totalValue: number;
  itemCount: number;
}

export interface ActivityEntry {
  id: string;
  action: "added" | "updated" | "moved" | "deleted";
  itemName: string;
  details: string;
  timestamp: string;
}

export interface WarrantyAlert {
  itemId: string;
  itemName: string;
  warrantyExpires: string;
  daysRemaining: number;
}

// ========== System Alerts Types ==========

export interface SystemAlert {
  id: string;
  severity: "critical" | "warning" | "info";
  category: "token" | "warranty" | "maintenance" | "plugin" | "system";
  title: string;
  description: string;
  itemId?: string;
  daysLeft?: number;
  actionUrl?: string;
  timestamp: string;
}

// ========== Paperless Types ==========

export interface PaperlessConnection {
  url: string;
  configured: boolean;
  lastSync: string;
  documentsLinked: number;
}

export interface DocumentLink {
  id: string;
  itemId: string;
  itemName: string;
  documentId: number;
  documentTitle: string;
  documentType: string;
  linkedDate: string;
}

// ========== Eye-Fi Types ==========

export interface EyeFiStatus {
  serverRunning: boolean;
  lastUpload: string;
  photosReceived: number;
  watchDirectory: string;
}

export interface EyeFiUpload {
  id: string;
  filename: string;
  timestamp: string;
  size: number;
  processed: boolean;
  scanResult?: VisionScanResult;
}

// ========== Excel Export Types ==========

export interface ExportRequest {
  format: "csv" | "tsv";
  dataTypes: ("items" | "locations" | "labels")[];
  fields: string[];
  startDate?: string;
  endDate?: string;
}

export interface ImportResult {
  recordsImported: number;
  recordsSkipped: number;
  errors: string[];
}

// ========== Maintenance Scheduler Types ==========

export interface MaintenanceSchedule {
  id: string;
  itemId: string;
  itemName: string;
  type: string;
  frequency: "one-time" | "weekly" | "monthly" | "quarterly" | "yearly";
  nextDue: string;
  lastCompleted?: string;
  assignedTo?: string;
  notes: string;
}

export interface ServiceProvider {
  id: string;
  name: string;
  phone: string;
  email: string;
  specialty: string;
  rating: number;
  notes: string;
}

export interface RepairEntry {
  id: string;
  itemId: string;
  itemName: string;
  date: string;
  providerId?: string;
  providerName?: string;
  cost: number;
  description: string;
  warrantyClaim: boolean;
}

// ========== Health & Monitoring Types ==========

export interface HealthReport {
  pluginName: string;
  status: "healthy" | "degraded" | "unhealthy" | "unknown";
  message: string;
  lastCheck: string;
  uptime: number;
  checks: Record<string, { name: string; status: string; message: string; duration: number }>;
}

export interface HealthSummary {
  totalPlugins: number;
  healthy: number;
  degraded: number;
  unhealthy: number;
  unknown: number;
  reports: HealthReport[];
}

export interface PluginMetrics {
  requestCount: number;
  errorCount: number;
  lastError: string;
  lastErrorTime: string;
  averageLatency: number;
  p95Latency: number;
  startTime: string;
  uptime: number;
}

// ========== Webhook Types ==========

export interface WebhookConfig {
  id: string;
  pluginName: string;
  url: string;
  secret: string;
  events: string[];
  active: boolean;
  createdAt: string;
  lastTriggered: string;
  failureCount: number;
  maxRetries: number;
}

// ========== Audit Types ==========

export interface AuditEntry {
  id: string;
  timestamp: string;
  pluginName: string;
  action: string;
  actor: string;
  details: Record<string, string>;
  severity: "info" | "warning" | "critical";
}

export interface AuditFilter {
  pluginName?: string;
  action?: string;
  severity?: string;
  since?: string;
  limit?: number;
}

// ========== Dependency Types ==========

export interface DependencyInfo {
  name: string;
  minVersion: string;
  optional: boolean;
  resolved: boolean;
  availableVersion?: string;
}

// ========== Session & Token Types ==========

export interface UserSession {
  id: string;
  device: string;
  browser: string;
  ip: string;
  lastActive: string;
  createdAt: string;
  current: boolean;
}

export interface ApiToken {
  id: string;
  name: string;
  scope: "read-only" | "read-write" | "admin";
  createdAt: string;
  lastUsed: string;
  expiresAt: string;
  requestCount24h?: number;
  requestCount7d?: number;
  requestCount30d?: number;
}

export interface ApiTokenCreated extends ApiToken {
  token: string; // Only returned on creation
}

export interface LoginHistoryEntry {
  id: string;
  timestamp: string;
  ip: string;
  status: "success" | "failed";
  device: string;
  method: string;
}

export interface SecurityAlertPreferences {
  notifyNewLogin: boolean;
  notifyPermissionChanges: boolean;
  notifyFailedLogins: boolean;
  failedLoginThreshold: number;
  alertEmail: string;
}

// ========== HA Bridge Types ==========

export interface HABridgeStatus {
  connected: boolean;
  lastSync: string;
  entitiesSynced: number;
  automationsCount: number;
}

export interface HAEntityMapping {
  id: string;
  entityId: string;
  entityName: string;
  itemId: string;
  itemName: string;
  linkType: string;
  syncStatus: "synced" | "pending" | "error";
  lastSynced: string;
}

export interface HAAutomation {
  id: string;
  name: string;
  trigger: string;
  action: string;
  enabled: boolean;
}

export interface HADiscoveredEntity {
  entityId: string;
  name: string;
  domain: string;
  state: string;
  mapped: boolean;
}

export class PluginsAPI extends BaseAPI {
  // ========== Plugin Management ==========

  getAll() {
    return this.http.get<PluginInfo[]>({ url: route("/plugins") });
  }

  get(name: string) {
    return this.http.get<PluginInfo>({ url: route(`/plugins/${name}`) });
  }

  enable(name: string) {
    return this.http.post<void, void>({ url: route(`/plugins/${name}/enable`) });
  }

  disable(name: string) {
    return this.http.post<void, void>({ url: route(`/plugins/${name}/disable`) });
  }

  // ========== Plugin Configuration ==========

  getConfig(name: string) {
    return this.http.get<PluginConfig[]>({ url: route(`/plugins/${name}/config`) });
  }

  saveConfig(name: string, values: Record<string, string>) {
    return this.http.put<Record<string, string>, void>({ url: route(`/plugins/${name}/config`), body: values });
  }

  // ========== Plugin Permissions ==========

  getPermissions(name: string) {
    return this.http.get<PluginPermission[]>({ url: route(`/plugins/${name}/permissions`) });
  }

  grantPermission(name: string, permission: string) {
    return this.http.post<{ permission: string }, void>({
      url: route(`/plugins/${name}/permissions/grant`),
      body: { permission },
    });
  }

  revokePermission(name: string, permission: string) {
    return this.http.post<{ permission: string }, void>({
      url: route(`/plugins/${name}/permissions/revoke`),
      body: { permission },
    });
  }

  // ========== Plugin Logs ==========

  getLogs(name: string) {
    return this.http.get<PluginLogEntry[]>({ url: route(`/plugins/${name}/logs`) });
  }

  // ========== Catalog (HACS-like) ==========

  getCatalog() {
    return this.http.get<CatalogPlugin[]>({ url: route("/plugins/catalog") });
  }

  refreshCatalog() {
    return this.http.post<void, void>({ url: route("/plugins/catalog/refresh") });
  }

  installPlugin(name: string) {
    return this.http.post<{ name: string }, void>({
      url: route("/plugins/catalog/install"),
      body: { name },
    });
  }

  // ========== Notification Platforms ==========

  getNotificationPlatforms() {
    return this.http.get<NotificationPlatform[]>({ url: route("/notifications/platforms") });
  }

  testNotification(platform: string) {
    return this.http.post<{ platform: string }, void>({
      url: route("/notifications/test"),
      body: { platform },
    });
  }

  getNotificationCategories() {
    return this.http.get<NotificationCategory[]>({ url: route("/notifications/categories") });
  }

  // ========== AI Vision ==========

  scanImages(req: VisionScanRequest) {
    return this.http.post<VisionScanRequest, VisionScanResult[]>({
      url: route("/plugins/ai-vision/scan"),
      body: req,
    });
  }

  getRecentScans() {
    return this.http.get<VisionScanResult[]>({ url: route("/plugins/ai-vision/scans") });
  }

  matchScanToItem(scanId: string, itemId: string) {
    return this.http.post<{ scanId: string; itemId: string }, VisionScanResult>({
      url: route("/plugins/ai-vision/match"),
      body: { scanId, itemId },
    });
  }

  dismissScan(scanId: string) {
    return this.http.delete<void>({ url: route(`/plugins/ai-vision/scans/${scanId}`) });
  }

  // ========== Label Printer ==========

  printLabel(req: PrintRequest) {
    return this.http.post<PrintRequest, PrintJob>({
      url: route("/plugins/label-printer/print"),
      body: req,
    });
  }

  getPrintPreview(req: PrintRequest) {
    return this.http.post<PrintRequest, PrintPreview>({
      url: route("/plugins/label-printer/preview"),
      body: req,
    });
  }

  getPrintHistory() {
    return this.http.get<PrintJob[]>({ url: route("/plugins/label-printer/history") });
  }

  cancelPrintJob(jobId: string) {
    return this.http.post<void, void>({ url: route(`/plugins/label-printer/jobs/${jobId}/cancel`) });
  }

  // ========== Lending ==========

  getActiveLoans() {
    return this.http.get<Loan[]>({ url: route("/plugins/lending/loans", { active: true }) });
  }

  getLoanHistory() {
    return this.http.get<Loan[]>({ url: route("/plugins/lending/loans") });
  }

  checkoutItem(req: CheckoutRequest) {
    return this.http.post<CheckoutRequest, Loan>({
      url: route("/plugins/lending/checkout"),
      body: req,
    });
  }

  returnItem(loanId: string, notes?: string) {
    return this.http.post<{ notes?: string }, Loan>({
      url: route(`/plugins/lending/loans/${loanId}/return`),
      body: { notes },
    });
  }

  getOverdueLoans() {
    return this.http.get<Loan[]>({ url: route("/plugins/lending/loans", { overdue: true }) });
  }

  // ========== Shopping ==========

  getShoppingList() {
    return this.http.get<ShoppingItem[]>({ url: route("/plugins/shopping/list") });
  }

  addShoppingItem(item: Omit<ShoppingItem, "id" | "purchased" | "addedDate">) {
    return this.http.post<typeof item, ShoppingItem>({
      url: route("/plugins/shopping/list"),
      body: item,
    });
  }

  updateShoppingItem(id: string, item: Partial<Omit<ShoppingItem, "id">>) {
    return this.http.put<typeof item, ShoppingItem>({
      url: route(`/plugins/shopping/list/${id}`),
      body: item,
    });
  }

  removeShoppingItem(id: string) {
    return this.http.delete<void>({ url: route(`/plugins/shopping/list/${id}`) });
  }

  markPurchased(id: string) {
    return this.http.post<void, ShoppingItem>({
      url: route(`/plugins/shopping/list/${id}/purchased`),
    });
  }

  getReorderRules() {
    return this.http.get<ReorderRule[]>({ url: route("/plugins/shopping/reorder-rules") });
  }

  addReorderRule(rule: Omit<ReorderRule, "id" | "itemName">) {
    return this.http.post<typeof rule, ReorderRule>({
      url: route("/plugins/shopping/reorder-rules"),
      body: rule,
    });
  }

  removeReorderRule(id: string) {
    return this.http.delete<void>({ url: route(`/plugins/shopping/reorder-rules/${id}`) });
  }

  // ========== Analytics ==========

  getOverview() {
    return this.http.get<AnalyticsOverview>({ url: route("/plugins/analytics/overview") });
  }

  getValueByLocation() {
    return this.http.get<ValueByLocation[]>({ url: route("/plugins/analytics/value-by-location") });
  }

  getActivityFeed(limit?: number) {
    return this.http.get<ActivityEntry[]>({
      url: route("/plugins/analytics/activity", limit !== undefined ? { limit } : {}),
    });
  }

  getWarrantyAlerts(days?: number) {
    return this.http.get<WarrantyAlert[]>({
      url: route("/plugins/analytics/warranty-alerts", days !== undefined ? { days } : {}),
    });
  }

  getSystemAlerts() {
    return this.http.get<SystemAlert[]>({
      url: route("/plugins/system-alerts"),
    });
  }

  // ========== Paperless ==========

  getPaperlessStatus() {
    return this.http.get<PaperlessConnection>({ url: route("/plugins/paperless/status") });
  }

  testPaperlessConnection() {
    return this.http.post<void, PaperlessConnection>({
      url: route("/plugins/paperless/test"),
    });
  }

  syncPaperless() {
    return this.http.post<void, PaperlessConnection>({
      url: route("/plugins/paperless/sync"),
    });
  }

  getDocumentLinks() {
    return this.http.get<DocumentLink[]>({ url: route("/plugins/paperless/links") });
  }

  linkDocument(itemId: string, documentId: number) {
    return this.http.post<{ itemId: string; documentId: number }, DocumentLink>({
      url: route("/plugins/paperless/links"),
      body: { itemId, documentId },
    });
  }

  unlinkDocument(linkId: string) {
    return this.http.delete<void>({ url: route(`/plugins/paperless/links/${linkId}`) });
  }

  // ========== Eye-Fi ==========

  getEyeFiStatus() {
    return this.http.get<EyeFiStatus>({ url: route("/plugins/eye-fi/status") });
  }

  getRecentUploads() {
    return this.http.get<EyeFiUpload[]>({ url: route("/plugins/eye-fi/uploads") });
  }

  testEyeFiServer() {
    return this.http.post<void, EyeFiStatus>({
      url: route("/plugins/eye-fi/test"),
    });
  }

  // ========== Excel Export ==========

  exportData(req: ExportRequest) {
    return this.http.post<ExportRequest, Blob>({
      url: route("/plugins/excel-export/export"),
      body: req,
    });
  }

  getAvailableFields() {
    return this.http.get<string[]>({ url: route("/plugins/excel-export/fields") });
  }

  importData(file: File, mapping: Record<string, string>, conflictMode: string) {
    const formData = new FormData();
    formData.append("file", file);
    formData.append("mapping", JSON.stringify(mapping));
    formData.append("conflictMode", conflictMode);

    return this.http.post<FormData, ImportResult>({
      url: route("/plugins/excel-export/import"),
      data: formData,
    });
  }

  // ========== Maintenance Scheduler ==========

  getSchedules() {
    return this.http.get<MaintenanceSchedule[]>({ url: route("/plugins/maintenance-scheduler/schedules") });
  }

  createSchedule(schedule: Omit<MaintenanceSchedule, "id" | "itemName" | "lastCompleted">) {
    return this.http.post<typeof schedule, MaintenanceSchedule>({
      url: route("/plugins/maintenance-scheduler/schedules"),
      body: schedule,
    });
  }

  deleteSchedule(id: string) {
    return this.http.delete<void>({ url: route(`/plugins/maintenance-scheduler/schedules/${id}`) });
  }

  getProviders() {
    return this.http.get<ServiceProvider[]>({ url: route("/plugins/maintenance-scheduler/providers") });
  }

  addProvider(provider: Omit<ServiceProvider, "id">) {
    return this.http.post<typeof provider, ServiceProvider>({
      url: route("/plugins/maintenance-scheduler/providers"),
      body: provider,
    });
  }

  getRepairLog() {
    return this.http.get<RepairEntry[]>({ url: route("/plugins/maintenance-scheduler/repairs") });
  }

  addRepairEntry(entry: Omit<RepairEntry, "id" | "itemName" | "providerName">) {
    return this.http.post<typeof entry, RepairEntry>({
      url: route("/plugins/maintenance-scheduler/repairs"),
      body: entry,
    });
  }

  // ========== Plugin Health & Monitoring ==========

  getHealthSummary() {
    return this.http.get<HealthSummary>({ url: route("/plugins/health") });
  }

  getPluginHealth(name: string) {
    return this.http.get<HealthReport>({ url: route(`/plugins/${name}/health`) });
  }

  getPluginMetrics(name: string) {
    return this.http.get<PluginMetrics>({ url: route(`/plugins/${name}/metrics`) });
  }

  getAllMetrics() {
    return this.http.get<Record<string, PluginMetrics>>({ url: route("/plugins/metrics") });
  }

  // ========== Webhooks ==========

  getWebhooks(pluginName: string) {
    return this.http.get<WebhookConfig[]>({ url: route(`/plugins/${pluginName}/webhooks`) });
  }

  createWebhook(pluginName: string, webhook: Omit<WebhookConfig, "id" | "createdAt" | "lastTriggered" | "failureCount">) {
    return this.http.post<typeof webhook, WebhookConfig>({
      url: route(`/plugins/${pluginName}/webhooks`),
      body: webhook,
    });
  }

  deleteWebhook(pluginName: string, webhookId: string) {
    return this.http.delete<void>({ url: route(`/plugins/${pluginName}/webhooks/${webhookId}`) });
  }

  // ========== Audit Trail ==========

  getAuditLog(filter?: AuditFilter) {
    const params = new URLSearchParams();
    if (filter?.pluginName) params.set("plugin", filter.pluginName);
    if (filter?.action) params.set("action", filter.action);
    if (filter?.severity) params.set("severity", filter.severity);
    if (filter?.limit) params.set("limit", String(filter.limit));
    if (filter?.since) params.set("since", filter.since);
    const qs = params.toString();
    return this.http.get<AuditEntry[]>({ url: route(`/plugins/audit${qs ? `?${qs}` : ""}`) });
  }

  exportAuditLog(format: "json" | "csv" = "json") {
    return this.http.get<string>({ url: route(`/plugins/audit/export?format=${format}`) });
  }

  // ========== Plugin Dependencies ==========

  getPluginDependencies(name: string) {
    return this.http.get<DependencyInfo[]>({ url: route(`/plugins/${name}/dependencies`) });
  }

  // ========== Session Management ==========

  getActiveSessions() {
    return this.http.get<UserSession[]>({ url: route("/users/self/sessions") });
  }

  revokeSession(sessionId: string) {
    return this.http.delete<void>({ url: route(`/users/self/sessions/${sessionId}`) });
  }

  revokeAllOtherSessions() {
    return this.http.post<void, void>({ url: route("/users/self/sessions/revoke-others") });
  }

  // ========== API Tokens ==========

  getApiTokens() {
    return this.http.get<ApiToken[]>({ url: route("/users/self/api-tokens") });
  }

  createApiToken(token: { name: string; expiresIn: string; scope: string }) {
    return this.http.post<typeof token, ApiTokenCreated>({
      url: route("/users/self/api-tokens"),
      body: token,
    });
  }

  revokeApiToken(tokenId: string) {
    return this.http.delete<void>({ url: route(`/users/self/api-tokens/${tokenId}`) });
  }

  // ========== Login History ==========

  getLoginHistory(page: number = 1, pageSize: number = 50, status?: string) {
    const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) });
    if (status) params.set("status", status);
    return this.http.get<LoginHistoryEntry[]>({ url: route(`/users/self/login-history?${params}`) });
  }

  // ========== Security Alerts ==========

  getSecurityAlertPreferences() {
    return this.http.get<SecurityAlertPreferences>({ url: route("/users/self/security-alerts") });
  }

  updateSecurityAlertPreferences(prefs: SecurityAlertPreferences) {
    return this.http.put<SecurityAlertPreferences, void>({
      url: route("/users/self/security-alerts"),
      body: prefs,
    });
  }

  // ========== HA Bridge ==========

  getHABridgeStatus() {
    return this.http.get<HABridgeStatus>({ url: route("/plugins/ha-bridge/status") });
  }

  getHABridgeEntities() {
    return this.http.get<HAEntityMapping[]>({ url: route("/plugins/ha-bridge/entities") });
  }

  mapHAEntity(mapping: { entityId: string; itemId: string; linkType: string }) {
    return this.http.post<typeof mapping, HAEntityMapping>({
      url: route("/plugins/ha-bridge/entities/map"),
      body: mapping,
    });
  }

  removeHAEntityMapping(id: string) {
    return this.http.delete<void>({ url: route(`/plugins/ha-bridge/entities/${id}`) });
  }

  getHAAutomations() {
    return this.http.get<HAAutomation[]>({ url: route("/plugins/ha-bridge/automations") });
  }

  createHAAutomation(automation: Omit<HAAutomation, "id">) {
    return this.http.post<typeof automation, HAAutomation>({
      url: route("/plugins/ha-bridge/automations"),
      body: automation,
    });
  }

  deleteHAAutomation(id: string) {
    return this.http.delete<void>({ url: route(`/plugins/ha-bridge/automations/${id}`) });
  }

  syncHA() {
    return this.http.post<void, { synced: number }>({ url: route("/plugins/ha-bridge/sync") });
  }

  discoverHAEntities() {
    return this.http.get<HADiscoveredEntity[]>({ url: route("/plugins/ha-bridge/discover") });
  }
}
