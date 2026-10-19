import { defineComponent, computed, ref, reactive, watch, mergeProps, unref, useSSRContext } from "vue";
import { ssrRenderAttrs, ssrRenderComponent, ssrInterpolate, ssrRenderList, ssrRenderClass, ssrRenderAttr, ssrIncludeBooleanAttr, ssrLooseContain } from "vue/server-renderer";
import { Plus, Pencil, X } from "lucide-vue-next";
import { h as useApi, d as usePermissions } from "../server.mjs";
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
  __name: "categories",
  __ssrInlineRender: true,
  setup(__props) {
    useApi();
    const { can } = usePermissions();
    const canManage = computed(() => can("categories.manage"));
    const categories = ref([]);
    const loading = ref(true);
    const error = ref("");
    const modalOpen = ref(false);
    const formMode = ref("create");
    ref(null);
    const saving = ref(false);
    const formError = ref("");
    const codeEdited = ref(false);
    const form = reactive({ name: "", code: "", description: "", is_active: true });
    watch(() => form.name, (name) => {
      if (formMode.value === "create" && !codeEdited.value) form.code = slugify(name);
    });
    watch(() => form.code, (code) => {
      if (formMode.value === "create" && code && code !== slugify(form.name)) codeEdited.value = true;
    });
    function slugify(value) {
      return value.toLowerCase().trim().replace(/[^a-z0-9]+/g, "-").replace(/^-|-$/g, "");
    }
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<section${ssrRenderAttrs(mergeProps({ class: "space-y-4" }, _attrs))}><header class="flex flex-wrap items-center justify-between gap-3"><div><p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">SETUP</p><h1 class="text-xl font-bold text-navy">Categories</h1><p class="mt-1 text-xs text-muted">Manage the categories available on expense forms.</p></div>`);
      if (unref(canManage)) {
        _push(`<button class="inline-flex items-center gap-2 rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover">`);
        _push(ssrRenderComponent(unref(Plus), { size: 16 }, null, _parent));
        _push(` Add category </button>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</header>`);
      if (unref(loading)) {
        _push(`<div class="rounded-xl border border-border bg-white p-6 text-center text-sm text-muted">Loading categories…</div>`);
      } else if (unref(error)) {
        _push(`<div class="flex items-center justify-between gap-3 rounded-xl border border-red-100 bg-white p-4 text-sm text-expense"><span>${ssrInterpolate(unref(error))}</span><button class="font-semibold text-primary">Try again</button></div>`);
      } else {
        _push(`<div class="overflow-hidden rounded-xl border border-border bg-white"><!--[-->`);
        ssrRenderList(unref(categories), (category) => {
          _push(`<article class="flex items-start justify-between gap-4 border-b border-border p-4 last:border-0"><div class="min-w-0"><div class="flex flex-wrap items-center gap-2"><h2 class="text-sm font-semibold text-navy">${ssrInterpolate(category.name)}</h2><span class="${ssrRenderClass([category.is_active ? "bg-green-50 text-income" : "bg-slate-100 text-muted", "rounded-full px-2 py-0.5 text-[9px] font-bold uppercase"])}">${ssrInterpolate(category.is_active ? "Active" : "Inactive")}</span>`);
          if (category.is_system) {
            _push(`<span class="rounded-full bg-blue-50 px-2 py-0.5 text-[9px] font-bold uppercase text-primary">System</span>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</div><p class="mt-1 font-mono text-[10px] text-muted">${ssrInterpolate(category.code)} · ${ssrInterpolate(category.type)}</p>`);
          if (category.description) {
            _push(`<p class="mt-1 text-xs text-muted">${ssrInterpolate(category.description)}</p>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</div>`);
          if (unref(canManage)) {
            _push(`<button class="inline-flex shrink-0 items-center gap-1 rounded-lg border border-border px-2.5 py-1.5 text-xs font-medium text-navy hover:bg-canvas">`);
            _push(ssrRenderComponent(unref(Pencil), { size: 13 }, null, _parent));
            _push(` Edit</button>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</article>`);
        });
        _push(`<!--]-->`);
        if (!unref(categories).length) {
          _push(`<div class="p-8 text-center text-sm text-muted">No expense categories yet. Add a category to make it available on expense forms.</div>`);
        } else {
          _push(`<!---->`);
        }
        _push(`</div>`);
      }
      if (unref(modalOpen)) {
        _push(`<div class="fixed inset-0 z-[90] flex items-end justify-center bg-slate-950/50 p-0 backdrop-blur-sm sm:items-center sm:p-5"><section class="w-full max-w-lg rounded-t-2xl border border-border bg-white shadow-2xl sm:rounded-2xl" role="dialog" aria-modal="true"${ssrRenderAttr("aria-label", unref(formMode) === "create" ? "Add category" : "Edit category")}><header class="flex items-center justify-between border-b border-border px-5 py-4"><div><p class="text-[9px] font-extrabold tracking-[0.14em] text-slate-400">${ssrInterpolate(unref(formMode) === "create" ? "NEW CATEGORY" : "UPDATE CATEGORY")}</p><h2 class="mt-1 text-base font-bold text-navy">${ssrInterpolate(unref(formMode) === "create" ? "Add category" : "Edit category")}</h2></div><button class="grid h-8 w-8 place-items-center rounded-lg bg-canvas text-muted hover:text-navy" aria-label="Close">`);
        _push(ssrRenderComponent(unref(X), { size: 17 }, null, _parent));
        _push(`</button></header><form class="space-y-3 p-5"><label class="block space-y-1.5 text-xs font-semibold text-navy"><span>Name</span><input${ssrRenderAttr("value", unref(form).name)} required minlength="2" maxlength="128" class="w-full rounded-lg border border-border px-3 py-2.5 font-normal" placeholder="e.g. Office supplies"></label><label class="block space-y-1.5 text-xs font-semibold text-navy"><span>Code</span><input${ssrRenderAttr("value", unref(form).code)} required minlength="2" maxlength="64" class="w-full rounded-lg border border-border px-3 py-2.5 font-mono font-normal" placeholder="e.g. office-supplies"><small class="block font-normal text-muted">${ssrInterpolate(unref(formMode) === "create" ? "Generated from the name; you can adjust it before saving." : "Changing the code keeps existing transactions linked to this category.")}</small></label><label class="block space-y-1.5 text-xs font-semibold text-navy"><span>Description <small class="font-normal text-muted">Optional</small></span><textarea rows="2" maxlength="512" class="w-full rounded-lg border border-border px-3 py-2.5 font-normal" placeholder="Add a short description">${ssrInterpolate(unref(form).description)}</textarea></label>`);
        if (unref(formMode) === "edit") {
          _push(`<label class="flex items-center gap-2 rounded-lg border border-border bg-canvas px-3 py-2.5 text-xs text-navy"><input${ssrIncludeBooleanAttr(Array.isArray(unref(form).is_active) ? ssrLooseContain(unref(form).is_active, null) : unref(form).is_active) ? " checked" : ""} type="checkbox" class="accent-primary"> Category is active and available on new forms </label>`);
        } else {
          _push(`<!---->`);
        }
        if (unref(formError)) {
          _push(`<p class="rounded-lg bg-red-50 px-3 py-2 text-xs text-expense" role="alert">${ssrInterpolate(unref(formError))}</p>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<footer class="flex justify-end gap-2 border-t border-border pt-4"><button type="button" class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>Cancel</button><button type="submit" class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-white disabled:opacity-60"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>${ssrInterpolate(unref(saving) ? "Saving…" : "Save category")}</button></footer></form></section></div>`);
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
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/admin/categories.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
export {
  _sfc_main as default
};
//# sourceMappingURL=categories-dn42pZlQ.js.map
