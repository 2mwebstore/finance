import { defineComponent, computed, ref, reactive, mergeProps, unref, useSSRContext } from "vue";
import { ssrRenderAttrs, ssrInterpolate, ssrRenderList, ssrRenderComponent, ssrRenderAttr, ssrIncludeBooleanAttr, ssrLooseContain } from "vue/server-renderer";
import { Pencil, Trash2, X } from "lucide-vue-next";
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
  __name: "roles",
  __ssrInlineRender: true,
  setup(__props) {
    useApi();
    const { can } = usePermissions();
    const canManage = computed(() => can("roles.manage"));
    const roles = ref([]);
    const permissions = ref([]);
    const loading = ref(true);
    const saving = ref(false);
    const error = ref("");
    const formError = ref("");
    const deleteError = ref("");
    const notice = ref("");
    const modalOpen = ref(false);
    const formMode = ref("create");
    const editingRole = ref(null);
    const deleteTarget = ref(null);
    const form = reactive({ code: "", name: "", description: "", permission_codes: [] });
    const permissionGroups = computed(() => {
      const groups = /* @__PURE__ */ new Map();
      for (const permission of permissions.value) {
        const module = permission.module || "General";
        groups.set(module, [...groups.get(module) || [], permission]);
      }
      return [...groups.entries()].map(([module, items]) => ({ module, items }));
    });
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<section${ssrRenderAttrs(mergeProps({ class: "space-y-4" }, _attrs))}><header class="flex items-center justify-between gap-3"><div><p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">ACCESS CONTROL</p><h1 class="text-xl font-bold text-navy">Roles &amp; permissions</h1><p class="mt-1 text-xs text-muted">Create roles and choose what each role can access.</p></div>`);
      if (unref(canManage)) {
        _push(`<button class="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover">+ Role</button>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</header>`);
      if (unref(notice)) {
        _push(`<p class="rounded-lg bg-green-50 px-3 py-2 text-sm text-income">${ssrInterpolate(unref(notice))}</p>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(loading)) {
        _push(`<div class="text-sm text-muted">Loading roles…</div>`);
      } else if (unref(error)) {
        _push(`<div class="rounded-lg bg-red-50 px-3 py-3 text-sm text-expense">${ssrInterpolate(unref(error))}</div>`);
      } else {
        _push(`<div class="space-y-3"><!--[-->`);
        ssrRenderList(unref(roles), (role) => {
          _push(`<article class="rounded-xl border border-border bg-white p-4"><div class="flex items-start justify-between gap-3"><div class="min-w-0"><div class="flex flex-wrap items-center gap-2"><h2 class="text-sm font-semibold text-navy">${ssrInterpolate(role.name)}</h2>`);
          if (role.is_system) {
            _push(`<span class="rounded-full bg-slate-100 px-2 py-0.5 text-[9px] font-bold uppercase text-muted">System</span>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</div><p class="mt-1 font-mono text-[10px] text-muted">${ssrInterpolate(role.code)}</p>`);
          if (role.description) {
            _push(`<p class="mt-1 text-xs text-muted">${ssrInterpolate(role.description)}</p>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</div>`);
          if (unref(canManage)) {
            _push(`<div class="flex shrink-0 gap-1.5"><button class="rounded-md border border-border px-2.5 py-1.5 text-[10px] font-medium text-navy hover:bg-canvas">`);
            _push(ssrRenderComponent(unref(Pencil), {
              size: 12,
              class: "mr-1 inline"
            }, null, _parent));
            _push(`Edit</button>`);
            if (!role.is_system) {
              _push(`<button class="rounded-md border border-red-100 px-2.5 py-1.5 text-[10px] font-medium text-expense hover:bg-red-50">`);
              _push(ssrRenderComponent(unref(Trash2), {
                size: 12,
                class: "mr-1 inline"
              }, null, _parent));
              _push(`Delete</button>`);
            } else {
              _push(`<!---->`);
            }
            _push(`</div>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</div>`);
          if (role.permissions?.length) {
            _push(`<div class="mt-3 flex flex-wrap gap-1.5"><!--[-->`);
            ssrRenderList(role.permissions, (permission) => {
              _push(`<span class="rounded-md border border-border bg-canvas px-2 py-1 font-mono text-[9px] text-navy">${ssrInterpolate(permission.code || permission.name)}</span>`);
            });
            _push(`<!--]--></div>`);
          } else {
            _push(`<p class="mt-3 text-xs text-muted">No permissions assigned.</p>`);
          }
          _push(`</article>`);
        });
        _push(`<!--]-->`);
        if (!unref(roles).length) {
          _push(`<div class="rounded-xl border border-border bg-white p-8 text-center text-sm text-muted">No roles found.</div>`);
        } else {
          _push(`<!---->`);
        }
        _push(`</div>`);
      }
      if (unref(modalOpen)) {
        _push(`<div class="fixed inset-0 z-[90] flex items-end justify-center bg-slate-950/50 p-0 backdrop-blur-sm sm:items-center sm:p-5"><section class="max-h-[94vh] w-full max-w-3xl overflow-y-auto rounded-t-2xl border border-border bg-white shadow-2xl sm:rounded-2xl" role="dialog" aria-modal="true"${ssrRenderAttr("aria-label", unref(formMode) === "create" ? "Create role" : "Edit role")}><header class="sticky top-0 z-10 flex items-center justify-between border-b border-border bg-white px-5 py-4"><div><p class="text-[9px] font-extrabold tracking-[0.14em] text-slate-400">${ssrInterpolate(unref(formMode) === "create" ? "NEW ROLE" : "UPDATE ROLE")}</p><h2 class="mt-1 text-base font-bold text-navy">${ssrInterpolate(unref(formMode) === "create" ? "Create role" : `Edit ${unref(editingRole)?.name}`)}</h2></div><button class="grid h-8 w-8 place-items-center rounded-lg bg-canvas text-muted hover:text-navy" aria-label="Close">`);
        _push(ssrRenderComponent(unref(X), { size: 17 }, null, _parent));
        _push(`</button></header><form class="space-y-4 p-5"><div class="grid grid-cols-1 gap-3 sm:grid-cols-2"><label class="space-y-1.5 text-xs font-semibold text-navy">Name<input${ssrRenderAttr("value", unref(form).name)} required minlength="2" maxlength="128" class="w-full rounded-lg border border-border px-3 py-2.5 text-sm font-normal"></label>`);
        if (unref(formMode) === "create") {
          _push(`<label class="space-y-1.5 text-xs font-semibold text-navy">Code<input${ssrRenderAttr("value", unref(form).code)} required minlength="2" maxlength="64" class="w-full rounded-lg border border-border px-3 py-2.5 font-mono text-sm font-normal" placeholder="e.g. store-manager"></label>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<label class="space-y-1.5 text-xs font-semibold text-navy sm:col-span-2">Description<textarea rows="2" maxlength="512" class="w-full rounded-lg border border-border px-3 py-2.5 text-sm font-normal">${ssrInterpolate(unref(form).description)}</textarea></label></div><div><h3 class="mb-2 text-xs font-semibold text-navy">Permissions <span class="font-normal text-muted">${ssrInterpolate(unref(form).permission_codes.length)} selected</span></h3>`);
        if (!unref(permissionGroups).length) {
          _push(`<div class="rounded-lg bg-canvas p-3 text-xs text-muted">No permission catalog is available.</div>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<div class="grid grid-cols-1 gap-3 md:grid-cols-2"><!--[-->`);
        ssrRenderList(unref(permissionGroups), (group) => {
          _push(`<fieldset class="rounded-lg border border-border p-3"><legend class="px-1 text-xs font-semibold text-navy">${ssrInterpolate(group.module)}</legend><!--[-->`);
          ssrRenderList(group.items, (permission) => {
            _push(`<label class="flex cursor-pointer items-start gap-2 py-1.5 text-xs text-navy"><input${ssrIncludeBooleanAttr(Array.isArray(unref(form).permission_codes) ? ssrLooseContain(unref(form).permission_codes, permission.code) : unref(form).permission_codes) ? " checked" : ""} type="checkbox"${ssrRenderAttr("value", permission.code)} class="mt-0.5 accent-primary"><span><span class="block font-medium">${ssrInterpolate(permission.name)}</span><span class="font-mono text-[9px] text-muted">${ssrInterpolate(permission.code)}</span></span></label>`);
          });
          _push(`<!--]--></fieldset>`);
        });
        _push(`<!--]--></div></div>`);
        if (unref(formError)) {
          _push(`<p class="rounded-lg bg-red-50 px-3 py-2 text-xs text-expense" role="alert">${ssrInterpolate(unref(formError))}</p>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<footer class="flex justify-end gap-2 border-t border-border pt-4"><button type="button" class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>Cancel</button><button type="submit" class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-white disabled:opacity-60"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>${ssrInterpolate(unref(saving) ? "Saving…" : "Save role")}</button></footer></form></section></div>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(deleteTarget)) {
        _push(`<div class="fixed inset-0 z-[100] flex items-center justify-center bg-slate-950/55 p-4 backdrop-blur-sm"><section class="w-full max-w-sm rounded-2xl border border-border bg-white p-5 shadow-2xl" role="alertdialog" aria-modal="true" aria-labelledby="delete-role-title"><h2 id="delete-role-title" class="text-base font-bold text-navy">Delete ${ssrInterpolate(unref(deleteTarget).name)}?</h2><p class="mt-2 text-sm text-muted">This role can be deleted only after it is removed from every user.</p>`);
        if (unref(deleteError)) {
          _push(`<p class="mt-3 rounded-lg bg-red-50 px-3 py-2 text-xs text-expense">${ssrInterpolate(unref(deleteError))}</p>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<footer class="mt-5 flex justify-end gap-2"><button class="rounded-lg border border-border px-3 py-2 text-xs font-semibold text-muted"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>Cancel</button><button class="rounded-lg bg-expense px-3 py-2 text-xs font-semibold text-white disabled:opacity-60"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>${ssrInterpolate(unref(saving) ? "Deleting…" : "Delete role")}</button></footer></section></div>`);
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
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/admin/roles.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
export {
  _sfc_main as default
};
//# sourceMappingURL=roles-D_XDrUlH.js.map
