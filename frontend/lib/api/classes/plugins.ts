import { BaseAPI, route } from "../base";

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

export class PluginsAPI extends BaseAPI {
  // Plugin Management
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

  // Configuration
  getConfig(name: string) {
    return this.http.get<PluginConfig[]>({ url: route(`/plugins/${name}/config`) });
  }

  saveConfig(name: string, values: Record<string, string>) {
    return this.http.put<Record<string, string>, void>({ url: route(`/plugins/${name}/config`), body: values });
  }

  // Permissions
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

  // Logs
  getLogs(name: string) {
    return this.http.get<PluginLogEntry[]>({ url: route(`/plugins/${name}/logs`) });
  }

  // Catalog (HACS-like)
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

  // Notification Platforms
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
}
