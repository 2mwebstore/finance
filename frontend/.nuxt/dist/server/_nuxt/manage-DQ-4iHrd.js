import { d as usePermissions, g as __nuxt_component_0, _ as _export_sfc } from "../server.mjs";
import { defineComponent, computed, mergeProps, withCtx, createVNode, useSSRContext } from "vue";
import { ssrRenderAttrs, ssrRenderComponent } from "vue/server-renderer";
import { Building2, ShieldCheck, KeyRound, UsersRound, ReceiptText, Tags } from "lucide-vue-next";
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
  __name: "manage",
  __ssrInlineRender: true,
  setup(__props) {
    const {
      can,
      canViewBranches,
      canViewUsers,
      canViewRoles,
      canViewAudit
    } = usePermissions();
    computed(() => {
      const items = [];
      if (canViewBranches.value) items.push({ label: "Branches", description: "Create and update business locations.", to: "/admin/branches", icon: Building2 });
      if (canViewRoles.value) items.push({ label: "Roles & Permissions", description: "Manage team roles and access.", to: "/admin/roles", icon: ShieldCheck });
      if (canViewRoles.value) items.push({ label: "Permission Codes", description: "Review the permissions available in the system.", to: "/admin/permissions", icon: KeyRound });
      if (canViewUsers.value) items.push({ label: "Users", description: "Manage user accounts, roles, and status.", to: "/admin/users", icon: UsersRound });
      if (canViewAudit.value) items.push({ label: "Audit Log", description: "Review account activity and filter by date.", to: "/admin/audit", icon: ReceiptText });
      if (can("categories.manage")) items.push({ label: "Categories", description: "Manage expense categories.", to: "/admin/categories", icon: Tags });
      return items;
    });
    return (_ctx, _push, _parent, _attrs) => {
      const _component_ClientOnly = __nuxt_component_0;
      _push(`<section${ssrRenderAttrs(mergeProps({ class: "space-y-4" }, _attrs))} data-v-00d40db0><header data-v-00d40db0><p class="text-[10px] font-bold tracking-[0.14em] text-slate-400" data-v-00d40db0>ADMINISTRATION</p><h1 class="text-xl font-bold text-navy" data-v-00d40db0>Manage</h1><p class="mt-1 text-xs text-muted" data-v-00d40db0>Choose an area to manage your business.</p></header>`);
      _push(ssrRenderComponent(_component_ClientOnly, null, {
        fallback: withCtx((_, _push2, _parent2, _scopeId) => {
          if (_push2) {
            _push2(`<div class="rounded-xl border border-border bg-white p-5 text-sm text-muted" data-v-00d40db0${_scopeId}>Loading management sections…</div>`);
          } else {
            return [
              createVNode("div", { class: "rounded-xl border border-border bg-white p-5 text-sm text-muted" }, "Loading management sections…")
            ];
          }
        })
      }, _parent));
      _push(`</section>`);
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/manage.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
const manage = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-00d40db0"]]);
export {
  manage as default
};
//# sourceMappingURL=manage-DQ-4iHrd.js.map
