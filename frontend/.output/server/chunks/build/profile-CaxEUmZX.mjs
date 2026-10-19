import { defineComponent, reactive, ref, mergeProps, unref, useSSRContext } from 'vue';
import { ssrRenderAttrs, ssrInterpolate, ssrRenderAttr, ssrIncludeBooleanAttr } from 'vue/server-renderer';
import { h as useApi, f as useAuth } from './server.mjs';
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
  __name: "profile",
  __ssrInlineRender: true,
  setup(__props) {
    useApi();
    const { user } = useAuth();
    const form = reactive({ current_password: "", new_password: "" });
    const confirmation = ref("");
    const saving = ref(false);
    const error = ref("");
    const notice = ref("");
    return (_ctx, _push, _parent, _attrs) => {
      var _a, _b, _c, _d;
      _push(`<section${ssrRenderAttrs(mergeProps({ class: "mx-auto max-w-2xl space-y-5" }, _attrs))}><header><p class="text-[10px] font-bold tracking-[0.14em] text-slate-400">ACCOUNT</p><h1 class="text-xl font-bold text-navy">Your profile</h1><p class="mt-1 text-sm text-muted">Manage your account details and sign-in PIN.</p></header><article class="rounded-xl border border-border bg-white p-4 sm:p-5"><h2 class="text-sm font-semibold text-navy">Account details</h2><dl class="mt-3 grid grid-cols-1 gap-3 text-sm sm:grid-cols-2"><div><dt class="text-xs text-muted">Full name</dt><dd class="mt-1 font-medium text-navy">${ssrInterpolate(((_a = unref(user)) == null ? void 0 : _a.full_name) || "\u2014")}</dd></div><div><dt class="text-xs text-muted">Username</dt><dd class="mt-1 font-medium text-navy">${ssrInterpolate(((_b = unref(user)) == null ? void 0 : _b.username) || "\u2014")}</dd></div><div><dt class="text-xs text-muted">Email</dt><dd class="mt-1 break-all font-medium text-navy">${ssrInterpolate(((_c = unref(user)) == null ? void 0 : _c.email) || "\u2014")}</dd></div><div><dt class="text-xs text-muted">Roles</dt><dd class="mt-1 font-medium text-navy">${ssrInterpolate((((_d = unref(user)) == null ? void 0 : _d.roles) || []).join(", ") || "\u2014")}</dd></div></dl></article><form class="space-y-4 rounded-xl border border-border bg-white p-4 sm:p-5"><div><h2 class="text-sm font-semibold text-navy">Change password</h2><p class="mt-1 text-xs text-muted">Your new password must be exactly 4 digits.</p></div><label class="block space-y-1 text-xs font-medium text-muted">Current password<input${ssrRenderAttr("value", unref(form).current_password)} type="password" required autocomplete="current-password" class="w-full rounded-lg border border-border px-3 py-2.5 text-sm text-navy"></label><div class="grid grid-cols-1 gap-3 sm:grid-cols-2"><label class="block space-y-1 text-xs font-medium text-muted">New 4-digit PIN<input${ssrRenderAttr("value", unref(form).new_password)} type="password" inputmode="numeric" pattern="[0-9]{4}" maxlength="4" minlength="4" required autocomplete="new-password" class="w-full rounded-lg border border-border px-3 py-2.5 text-sm text-navy" placeholder="4 digits"></label><label class="block space-y-1 text-xs font-medium text-muted">Confirm new PIN<input${ssrRenderAttr("value", unref(confirmation))} type="password" inputmode="numeric" pattern="[0-9]{4}" maxlength="4" minlength="4" required autocomplete="new-password" class="w-full rounded-lg border border-border px-3 py-2.5 text-sm text-navy" placeholder="Enter the PIN again"></label></div>`);
      if (unref(error)) {
        _push(`<p class="rounded-lg bg-red-50 px-3 py-2 text-xs text-expense" role="alert">${ssrInterpolate(unref(error))}</p>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(notice)) {
        _push(`<p class="rounded-lg bg-green-50 px-3 py-2 text-xs text-income" role="status">${ssrInterpolate(unref(notice))}</p>`);
      } else {
        _push(`<!---->`);
      }
      _push(`<button type="submit"${ssrIncludeBooleanAttr(unref(saving)) ? " disabled" : ""} class="w-full rounded-lg bg-primary px-4 py-2.5 text-sm font-semibold text-white disabled:opacity-60 sm:w-auto">${ssrInterpolate(unref(saving) ? "Updating\u2026" : "Update password")}</button></form></section>`);
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/profile.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};

export { _sfc_main as default };
//# sourceMappingURL=profile-CaxEUmZX.mjs.map
