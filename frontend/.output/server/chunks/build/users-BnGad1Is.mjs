import { _ as __nuxt_component_1 } from './SearchSelect-BFyTiPOW.mjs';
import { defineComponent, computed, ref, reactive, mergeProps, unref, useSSRContext } from 'vue';
import { ssrRenderAttrs, ssrInterpolate, ssrRenderAttr, ssrRenderComponent, ssrIncludeBooleanAttr, ssrRenderList, ssrRenderClass } from 'vue/server-renderer';
import { h as useApi, f as useAuth, d as usePermissions } from './server.mjs';
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
  __name: "users",
  __ssrInlineRender: true,
  setup(__props) {
    useApi();
    const { user: currentUser } = useAuth();
    const { can, canCreateUser } = usePermissions();
    const canCreate = canCreateUser;
    const canEditUser = computed(() => can("users.edit", "users.manage"));
    const canManageUsers = computed(() => can("users.manage"));
    const currentUserId = computed(() => {
      var _a;
      return ((_a = currentUser.value) == null ? void 0 : _a.id) || "";
    });
    const items = ref([]);
    const roles = ref([]);
    const branches = ref([]);
    const loading = ref(true);
    const error = ref("");
    const showCreate = ref(false);
    const saving = ref(false);
    const createError = ref("");
    const editError = ref("");
    const notice = ref("");
    const editingUser = ref(null);
    const passwordTarget = ref(null);
    const passwordError = ref("");
    const expandedPerms = ref({});
    const createForm = reactive({ username: "", email: "", full_name: "", password: "", role_code: "", branch_id: "" });
    const editForm = reactive({ email: "", full_name: "", role_codes: [], branch_ids: [] });
    const passwordForm = reactive({ password: "", confirm: "" });
    const assignableRoles = computed(() => roles.value.filter((role) => {
      var _a;
      return role.code !== "super_admin" || ((_a = currentUser.value) == null ? void 0 : _a.is_super_admin);
    }));
    const roleOptions = computed(() => assignableRoles.value.map((role) => ({ value: role.code, label: role.name })));
    const branchOptions = computed(() => branches.value.map((branch) => ({ value: branch.id, label: `${branch.code} \u2014 ${branch.name}` })));
    const isEditingSelf = computed(() => {
      var _a;
      return ((_a = editingUser.value) == null ? void 0 : _a.id) === currentUserId.value;
    });
    function permCount(item) {
      var _a;
      const codes = (_a = item.permission_codes) != null ? _a : item.roles.flatMap((role) => {
        var _a2;
        return (_a2 = role.permission_codes) != null ? _a2 : [];
      });
      return new Set(codes).size;
    }
    return (_ctx, _push, _parent, _attrs) => {
      const _component_SearchSelect = __nuxt_component_1;
      _push(`<section${ssrRenderAttrs(mergeProps({ class: "space-y-4" }, _attrs))}><header class="flex items-center justify-between gap-3"><div><p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">ACCESS CONTROL</p><h1 class="text-xl font-bold text-navy">Users</h1></div>`);
      if (unref(canCreate)) {
        _push(`<button class="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white hover:bg-primary-hover">${ssrInterpolate(unref(showCreate) ? "Cancel" : "+ User")}</button>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</header>`);
      if (unref(showCreate)) {
        _push(`<form class="space-y-3 rounded-xl border border-border bg-white p-4"><h2 class="text-sm font-semibold text-navy">Create user</h2><div class="grid grid-cols-1 gap-3 sm:grid-cols-2"><label class="space-y-1 text-xs font-medium text-muted">Username<input${ssrRenderAttr("value", unref(createForm).username)} required minlength="3" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy"></label><label class="space-y-1 text-xs font-medium text-muted">Email<input${ssrRenderAttr("value", unref(createForm).email)} type="email" required class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy"></label><label class="space-y-1 text-xs font-medium text-muted">Full name<input${ssrRenderAttr("value", unref(createForm).full_name)} required minlength="2" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy"></label><label class="space-y-1 text-xs font-medium text-muted">Temporary 4-digit PIN<input${ssrRenderAttr("value", unref(createForm).password)} type="password" inputmode="numeric" pattern="[0-9]{4}" maxlength="4" minlength="4" required autocomplete="new-password" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="4 digits"></label><div class="space-y-1 text-xs font-medium text-muted"><span class="block">Role</span>`);
        _push(ssrRenderComponent(_component_SearchSelect, {
          modelValue: unref(createForm).role_code,
          "onUpdate:modelValue": ($event) => unref(createForm).role_code = $event,
          options: unref(roleOptions),
          placeholder: "Select a role",
          "search-placeholder": "Search roles\u2026",
          "aria-label": "User role"
        }, null, _parent));
        _push(`</div><div class="space-y-1 text-xs font-medium text-muted"><span class="block">Branch</span>`);
        _push(ssrRenderComponent(_component_SearchSelect, {
          modelValue: unref(createForm).branch_id,
          "onUpdate:modelValue": ($event) => unref(createForm).branch_id = $event,
          options: unref(branchOptions),
          placeholder: "Select a branch",
          "search-placeholder": "Search branches\u2026",
          "aria-label": "User branch"
        }, null, _parent));
        _push(`</div></div>`);
        if (unref(createError)) {
          _push(`<p class="text-sm text-expense">${ssrInterpolate(unref(createError))}</p>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<button type="submit"${ssrIncludeBooleanAttr(unref(saving) || !unref(createForm).role_code || !unref(createForm).branch_id) ? " disabled" : ""} class="rounded-lg bg-primary px-4 py-2 text-sm text-white disabled:opacity-60">${ssrInterpolate(unref(saving) ? "Creating\u2026" : "Create user")}</button></form>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(editingUser)) {
        _push(`<form class="space-y-3 rounded-xl border border-blue-100 bg-white p-4 shadow-sm"><div class="flex items-start justify-between gap-3"><div><h2 class="text-sm font-semibold text-navy">Edit ${ssrInterpolate(unref(editingUser).username)}</h2><p class="mt-1 text-xs text-muted">Update profile and access details.</p></div><button type="button" class="text-xs font-medium text-muted hover:text-navy">Cancel</button></div><div class="grid grid-cols-1 gap-3 sm:grid-cols-2"><label class="space-y-1 text-xs font-medium text-muted">Full name<input${ssrRenderAttr("value", unref(editForm).full_name)} required minlength="2" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy"></label><label class="space-y-1 text-xs font-medium text-muted">Email<input${ssrRenderAttr("value", unref(editForm).email)} type="email" required class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy"></label>`);
        if (unref(canManageUsers) && !unref(isEditingSelf)) {
          _push(`<div class="space-y-1 text-xs font-medium text-muted"><span class="block">Roles <small>(select one or more)</small></span>`);
          _push(ssrRenderComponent(_component_SearchSelect, {
            modelValue: unref(editForm).role_codes,
            "onUpdate:modelValue": ($event) => unref(editForm).role_codes = $event,
            options: unref(roleOptions),
            multiple: "",
            placeholder: "Select roles",
            "search-placeholder": "Search roles\u2026",
            "aria-label": "User roles"
          }, null, _parent));
          _push(`</div>`);
        } else {
          _push(`<!---->`);
        }
        if (unref(canManageUsers) && !unref(isEditingSelf) && unref(branches).length) {
          _push(`<div class="space-y-1 text-xs font-medium text-muted"><span class="block">Branches <small>(select one or more)</small></span>`);
          _push(ssrRenderComponent(_component_SearchSelect, {
            modelValue: unref(editForm).branch_ids,
            "onUpdate:modelValue": ($event) => unref(editForm).branch_ids = $event,
            options: unref(branchOptions),
            multiple: "",
            placeholder: "Select branches",
            "search-placeholder": "Search branches\u2026",
            "aria-label": "User branches"
          }, null, _parent));
          _push(`</div>`);
        } else {
          _push(`<!---->`);
        }
        _push(`</div>`);
        if (unref(editError)) {
          _push(`<p class="text-sm text-expense">${ssrInterpolate(unref(editError))}</p>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<button type="submit"${ssrIncludeBooleanAttr(unref(saving) || unref(canManageUsers) && !unref(isEditingSelf) && (!unref(editForm).role_codes.length || unref(branches).length > 0 && !unref(editForm).branch_ids.length)) ? " disabled" : ""} class="rounded-lg bg-primary px-4 py-2 text-sm text-white disabled:opacity-60">${ssrInterpolate(unref(saving) ? "Saving\u2026" : "Save changes")}</button></form>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(notice)) {
        _push(`<p class="rounded-lg bg-green-50 px-3 py-2 text-sm text-income">${ssrInterpolate(unref(notice))}</p>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(loading)) {
        _push(`<div class="text-sm text-muted">Loading\u2026</div>`);
      } else if (unref(error)) {
        _push(`<div class="text-sm text-expense">${ssrInterpolate(unref(error))}</div>`);
      } else {
        _push(`<ul class="divide-y divide-border rounded-xl border border-border bg-white"><!--[-->`);
        ssrRenderList(unref(items), (item) => {
          _push(`<li class="p-4"><div class="flex flex-col gap-3 sm:flex-row sm:justify-between"><div class="min-w-0"><div class="text-sm font-semibold text-navy">${ssrInterpolate(item.full_name)}</div><div class="text-xs text-muted">@${ssrInterpolate(item.username)} \xB7 ${ssrInterpolate(item.email)}</div><div class="mt-2 flex flex-wrap gap-1.5"><!--[-->`);
          ssrRenderList(item.roles || [], (role) => {
            _push(`<span class="rounded-full bg-blue-50 px-2 py-1 text-[10px] font-medium text-primary">${ssrInterpolate(role.name)}</span>`);
          });
          _push(`<!--]-->`);
          if (!(item.roles || []).length) {
            _push(`<span class="text-xs text-muted">No role assigned</span>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</div><p class="mt-2 text-xs text-muted">Branches: ${ssrInterpolate((item.branches || []).map((branch) => branch.code).join(", ") || "\u2014")}</p><div class="mt-2"><button type="button" class="text-[10px] font-semibold text-primary">${ssrInterpolate(unref(expandedPerms)[item.id] ? "Hide permissions" : `Permissions (${permCount(item)})`)}</button>`);
          if (unref(expandedPerms)[item.id]) {
            _push(`<div class="mt-1.5 flex flex-wrap gap-1">`);
            if (item.is_super_admin) {
              _push(`<span class="rounded border border-primary/20 bg-primary/10 px-1.5 py-0.5 text-[10px] text-primary">All permissions</span>`);
            } else {
              _push(`<!---->`);
            }
            _push(`<!--[-->`);
            ssrRenderList(item.permission_codes || [], (code) => {
              _push(`<span class="rounded border border-border bg-canvas px-1.5 py-0.5 font-mono text-[10px] text-navy">${ssrInterpolate(code)}</span>`);
            });
            _push(`<!--]-->`);
            if (!item.is_super_admin && !(item.permission_codes || []).length) {
              _push(`<span class="text-[10px] text-muted">No permissions</span>`);
            } else {
              _push(`<!---->`);
            }
            _push(`</div>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</div></div><div class="flex shrink-0 flex-row flex-wrap items-center gap-2 sm:flex-col sm:items-end"><span class="${ssrRenderClass([item.is_active ? "bg-green-50 text-income" : "bg-slate-100 text-muted", "rounded-full px-2 py-1 text-[10px] font-semibold uppercase"])}">${ssrInterpolate(item.is_active ? "Active" : "Inactive")}</span>`);
          if (item.is_super_admin) {
            _push(`<span class="text-[10px] font-medium text-primary">Super Admin</span>`);
          } else {
            _push(`<!---->`);
          }
          _push(`<div class="flex flex-wrap justify-start gap-1.5 sm:justify-end">`);
          if (unref(canEditUser)) {
            _push(`<button class="rounded-md border border-border px-2 py-1 text-[10px] font-medium text-navy hover:bg-canvas">Edit</button>`);
          } else {
            _push(`<!---->`);
          }
          if (unref(canManageUsers) && item.id !== unref(currentUserId)) {
            _push(`<button class="rounded-md border border-border px-2 py-1 text-[10px] font-medium text-primary hover:bg-blue-50">Change PIN</button>`);
          } else {
            _push(`<!---->`);
          }
          if (unref(canManageUsers) && item.id !== unref(currentUserId)) {
            _push(`<button class="${ssrRenderClass([item.is_active ? "border-red-100 text-expense hover:bg-red-50" : "border-green-100 text-income hover:bg-green-50", "rounded-md border px-2 py-1 text-[10px] font-medium"])}"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>${ssrInterpolate(item.is_active ? "Deactivate" : "Activate")}</button>`);
          } else {
            _push(`<!---->`);
          }
          _push(`</div></div></div></li>`);
        });
        _push(`<!--]-->`);
        if (!unref(items).length) {
          _push(`<li class="p-6 text-center text-sm text-muted">No users</li>`);
        } else {
          _push(`<!---->`);
        }
        _push(`</ul>`);
      }
      if (unref(passwordTarget)) {
        _push(`<div class="fixed inset-0 z-[100] flex items-end justify-center bg-slate-950/55 p-0 backdrop-blur-sm sm:items-center sm:p-4"><section class="w-full max-w-md rounded-t-2xl border border-border bg-white p-5 shadow-2xl sm:rounded-2xl" role="dialog" aria-modal="true" aria-labelledby="change-user-pin-title"><header class="mb-4 flex items-start justify-between gap-3"><div><h2 id="change-user-pin-title" class="text-base font-bold text-navy">Change ${ssrInterpolate(unref(passwordTarget).full_name)}\u2019s PIN</h2><p class="mt-1 text-xs text-muted">Enter a new 4-digit numeric PIN. Their active sessions will be revoked.</p></div><button type="button" class="text-xs text-muted"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>Close</button></header><form class="space-y-3"><label class="block space-y-1 text-xs font-medium text-muted">New PIN<input${ssrRenderAttr("value", unref(passwordForm).password)} type="password" inputmode="numeric" pattern="[0-9]{4}" maxlength="4" minlength="4" required autocomplete="new-password" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="4 digits"></label><label class="block space-y-1 text-xs font-medium text-muted">Confirm PIN<input${ssrRenderAttr("value", unref(passwordForm).confirm)} type="password" inputmode="numeric" pattern="[0-9]{4}" maxlength="4" minlength="4" required autocomplete="new-password" class="w-full rounded-lg border border-border px-3 py-2 text-sm text-navy" placeholder="Enter the PIN again"></label>`);
        if (unref(passwordError)) {
          _push(`<p class="rounded-lg bg-red-50 px-3 py-2 text-xs text-expense" role="alert">${ssrInterpolate(unref(passwordError))}</p>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<footer class="flex flex-col-reverse gap-2 border-t border-border pt-3 sm:flex-row sm:justify-end"><button type="button" class="rounded-lg border border-border px-4 py-2 text-xs font-semibold text-muted"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>Cancel</button><button type="submit" class="rounded-lg bg-primary px-4 py-2 text-xs font-semibold text-white disabled:opacity-60"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""}>${ssrInterpolate(unref(saving) ? "Saving\u2026" : "Update PIN")}</button></footer></form></section></div>`);
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
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/admin/users.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};

export { _sfc_main as default };
//# sourceMappingURL=users-BnGad1Is.mjs.map
