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
}
