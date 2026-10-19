import { c as currentMonthRange, _ as __nuxt_component_0 } from "./DateRangePicker-BmLhEXsC.js";
import { _ as __nuxt_component_1 } from "./SearchSelect-BFyTiPOW.js";
import { _ as __nuxt_component_2 } from "./TransactionModal-D84yXN2T.js";
import { defineComponent, ref, watch, mergeProps, unref, useSSRContext, computed, isRef } from "vue";
import { ssrRenderAttrs, ssrRenderComponent, ssrInterpolate, ssrIncludeBooleanAttr, ssrRenderList, ssrRenderClass, ssrRenderAttr } from "vue/server-renderer";
import { Trash2, X, Plus, CircleDollarSign, ReceiptText, ArrowDownLeft, ArrowUpRight, Pencil } from "lucide-vue-next";
import { h as useApi, _ as _export_sfc, i as useRoute, d as usePermissions } from "../server.mjs";
import { useMediaQuery, useIntersectionObserver } from "@vueuse/core";
import { f as formatMoney } from "./money-BqQfZu3a.js";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/ofetch/dist/node.mjs";
import "#internal/nuxt/paths";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/hookable/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/unctx/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/h3/dist/index.mjs";
import "vue-router";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/defu/dist/defu.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/ufo/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/klona/dist/index.mjs";
import "@vuetify/v0/utilities";
const _sfc_main$1 = /* @__PURE__ */ defineComponent({
  __name: "DeleteTransactionModal",
  __ssrInlineRender: true,
  props: {
    modelValue: { type: Boolean },
    transaction: {}
  },
  emits: ["update:modelValue", "deleted"],
  setup(__props, { emit: __emit }) {
    const props = __props;
    useApi();
    const reason = ref("");
    const error = ref("");
    const deleting = ref(false);
    watch(() => [props.modelValue, props.transaction?.id], ([isOpen]) => {
      if (isOpen) {
        reason.value = "";
        error.value = "";
      }
    });
    return (_ctx, _push, _parent, _attrs) => {
      if (__props.modelValue && __props.transaction) {
        _push(`<div${ssrRenderAttrs(mergeProps({ class: "delete-backdrop" }, _attrs))} data-v-143b1a65><section class="delete-dialog" role="alertdialog" aria-modal="true" aria-labelledby="delete-title" data-v-143b1a65><div class="delete-icon" data-v-143b1a65>`);
        _push(ssrRenderComponent(unref(Trash2), { size: 21 }, null, _parent));
        _push(`</div><button type="button" class="close-button" aria-label="Close dialog" data-v-143b1a65>`);
        _push(ssrRenderComponent(unref(X), { size: 18 }, null, _parent));
        _push(`</button><p class="delete-eyebrow" data-v-143b1a65>REMOVE TRANSACTION</p><h2 id="delete-title" data-v-143b1a65>Delete this transaction?</h2><p class="delete-copy" data-v-143b1a65><strong data-v-143b1a65>${ssrInterpolate(__props.transaction.transaction_number)}</strong> will be voided and removed from active financial totals. Its record stays in the audit history. </p><label class="reason-field" data-v-143b1a65><span data-v-143b1a65>Reason <small data-v-143b1a65>Optional</small></span><textarea rows="2" maxlength="512" placeholder="Add a short reason" data-v-143b1a65>${ssrInterpolate(unref(reason))}</textarea></label>`);
        if (unref(error)) {
          _push(`<p class="delete-error" role="alert" data-v-143b1a65>${ssrInterpolate(unref(error))}</p>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<div class="delete-actions" data-v-143b1a65><button type="button" class="cancel-button"${ssrIncludeBooleanAttr(unref(deleting)) ? " disabled" : ""} data-v-143b1a65>Cancel</button><button type="button" class="confirm-button"${ssrIncludeBooleanAttr(unref(deleting)) ? " disabled" : ""} data-v-143b1a65>`);
        _push(ssrRenderComponent(unref(Trash2), { size: 15 }, null, _parent));
        _push(`${ssrInterpolate(unref(deleting) ? "Deleting…" : "Delete transaction")}</button></div></section></div>`);
      } else {
        _push(`<!---->`);
      }
    };
  }
});
const _sfc_setup$1 = _sfc_main$1.setup;
_sfc_main$1.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("components/DeleteTransactionModal.vue");
  return _sfc_setup$1 ? _sfc_setup$1(props, ctx) : void 0;
};
const __nuxt_component_3 = /* @__PURE__ */ _export_sfc(_sfc_main$1, [["__scopeId", "data-v-143b1a65"]]);
const _sfc_main = /* @__PURE__ */ defineComponent({
  __name: "index",
  __ssrInlineRender: true,
  setup(__props) {
    const { api } = useApi();
    const route = useRoute();
    const {
      canCreateIncome,
      canCreateExpense,
      canEditIncome,
      canEditExpense,
      canVoidIncome,
      canVoidExpense
    } = usePermissions();
    const typeFilter = ref(typeof route.query.type === "string" ? route.query.type : "");
    const searchTerm = ref(typeof route.query.search === "string" ? route.query.search : "");
    const dateRange = ref(currentMonthRange());
    const currency = ref("");
    const page = ref(1);
    const pageSize = ref(10);
    const total = ref(0);
    const totals = ref(null);
    const loadingMore = ref(false);
    const loadMoreSentinel = ref(null);
    const isMobile = useMediaQuery("(max-width: 760px)");
    let requestSequence = 0;
    let filterTimer;
    const routeType = computed(() => route.query.type === "income" || route.query.type === "expense" ? route.query.type : "");
    const pageTitle = computed(() => routeType.value === "income" ? "Income" : routeType.value === "expense" ? "Expense" : "Transactions");
    const headingCopy = computed(() => routeType.value ? `View, update, and manage ${routeType.value} records.` : "View, update, and manage transaction records.");
    const emptyTitle = computed(() => routeType.value ? `No ${routeType.value} found` : "No transactions found");
    const emptyCopy = computed(() => routeType.value ? `New ${routeType.value} records will appear here.` : "New income and expense records will appear here.");
    const items = ref([]);
    const pageCount = computed(() => Math.max(1, Math.ceil(total.value / pageSize.value)));
    const firstItem = computed(() => total.value ? (page.value - 1) * pageSize.value + 1 : 0);
    const lastItem = computed(() => Math.min(page.value * pageSize.value, total.value));
    const summaryRows = computed(() => {
      if (!totals.value) return [];
      const rows = [
        { currency: "USD", income: totals.value.income_usd, expense: totals.value.expense_usd, net: totals.value.net_usd },
        { currency: "KHR", income: totals.value.income_khr, expense: totals.value.expense_khr, net: totals.value.net_khr }
      ];
      return currency.value ? rows.filter((row) => row.currency === currency.value) : rows;
    });
    const loading = ref(true);
    const err = ref("");
    const transactionModalOpen = ref(false);
    const modalMode = ref("create");
    const modalType = ref("income");
    const selectedTransaction = ref(null);
    const deleteModalOpen = ref(false);
    const transactionToDelete = ref(null);
    const filters = [
      { value: "", label: "All activity" },
      { value: "income", label: "Income" },
      { value: "expense", label: "Expense" }
    ];
    const currencyOptions = [
      { value: "", label: "All currencies" },
      { value: "USD", label: "USD ($)" },
      { value: "KHR", label: "KHR (៛)" }
    ];
    const pageSizeOptions = [10, 20, 50].map((value) => ({ value: String(value), label: `${value} rows` }));
    function formatDate(value) {
      const date = new Date(value);
      return Number.isNaN(date.valueOf()) ? value : new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", year: "numeric" }).format(date);
    }
    function displayStatus(value) {
      return value ? value.charAt(0).toUpperCase() + value.slice(1) : "Approved";
    }
    function statusClass(value) {
      return `status-${value.toLowerCase()}`;
    }
    function canEdit(transaction) {
      const isEditable = transaction.status === "approved" || transaction.status === "pending";
      return isEditable && (transaction.type === "income" ? canEditIncome.value : canEditExpense.value);
    }
    function canDelete(transaction) {
      const isVoided = transaction.status === "voided" || transaction.status === "reversed";
      return !isVoided && (transaction.type === "income" ? canVoidIncome.value : canVoidExpense.value);
    }
    async function load(append = false) {
      if (append && (loading.value || loadingMore.value || items.value.length >= total.value)) return;
      const requestId = ++requestSequence;
      if (append) loadingMore.value = true;
      else {
        loading.value = true;
        loadingMore.value = false;
      }
      err.value = "";
      try {
        const listQuery = {
          type: routeType.value || typeFilter.value || void 0,
          search: searchTerm.value || void 0,
          date_from: dateRange.value.from || void 0,
          date_to: dateRange.value.to || void 0,
          currency: currency.value || void 0,
          page: page.value,
          page_size: pageSize.value
        };
        const [result, summary] = await Promise.all([
          api("/transactions", { query: listQuery }),
          append ? Promise.resolve(null) : api("/transactions/totals", {
            query: {
              type: listQuery.type,
              search: listQuery.search,
              date_from: listQuery.date_from,
              date_to: listQuery.date_to,
              currency: listQuery.currency
            }
          })
        ]);
        if (requestId !== requestSequence) return;
        items.value = append ? [...items.value, ...(result.items || []).filter((item) => !items.value.some((existing) => existing.id === item.id))] : result.items || [];
        total.value = result.total || 0;
        if (summary) totals.value = summary;
      } catch (e) {
        if (requestId !== requestSequence) return;
        err.value = e?.data?.error || "Could not load transactions.";
      } finally {
        if (requestId === requestSequence) {
          loading.value = false;
          loadingMore.value = false;
        }
      }
    }
    function loadMore() {
      if (!isMobile.value || loading.value || loadingMore.value || items.value.length >= total.value) return;
      page.value += 1;
      void load(true);
    }
    useIntersectionObserver(loadMoreSentinel, ([entry]) => {
      if (entry?.isIntersecting) loadMore();
    }, { rootMargin: "500px 0px" });
    function scheduleFilterLoad() {
      if (filterTimer) clearTimeout(filterTimer);
      filterTimer = setTimeout(() => {
        page.value = 1;
        items.value = [];
        void load();
      }, 250);
    }
    function setPageSize(value) {
      pageSize.value = Number(value) || 10;
      page.value = 1;
      load();
    }
    async function onTransactionSaved() {
      selectedTransaction.value = null;
      await load();
    }
    async function onTransactionDeleted() {
      transactionToDelete.value = null;
      await load();
    }
    watch(() => [route.query.type, route.query.search], ([type, search]) => {
      typeFilter.value = typeof type === "string" ? type : "";
      searchTerm.value = typeof search === "string" ? search : "";
      page.value = 1;
      items.value = [];
      load();
    });
    watch(() => [dateRange.value.from, dateRange.value.to, currency.value], scheduleFilterLoad);
    return (_ctx, _push, _parent, _attrs) => {
      const _component_DateRangePicker = __nuxt_component_0;
      const _component_SearchSelect = __nuxt_component_1;
      const _component_TransactionModal = __nuxt_component_2;
      const _component_DeleteTransactionModal = __nuxt_component_3;
      _push(`<div${ssrRenderAttrs(mergeProps({ class: "transactions-page" }, _attrs))} data-v-d65881e9><div class="page-heading" data-v-d65881e9><div data-v-d65881e9><p class="eyebrow" data-v-d65881e9>FINANCIAL ACTIVITY</p><h1 data-v-d65881e9>${ssrInterpolate(unref(pageTitle))}</h1><p class="heading-copy" data-v-d65881e9>${ssrInterpolate(unref(headingCopy))}</p></div><div class="create-actions" data-v-d65881e9>`);
      if (unref(canCreateIncome) && unref(routeType) !== "expense") {
        _push(`<button class="create-button income-create" data-v-d65881e9>`);
        _push(ssrRenderComponent(unref(Plus), { size: 16 }, null, _parent));
        _push(` Add Income</button>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(canCreateExpense) && unref(routeType) !== "income") {
        _push(`<button class="create-button expense-create" data-v-d65881e9>`);
        _push(ssrRenderComponent(unref(Plus), { size: 16 }, null, _parent));
        _push(` Add Expense</button>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</div></div><div class="list-toolbar" data-v-d65881e9>`);
      if (!unref(routeType)) {
        _push(`<div class="type-filters" aria-label="Transaction type filter" data-v-d65881e9><!--[-->`);
        ssrRenderList(filters, (filter) => {
          _push(`<button class="${ssrRenderClass(unref(typeFilter) === filter.value ? "active" : "")}" data-v-d65881e9>${ssrInterpolate(filter.label)}</button>`);
        });
        _push(`<!--]--></div>`);
      } else {
        _push(`<span class="type-scope" data-v-d65881e9>${ssrInterpolate(unref(routeType) === "income" ? "Income only" : "Expense only")}</span>`);
      }
      if (unref(searchTerm)) {
        _push(`<p class="search-caption" data-v-d65881e9>Search results for <strong data-v-d65881e9>“${ssrInterpolate(unref(searchTerm))}”</strong></p>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</div><div class="list-filters" data-v-d65881e9><div class="filter-field date-range-filter" data-v-d65881e9><span data-v-d65881e9>Date range</span>`);
      _push(ssrRenderComponent(_component_DateRangePicker, {
        modelValue: unref(dateRange),
        "onUpdate:modelValue": ($event) => isRef(dateRange) ? dateRange.value = $event : null
      }, null, _parent));
      _push(`</div><div class="filter-field currency-filter" data-v-d65881e9><span data-v-d65881e9>Currency</span>`);
      _push(ssrRenderComponent(_component_SearchSelect, {
        modelValue: unref(currency),
        "onUpdate:modelValue": ($event) => isRef(currency) ? currency.value = $event : null,
        options: currencyOptions,
        placeholder: "All currencies",
        "aria-label": "Currency filter"
      }, null, _parent));
      _push(`</div><div class="filter-actions" data-v-d65881e9><button type="button" class="reset-filter-button" data-v-d65881e9>Reset</button></div></div>`);
      if (unref(summaryRows).length) {
        _push(`<section class="totals-summary" aria-label="Income minus expense totals" data-v-d65881e9><!--[-->`);
        ssrRenderList(unref(summaryRows), (row) => {
          _push(`<article class="totals-card" data-v-d65881e9><div class="totals-card-heading" data-v-d65881e9>`);
          _push(ssrRenderComponent(unref(CircleDollarSign), { size: 17 }, null, _parent));
          _push(`<span data-v-d65881e9>${ssrInterpolate(row.currency)} · approved total</span></div><div class="totals-equation" data-v-d65881e9><div data-v-d65881e9><small data-v-d65881e9>Income</small><strong class="income-text" data-v-d65881e9>${ssrInterpolate(unref(formatMoney)(row.income, row.currency))}</strong></div><span class="equation-operator" data-v-d65881e9>−</span><div data-v-d65881e9><small data-v-d65881e9>Expense</small><strong class="expense-text" data-v-d65881e9>${ssrInterpolate(unref(formatMoney)(row.expense, row.currency))}</strong></div><span class="equation-operator" data-v-d65881e9>=</span><div data-v-d65881e9><small data-v-d65881e9>Net</small><strong class="${ssrRenderClass(Number(row.net) >= 0 ? "income-text" : "expense-text")}" data-v-d65881e9>${ssrInterpolate(unref(formatMoney)(row.net, row.currency))}</strong></div></div></article>`);
        });
        _push(`<!--]--></section>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(loading)) {
        _push(`<div class="list-state" data-v-d65881e9><span class="mini-spinner" data-v-d65881e9></span> Loading transactions…</div>`);
      } else if (unref(err)) {
        _push(`<div class="list-state list-error" data-v-d65881e9><p data-v-d65881e9>${ssrInterpolate(unref(err))}</p><button data-v-d65881e9>Try again</button></div>`);
      } else if (!unref(items).length) {
        _push(`<div class="list-state empty-state" data-v-d65881e9><span class="empty-icon" data-v-d65881e9>`);
        _push(ssrRenderComponent(unref(ReceiptText), { size: 22 }, null, _parent));
        _push(`</span><strong data-v-d65881e9>${ssrInterpolate(unref(emptyTitle))}</strong><span data-v-d65881e9>${ssrInterpolate(unref(emptyCopy))}</span></div>`);
      } else {
        _push(`<div class="transaction-list" data-v-d65881e9><!--[-->`);
        ssrRenderList(unref(items), (transaction) => {
          _push(`<article class="transaction-row" data-v-d65881e9><div class="${ssrRenderClass([transaction.type === "income" ? "income-tone" : "expense-tone", "transaction-type-icon"])}" data-v-d65881e9>`);
          if (transaction.type === "income") {
            _push(ssrRenderComponent(unref(ArrowDownLeft), { size: 18 }, null, _parent));
          } else {
            _push(ssrRenderComponent(unref(ArrowUpRight), { size: 18 }, null, _parent));
          }
          _push(`</div><div class="transaction-info" data-v-d65881e9><div class="transaction-title-line" data-v-d65881e9><strong data-v-d65881e9>${ssrInterpolate(transaction.description || (transaction.type === "expense" ? transaction.category?.name : "") || transaction.transaction_number)}</strong><span class="${ssrRenderClass([statusClass(transaction.status), "status-badge"])}" data-v-d65881e9>${ssrInterpolate(displayStatus(transaction.status))}</span></div><div class="transaction-meta" data-v-d65881e9><span data-v-d65881e9>${ssrInterpolate(transaction.transaction_number)}</span>`);
          if (transaction.type === "expense") {
            _push(`<!--[--><i data-v-d65881e9></i><span data-v-d65881e9>${ssrInterpolate(transaction.category?.name || "Uncategorized")}</span><!--]-->`);
          } else {
            _push(`<!---->`);
          }
          _push(`<i data-v-d65881e9></i><span data-v-d65881e9>${ssrInterpolate(transaction.branch?.name || transaction.branch?.code || "No branch")}</span><i data-v-d65881e9></i><span data-v-d65881e9>${ssrInterpolate(formatDate(transaction.transaction_date))}</span></div></div><div class="${ssrRenderClass([transaction.type === "income" ? "income-text" : "expense-text", "transaction-amount"])}" data-v-d65881e9><span data-v-d65881e9>${ssrInterpolate(transaction.type === "income" ? "+" : "−")}${ssrInterpolate(unref(formatMoney)(transaction.amount, transaction.currency))}</span><small data-v-d65881e9>${ssrInterpolate(transaction.type)}</small></div><div class="row-actions" data-v-d65881e9>`);
          if (canEdit(transaction)) {
            _push(`<button class="row-action edit-action"${ssrRenderAttr("aria-label", `Edit ${transaction.transaction_number}`)} title="Edit" data-v-d65881e9>`);
            _push(ssrRenderComponent(unref(Pencil), { size: 15 }, null, _parent));
            _push(`</button>`);
          } else {
            _push(`<!---->`);
          }
          if (canDelete(transaction)) {
            _push(`<button class="row-action delete-action"${ssrRenderAttr("aria-label", `Delete ${transaction.transaction_number}`)} title="Delete" data-v-d65881e9>`);
            _push(ssrRenderComponent(unref(Trash2), { size: 15 }, null, _parent));
            _push(`</button>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</div></article>`);
        });
        _push(`<!--]--></div>`);
      }
      if (unref(isMobile) && !unref(loading) && !unref(err) && unref(items).length) {
        _push(`<div class="mobile-load-more" aria-live="polite" data-v-d65881e9>`);
        if (unref(loadingMore)) {
          _push(`<span data-v-d65881e9><span class="mini-spinner" data-v-d65881e9></span> Loading more transactions…</span>`);
        } else if (unref(items).length < unref(total)) {
          _push(`<span data-v-d65881e9>Scroll for more</span>`);
        } else {
          _push(`<span data-v-d65881e9>All ${ssrInterpolate(unref(total))} transactions loaded</span>`);
        }
        _push(`</div>`);
      } else {
        _push(`<!---->`);
      }
      if (!unref(isMobile) && !unref(loading) && !unref(err) && unref(total) > 0) {
        _push(`<div class="pagination-bar" data-v-d65881e9><span class="pagination-summary" data-v-d65881e9>Showing ${ssrInterpolate(unref(firstItem))}–${ssrInterpolate(unref(lastItem))} of ${ssrInterpolate(unref(total))}</span><div class="pagination-controls" data-v-d65881e9><div class="page-size-control" data-v-d65881e9><span data-v-d65881e9>Rows</span>`);
        _push(ssrRenderComponent(_component_SearchSelect, {
          "model-value": String(unref(pageSize)),
          options: unref(pageSizeOptions),
          "aria-label": "Rows per page",
          "onUpdate:modelValue": setPageSize
        }, null, _parent));
        _push(`</div><button class="page-button"${ssrIncludeBooleanAttr(unref(page) <= 1) ? " disabled" : ""} aria-label="Previous page" data-v-d65881e9>Previous</button><span class="page-indicator" data-v-d65881e9>Page ${ssrInterpolate(unref(page))} of ${ssrInterpolate(unref(pageCount))}</span><button class="page-button"${ssrIncludeBooleanAttr(unref(page) >= unref(pageCount)) ? " disabled" : ""} aria-label="Next page" data-v-d65881e9>Next</button></div></div>`);
      } else {
        _push(`<!---->`);
      }
      _push(ssrRenderComponent(_component_TransactionModal, {
        modelValue: unref(transactionModalOpen),
        "onUpdate:modelValue": ($event) => isRef(transactionModalOpen) ? transactionModalOpen.value = $event : null,
        mode: unref(modalMode),
        type: unref(modalType),
        transaction: unref(selectedTransaction),
        onSaved: onTransactionSaved
      }, null, _parent));
      _push(ssrRenderComponent(_component_DeleteTransactionModal, {
        modelValue: unref(deleteModalOpen),
        "onUpdate:modelValue": ($event) => isRef(deleteModalOpen) ? deleteModalOpen.value = $event : null,
        transaction: unref(transactionToDelete),
        onDeleted: onTransactionDeleted
      }, null, _parent));
      _push(`</div>`);
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/transactions/index.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
const index = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-d65881e9"]]);
export {
  index as default
};
//# sourceMappingURL=index-Ci5AUSnw.js.map
