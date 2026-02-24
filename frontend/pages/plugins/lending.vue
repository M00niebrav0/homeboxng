<script setup lang="ts">
  import BaseContainer from "@/components/Base/Container.vue";
  import BaseCard from "@/components/Base/Card.vue";
  import Subtitle from "~/components/global/Subtitle.vue";

  definePageMeta({
    middleware: ["auth"],
  });
  useHead({
    title: "HomeBoxNG | Lending Tracker",
  });

  const api = useUserApi();

  // Interfaces
  interface Loan {
    id: string;
    itemId: string;
    itemName: string;
    borrowerName: string;
    checkoutDate: string;
    dueDate: string;
    returnDate?: string;
    notes: string;
    conditionNotes?: string;
    overdue: boolean;
  }

  // Active loans
  const { data: activeLoans, refresh: refreshLoans } = useAsyncData("lending-active", async () => {
    const { data } = await api.http.get<Loan[]>({
      url: "/api/v1/plugins/lending/active",
    });
    return data || [];
  });

  // Loan history
  const { data: loanHistory, refresh: refreshHistory } = useAsyncData("lending-history", async () => {
    const { data } = await api.http.get<Loan[]>({
      url: "/api/v1/plugins/lending/history",
    });
    return data || [];
  });

  // Overdue items
  const overdueCount = computed(() => {
    return activeLoans.value?.filter(l => l.overdue).length ?? 0;
  });

  // Checkout form state
  const checkoutForm = reactive({
    searchQuery: "",
    searchResults: [] as Array<{ id: string; name: string; location: string }>,
    selectedItem: null as { id: string; name: string; location: string } | null,
    borrowerName: "",
    dueDate: "",
    notes: "",
    searching: false,
    submitting: false,
  });

  // Return form state
  const returnForm = reactive({
    selectedLoanId: "",
    conditionNotes: "",
    submitting: false,
  });

  // Search items for checkout
  let searchTimeout: ReturnType<typeof setTimeout>;

  watch(() => checkoutForm.searchQuery, (query) => {
    clearTimeout(searchTimeout);
    if (query.length < 2) {
      checkoutForm.searchResults = [];
      return;
    }
    searchTimeout = setTimeout(async () => {
      checkoutForm.searching = true;
      try {
        const { data } = await api.items.getAll({
          q: query,
          page: 1,
          pageSize: 10,
        });
        checkoutForm.searchResults = (data?.items || []).map((item: any) => ({
          id: item.id,
          name: item.name,
          location: item.location?.name || "No location",
        }));
      } finally {
        checkoutForm.searching = false;
      }
    }, 300);
  });

  function selectCheckoutItem(item: { id: string; name: string; location: string }) {
    checkoutForm.selectedItem = item;
    checkoutForm.searchQuery = item.name;
    checkoutForm.searchResults = [];
  }

  function clearCheckoutItem() {
    checkoutForm.selectedItem = null;
    checkoutForm.searchQuery = "";
  }

  async function submitCheckout() {
    if (!checkoutForm.selectedItem || !checkoutForm.borrowerName || !checkoutForm.dueDate) return;

    checkoutForm.submitting = true;
    try {
      await api.http.post<{
        itemId: string;
        borrowerName: string;
        dueDate: string;
        notes: string;
      }, void>({
        url: "/api/v1/plugins/lending/checkout",
        body: {
          itemId: checkoutForm.selectedItem.id,
          borrowerName: checkoutForm.borrowerName,
          dueDate: checkoutForm.dueDate,
          notes: checkoutForm.notes,
        },
      });

      // Reset form
      checkoutForm.selectedItem = null;
      checkoutForm.searchQuery = "";
      checkoutForm.borrowerName = "";
      checkoutForm.dueDate = "";
      checkoutForm.notes = "";

      refreshLoans();
    } finally {
      checkoutForm.submitting = false;
    }
  }

  async function submitReturn() {
    if (!returnForm.selectedLoanId) return;

    returnForm.submitting = true;
    try {
      await api.http.post<{
        loanId: string;
        conditionNotes: string;
      }, void>({
        url: "/api/v1/plugins/lending/return",
        body: {
          loanId: returnForm.selectedLoanId,
          conditionNotes: returnForm.conditionNotes,
        },
      });

      // Reset form
      returnForm.selectedLoanId = "";
      returnForm.conditionNotes = "";

      refreshLoans();
      refreshHistory();
    } finally {
      returnForm.submitting = false;
    }
  }

  function formatDate(dateStr: string): string {
    return new Date(dateStr).toLocaleDateString();
  }

  function daysSince(dateStr: string): number {
    const date = new Date(dateStr);
    const now = new Date();
    return Math.floor((now.getTime() - date.getTime()) / 86400000);
  }

  function daysUntil(dateStr: string): number {
    const date = new Date(dateStr);
    const now = new Date();
    return Math.floor((date.getTime() - now.getTime()) / 86400000);
  }

  function loanDuration(checkout: string, returnDate?: string): string {
    const start = new Date(checkout);
    const end = returnDate ? new Date(returnDate) : new Date();
    const days = Math.floor((end.getTime() - start.getTime()) / 86400000);
    if (days === 0) return "Today";
    if (days === 1) return "1 day";
    return `${days} days`;
  }
