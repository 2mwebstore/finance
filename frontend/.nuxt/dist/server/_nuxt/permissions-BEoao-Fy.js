import { defineComponent, ref, computed, mergeProps, unref, useSSRContext } from "vue";
import { ssrRenderAttrs, ssrInterpolate, ssrIncludeBooleanAttr, ssrRenderList, ssrRenderClass } from "vue/server-renderer";
import { j as usePermissionsStore, d as usePermissions } from "../server.mjs";
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
  __name: "permissions",
  __ssrInlineRender: true,
  setup(__props) {
    const store = usePermissionsStore();
    const { catalog, catalogSynced, catalogSyncError, can } = usePermissions();
    const moduleFilter = ref("");
    const syncing = ref(false);
    const moduleOptions = computed(() => {
      const set = new Set(catalog.value.map((c) => c.module));
      return ["", ...Array.from(set).sort()];
    });
    const filtered = computed(
      () => moduleFilter.value ? catalog.value.filter((c) => c.module === moduleFilter.value) : catalog.value
    );
    function formatTime(d) {
      try {
        return new Date(d).toLocaleString();
      } catch {
        return d;
      }
    }
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<div${ssrRenderAttrs(mergeProps({ class: "space-y-4" }, _attrs))}><div class="flex flex-wrap items-center justify-between gap-2"><h1 class="text-xl font-bold text-navy">Permission codes</h1><div class="flex items-center gap-2"><span class="text-xs text-muted">${ssrInterpolate(unref(catalog).length)} codes</span><button class="rounded-lg border border-border bg-white px-3 py-1.5 text-xs font-medium hover:bg-canvas"${ssrIncludeBooleanAttr(unref(syncing)) ? " disabled" : ""}>${ssrInterpolate(unref(syncing) ? "Syncing…" : "Sync from API")}</button></div></div><div class="text-xs text-muted flex flex-wrap gap-3">`);
      if (unref(catalogSynced)) {
        _push(`<span class="text-income">● Synced from API</span>`);
      } else {
        _push(`<span class="text-muted">○ Using static catalog</span>`);
      }
      if (unref(catalogSyncError)) {
        _push(`<span class="text-expense">${ssrInterpolate(unref(catalogSyncError))}</span>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(store).lastSyncedAt) {
        _push(`<span>Last: ${ssrInterpolate(formatTime(unref(store).lastSyncedAt))}</span>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</div><p class="text-sm text-muted"> Codes are enforced by the API. This list is loaded from <code class="text-xs bg-canvas px-1 rounded">GET /api/v1/permissions</code> when your role allows it. </p><div class="flex gap-2 overflow-x-auto pb-1"><!--[-->`);
      ssrRenderList(unref(moduleOptions), (m) => {
        _push(`<button class="${ssrRenderClass([unref(moduleFilter) === m ? "bg-primary text-white border-primary" : "bg-white text-muted border-border", "shrink-0 rounded-full px-3 py-1.5 text-xs font-medium border"])}">${ssrInterpolate(m || "All")}</button>`);
      });
      _push(`<!--]--></div><div class="bg-white rounded-xl border border-border overflow-hidden"><table class="w-full text-sm"><thead class="bg-canvas text-left text-xs text-muted"><tr><th class="px-3 py-2 font-medium">Code</th><th class="px-3 py-2 font-medium hidden sm:table-cell">Name</th><th class="px-3 py-2 font-medium">Module</th><th class="px-3 py-2 font-medium hidden md:table-cell">You</th></tr></thead><tbody class="divide-y divide-border"><!--[-->`);
      ssrRenderList(unref(filtered), (row) => {
        _push(`<tr><td class="px-3 py-2 font-mono text-xs text-primary">${ssrInterpolate(row.code)}</td><td class="px-3 py-2 hidden sm:table-cell">${ssrInterpolate(row.name)}</td><td class="px-3 py-2 text-xs text-muted">${ssrInterpolate(row.module)}</td><td class="px-3 py-2 hidden md:table-cell">`);
        if (unref(can)(row.code)) {
          _push(`<span class="text-[10px] text-income font-semibold">YES</span>`);
        } else {
          _push(`<span class="text-[10px] text-muted">—</span>`);
        }
        _push(`</td></tr>`);
      });
      _push(`<!--]--></tbody></table></div></div>`);
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/admin/permissions.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
export {
  _sfc_main as default
};
//# sourceMappingURL=permissions-BEoao-Fy.js.map
