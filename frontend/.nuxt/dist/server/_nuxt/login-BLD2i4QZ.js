import { defineComponent, ref, unref, useSSRContext } from "vue";
import { ssrRenderAttrs, ssrRenderAttr, ssrInterpolate, ssrIncludeBooleanAttr } from "vue/server-renderer";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/hookable/dist/index.mjs";
import { f as useAuth, d as usePermissions } from "../server.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/ofetch/dist/node.mjs";
import "#internal/nuxt/paths";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/unctx/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/h3/dist/index.mjs";
import "vue-router";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/defu/dist/defu.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/ufo/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/klona/dist/index.mjs";
import "@vuetify/v0/utilities";
const _sfc_main = /* @__PURE__ */ defineComponent({
  __name: "login",
  __ssrInlineRender: true,
  setup(__props) {
    useAuth();
    usePermissions();
    const identifier = ref("");
    const password = ref("");
    const error = ref("");
    const loading = ref(false);
    const showDemoCredentials = false;
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<div${ssrRenderAttrs(_attrs)}><div class="login-heading"><p class="login-eyebrow">WELCOME BACK</p><h2>Sign in to your workspace</h2><p>Use your account credentials to continue.</p></div><form class="bg-white rounded-2xl shadow-xl p-6 space-y-4"><div><label class="block text-sm font-medium text-navy mb-1">Username or email</label><input${ssrRenderAttr("value", unref(identifier))} type="text" autocomplete="username" required class="w-full rounded-lg border border-border px-3 py-2.5 text-base focus:outline-none focus:ring-2 focus:ring-primary" placeholder="manager1"></div><div><label class="block text-sm font-medium text-navy mb-1">Password</label><input${ssrRenderAttr("value", unref(password))} type="password" inputmode="numeric" autocomplete="current-password" required class="w-full rounded-lg border border-border px-3 py-2.5 text-base focus:outline-none focus:ring-2 focus:ring-primary"></div>`);
      if (unref(showDemoCredentials)) {
        _push(`<p class="text-center text-xs text-muted"> Demo login: <span class="font-medium text-navy">manager1</span> / PIN <span class="font-medium text-navy">2468</span></p>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(error)) {
        _push(`<p class="text-sm text-expense">${ssrInterpolate(unref(error))}</p>`);
      } else {
        _push(`<!---->`);
      }
      _push(`<button type="submit"${ssrIncludeBooleanAttr(unref(loading)) ? " disabled" : ""} class="w-full rounded-lg bg-primary hover:bg-primary-hover text-white font-medium py-2.5 transition disabled:opacity-60">${ssrInterpolate(unref(loading) ? "Signing in…" : "Sign in")}</button></form></div>`);
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/login.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
export {
  _sfc_main as default
};
//# sourceMappingURL=login-BLD2i4QZ.js.map