</script>

<template>
  <div>
    <BaseContainer class="flex flex-col gap-6">
      <!-- Header -->
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold">Lending Tracker</h1>
          <p class="text-sm opacity-70">
            Track items checked out to others and manage returns.
          </p>
        </div>
        <NuxtLink to="/plugins" class="btn btn-sm btn-outline">
          Back to Plugins
        </NuxtLink>
      </div>

      <!-- Overdue Alert Banner -->
      <div
        v-if="overdueCount > 0"
        class="alert alert-error shadow-lg"
      >
        <svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
        </svg>
        <div>
          <h3 class="font-bold">{{ overdueCount }} Overdue {{ overdueCount === 1 ? "Item" : "Items" }}</h3>
          <p class="text-sm">Some checked-out items have passed their due date. Please follow up with borrowers.</p>
        </div>
      </div>

      <!-- Active Loans -->
      <section>
        <Subtitle>Active Loans ({{ activeLoans?.length ?? 0 }})</Subtitle>
        <div v-if="!activeLoans || activeLoans.length === 0" class="text-center py-8 opacity-50">
          <p>No items are currently checked out.</p>
        </div>
        <div v-else class="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
          <BaseCard v-for="loan in activeLoans" :key="loan.id">
            <div class="p-4">
              <div class="flex items-start justify-between">
                <div class="min-w-0 flex-1">
                  <h3 class="font-semibold truncate">{{ loan.itemName }}</h3>
                  <p class="text-sm opacity-60 mt-1">
                    Borrowed by <span class="font-medium">{{ loan.borrowerName }}</span>
                  </p>
                </div>
                <span
                  v-if="loan.overdue"
                  class="badge badge-error badge-sm flex-shrink-0 ml-2"
                >
                  OVERDUE
                </span>
              </div>

              <div class="mt-3 space-y-1 text-sm">
                <div class="flex justify-between">
                  <span class="opacity-50">Checked out:</span>
                  <span>{{ formatDate(loan.checkoutDate) }}</span>
                </div>
                <div class="flex justify-between">
                  <span class="opacity-50">Due:</span>
                  <span
                    :class="{
                      'text-error font-bold': loan.overdue,
                      'text-warning': !loan.overdue && daysUntil(loan.dueDate) <= 3,
                    }"
                  >
                    {{ formatDate(loan.dueDate) }}
                    <template v-if="loan.overdue">
                      ({{ daysSince(loan.dueDate) }}d overdue)
                    </template>
                    <template v-else-if="daysUntil(loan.dueDate) <= 3">
                      ({{ daysUntil(loan.dueDate) }}d left)
                    </template>
                  </span>
                </div>
                <div class="flex justify-between">
                  <span class="opacity-50">Duration:</span>
                  <span>{{ loanDuration(loan.checkoutDate) }}</span>
                </div>
              </div>

              <p v-if="loan.notes" class="mt-2 text-xs opacity-40 line-clamp-2">{{ loan.notes }}</p>
            </div>
          </BaseCard>
        </div>
      </section>

      <div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <!-- Checkout Form -->
        <section>
          <Subtitle>Check Out Item</Subtitle>
          <BaseCard>
            <div class="p-4 space-y-4">
              <!-- Item Search -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Item</span>
                </label>
                <div v-if="checkoutForm.selectedItem" class="flex items-center gap-2">
                  <div class="flex-1 bg-base-200 rounded-lg p-2">
                    <p class="text-sm font-medium">{{ checkoutForm.selectedItem.name }}</p>
                    <p class="text-xs opacity-50">{{ checkoutForm.selectedItem.location }}</p>
                  </div>
                  <button class="btn btn-xs btn-ghost" @click="clearCheckoutItem">
                    Clear
                  </button>
                </div>
                <div v-else class="relative">
                  <input
                    v-model="checkoutForm.searchQuery"
                    type="text"
                    placeholder="Search for an item..."
                    class="input input-bordered input-sm w-full"
                  />
                  <span v-if="checkoutForm.searching" class="absolute right-3 top-2">
                    <span class="loading loading-spinner loading-xs" />
                  </span>

                  <div
                    v-if="checkoutForm.searchResults.length > 0"
                    class="absolute z-10 mt-1 w-full border border-base-300 rounded-lg bg-base-100 shadow-lg max-h-48 overflow-y-auto"
                  >
                    <button
                      v-for="item in checkoutForm.searchResults"
                      :key="item.id"
                      class="w-full text-left px-3 py-2 hover:bg-base-200 transition-colors"
                      @click="selectCheckoutItem(item)"
                    >
                      <p class="text-sm font-medium">{{ item.name }}</p>
                      <p class="text-xs opacity-50">{{ item.location }}</p>
                    </button>
                  </div>
                </div>
              </div>

              <!-- Borrower Name -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Borrower Name</span>
                </label>
                <input
                  v-model="checkoutForm.borrowerName"
                  type="text"
                  placeholder="Who is borrowing this item?"
                  class="input input-bordered input-sm w-full"
                />
              </div>

              <!-- Due Date -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Due Date</span>
                </label>
                <input
                  v-model="checkoutForm.dueDate"
                  type="date"
                  class="input input-bordered input-sm w-full"
                />
              </div>

              <!-- Notes -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Notes</span>
                </label>
                <textarea
                  v-model="checkoutForm.notes"
                  placeholder="Optional notes about this checkout..."
                  class="textarea textarea-bordered textarea-sm w-full"
                  rows="2"
                />
              </div>

              <button
                class="btn btn-primary btn-sm w-full"
                :disabled="!checkoutForm.selectedItem || !checkoutForm.borrowerName || !checkoutForm.dueDate || checkoutForm.submitting"
                @click="submitCheckout"
              >
                {{ checkoutForm.submitting ? "Processing..." : "Check Out" }}
              </button>
            </div>
          </BaseCard>
        </section>

        <!-- Return Form -->
        <section>
          <Subtitle>Return Item</Subtitle>
          <BaseCard>
            <div class="p-4 space-y-4">
              <!-- Select Active Loan -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Select Loan</span>
                </label>
                <select
                  v-model="returnForm.selectedLoanId"
                  class="select select-bordered select-sm w-full"
                >
                  <option value="" disabled>Choose an active loan...</option>
                  <option
                    v-for="loan in activeLoans"
                    :key="loan.id"
                    :value="loan.id"
                  >
                    {{ loan.itemName }} - {{ loan.borrowerName }}
                    {{ loan.overdue ? "(OVERDUE)" : "" }}
                  </option>
                </select>
              </div>

              <!-- Condition Notes -->
              <div class="form-control">
                <label class="label">
                  <span class="label-text font-medium">Condition Notes</span>
                </label>
                <textarea
                  v-model="returnForm.conditionNotes"
                  placeholder="Describe the condition of the returned item..."
                  class="textarea textarea-bordered textarea-sm w-full"
                  rows="3"
                />
              </div>

              <button
                class="btn btn-success btn-sm w-full"
                :disabled="!returnForm.selectedLoanId || returnForm.submitting"
                @click="submitReturn"
              >
                {{ returnForm.submitting ? "Processing..." : "Return Item" }}
              </button>
            </div>
          </BaseCard>
        </section>
      </div>

      <!-- Loan History -->
      <section>
        <Subtitle>Loan History</Subtitle>
        <div v-if="!loanHistory || loanHistory.length === 0" class="text-center py-8 opacity-50">
          <p>No loan history yet.</p>
        </div>
        <div v-else class="overflow-x-auto">
          <table class="table table-sm w-full">
            <thead>
              <tr>
                <th>Item</th>
                <th>Borrower</th>
                <th>Checked Out</th>
                <th>Returned</th>
                <th>Duration</th>
                <th>Condition</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="loan in loanHistory" :key="loan.id">
                <td class="font-medium">{{ loan.itemName }}</td>
                <td>{{ loan.borrowerName }}</td>
                <td class="text-xs">{{ formatDate(loan.checkoutDate) }}</td>
                <td class="text-xs">{{ loan.returnDate ? formatDate(loan.returnDate) : "-" }}</td>
                <td class="text-xs">{{ loanDuration(loan.checkoutDate, loan.returnDate) }}</td>
                <td class="text-xs opacity-60 max-w-[200px] truncate">
                  {{ loan.conditionNotes || "-" }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </BaseContainer>
  </div>
</template>
