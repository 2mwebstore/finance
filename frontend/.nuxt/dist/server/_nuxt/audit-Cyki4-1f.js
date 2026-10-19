import { c as currentMonthRange, _ as __nuxt_component_0 } from "./DateRangePicker-BmLhEXsC.js";
import { _ as __nuxt_component_1 } from "./SearchSelect-BFyTiPOW.js";
import { defineComponent, ref, watch, mergeProps, isRef, unref, useSSRContext } from "vue";
import { ssrRenderAttrs, ssrRenderComponent, ssrInterpolate, ssrRenderList, ssrRenderClass } from "vue/server-renderer";
import { h as useApi } from "../server.mjs";
import "@vueuse/core";
import "lucide-vue-next";
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
const _sfc_main = /* @__PURE__ */ defineComponent({
  __name: "audit",
  __ssrInlineRender: true,
  setup(__props) {
    const { api } = useApi();
    const items = ref([]);
    const total = ref(0);
    const loading = ref(true);
    const err = ref("");
    const actionFilter = ref("");
    const dateRange = ref(currentMonthRange());
    const actionOptions = [
      { value: "", label: "All actions" },
      { value: "auth.login", label: "Login" },
      { value: "auth.login_failed", label: "Login failed" },
      { value: "transaction.create", label: "Transaction created" },
      { value: "transaction.void", label: "Transaction voided" },
      { value: "transaction.approve", label: "Transaction approved" },
      { value: "transaction.reverse", label: "Transaction reversed" },
      { value: "report.export", label: "Report export" },
      { value: "settings.change", label: "Settings change" }
    ];
    function formatTime(d) {
      try {
        return new Date(d).toLocaleString("en-GB", { dateStyle: "short", timeStyle: "medium" });
      } catch {
        return d;
      }
    }
    async function load() {
      loading.value = true;
      err.value = "";
      try {
        const res = await api("/audit-logs", {
          query: {
            action: actionFilter.value || void 0,
            date_from: dateRange.value.from || void 0,
            date_to: dateRange.value.to || void 0,
            page_size: 50
          }
        });
        items.value = res.items || [];
        total.value = res.total || 0;
      } catch (e) {
        err.value = e?.data?.error || "Failed to load audit log";
      } finally {
        loading.value = false;
      }
    }
    watch(() => [actionFilter.value, dateRange.value.from, dateRange.value.to], () => {
      void load();
    });
    return (_ctx, _push, _parent, _attrs) => {
      const _component_DateRangePicker = __nuxt_component_0;
      const _component_SearchSelect = __nuxt_component_1;
      _push(`<div${ssrRenderAttrs(mergeProps({ class: "space-y-4" }, _attrs))}><div class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between"><h1 class="text-xl font-bold text-navy">Audit log</h1><div class="grid grid-cols-1 gap-2 sm:grid-cols-2"><label class="min-w-0 space-y-1"><span class="block text-xs font-semibold text-muted">Date range</span>`);
      _push(ssrRenderComponent(_component_DateRangePicker, {
        modelValue: unref(dateRange),
        "onUpdate:modelValue": ($event) => isRef(dateRange) ? dateRange.value = $event : null
      }, null, _parent));
      _push(`</label><label class="min-w-0 space-y-1"><span class="block text-xs font-semibold text-muted">Action</span>`);
      _push(ssrRenderComponent(_component_SearchSelect, {
        modelValue: unref(actionFilter),
        "onUpdate:modelValue": ($event) => isRef(actionFilter) ? actionFilter.value = $event : null,
        options: actionOptions,
        "aria-label": "Filter audit actions",
        class: "w-full sm:w-56"
      }, null, _parent));
      _push(`</label></div></div>`);
      if (unref(loading)) {
        _push(`<div class="text-sm text-muted">Loading…</div>`);
      } else if (unref(err)) {
        _push(`<div class="text-sm text-expense">${ssrInterpolate(unref(err))}</div>`);
      } else {
        _push(`<ul class="bg-white rounded-xl border border-border divide-y divide-border"><!--[-->`);
        ssrRenderList(unref(items), (a) => {
          _push(`<li class="p-3 text-sm"><div class="flex justify-between gap-2"><span class="font-mono text-xs text-primary">${ssrInterpolate(a.action)}</span><span class="${ssrRenderClass([a.outcome === "success" ? "text-income" : "text-expense", "text-[10px] uppercase font-semibold"])}">${ssrInterpolate(a.outcome)}</span></div><div class="text-xs text-muted mt-0.5">${ssrInterpolate(formatTime(a.created_at))} `);
          if (a.actor_email) {
            _push(`<span> · ${ssrInterpolate(a.actor_email)}</span>`);
          } else {
            _push(`<!---->`);
          }
          if (a.ip_address) {
            _push(`<span> · ${ssrInterpolate(a.ip_address)}</span>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</div>`);
          if (a.entity_type) {
            _push(`<div class="text-xs text-muted">${ssrInterpolate(a.entity_type)} ${ssrInterpolate(a.entity_id ? `#${a.entity_id.slice(0, 8)}` : "")}</div>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</li>`);
        });
        _push(`<!--]-->`);
        if (!unref(items).length) {
          _push(`<li class="p-6 text-center text-sm text-muted">No audit events</li>`);
        } else {
          _push(`<!---->`);
        }
        _push(`</ul>`);
      }
      _push(`<p class="text-xs text-muted">Showing ${ssrInterpolate(unref(items).length)} of ${ssrInterpolate(unref(total))}</p></div>`);
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/admin/audit.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
export {
  _sfc_main as default
};
//# sourceMappingURL=audit-Cyki4-1f.js.map
