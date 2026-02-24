export interface WidgetConfig {
  id: string;
  type: WidgetType;
  title: string;
  col: number;
  row: number;
  visible: boolean;
}

export type WidgetType =
  | "quick-stats"
  | "recent-items"
  | "low-stock"
  | "warranty-tracker"
  | "maintenance-due"
  | "plugin-status"
  | "quick-actions"
  | "activity-feed"
  | "value-by-location"
  | "storage-map";

export const WIDGET_DEFINITIONS: Record<WidgetType, { title: string; description: string }> = {
  "quick-stats": { title: "Quick Stats", description: "Total items, value, locations, and tags" },
  "recent-items": { title: "Recent Items", description: "Last 10 items added or modified" },
  "low-stock": { title: "Low Stock Alerts", description: "Items at or below reorder threshold" },
  "warranty-tracker": { title: "Warranty Tracker", description: "Warranties expiring in the next 90 days" },
  "maintenance-due": { title: "Maintenance Due", description: "Upcoming and overdue maintenance tasks" },
  "plugin-status": { title: "Plugin Status", description: "Status of all installed plugins" },
  "quick-actions": { title: "Quick Actions", description: "Shortcuts to common tasks" },
  "activity-feed": { title: "Activity Feed", description: "Recent activity across the system" },
  "value-by-location": { title: "Value by Location", description: "Item value distribution by location" },
  "storage-map": { title: "Storage Map", description: "Visual tree of your location hierarchy" },
};

const STORAGE_KEY = "homeboxng-dashboard-layout";

function getDefaultLayout(): WidgetConfig[] {
  return [
    { id: "w-1", type: "quick-stats", title: "Quick Stats", col: 0, row: 0, visible: true },
    { id: "w-2", type: "recent-items", title: "Recent Items", col: 1, row: 0, visible: true },
    { id: "w-3", type: "quick-actions", title: "Quick Actions", col: 2, row: 0, visible: true },
    { id: "w-4", type: "warranty-tracker", title: "Warranty Tracker", col: 0, row: 1, visible: true },
    { id: "w-5", type: "activity-feed", title: "Activity Feed", col: 1, row: 1, visible: true },
    { id: "w-6", type: "value-by-location", title: "Value by Location", col: 2, row: 1, visible: true },
    { id: "w-7", type: "low-stock", title: "Low Stock Alerts", col: 0, row: 2, visible: false },
    { id: "w-8", type: "maintenance-due", title: "Maintenance Due", col: 1, row: 2, visible: false },
    { id: "w-9", type: "plugin-status", title: "Plugin Status", col: 2, row: 2, visible: false },
    { id: "w-10", type: "storage-map", title: "Storage Map", col: 0, row: 3, visible: false },
  ];
}

export function useWidgetLayout() {
  const widgets = ref<WidgetConfig[]>(getDefaultLayout());
  const isCustomizing = ref(false);

  function loadLayout() {
    if (import.meta.server) return;
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (stored) {
        const parsed = JSON.parse(stored) as WidgetConfig[];
        if (Array.isArray(parsed) && parsed.length > 0) {
          // Merge with defaults to pick up any new widget types added in updates
          const storedTypes = new Set(parsed.map(w => w.type));
          const defaults = getDefaultLayout();
          const missing = defaults.filter(d => !storedTypes.has(d.type));
          widgets.value = [...parsed, ...missing];
          return;
        }
      }
    } catch {
      // Ignore parse errors, fall through to defaults
    }
    widgets.value = getDefaultLayout();
  }

  function saveLayout() {
    if (import.meta.server) return;
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(widgets.value));
    } catch {
      // Silently fail if storage is full
    }
  }

  function addWidget(type: WidgetType) {
    const existing = widgets.value.find(w => w.type === type);
    if (existing) {
      existing.visible = true;
    } else {
      const def = WIDGET_DEFINITIONS[type];
      const maxRow = Math.max(0, ...widgets.value.map(w => w.row));
      widgets.value.push({
        id: `w-${Date.now()}`,
        type,
        title: def.title,
        col: 0,
        row: maxRow + 1,
        visible: true,
      });
    }
    saveLayout();
  }

  function removeWidget(id: string) {
    const widget = widgets.value.find(w => w.id === id);
    if (widget) {
      widget.visible = false;
      saveLayout();
    }
  }

  function moveWidget(id: string, direction: "up" | "down") {
    const visible = widgets.value.filter(w => w.visible);
    const idx = visible.findIndex(w => w.id === id);
    if (idx === -1) return;

    const swapIdx = direction === "up" ? idx - 1 : idx + 1;
    if (swapIdx < 0 || swapIdx >= visible.length) return;

    const a = visible[idx];
    const b = visible[swapIdx];
    if (!a || !b) return;
    const tmpRow = a.row;
    const tmpCol = a.col;
    a.row = b.row;
    a.col = b.col;
    b.row = tmpRow;
    b.col = tmpCol;
    saveLayout();
  }

  function resetLayout() {
    widgets.value = getDefaultLayout();
    saveLayout();
  }

  const visibleWidgets = computed(() => {
    return widgets.value
      .filter(w => w.visible)
      .sort((a, b) => a.row !== b.row ? a.row - b.row : a.col - b.col);
  });

  const hiddenWidgets = computed(() => {
    return widgets.value.filter(w => !w.visible);
  });

  // Load on init (client only)
  if (!import.meta.server) {
    loadLayout();
  }

  return {
    widgets,
    visibleWidgets,
    hiddenWidgets,
    isCustomizing,
    addWidget,
    removeWidget,
    moveWidget,
    resetLayout,
    saveLayout,
    loadLayout,
  };
}
