import { _ as __nuxt_component_2 } from './TransactionModal-D84yXN2T.mjs';
import { defineComponent, ref, computed, mergeProps, unref, useSSRContext } from 'vue';
import { ssrRenderAttrs, ssrRenderComponent } from 'vue/server-renderer';
import { _ as _export_sfc, i as useRoute$1, d as usePermissions, n as navigateTo } from './server.mjs';
import './SearchSelect-BFyTiPOW.mjs';
import '@vueuse/core';
import 'lucide-vue-next';
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
  __name: "new",
  __ssrInlineRender: true,
  setup(__props) {
    const route = useRoute$1();
    const { canCreateIncome, canCreateExpense, canViewTransactions, canViewDashboard } = usePermissions();
    const modalOpen = ref(true);
    const transactionType = computed(() => {
      if (route.query.type === "expense" && canCreateExpense.value) return "expense";
      if (route.query.type === "income" && canCreateIncome.value) return "income";
      if (canCreateIncome.value) return "income";
      return "expense";
    });
    function returnToPreviousPage() {
      if (canViewTransactions.value) return navigateTo("/transactions");
      if (canViewDashboard.value) return navigateTo("/dashboard");
      return navigateTo("/more");
    }
    function onModalChange(open) {
      modalOpen.value = open;
      if (!open) returnToPreviousPage();
    }
    return (_ctx, _push, _parent, _attrs) => {
      const _component_TransactionModal = __nuxt_component_2;
      _push(`<div${ssrRenderAttrs(mergeProps({ class: "new-transaction-route" }, _attrs))} data-v-21e13235>`);
      _push(ssrRenderComponent(_component_TransactionModal, {
        "model-value": unref(modalOpen),
        type: unref(transactionType),
        mode: "create",
        "onUpdate:modelValue": onModalChange
      }, null, _parent));
      _push(`</div>`);
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/transactions/new.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
const _new = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-21e13235"]]);

export { _new as default };
//# sourceMappingURL=new-BHgMEuLn.mjs.map
