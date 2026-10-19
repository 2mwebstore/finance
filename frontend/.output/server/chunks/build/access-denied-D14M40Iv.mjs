import { _ as __nuxt_component_0 } from './nuxt-link-Cf-wdoG-.mjs';
import { mergeProps, withCtx, createTextVNode, useSSRContext } from 'vue';
import { ssrRenderAttrs, ssrRenderComponent } from 'vue/server-renderer';
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

const _sfc_main = {};
function _sfc_ssrRender(_ctx, _push, _parent, _attrs) {
  const _component_NuxtLink = __nuxt_component_0;
  _push(`<div${ssrRenderAttrs(mergeProps({ class: "text-center py-20 px-4" }, _attrs))}><div class="text-4xl mb-3">\u{1F512}</div><h1 class="text-xl font-bold text-navy">Access denied</h1><p class="text-muted text-sm mt-2 max-w-sm mx-auto"> Your role does not include permission to view this page. Contact an administrator if you need access. </p><div class="mt-6 flex flex-wrap justify-center gap-3">`);
  _push(ssrRenderComponent(_component_NuxtLink, {
    to: "/dashboard",
    class: "rounded-lg bg-primary text-white text-sm font-medium px-4 py-2"
  }, {
    default: withCtx((_, _push2, _parent2, _scopeId) => {
      if (_push2) {
        _push2(` Dashboard `);
      } else {
        return [
          createTextVNode(" Dashboard ")
        ];
      }
    }),
    _: 1
  }, _parent));
  _push(ssrRenderComponent(_component_NuxtLink, {
    to: "/more",
    class: "rounded-lg border border-border bg-white text-sm font-medium px-4 py-2"
  }, {
    default: withCtx((_, _push2, _parent2, _scopeId) => {
      if (_push2) {
        _push2(` My permissions `);
      } else {
        return [
          createTextVNode(" My permissions ")
        ];
      }
    }),
    _: 1
  }, _parent));
  _push(`</div></div>`);
}
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/access-denied.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
const accessDenied = /* @__PURE__ */ _export_sfc(_sfc_main, [["ssrRender", _sfc_ssrRender]]);

export { accessDenied as default };
//# sourceMappingURL=access-denied-D14M40Iv.mjs.map
