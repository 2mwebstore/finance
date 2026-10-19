import { c as currentMonthRange, _ as __nuxt_component_0 } from './DateRangePicker-BmLhEXsC.mjs';
import { defineComponent, ref, watch, mergeProps, unref, isRef, useSSRContext } from 'vue';
import { ssrRenderAttrs, ssrIncludeBooleanAttr, ssrRenderComponent, ssrInterpolate } from 'vue/server-renderer';
import { Download } from 'lucide-vue-next';
import { f as formatMoney } from './money-BqQfZu3a.mjs';
import { _ as _export_sfc, h as useApi, f as useAuth, d as usePermissions } from './server.mjs';
import '@vueuse/core';
import '../nitro/nitro.mjs';
import 'node:http';
import 'node:https';
import 'node:events';
import 'node:buffer';
import 'node:fs';
import 'node:path';
import 'node:crypto';
import 'node:url';
import '../routes/renderer.mjs';
import 'vue-bundle-renderer/runtime';
import 'unhead/server';
import 'devalue';
import 'unhead/utils';
import 'unhead/plugins';
import 'vue-router';
import '@vuetify/v0/utilities';

const _sfc_main = /* @__PURE__ */ defineComponent({
  __name: "reports",
  __ssrInlineRender: true,
  setup(__props) {
    const { api } = useApi();
    useAuth();
    const { canExportReports } = usePermissions();
    const dateRange = ref(currentMonthRange());
    const data = ref(null);
    const loading = ref(true);
    const err = ref("");
    const exporting = ref(false);
    async function load() {
      var _a;
      loading.value = true;
      err.value = "";
      try {
        data.value = await api("/dashboard/summary", {
          query: { period: "custom", date_from: dateRange.value.from, date_to: dateRange.value.to }
        });
      } catch (e) {
        err.value = ((_a = e == null ? void 0 : e.data) == null ? void 0 : _a.error) || "Failed to load report";
      } finally {
        loading.value = false;
      }
    }
    watch(() => [dateRange.value.from, dateRange.value.to], ([from, to]) => {
      if (!from || !to) {
        dateRange.value = currentMonthRange();
        return;
      }
      void load();
    });
    return (_ctx, _push, _parent, _attrs) => {
      const _component_DateRangePicker = __nuxt_component_0;
      _push(`<section${ssrRenderAttrs(mergeProps({ class: "reports-page" }, _attrs))} data-v-7bd86687><header class="reports-heading" data-v-7bd86687><div data-v-7bd86687><p class="eyebrow" data-v-7bd86687>FINANCIAL INSIGHTS</p><h1 data-v-7bd86687>Reports</h1><p class="heading-copy" data-v-7bd86687>Summarized income and expenses for any date range.</p></div>`);
      if (unref(canExportReports)) {
        _push(`<button class="export-button"${ssrIncludeBooleanAttr(unref(exporting)) ? " disabled" : ""} data-v-7bd86687>`);
        _push(ssrRenderComponent(unref(Download), { size: 16 }, null, _parent));
        _push(` ${ssrInterpolate(unref(exporting) ? "Exporting\u2026" : "Export CSV")}</button>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</header><section class="reports-controls" data-v-7bd86687><div class="control-field" data-v-7bd86687><span data-v-7bd86687>Report date range</span>`);
      _push(ssrRenderComponent(_component_DateRangePicker, {
        modelValue: unref(dateRange),
        "onUpdate:modelValue": ($event) => isRef(dateRange) ? dateRange.value = $event : null
      }, null, _parent));
      _push(`</div><p class="control-hint" data-v-7bd86687>Updates automatically when both dates are selected.</p></section>`);
      if (unref(loading)) {
        _push(`<div class="state-card" data-v-7bd86687><span class="mini-spinner" data-v-7bd86687></span> Loading report\u2026</div>`);
      } else if (unref(err)) {
        _push(`<div class="state-card state-error" data-v-7bd86687><p data-v-7bd86687>${ssrInterpolate(unref(err))}</p><button data-v-7bd86687>Try again</button></div>`);
      } else if (unref(data)) {
        _push(`<div class="reports-grid" data-v-7bd86687><article class="summary-card usd-card" data-v-7bd86687><div class="summary-top" data-v-7bd86687><span class="summary-icon" data-v-7bd86687>USD</span><small data-v-7bd86687>United States Dollar</small></div><strong data-v-7bd86687>${ssrInterpolate(unref(formatMoney)(unref(data).net_usd, "USD"))}</strong><span data-v-7bd86687>Net balance</span></article><article class="summary-card khr-card" data-v-7bd86687><div class="summary-top" data-v-7bd86687><span class="summary-icon" data-v-7bd86687>KHR</span><small data-v-7bd86687>Cambodian Riel</small></div><strong data-v-7bd86687>${ssrInterpolate(unref(formatMoney)(unref(data).net_khr, "KHR"))}</strong><span data-v-7bd86687>Net balance</span></article><article class="breakdown-card" data-v-7bd86687><h2 data-v-7bd86687>Income and expense breakdown</h2><div class="breakdown-row" data-v-7bd86687><span data-v-7bd86687>Income \xB7 USD</span><strong class="income-text" data-v-7bd86687>${ssrInterpolate(unref(formatMoney)(unref(data).income_usd, "USD"))}</strong></div><div class="breakdown-row" data-v-7bd86687><span data-v-7bd86687>Expense \xB7 USD</span><strong class="expense-text" data-v-7bd86687>${ssrInterpolate(unref(formatMoney)(unref(data).expense_usd, "USD"))}</strong></div><div class="breakdown-row" data-v-7bd86687><span data-v-7bd86687>Income \xB7 KHR</span><strong class="income-text" data-v-7bd86687>${ssrInterpolate(unref(formatMoney)(unref(data).income_khr, "KHR"))}</strong></div><div class="breakdown-row" data-v-7bd86687><span data-v-7bd86687>Expense \xB7 KHR</span><strong class="expense-text" data-v-7bd86687>${ssrInterpolate(unref(formatMoney)(unref(data).expense_khr, "KHR"))}</strong></div><div class="breakdown-row total-row" data-v-7bd86687><span data-v-7bd86687>Approved transactions</span><strong data-v-7bd86687>${ssrInterpolate(unref(data).transaction_count)}</strong></div></article></div>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</section>`);
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/reports.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
const reports = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-7bd86687"]]);

export { reports as default };
//# sourceMappingURL=reports-CI8BDXv_.mjs.map
