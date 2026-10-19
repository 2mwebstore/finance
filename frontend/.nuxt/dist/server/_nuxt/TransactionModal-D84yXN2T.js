import { _ as __nuxt_component_1 } from "./SearchSelect-BFyTiPOW.js";
import { defineComponent, ref, watch, mergeProps, unref, useSSRContext, reactive, computed } from "vue";
import { ssrRenderAttrs, ssrRenderAttr, ssrRenderClass, ssrRenderComponent, ssrInterpolate, ssrIncludeBooleanAttr, ssrLooseContain } from "vue/server-renderer";
import { ArrowDownLeft, ArrowUpRight, X } from "lucide-vue-next";
import { h as useApi, f as useAuth, d as usePermissions, _ as _export_sfc } from "../server.mjs";
const _sfc_main$1 = /* @__PURE__ */ defineComponent({
  __name: "MoneyInput",
  __ssrInlineRender: true,
  props: {
    modelValue: {},
    currency: {},
    ariaLabel: {}
  },
  emits: ["update:modelValue"],
  setup(__props, { emit: __emit }) {
    const props = __props;
    const emit = __emit;
    const input = ref(null);
    const focused = ref(false);
    const displayValue = ref(formatEntry(props.modelValue, props.currency, false));
    function normalize(value, currency) {
      const cleaned = value.replace(/,/g, "").replace(/[^\d.]/g, "");
      if (currency === "KHR") return cleaned.split(".")[0].replace(/\D/g, "");
      const dot = cleaned.indexOf(".");
      if (dot < 0) return cleaned;
      const whole = cleaned.slice(0, dot).replace(/\./g, "");
      const fraction = cleaned.slice(dot + 1).replace(/\./g, "").slice(0, 2);
      return `${whole}.${fraction}`;
    }
    function groupDigits(value) {
      return value.replace(/\B(?=(\d{3})+(?!\d))/g, ",");
    }
    function formatEntry(value, currency, complete = false) {
      const raw = normalize(value, currency);
      if (!raw) return "";
      if (currency === "KHR") return groupDigits(raw);
      const hasDot = raw.includes(".");
      let [whole, fraction = ""] = raw.split(".");
      if (!whole) whole = "0";
      if (complete) fraction = fraction.padEnd(2, "0").slice(0, 2);
      return `${groupDigits(whole)}${hasDot || complete ? `.${fraction}` : ""}`;
    }
    watch(() => [props.modelValue, props.currency], ([value, currency]) => {
      if (!focused.value) displayValue.value = formatEntry(String(value), currency, true);
    });
    watch(() => props.currency, (currency, previousCurrency) => {
      if (previousCurrency && currency !== previousCurrency) {
        const amount = normalize(props.modelValue, currency);
        emit("update:modelValue", amount);
        displayValue.value = formatEntry(amount, currency);
      }
    });
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<input${ssrRenderAttrs(mergeProps({
        ref_key: "input",
        ref: input,
        value: unref(displayValue),
        type: "text",
        inputmode: "decimal",
        autocomplete: "off",
        placeholder: __props.currency === "USD" ? "0.00" : "0",
        "aria-label": __props.ariaLabel
      }, _attrs))}>`);
    };
  }
});
const _sfc_setup$1 = _sfc_main$1.setup;
_sfc_main$1.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("components/MoneyInput.vue");
  return _sfc_setup$1 ? _sfc_setup$1(props, ctx) : void 0;
};
const _sfc_main = /* @__PURE__ */ defineComponent({
  __name: "TransactionModal",
  __ssrInlineRender: true,
  props: {
    modelValue: { type: Boolean },
    mode: { default: "create" },
    type: { default: "income" },
    transaction: { default: null }
  },
  emits: ["update:modelValue", "saved"],
  setup(__props, { emit: __emit }) {
    const props = __props;
    const { api } = useApi();
    const { user } = useAuth();
    const { canViewBranches } = usePermissions();
    const branches = ref([]);
    const categories = ref([]);
    const form = reactive({
      type: "income",
      currency: "USD",
      amount: "",
      transaction_date: localDate(),
      branch_id: "",
      category_id: "",
      description: "",
      counterparty: "",
      pending: false
    });
    const loadingOptions = ref(true);
    const categoryLoading = ref(false);
    const saving = ref(false);
    const error = ref("");
    const branchOptions = computed(() => branches.value.map((branch) => ({ value: branch.id, label: `${branch.code} — ${branch.name}` })));
    const categoryOptions = computed(() => categories.value.map((category) => ({
      value: category.id,
      label: `${category.name}${category.is_active ? "" : " (inactive)"}`
    })));
    const currencyOptions = [
      { value: "USD", label: "USD ($)" },
      { value: "KHR", label: "KHR (៛)" }
    ];
    const transactionType = computed(() => props.mode === "edit" ? props.transaction?.type || props.type : props.type);
    const actionLabel = computed(() => props.mode === "edit" ? "Edit" : "Add");
    function localDate() {
      const now = /* @__PURE__ */ new Date();
      return `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, "0")}-${String(now.getDate()).padStart(2, "0")}`;
    }
    function initializeForm() {
      const txn = props.mode === "edit" ? props.transaction : null;
      form.type = transactionType.value;
      form.currency = txn?.currency || "USD";
      form.amount = txn?.amount || "";
      form.transaction_date = txn?.transaction_date?.slice(0, 10) || localDate();
      form.branch_id = txn?.branch_id || user.value?.branches?.[0]?.id || "";
      form.category_id = txn?.category_id || "";
      form.description = txn?.description || "";
      form.counterparty = txn?.counterparty || "";
      form.pending = false;
      error.value = "";
    }
    async function loadCategories() {
      if (transactionType.value !== "expense") {
        categories.value = [];
        categoryLoading.value = false;
        form.category_id = "";
        return;
      }
      categoryLoading.value = true;
      try {
        const result = await api("/categories", {
          query: {
            type: transactionType.value,
            active: props.mode === "edit" ? void 0 : "true"
          }
        });
        categories.value = (result.items || []).filter((category) => category.type === transactionType.value);
        if (props.mode === "create" && form.category_id && !categories.value.some((category) => category.id === form.category_id)) {
          form.category_id = "";
        }
      } catch (e) {
        categories.value = [];
        error.value = e?.data?.error || "Could not load categories.";
      } finally {
        categoryLoading.value = false;
      }
    }
    async function loadBranches() {
      branches.value = user.value?.branches || [];
      if (canViewBranches.value) {
        try {
          const result = await api("/branches", { query: { is_active: true, page_size: 100 } });
          branches.value = result.items || branches.value;
        } catch {
        }
      }
      if (!form.branch_id && branches.value.length) form.branch_id = branches.value[0].id;
    }
    async function prepareModal() {
      initializeForm();
      loadingOptions.value = true;
      await Promise.all([loadCategories(), props.mode === "create" ? loadBranches() : Promise.resolve()]);
      loadingOptions.value = false;
    }
    watch(() => [props.modelValue, props.mode, props.type, props.transaction?.id], ([isOpen]) => {
      if (isOpen) prepareModal();
    });
    return (_ctx, _push, _parent, _attrs) => {
      const _component_SearchSelect = __nuxt_component_1;
      const _component_MoneyInput = _sfc_main$1;
      if (__props.modelValue) {
        _push(`<div${ssrRenderAttrs(mergeProps({ class: "modal-backdrop" }, _attrs))} data-v-aada020f><section class="transaction-modal" role="dialog" aria-modal="true"${ssrRenderAttr("aria-label", `${unref(actionLabel)} ${unref(transactionType)}`)} data-v-aada020f><header class="modal-header" data-v-aada020f><div class="modal-title-group" data-v-aada020f><span class="${ssrRenderClass([unref(transactionType) === "income" ? "income-icon" : "expense-icon", "modal-icon"])}" data-v-aada020f>`);
        if (unref(transactionType) === "income") {
          _push(ssrRenderComponent(unref(ArrowDownLeft), { size: 20 }, null, _parent));
        } else {
          _push(ssrRenderComponent(unref(ArrowUpRight), { size: 20 }, null, _parent));
        }
        _push(`</span><div data-v-aada020f><p class="modal-eyebrow" data-v-aada020f>${ssrInterpolate(__props.mode === "edit" ? "UPDATE RECORD" : "NEW RECORD")}</p><h2 data-v-aada020f>${ssrInterpolate(unref(actionLabel))} ${ssrInterpolate(unref(transactionType) === "income" ? "Income" : "Expense")}</h2></div></div><button type="button" class="icon-button" aria-label="Close dialog" data-v-aada020f>`);
        _push(ssrRenderComponent(unref(X), { size: 19 }, null, _parent));
        _push(`</button></header>`);
        if (unref(loadingOptions)) {
          _push(`<div class="modal-loading" data-v-aada020f>Loading form options…</div>`);
        } else {
          _push(`<form class="transaction-form" data-v-aada020f><div class="form-grid" data-v-aada020f><label class="form-field amount-field" data-v-aada020f><span data-v-aada020f>Amount</span><div class="amount-control" data-v-aada020f>`);
          _push(ssrRenderComponent(_component_SearchSelect, {
            modelValue: unref(form).currency,
            "onUpdate:modelValue": ($event) => unref(form).currency = $event,
            options: currencyOptions,
            "aria-label": "Currency",
            disabled: __props.mode === "edit",
            class: "currency-select"
          }, null, _parent));
          _push(ssrRenderComponent(_component_MoneyInput, {
            modelValue: unref(form).amount,
            "onUpdate:modelValue": ($event) => unref(form).amount = $event,
            currency: unref(form).currency,
            "aria-label": "Transaction amount",
            required: ""
          }, null, _parent));
          _push(`</div></label><label class="form-field" data-v-aada020f><span data-v-aada020f>Date</span><input${ssrRenderAttr("value", unref(form).transaction_date)} type="date" required data-v-aada020f></label>`);
          if (__props.mode === "create") {
            _push(`<div class="form-field full-field" data-v-aada020f><span data-v-aada020f>Branch</span>`);
            _push(ssrRenderComponent(_component_SearchSelect, {
              modelValue: unref(form).branch_id,
              "onUpdate:modelValue": ($event) => unref(form).branch_id = $event,
              options: unref(branchOptions),
              placeholder: "Select a branch",
              "search-placeholder": "Search branches…",
              "aria-label": "Branch"
            }, null, _parent));
            if (!unref(branches).length) {
              _push(`<small class="field-hint" data-v-aada020f>No accessible branches are available.</small>`);
            } else {
              _push(`<!---->`);
            }
            _push(`</div>`);
          } else {
            _push(`<!---->`);
          }
          if (unref(transactionType) === "expense") {
            _push(`<div class="form-field full-field" data-v-aada020f><span data-v-aada020f>Category</span>`);
            _push(ssrRenderComponent(_component_SearchSelect, {
              modelValue: unref(form).category_id,
              "onUpdate:modelValue": ($event) => unref(form).category_id = $event,
              options: unref(categoryOptions),
              placeholder: "Select a category",
              "search-placeholder": "Search categories…",
              "aria-label": "Expense category",
              disabled: !unref(categories).length
            }, null, _parent));
            if (!unref(categories).length && !unref(categoryLoading)) {
              _push(`<small class="field-hint" data-v-aada020f>There are no expense categories available.</small>`);
            } else {
              _push(`<!---->`);
            }
            _push(`</div>`);
          } else {
            _push(`<!---->`);
          }
          _push(`<label class="form-field full-field" data-v-aada020f><span data-v-aada020f>Description <small data-v-aada020f>Optional</small></span><textarea rows="2" maxlength="1024" placeholder="What is this transaction for?" data-v-aada020f>${ssrInterpolate(unref(form).description)}</textarea></label><label class="form-field full-field" data-v-aada020f><span data-v-aada020f>Customer or supplier <small data-v-aada020f>Optional</small></span><input${ssrRenderAttr("value", unref(form).counterparty)} maxlength="255" placeholder="Name of payer or payee" data-v-aada020f></label></div>`);
          if (__props.mode === "create") {
            _push(`<label class="pending-option" data-v-aada020f><input${ssrIncludeBooleanAttr(Array.isArray(unref(form).pending) ? ssrLooseContain(unref(form).pending, null) : unref(form).pending) ? " checked" : ""} type="checkbox" data-v-aada020f><span data-v-aada020f><strong data-v-aada020f>Submit for approval</strong><small data-v-aada020f>This transaction stays pending until approved.</small></span></label>`);
          } else {
            _push(`<!---->`);
          }
          if (unref(error)) {
            _push(`<p class="form-error" role="alert" data-v-aada020f>${ssrInterpolate(unref(error))}</p>`);
          } else {
            _push(`<!---->`);
          }
          _push(`<div class="modal-actions" data-v-aada020f><button type="button" class="cancel-button"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""} data-v-aada020f>Cancel</button><button type="submit" class="${ssrRenderClass([unref(transactionType) === "income" ? "income-button" : "expense-button", "save-button"])}"${ssrIncludeBooleanAttr(unref(saving) || unref(transactionType) === "expense" && !unref(categories).length || __props.mode === "create" && (!unref(branches).length || !unref(form).branch_id) || unref(transactionType) === "expense" && !unref(form).category_id) ? " disabled" : ""} data-v-aada020f>${ssrInterpolate(unref(saving) ? "Saving…" : __props.mode === "edit" ? "Save changes" : `Save ${unref(transactionType)}`)}</button></div></form>`);
        }
        _push(`</section></div>`);
      } else {
        _push(`<!---->`);
      }
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("components/TransactionModal.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
const __nuxt_component_2 = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-aada020f"]]);
export {
  __nuxt_component_2 as _
};
//# sourceMappingURL=TransactionModal-D84yXN2T.js.map
