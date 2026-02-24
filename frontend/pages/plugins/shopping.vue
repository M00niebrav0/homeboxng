<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";
  import { route } from "~/lib/api/base";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Shopping List",
  });

  const api = useUserApi();

  // --- Types ---

  interface ShoppingItem {
    id: string;
    name: string;
    quantity: number;
    store: string;
    priority: "high" | "medium" | "low";
    notes: string;
    purchased: boolean;
    purchasedAt?: string;
    cost?: number;
  }

  interface ReorderRule {
    id: string;
    itemName: string;
    minQuantity: number;
    reorderQuantity: number;
    preferredStore: string;
  }

  // --- State ---

  const shoppingList = ref<ShoppingItem[]>([]);
  const reorderRules = ref<ReorderRule[]>([]);
  const purchaseHistory = ref<ShoppingItem[]>([]);
  const showHistory = ref(false);
  const showReorderForm = ref(false);
  const loading = ref(false);

  const stores = ref<string[]>(["Amazon", "Home Depot", "Walmart", "Target", "Lowes", "Costco", "Other"]);

  // --- Add Item Form ---

  const newItem = reactive({
    name: "",
    quantity: 1,
    store: "",
    priority: "medium" as "high" | "medium" | "low",
    notes: "",
  });

  // --- Reorder Rule Form ---

  const newRule = reactive({
    itemName: "",
    minQuantity: 1,
    reorderQuantity: 1,
    preferredStore: "",
  });

  // --- Data Loading ---

  const { refresh: refreshList } = useAsyncData("shopping-list", async () => {
    try {
      const { data } = await api.http.get<ShoppingItem[]>({ url: route("/plugins/shopping/items") });
      shoppingList.value = (data || []).filter((i: ShoppingItem) => !i.purchased);
      return data;
    } catch {
      return [];
    }
  });

  const { refresh: refreshRules } = useAsyncData("reorder-rules", async () => {
    try {
      const { data } = await api.http.get<ReorderRule[]>({ url: route("/plugins/shopping/reorder-rules") });
      reorderRules.value = data || [];
      return data;
    } catch {
      return [];
    }
  });

  const { refresh: refreshHistory } = useAsyncData("purchase-history", async () => {
    try {
      const { data } = await api.http.get<ShoppingItem[]>({ url: route("/plugins/shopping/history") });
      purchaseHistory.value = data || [];
      return data;
    } catch {
      return [];
    }
  });

  // --- Computed ---

  const groupedByStore = computed(() => {
    const groups: Record<string, ShoppingItem[]> = {};
    for (const item of shoppingList.value) {
      const store = item.store || "Unassigned";
      if (!groups[store]) {
        groups[store] = [];
      }
      groups[store].push(item);
    }
    return groups;
  });

  const totalItems = computed(() => shoppingList.value.length);

  const highPriorityCount = computed(() => shoppingList.value.filter(i => i.priority === "high").length);

  const storeCount = computed(() => Object.keys(groupedByStore.value).length);

  // --- Methods ---

  function priorityBadgeClass(priority: string): string {
    if (priority === "high") return "badge-error";
    if (priority === "medium") return "badge-warning";
    return "badge-info";
  }

  async function addItem() {
    if (!newItem.name.trim()) return;

    loading.value = true;
    try {
      await api.http.post<typeof newItem, void>({
        url: route("/plugins/shopping/items"),
        body: { ...newItem },
      });
      newItem.name = "";
      newItem.quantity = 1;
      newItem.store = "";
      newItem.priority = "medium";
      newItem.notes = "";
      await refreshList();
    } finally {
      loading.value = false;
    }
  }

  async function togglePurchased(item: ShoppingItem) {
    loading.value = true;
    try {
      await api.http.post<{ purchased: boolean }, void>({
        url: route(`/plugins/shopping/items/${item.id}/purchase`),
        body: { purchased: !item.purchased },
      });
      await refreshList();
      await refreshHistory();
    } finally {
      loading.value = false;
    }
  }

  async function removeItem(item: ShoppingItem) {
    loading.value = true;
    try {
      await api.http.delete<void>({ url: route(`/plugins/shopping/items/${item.id}`) });
      await refreshList();
    } finally {
      loading.value = false;
    }
  }

  async function addReorderRule() {
    if (!newRule.itemName.trim()) return;

    loading.value = true;
    try {
      await api.http.post<typeof newRule, void>({
        url: route("/plugins/shopping/reorder-rules"),
        body: { ...newRule },
      });
      newRule.itemName = "";
      newRule.minQuantity = 1;
      newRule.reorderQuantity = 1;
      newRule.preferredStore = "";
      showReorderForm.value = false;
      await refreshRules();
    } finally {
      loading.value = false;
    }
  }

  async function removeReorderRule(rule: ReorderRule) {
    loading.value = true;
    try {
      await api.http.delete<void>({ url: route(`/plugins/shopping/reorder-rules/${rule.id}`) });
      await refreshRules();
    } finally {
      loading.value = false;
    }
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Shopping List</h1>
          <p class="text-sm opacity-70">
            Manage your shopping list, reorder rules, and purchase history.
          </p>
        </div>
        <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
          Back to Plugins
        </NuxtLink>
      </div>

      <!-- Stats -->
      <div class="stats shadow w-full">
        <div class="stat">
          <div class="stat-title">Items to Buy</div>
          <div class="stat-value text-primary">{{ totalItems }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">High Priority</div>
          <div class="stat-value text-error">{{ highPriorityCount }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Stores</div>
          <div class="stat-value">{{ storeCount }}</div>
        </div>
        <div class="stat">
          <div class="stat-title">Reorder Rules</div>
          <div class="stat-value text-info">{{ reorderRules.length }}</div>
        </div>
      </div>

      <!-- Add Item Form -->
      <section>
        <Subtitle>Add Item</Subtitle>
        <BaseCard>
          <div class="p-4">
            <form class="flex flex-col gap-3" @submit.prevent="addItem">
              <div class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-4">
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Item Name</span>
                  </label>
                  <input
                    v-model="newItem.name"
                    type="text"
                    placeholder="Enter item name"
                    class="input input-bordered input-sm w-full"
                    required
                  />
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Quantity</span>
                  </label>
                  <input
                    v-model.number="newItem.quantity"
                    type="number"
                    min="1"
                    class="input input-bordered input-sm w-full"
                  />
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Store</span>
                  </label>
                  <select v-model="newItem.store" class="select select-bordered select-sm w-full">
                    <option value="">Select store</option>
                    <option v-for="store in stores" :key="store" :value="store">{{ store }}</option>
                  </select>
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Priority</span>
                  </label>
                  <select v-model="newItem.priority" class="select select-bordered select-sm w-full">
                    <option value="high">High</option>
                    <option value="medium">Medium</option>
                    <option value="low">Low</option>
                  </select>
                </div>
              </div>
              <div class="form-control">
                <label class="label">
                  <span class="label-text">Notes</span>
                </label>
                <input
                  v-model="newItem.notes"
                  type="text"
                  placeholder="Optional notes"
                  class="input input-bordered input-sm w-full"
                />
              </div>
              <div class="flex justify-end">
                <button type="submit" class="btn btn-sm btn-primary" :disabled="loading || !newItem.name.trim()">
                  Add to List
                </button>
              </div>
            </form>
          </div>
        </BaseCard>
      </section>

      <!-- Active Shopping List - Grouped by Store -->
      <section>
        <Subtitle>Shopping List by Store</Subtitle>
        <div v-if="Object.keys(groupedByStore).length === 0" class="text-center py-8 opacity-50">
          <p>No items in your shopping list. Add some above.</p>
        </div>
        <div v-else class="flex flex-col gap-4">
          <BaseCard v-for="(items, storeName) in groupedByStore" :key="storeName">
            <template #title>
              <span class="flex items-center gap-2">
                {{ storeName }}
                <span class="badge badge-sm badge-outline">{{ items.length }} items</span>
              </span>
            </template>
            <div class="overflow-x-auto">
              <table class="table table-sm w-full">
                <thead>
                  <tr>
                    <th class="w-10">Done</th>
                    <th>Item</th>
                    <th class="w-20">Qty</th>
                    <th class="w-24">Priority</th>
                    <th>Notes</th>
                    <th class="w-16">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="item in items" :key="item.id" :class="{ 'opacity-40 line-through': item.purchased }">
                    <td>
                      <input
                        type="checkbox"
                        class="checkbox checkbox-sm checkbox-primary"
                        :checked="item.purchased"
                        @change="togglePurchased(item)"
                      />
                    </td>
                    <td class="font-medium">{{ item.name }}</td>
                    <td>{{ item.quantity }}</td>
                    <td>
                      <span class="badge badge-sm" :class="priorityBadgeClass(item.priority)">
                        {{ item.priority }}
                      </span>
                    </td>
                    <td class="text-sm opacity-70">{{ item.notes }}</td>
                    <td>
                      <button class="btn btn-xs btn-ghost text-error" @click="removeItem(item)">
                        Remove
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </BaseCard>
        </div>
      </section>

      <!-- Reorder Rules -->
      <section>
        <div class="flex items-center justify-between mb-3">
          <Subtitle>Reorder Rules</Subtitle>
          <button class="btn btn-sm btn-outline" @click="showReorderForm = !showReorderForm">
            {{ showReorderForm ? "Cancel" : "Add Rule" }}
          </button>
        </div>

        <!-- Add Reorder Rule Form -->
        <BaseCard v-if="showReorderForm" class="mb-4">
          <div class="p-4">
            <form class="flex flex-col gap-3" @submit.prevent="addReorderRule">
              <div class="grid grid-cols-1 gap-3 md:grid-cols-2 lg:grid-cols-4">
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Item Name</span>
                  </label>
                  <input
                    v-model="newRule.itemName"
                    type="text"
                    placeholder="Item to track"
                    class="input input-bordered input-sm w-full"
                    required
                  />
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Min Quantity Threshold</span>
                  </label>
                  <input
                    v-model.number="newRule.minQuantity"
                    type="number"
                    min="0"
                    class="input input-bordered input-sm w-full"
                  />
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Reorder Quantity</span>
                  </label>
                  <input
                    v-model.number="newRule.reorderQuantity"
                    type="number"
                    min="1"
                    class="input input-bordered input-sm w-full"
                  />
                </div>
                <div class="form-control">
                  <label class="label">
                    <span class="label-text">Preferred Store</span>
                  </label>
                  <select v-model="newRule.preferredStore" class="select select-bordered select-sm w-full">
                    <option value="">Select store</option>
                    <option v-for="store in stores" :key="store" :value="store">{{ store }}</option>
                  </select>
                </div>
              </div>
              <div class="flex justify-end">
                <button type="submit" class="btn btn-sm btn-primary" :disabled="loading || !newRule.itemName.trim()">
                  Save Rule
                </button>
              </div>
            </form>
          </div>
        </BaseCard>

        <!-- Reorder Rules Table -->
        <BaseCard>
          <div v-if="reorderRules.length === 0" class="p-6 text-center opacity-50">
            <p>No reorder rules configured. Add one to automatically track minimum stock levels.</p>
          </div>
          <div v-else class="overflow-x-auto">
            <table class="table table-sm w-full">
              <thead>
                <tr>
                  <th>Item</th>
                  <th>Min Quantity</th>
                  <th>Reorder Qty</th>
                  <th>Preferred Store</th>
                  <th class="w-16">Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="rule in reorderRules" :key="rule.id">
                  <td class="font-medium">{{ rule.itemName }}</td>
                  <td>{{ rule.minQuantity }}</td>
                  <td>{{ rule.reorderQuantity }}</td>
                  <td>{{ rule.preferredStore || "Any" }}</td>
                  <td>
                    <button class="btn btn-xs btn-ghost text-error" @click="removeReorderRule(rule)">
                      Remove
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
        </BaseCard>
      </section>

      <!-- Purchase History -->
      <section>
        <div class="flex items-center justify-between mb-3">
          <Subtitle>Purchase History</Subtitle>
          <button class="btn btn-sm btn-ghost" @click="showHistory = !showHistory">
            {{ showHistory ? "Hide" : "Show" }} History
          </button>
        </div>
        <BaseCard v-if="showHistory">
          <div v-if="purchaseHistory.length === 0" class="p-6 text-center opacity-50">
            <p>No purchase history yet.</p>
          </div>
          <div v-else class="overflow-x-auto">
            <table class="table table-sm w-full">
              <thead>
                <tr>
                  <th>Item</th>
                  <th>Quantity</th>
                  <th>Store</th>
                  <th>Purchased</th>
                  <th>Cost</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="item in purchaseHistory" :key="item.id">
                  <td class="font-medium">{{ item.name }}</td>
                  <td>{{ item.quantity }}</td>
                  <td>{{ item.store || "N/A" }}</td>
                  <td class="text-sm opacity-70">{{ item.purchasedAt || "Unknown" }}</td>
                  <td>{{ item.cost != null ? `$${item.cost.toFixed(2)}` : "N/A" }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </BaseCard>
      </section>
    </BaseContainer>
  </div>
</template>
