import { defineComponent, mergeProps, unref, useSSRContext } from 'vue';
import { ssrRenderAttrs, ssrRenderComponent, ssrRenderSlot } from 'vue/server-renderer';
import { ChartNoAxesCombined, Check } from 'lucide-vue-next';
import { _ as _export_sfc } from './server.mjs';
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
  __name: "auth",
  __ssrInlineRender: true,
  setup(__props) {
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<div${ssrRenderAttrs(mergeProps({ class: "auth-shell" }, _attrs))} data-v-f512c5f1><div class="auth-glow auth-glow-one" data-v-f512c5f1></div><div class="auth-glow auth-glow-two" data-v-f512c5f1></div><div class="auth-layout" data-v-f512c5f1><div class="auth-intro" data-v-f512c5f1><div class="auth-brand" data-v-f512c5f1><span class="auth-brand-mark" data-v-f512c5f1>`);
      _push(ssrRenderComponent(unref(ChartNoAxesCombined), { size: 26 }, null, _parent));
      _push(`</span><span data-v-f512c5f1>Income &amp; Expense</span></div><p class="auth-eyebrow" data-v-f512c5f1>BUSINESS FINANCE</p><h1 data-v-f512c5f1>Make every business decision with clarity.</h1><p class="auth-copy" data-v-f512c5f1>Track income, expenses, branches, and reports in one simple workspace.</p><div class="auth-points" data-v-f512c5f1><span data-v-f512c5f1>`);
      _push(ssrRenderComponent(unref(Check), { size: 15 }, null, _parent));
      _push(` Real-time overview</span><span data-v-f512c5f1>`);
      _push(ssrRenderComponent(unref(Check), { size: 15 }, null, _parent));
      _push(` Secure access control</span></div></div><div class="auth-card" data-v-f512c5f1>`);
      ssrRenderSlot(_ctx.$slots, "default", {}, null, _push, _parent);
      _push(`</div></div></div>`);
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("layouts/auth.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
const auth = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-f512c5f1"]]);

export { auth as default };
//# sourceMappingURL=auth-BAYWvNQB.mjs.map
