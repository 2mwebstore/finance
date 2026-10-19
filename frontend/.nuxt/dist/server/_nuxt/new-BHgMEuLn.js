import { _ as __nuxt_component_2 } from "./TransactionModal-D84yXN2T.js";
import { defineComponent, ref, computed, mergeProps, unref, useSSRContext } from "vue";
import { ssrRenderAttrs, ssrRenderComponent } from "vue/server-renderer";
import { i as useRoute, d as usePermissions, n as navigateTo, _ as _export_sfc } from "../server.mjs";
import "./SearchSelect-BFyTiPOW.js";
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
  __name: "new",
  __ssrInlineRender: true,
  setup(__props) {
    const route = useRoute();
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
export {
  _new as default
};
//# sourceMappingURL=new-BHgMEuLn.js.map
