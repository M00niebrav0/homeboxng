<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import { route } from "~/lib/api/base";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Maintenance Scheduler",
  });

  const api = useUserApi();

  // --- Types ---

  interface ScheduleEntry {
    id: string;
    itemId: string;
    itemName: string;
    maintenanceType: string;
    frequency: string;
    nextDue: string;
    lastCompleted: string;
    notes: string;
    assignedTo: string;
  }

  interface ServiceProvider {
    id: string;
    name: string;
    phone: string;
    email: string;
    specialty: string;
    rating: number;
    notes: string;
  }

  interface RepairLogEntry {
    id: string;
    itemId: string;
    itemName: string;
    date: string;
    providerId: string;
    providerName: string;
    cost: number;
    description: string;
    warrantyClaim: boolean;
  }

  interface CalendarEvent {
    date: string;
    items: Array<{ id: string; itemName: string; type: string }>;
  }

  // --- State ---

  const schedules = ref<ScheduleEntry[]>([]);
  const providers = ref<ServiceProvider[]>([]);
  const repairLog = ref<RepairLogEntry[]>([]);
  const calendarEvents = ref<CalendarEvent[]>([]);
  const loading = ref(false);

  const showCreateSchedule = ref(false);
  const showAddProvider = ref(false);

  // Create schedule form
  const newSchedule = reactive({
    itemId: "",
    itemName: "",
    maintenanceType: "inspection",
    frequency: "monthly",
    startDate: "",
    notes: "",
    assignedTo: "",
  });

  // Add provider form
  const newProvider = reactive({
    name: "",
    phone: "",
    email: "",
    specialty: "general",
    rating: 3,
    notes: "",
  });

  // Item search for schedule creation
  const itemSearch = ref("");
  const itemSearchResults = ref<Array<{ id: string; name: string }>>([]);

  const maintenanceTypes = [
    { value: "inspection", label: "Inspection" },
    { value: "cleaning", label: "Cleaning" },
    { value: "calibration", label: "Calibration" },
    { value: "repair", label: "Repair" },
    { value: "replacement", label: "Replacement" },
    { value: "lubrication", label: "Lubrication" },
    { value: "testing", label: "Testing" },
    { value: "other", label: "Other" },
  ];

  const frequencyOptions = [
    { value: "one-time", label: "One-time" },
    { value: "weekly", label: "Weekly" },
    { value: "monthly", label: "Monthly" },
    { value: "quarterly", label: "Quarterly" },
    { value: "yearly", label: "Yearly" },
  ];

  const specialtyOptions = [
    { value: "general", label: "General" },
    { value: "electrical", label: "Electrical" },
    { value: "plumbing", label: "Plumbing" },
    { value: "hvac", label: "HVAC" },
    { value: "appliance", label: "Appliance Repair" },
    { value: "automotive", label: "Automotive" },
    { value: "electronics", label: "Electronics" },
    { value: "other", label: "Other" },
  ];

  // --- Calendar ---

  const today = new Date();
  const calendarStartDate = ref(new Date(today.getFullYear(), today.getMonth(), today.getDate()));

  const calendarDays = computed(() => {
    const days: Array<{ date: Date; dateStr: string; events: CalendarEvent["items"]; isToday: boolean; isPast: boolean }> = [];
    const start = new Date(calendarStartDate.value);

    // Align to start of week (Sunday)
    const dayOfWeek = start.getDay();
    start.setDate(start.getDate() - dayOfWeek);

    for (let i = 0; i < 35; i++) {
      const d = new Date(start);
      d.setDate(start.getDate() + i);
      const dateStr = d.toISOString().slice(0, 10);
      const eventData = calendarEvents.value.find(e => e.date === dateStr);

      days.push({
        date: d,
        dateStr,
        events: eventData?.items || [],
        isToday: dateStr === today.toISOString().slice(0, 10),
        isPast: d < today && dateStr !== today.toISOString().slice(0, 10),
      });
    }
    return days;
  });

  const calendarTitle = computed(() => {
    const start = calendarDays.value[0]?.date;
    const end = calendarDays.value[calendarDays.value.length - 1]?.date;
    if (!start || !end) return "";
    const startMonth = start.toLocaleDateString("en-US", { month: "long", year: "numeric" });
    const endMonth = end.toLocaleDateString("en-US", { month: "long", year: "numeric" });
    return startMonth === endMonth ? startMonth : `${startMonth} - ${endMonth}`;
  });

  function prevWeek() {
    const d = new Date(calendarStartDate.value);
    d.setDate(d.getDate() - 7);
    calendarStartDate.value = d;
  }

  function nextWeek() {
    const d = new Date(calendarStartDate.value);
    d.setDate(d.getDate() + 7);
    calendarStartDate.value = d;
  }

  function goToToday() {
    calendarStartDate.value = new Date(today.getFullYear(), today.getMonth(), today.getDate());
  }

  // --- Data Loading ---

  const { refresh: refreshSchedules } = useAsyncData("maint-schedules", async () => {
    try {
      const { data } = await api.http.get<ScheduleEntry[]>({ url: route("/plugins/maintenance-scheduler/schedules") });
      schedules.value = data || [];
      return data;
    } catch {
      return [];
    }
  });

  const { refresh: refreshProviders } = useAsyncData("maint-providers", async () => {
    try {
      const { data } = await api.http.get<ServiceProvider[]>({ url: route("/plugins/maintenance-scheduler/providers") });
      providers.value = data || [];
      return data;
    } catch {
      return [];
    }
  });

  useAsyncData("maint-repair-log", async () => {
    try {
      const { data } = await api.http.get<RepairLogEntry[]>({ url: route("/plugins/maintenance-scheduler/repairs") });
      repairLog.value = data || [];
      return data;
    } catch {
      return [];
    }
  });

  useAsyncData("maint-calendar", async () => {
    try {
      const { data } = await api.http.get<CalendarEvent[]>({ url: route("/plugins/maintenance-scheduler/calendar") });
      calendarEvents.value = data || [];
      return data;
    } catch {
      return [];
    }
  });

  // --- Methods ---

  async function searchItems() {
    if (!itemSearch.value.trim()) return;
    try {
      const { data } = await api.http.get<Array<{ id: string; name: string }>>({
        url: route("/plugins/maintenance-scheduler/search/items", { q: itemSearch.value }),
      });
      itemSearchResults.value = data || [];
    } catch {
      itemSearchResults.value = [];
    }
  }

  function selectItem(item: { id: string; name: string }) {
    newSchedule.itemId = item.id;
    newSchedule.itemName = item.name;
    itemSearch.value = item.name;
    itemSearchResults.value = [];
  }

  async function createSchedule() {
    if (!newSchedule.itemId || !newSchedule.startDate) return;

    loading.value = true;
    try {
      await api.http.post<typeof newSchedule, void>({
        url: route("/plugins/maintenance-scheduler/schedules"),
        body: { ...newSchedule },
      });
      newSchedule.itemId = "";
      newSchedule.itemName = "";
      newSchedule.maintenanceType = "inspection";
      newSchedule.frequency = "monthly";
      newSchedule.startDate = "";
      newSchedule.notes = "";
      newSchedule.assignedTo = "";
      itemSearch.value = "";
      showCreateSchedule.value = false;
      await refreshSchedules();
    } finally {
      loading.value = false;
    }
  }

  async function removeSchedule(schedule: ScheduleEntry) {
    loading.value = true;
    try {
      await api.http.delete<void>({ url: route(`/plugins/maintenance-scheduler/schedules/${schedule.id}`) });
      await refreshSchedules();
    } finally {
      loading.value = false;
    }
  }

  async function markCompleted(schedule: ScheduleEntry) {
    loading.value = true;
    try {
      await api.http.post<void, void>({
        url: route(`/plugins/maintenance-scheduler/schedules/${schedule.id}/complete`),
      });
      await refreshSchedules();
    } finally {
      loading.value = false;
    }
  }

  async function addProvider() {
    if (!newProvider.name.trim()) return;

    loading.value = true;
    try {
      await api.http.post<typeof newProvider, void>({
        url: route("/plugins/maintenance-scheduler/providers"),
        body: { ...newProvider },
      });
      newProvider.name = "";
      newProvider.phone = "";
      newProvider.email = "";
      newProvider.specialty = "general";
      newProvider.rating = 3;
      newProvider.notes = "";
      showAddProvider.value = false;
      await refreshProviders();
    } finally {
      loading.value = false;
    }
  }

  async function removeProvider(provider: ServiceProvider) {
    loading.value = true;
    try {
      await api.http.delete<void>({ url: route(`/plugins/maintenance-scheduler/providers/${provider.id}`) });
      await refreshProviders();
    } finally {
      loading.value = false;
    }
  }

  function maintenanceTypeBadgeClass(type: string): string {
    const classes: Record<string, string> = {
      inspection: "badge-info",
      cleaning: "badge-success",
      calibration: "badge-warning",
      repair: "badge-error",
      replacement: "badge-error",
      lubrication: "badge-info",
      testing: "badge-warning",
    };
    return classes[type] || "badge-ghost";
  }

  function renderStars(rating: number): string {
    return "\u2605".repeat(rating) + "\u2606".repeat(5 - rating);
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Maintenance Scheduler</h1>
          <p class="text-sm opacity-70">
            Schedule recurring maintenance, track service providers, and log repairs.
          </p>
        </div>
        <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
          Back to Plugins
        </NuxtLink>
      </div>

      <!-- Stats -->
      <div class="stats shadow w-full">
        <div class="stat">
          <div class="stat-title">Active Schedules</div>
          <div class="stat-value text-primary">{{ schedules.length }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Service Providers</div>
          <div class="stat-value">{{ providers.length }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Total Repairs</div>
          <div class="stat-value text-info">{{ repairLog.length }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Repair Costs</div>
          <div class="stat-value text-warning">
            ${{ repairLog.reduce((sum, r) => sum + (r.cost || 0), 0).toFixed(0) }}
          </div>
        </div>
      </div>

      <!-- Upcoming Schedule Calendar -->
      <section>
        <div class="flex items-center justify-between mb-3">
          <Subtitle>Upcoming Schedule</Subtitle>
          <div class="flex gap-2">
            <button class="btn btn-xs btn-outline" @click="prevWeek">Prev</button>
            <button class="btn btn-xs btn-outline" @click="goToToday">Today</button>
            <button class="btn btn-xs btn-outline" @click="nextWeek">Next</button>
          </div>
        </div>
        <BaseCard>
          <div class="p-4">
            <p class="text-sm font-medium mb-3 text-center">{{ calendarTitle }}</p>
            <!-- Day Headers -->
            <div class="grid grid-cols-7 gap-1 mb-1">
              <div
                v-for="day in ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']"
                :key="day"
                class="text-center text-xs font-semibold opacity-60 py-1"
              >
                {{ day }}
              </div>
            </div>
            <!-- Calendar Grid -->
            <div class="grid grid-cols-7 gap-1">
              <div
                v-for="day in calendarDays"
                :key="day.dateStr"
                class="border rounded p-1 min-h-[60px] text-xs"
                :class="{
                  'bg-primary/10 border-primary': day.isToday,
                  'opacity-40': day.isPast,
                  'bg-base-100': !day.isToday && !day.isPast,
                }"
              >
                <div class="font-semibold" :class="day.isToday ? 'text-primary' : 'opacity-70'">
                  {{ day.date.getDate() }}
                </div>
                <div v-for="event in day.events.slice(0, 3)" :key="event.id" class="mt-0.5">
                  <span
                    class="inline-block w-full rounded px-1 truncate text-[10px]"
                    :class="maintenanceTypeBadgeClass(event.type)"
                  >
                    {{ event.itemName }}
                  </span>
                </div>
                <div v-if="day.events.length > 3" class="text-[10px] opacity-50 mt-0.5">
                  +{{ day.events.length - 3 }} more
                </div>
              </div>
            </div>
          </div>
        </BaseCard>
      </section>

      <!-- Create Schedule -->
      <section>
        <div class="flex items-center justify-between mb-3">
          <Subtitle>Create Schedule</Subtitle>
          <button class="btn btn-sm btn-outline" @click="showCreateSchedule = !showCreateSchedule">
            {{ showCreateSchedule ? "Cancel" : "New Schedule" }}
          </button>
        </div>
        <BaseCard v-if="showCreateSchedule">
          <div class="p-4">
            <form class="flex flex-col gap-4" @submit.prevent="createSchedule">
              <div class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
                <!-- Item Selector -->
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Item</span>
                  </label>
                  <div class="flex gap-2">
                    <input
                      v-model="itemSearch"
                      type="text"
                      placeholder="Search for an item..."
                      class="input input-bordered input-sm flex-1"
                      @keyup.enter.prevent="searchItems"
                    />
                    <button type="button" class="btn btn-sm btn-outline" @click="searchItems">Search</button>
                  </div>
                  <div v-if="itemSearchResults.length > 0" class="mt-1 border rounded max-h-32 overflow-y-auto">
                    <button
                      v-for="item in itemSearchResults"
                      :key="item.id"
                      type="button"
                      class="block w-full text-left px-3 py-1 text-sm hover:bg-base-200"
                      @click="selectItem(item)"
                    >
                      {{ item.name }}
                    </button>
                  </div>
                  <p v-if="newSchedule.itemName" class="text-xs mt-1 text-success">
                    Selected: {{ newSchedule.itemName }}
                  </p>
                </div>

                <!-- Maintenance Type -->
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Maintenance Type</span>
                  </label>
                  <select v-model="newSchedule.maintenanceType" class="select select-bordered select-sm w-full">
                    <option v-for="opt in maintenanceTypes" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                  </select>
                </div>

                <!-- Frequency -->
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Frequency</span>
                  </label>
                  <select v-model="newSchedule.frequency" class="select select-bordered select-sm w-full">
                    <option v-for="opt in frequencyOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                  </select>
                </div>

                <!-- Start Date -->
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Start Date</span>
                  </label>
                  <input
                    v-model="newSchedule.startDate"
                    type="date"
                    class="input input-bordered input-sm w-full"
                    required
                  />
                </div>

                <!-- Assigned To -->
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Assigned To</span>
                  </label>
                  <input
                    v-model="newSchedule.assignedTo"
                    type="text"
                    placeholder="Person or team"
                    class="input input-bordered input-sm w-full"
                  />
                </div>

                <!-- Notes -->
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Notes</span>
                  </label>
                  <input
                    v-model="newSchedule.notes"
                    type="text"
                    placeholder="Additional notes"
                    class="input input-bordered input-sm w-full"
                  />
                </div>
              </div>
              <div class="flex justify-end">
                <button
                  type="submit"
                  class="btn btn-sm btn-primary"
                  :disabled="loading || !newSchedule.itemId || !newSchedule.startDate"
                >
                  Create Schedule
                </button>
              </div>
            </form>
          </div>
        </BaseCard>
      </section>

      <!-- Active Schedules -->
      <section>
        <Subtitle>Active Schedules</Subtitle>
        <BaseCard>
          <div v-if="schedules.length === 0" class="p-6 text-center opacity-50">
            <p>No active maintenance schedules. Create one to get started.</p>
          </div>
          <div v-else class="overflow-x-auto">
            <table class="table table-sm w-full">
              <thead>
                <tr>
                  <th>Item</th>
                  <th>Type</th>
                  <th>Frequency</th>
                  <th>Next Due</th>
                  <th>Last Completed</th>
                  <th>Assigned</th>
                  <th class="w-32">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="schedule in schedules" :key="schedule.id">
                  <td class="font-medium">{{ schedule.itemName }}</td>
                  <td>
                    <span class="badge badge-sm" :class="maintenanceTypeBadgeClass(schedule.maintenanceType)">
                      {{ schedule.maintenanceType }}
                    </span>
                  </td>
                  <td class="text-sm">{{ schedule.frequency }}</td>
                  <td class="text-sm">{{ schedule.nextDue }}</td>
                  <td class="text-sm opacity-70">{{ schedule.lastCompleted || "Never" }}</td>
                  <td class="text-sm">{{ schedule.assignedTo || "Unassigned" }}</td>
                  <td>
                    <div class="flex gap-1">
                      <button class="btn btn-xs btn-success" @click="markCompleted(schedule)">
                        Done
                      </button>
                      <button class="btn btn-xs btn-ghost text-error" @click="removeSchedule(schedule)">
                        Remove
                      </button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </BaseCard>
      </section>

      <!-- Service Providers -->
      <section>
        <div class="flex items-center justify-between mb-3">
          <Subtitle>Service Providers</Subtitle>
          <button class="btn btn-sm btn-outline" @click="showAddProvider = !showAddProvider">
            {{ showAddProvider ? "Cancel" : "Add Provider" }}
          </button>
        </div>

        <!-- Add Provider Form -->
        <BaseCard v-if="showAddProvider" class="mb-4">
          <div class="p-4">
            <form class="flex flex-col gap-3" @submit.prevent="addProvider">
              <div class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Name</span>
                  </label>
                  <input
                    v-model="newProvider.name"
                    type="text"
                    placeholder="Provider name"
                    class="input input-bordered input-sm w-full"
                    required
                  />
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Phone</span>
                  </label>
                  <input
                    v-model="newProvider.phone"
                    type="tel"
                    placeholder="(555) 123-4567"
                    class="input input-bordered input-sm w-full"
                  />
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Email</span>
                  </label>
                  <input
                    v-model="newProvider.email"
                    type="email"
                    placeholder="provider@example.com"
                    class="input input-bordered input-sm w-full"
                  />
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Specialty</span>
                  </label>
                  <select v-model="newProvider.specialty" class="select select-bordered select-sm w-full">
                    <option v-for="opt in specialtyOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
                  </select>
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Rating (1-5)</span>
                  </label>
                  <input
                    v-model.number="newProvider.rating"
                    type="range"
                    min="1"
                    max="5"
                    class="range range-primary range-sm"
                  />
                  <div class="text-xs text-center mt-1">{{ renderStars(newProvider.rating) }}</div>
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Notes</span>
                  </label>
                  <input
                    v-model="newProvider.notes"
                    type="text"
                    placeholder="Additional notes"
                    class="input input-bordered input-sm w-full"
                  />
                </div>
              </div>
              <div class="flex justify-end">
                <button type="submit" class="btn btn-sm btn-primary" :disabled="loading || !newProvider.name.trim()">
                  Add Provider
                </button>
              </div>
            </form>
          </div>
        </BaseCard>

        <!-- Provider Cards -->
        <div v-if="providers.length === 0 && !showAddProvider" class="text-center py-8 opacity-50">
          <p>No service providers configured. Add one to track your maintenance contacts.</p>
        </div>
        <div v-else class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-3">
          <BaseCard v-for="provider in providers" :key="provider.id">
            <div class="p-4">
              <div class="flex items-start justify-between">
                <div>
                  <h3 class="font-semibold">{{ provider.name }}</h3>
                  <span class="badge badge-xs badge-outline mt-1">{{ provider.specialty }}</span>
                </div>
                <button class="btn btn-xs btn-ghost text-error" @click="removeProvider(provider)">
                  Remove
                </button>
              </div>
              <div class="mt-3 space-y-1 text-sm">
                <p v-if="provider.phone" class="opacity-70">Phone: {{ provider.phone }}</p>
                <p v-if="provider.email" class="opacity-70">Email: {{ provider.email }}</p>
                <p class="text-warning">{{ renderStars(provider.rating) }}</p>
                <p v-if="provider.notes" class="opacity-50 text-xs">{{ provider.notes }}</p>
              </div>
            </div>
          </BaseCard>
        </div>
      </section>

      <!-- Repair Log -->
      <section>
        <Subtitle>Repair Log</Subtitle>
        <BaseCard>
          <div v-if="repairLog.length === 0" class="p-6 text-center opacity-50">
            <p>No repairs logged yet. Repairs are automatically logged when schedules are completed.</p>
          </div>
          <div v-else class="overflow-x-auto">
            <table class="table table-sm w-full">
              <thead>
                <tr>
                  <th>Item</th>
                  <th>Date</th>
                  <th>Provider</th>
                  <th>Cost</th>
                  <th>Description</th>
                  <th>Warranty</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="repair in repairLog" :key="repair.id">
                  <td class="font-medium">{{ repair.itemName }}</td>
                  <td class="text-sm">{{ repair.date }}</td>
                  <td class="text-sm">{{ repair.providerName || "Self" }}</td>
                  <td class="text-sm">${{ repair.cost.toFixed(2) }}</td>
                  <td class="text-sm max-w-[200px] truncate">{{ repair.description }}</td>
                  <td>
                    <span
                      class="badge badge-sm"
                      :class="repair.warrantyClaim ? 'badge-success' : 'badge-ghost'"
                    >
                      {{ repair.warrantyClaim ? "Yes" : "No" }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </BaseCard>
      </section>
    </BaseContainer>
  </div>
</template>
