import { defineComponent, computed, ref, reactive, mergeProps, unref, useSSRContext } from 'vue';
import { ssrRenderAttrs, ssrInterpolate, ssrIncludeBooleanAttr, ssrRenderAttr, ssrRenderList, ssrRenderClass } from 'vue/server-renderer';
import { h as useApi, d as usePermissions } from './server.mjs';
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
  __name: "branches",
  __ssrInlineRender: true,
  setup(__props) {
    useApi();
    const { can, canCreateBranch } = usePermissions();
    const canCreate = canCreateBranch;
    const canEdit = computed(() => can("branches.edit", "branches.manage"));
    const canManage = computed(() => can("branches.manage"));
    const items = ref([]);
    const loading = ref(true);
    const err = ref("");
    const notice = ref("");
    const showCreate = ref(false);
    const saving = ref(false);
    const formError = ref("");
    const archiveError = ref("");
    const editing = ref(null);
    const archiveTarget = ref(null);
    const form = reactive({ code: "", name: "", address: "", phone: "", email: "" });
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<section${ssrRenderAttrs(mergeProps({ class: "space-y-4" }, _attrs))}><header class="flex flex-wrap items-center justify-between gap-3"><div><p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">BUSINESS LOCATIONS</p><h1 class="text-xl font-bold text-navy">Branches</h1><p class="mt-1 text-xs text-muted">Create, view, update, and archive business locations.</p></div>`);
      if (unref(canCreate)) {
        _push(`<button class="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover">${ssrInterpolate(unref(showCreate) ? "Cancel" : "+ Branch")}</button>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</header>`);
      if (unref(showCreate) || unref(editing)) {
        _push(`<form class="space-y-3 rounded-xl border border-border bg-white p-4"><div class="flex items-start justify-between gap-3"><div><h2 class="text-sm font-semibold text-navy">${ssrInterpolate(unref(editing) ? `Edit ${unref(editing).name}` : "Create branch")}</h2><p class="mt-1 text-xs text-muted">Update the location details used on transactions.</p></div><button type="button" class="text-xs font-medium text-muted"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>Cancel</button></div><div class="grid grid-cols-1 gap-3 sm:grid-cols-2"><label class="space-y-1 text-xs font-medium text-muted">Branch code<input${ssrRenderAttr("value", unref(form).code)} required minlength="2" maxlength="32" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="PP-01"></label><label class="space-y-1 text-xs font-medium text-muted">Branch name<input${ssrRenderAttr("value", unref(form).name)} required minlength="2" maxlength="128" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="Phnom Penh HQ"></label><label class="space-y-1 text-xs font-medium text-muted">Address<input${ssrRenderAttr("value", unref(form).address)} maxlength="512" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy"></label><label class="space-y-1 text-xs font-medium text-muted">Phone<input${ssrRenderAttr("value", unref(form).phone)} maxlength="32" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy"></label><label class="space-y-1 text-xs font-medium text-muted sm:col-span-2">Email<input${ssrRenderAttr("value", unref(form).email)} type="email" maxlength="255" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy"></label></div>`);
        if (unref(formError)) {
          _push(`<p class="text-sm text-expense" role="alert">${ssrInterpolate(unref(formError))}</p>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<button type="submit"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""} class="rounded-lg bg-primary px-4 py-2 text-sm text-white disabled:opacity-60">${ssrInterpolate(unref(saving) ? "Saving\u2026" : unref(editing) ? "Save changes" : "Create branch")}</button></form>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(notice)) {
        _push(`<p class="rounded-lg bg-green-50 px-3 py-2 text-sm text-income" role="status">${ssrInterpolate(unref(notice))}</p>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(loading)) {
        _push(`<div class="text-sm text-muted">Loading\u2026</div>`);
      } else if (unref(err)) {
        _push(`<div class="text-sm text-expense">${ssrInterpolate(unref(err))}</div>`);
      } else {
        _push(`<ul class="divide-y divide-border rounded-xl border border-border bg-white"><!--[-->`);
        ssrRenderList(unref(items), (branch) => {
          _push(`<li class="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between"><div class="min-w-0"><div class="flex flex-wrap items-center gap-2"><h2 class="text-sm font-semibold text-navy">${ssrInterpolate(branch.name)}</h2><span class="${ssrRenderClass([branch.is_active ? "bg-green-50 text-income" : "bg-slate-100 text-muted", "rounded-full px-2 py-1 text-[10px] font-semibold uppercase"])}">${ssrInterpolate(branch.is_active ? "Active" : "Inactive")}</span></div><p class="mt-1 font-mono text-xs text-muted">${ssrInterpolate(branch.code)}</p>`);
          if (branch.address || branch.phone || branch.email) {
            _push(`<p class="mt-1 break-words text-xs text-muted">`);
            if (branch.address) {
              _push(`<span>${ssrInterpolate(branch.address)}</span>`);
            } else {
              _push(`<!---->`);
            }
            if (branch.phone) {
              _push(`<span> \xB7 ${ssrInterpolate(branch.phone)}</span>`);
            } else {
              _push(`<!---->`);
            }
            if (branch.email) {
              _push(`<span> \xB7 ${ssrInterpolate(branch.email)}</span>`);
            } else {
              _push(`<!---->`);
            }
            _push(`</p>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</div>`);
          if (unref(canEdit) || unref(canManage)) {
            _push(`<div class="flex flex-wrap gap-2 sm:shrink-0 sm:justify-end">`);
            if (unref(canEdit)) {
              _push(`<button type="button" class="rounded-md border border-border px-3 py-1.5 text-xs font-medium text-navy hover:bg-canvas">Edit</button>`);
            } else {
              _push(`<!---->`);
            }
            if (unref(canManage) && branch.is_active) {
              _push(`<button type="button" class="rounded-md border border-red-100 px-3 py-1.5 text-xs font-medium text-expense hover:bg-red-50">Delete</button>`);
            } else if (unref(canManage)) {
              _push(`<button type="button" class="rounded-md border border-green-100 px-3 py-1.5 text-xs font-medium text-income hover:bg-green-50"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>Restore</button>`);
            } else {
              _push(`<!---->`);
            }
            _push(`</div>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</li>`);
        });
        _push(`<!--]-->`);
        if (!unref(items).length) {
          _push(`<li class="p-6 text-center text-sm text-muted">No branches</li>`);
        } else {
          _push(`<!---->`);
        }
        _push(`</ul>`);
      }
      if (unref(archiveTarget)) {
        _push(`<div class="fixed inset-0 z-[100] flex items-end justify-center bg-slate-950/55 p-0 backdrop-blur-sm sm:items-center sm:p-4"><section class="w-full max-w-md rounded-t-2xl border border-border bg-white p-5 shadow-2xl sm:rounded-2xl" role="dialog" aria-modal="true" aria-labelledby="archive-branch-title"><h2 id="archive-branch-title" class="text-base font-bold text-navy">Delete ${ssrInterpolate(unref(archiveTarget).name)}?</h2><p class="mt-2 text-sm text-muted">This archives the branch and keeps its transaction history. You can restore it later.</p>`);
        if (unref(archiveError)) {
          _push(`<p class="mt-3 text-xs text-expense" role="alert">${ssrInterpolate(unref(archiveError))}</p>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<footer class="mt-5 flex flex-col-reverse gap-2 border-t border-border pt-3 sm:flex-row sm:justify-end"><button type="button" class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>Cancel</button><button type="button" class="rounded-lg bg-expense px-4 py-2 text-xs font-semibold text-white disabled:opacity-60"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>${ssrInterpolate(unref(saving) ? "Deleting\u2026" : "Delete branch")}</button></footer></section></div>`);
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
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/admin/branches.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};

export { _sfc_main as default };
//# sourceMappingURL=branches-uzytKBXM.mjs.map
