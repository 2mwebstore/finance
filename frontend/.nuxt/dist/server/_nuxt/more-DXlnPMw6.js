import { defineComponent, computed, unref, useSSRContext, mergeProps, withCtx, createTextVNode, createVNode } from "vue";
import { ssrRenderSlot, ssrRenderAttrs, ssrInterpolate, ssrRenderComponent, ssrRenderList } from "vue/server-renderer";
import { d as usePermissions, f as useAuth } from "../server.mjs";
import { _ as __nuxt_component_0 } from "./nuxt-link-Cf-wdoG-.js";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/hookable/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/ofetch/dist/node.mjs";
import "#internal/nuxt/paths";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/unctx/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/h3/dist/index.mjs";
import "vue-router";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/defu/dist/defu.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/ufo/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/klona/dist/index.mjs";
import "@vuetify/v0/utilities";
const _sfc_main$1 = /* @__PURE__ */ defineComponent({
  __name: "Can",
  __ssrInlineRender: true,
  props: {
    perm: {},
    any: {},
    all: {}
  },
  setup(__props) {
    const props = __props;
    const { can, canAll } = usePermissions();
    const allowed = computed(() => {
      if (props.all?.length) return canAll(...props.all);
      if (props.any?.length) return can(...props.any);
      if (props.perm) return can(props.perm);
      return true;
    });
    return (_ctx, _push, _parent, _attrs) => {
      if (unref(allowed)) {
        ssrRenderSlot(_ctx.$slots, "default", {}, null, _push, _parent);
      } else {
        ssrRenderSlot(_ctx.$slots, "fallback", {}, null, _push, _parent);
      }
    };
  }
});
const _sfc_setup$1 = _sfc_main$1.setup;
_sfc_main$1.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("components/Can.vue");
  return _sfc_setup$1 ? _sfc_setup$1(props, ctx) : void 0;
};
const _sfc_main = /* @__PURE__ */ defineComponent({
  __name: "more",
  __ssrInlineRender: true,
  setup(__props) {
    const { user } = useAuth();
    const {
      isSuperAdmin,
      permissions,
      canViewBranches,
      canViewUsers,
      canViewRoles,
      canViewAudit
    } = usePermissions();
    const showAdmin = computed(
      () => canViewBranches.value || canViewUsers.value || canViewRoles.value || canViewAudit.value
    );
    return (_ctx, _push, _parent, _attrs) => {
      const _component_Can = _sfc_main$1;
      const _component_NuxtLink = __nuxt_component_0;
      _push(`<div${ssrRenderAttrs(mergeProps({ class: "space-y-4" }, _attrs))}><h1 class="text-xl font-bold text-navy">More</h1><div class="bg-white rounded-xl border border-border divide-y divide-border"><div class="p-4"><div class="text-sm font-medium">${ssrInterpolate(unref(user)?.full_name)}</div><div class="text-xs text-muted">${ssrInterpolate(unref(user)?.email)}</div><div class="text-xs text-muted mt-1">${ssrInterpolate((unref(user)?.roles || []).join(", "))}</div>`);
      if (unref(isSuperAdmin)) {
        _push(`<div class="text-[10px] text-primary font-semibold mt-1">Super Admin</div>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</div>`);
      _push(ssrRenderComponent(_component_Can, { perm: "dashboard.view" }, {
        default: withCtx((_, _push2, _parent2, _scopeId) => {
          if (_push2) {
            _push2(ssrRenderComponent(_component_NuxtLink, {
              to: "/dashboard",
              class: "block p-4 text-sm hover:bg-canvas"
            }, {
              default: withCtx((_2, _push3, _parent3, _scopeId2) => {
                if (_push3) {
                  _push3(`Dashboard`);
                } else {
                  return [
                    createTextVNode("Dashboard")
                  ];
                }
              }),
              _: 1
            }, _parent2, _scopeId));
          } else {
            return [
              createVNode(_component_NuxtLink, {
                to: "/dashboard",
                class: "block p-4 text-sm hover:bg-canvas"
              }, {
                default: withCtx(() => [
                  createTextVNode("Dashboard")
                ]),
                _: 1
              })
            ];
          }
        }),
        _: 1
      }, _parent));
      _push(ssrRenderComponent(_component_Can, { any: ["transactions.view", "income.view", "expense.view"] }, {
        default: withCtx((_, _push2, _parent2, _scopeId) => {
          if (_push2) {
            _push2(ssrRenderComponent(_component_NuxtLink, {
              to: "/transactions",
              class: "block p-4 text-sm hover:bg-canvas"
            }, {
              default: withCtx((_2, _push3, _parent3, _scopeId2) => {
                if (_push3) {
                  _push3(`Transactions`);
                } else {
                  return [
                    createTextVNode("Transactions")
                  ];
                }
              }),
              _: 1
            }, _parent2, _scopeId));
          } else {
            return [
              createVNode(_component_NuxtLink, {
                to: "/transactions",
                class: "block p-4 text-sm hover:bg-canvas"
              }, {
                default: withCtx(() => [
                  createTextVNode("Transactions")
                ]),
                _: 1
              })
            ];
          }
        }),
        _: 1
      }, _parent));
      _push(ssrRenderComponent(_component_Can, { any: ["income.create", "expense.create"] }, {
        default: withCtx((_, _push2, _parent2, _scopeId) => {
          if (_push2) {
            _push2(ssrRenderComponent(_component_NuxtLink, {
              to: "/transactions/new",
              class: "block p-4 text-sm hover:bg-canvas"
            }, {
              default: withCtx((_2, _push3, _parent3, _scopeId2) => {
                if (_push3) {
                  _push3(`Add transaction`);
                } else {
                  return [
                    createTextVNode("Add transaction")
                  ];
                }
              }),
              _: 1
            }, _parent2, _scopeId));
          } else {
            return [
              createVNode(_component_NuxtLink, {
                to: "/transactions/new",
                class: "block p-4 text-sm hover:bg-canvas"
              }, {
                default: withCtx(() => [
                  createTextVNode("Add transaction")
                ]),
                _: 1
              })
            ];
          }
        }),
        _: 1
      }, _parent));
      _push(ssrRenderComponent(_component_Can, { any: ["reports.view", "dashboard.view"] }, {
        default: withCtx((_, _push2, _parent2, _scopeId) => {
          if (_push2) {
            _push2(ssrRenderComponent(_component_NuxtLink, {
              to: "/reports",
              class: "block p-4 text-sm hover:bg-canvas"
            }, {
              default: withCtx((_2, _push3, _parent3, _scopeId2) => {
                if (_push3) {
                  _push3(`Reports`);
                } else {
                  return [
                    createTextVNode("Reports")
                  ];
                }
              }),
              _: 1
            }, _parent2, _scopeId));
          } else {
            return [
              createVNode(_component_NuxtLink, {
                to: "/reports",
                class: "block p-4 text-sm hover:bg-canvas"
              }, {
                default: withCtx(() => [
                  createTextVNode("Reports")
                ]),
                _: 1
              })
            ];
          }
        }),
        _: 1
      }, _parent));
      _push(`</div>`);
      if (unref(showAdmin)) {
        _push(`<div class="bg-white rounded-xl border border-border divide-y divide-border"><div class="px-4 py-2 text-xs font-semibold text-muted uppercase tracking-wide">Administration</div>`);
        _push(ssrRenderComponent(_component_Can, { perm: "branches.view" }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              _push2(ssrRenderComponent(_component_NuxtLink, {
                to: "/admin/branches",
                class: "block p-4 text-sm hover:bg-canvas"
              }, {
                default: withCtx((_2, _push3, _parent3, _scopeId2) => {
                  if (_push3) {
                    _push3(`Branches`);
                  } else {
                    return [
                      createTextVNode("Branches")
                    ];
                  }
                }),
                _: 1
              }, _parent2, _scopeId));
            } else {
              return [
                createVNode(_component_NuxtLink, {
                  to: "/admin/branches",
                  class: "block p-4 text-sm hover:bg-canvas"
                }, {
                  default: withCtx(() => [
                    createTextVNode("Branches")
                  ]),
                  _: 1
                })
              ];
            }
          }),
          _: 1
        }, _parent));
        _push(ssrRenderComponent(_component_Can, { perm: "users.view" }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              _push2(ssrRenderComponent(_component_NuxtLink, {
                to: "/admin/users",
                class: "block p-4 text-sm hover:bg-canvas"
              }, {
                default: withCtx((_2, _push3, _parent3, _scopeId2) => {
                  if (_push3) {
                    _push3(`Users`);
                  } else {
                    return [
                      createTextVNode("Users")
                    ];
                  }
                }),
                _: 1
              }, _parent2, _scopeId));
            } else {
              return [
                createVNode(_component_NuxtLink, {
                  to: "/admin/users",
                  class: "block p-4 text-sm hover:bg-canvas"
                }, {
                  default: withCtx(() => [
                    createTextVNode("Users")
                  ]),
                  _: 1
                })
              ];
            }
          }),
          _: 1
        }, _parent));
        _push(ssrRenderComponent(_component_Can, { perm: "roles.view" }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              _push2(ssrRenderComponent(_component_NuxtLink, {
                to: "/admin/roles",
                class: "block p-4 text-sm hover:bg-canvas"
              }, {
                default: withCtx((_2, _push3, _parent3, _scopeId2) => {
                  if (_push3) {
                    _push3(`Roles`);
                  } else {
                    return [
                      createTextVNode("Roles")
                    ];
                  }
                }),
                _: 1
              }, _parent2, _scopeId));
            } else {
              return [
                createVNode(_component_NuxtLink, {
                  to: "/admin/roles",
                  class: "block p-4 text-sm hover:bg-canvas"
                }, {
                  default: withCtx(() => [
                    createTextVNode("Roles")
                  ]),
                  _: 1
                })
              ];
            }
          }),
          _: 1
        }, _parent));
        _push(ssrRenderComponent(_component_Can, { perm: "roles.view" }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              _push2(ssrRenderComponent(_component_NuxtLink, {
                to: "/admin/permissions",
                class: "block p-4 text-sm hover:bg-canvas"
              }, {
                default: withCtx((_2, _push3, _parent3, _scopeId2) => {
                  if (_push3) {
                    _push3(`Permission codes`);
                  } else {
                    return [
                      createTextVNode("Permission codes")
                    ];
                  }
                }),
                _: 1
              }, _parent2, _scopeId));
            } else {
              return [
                createVNode(_component_NuxtLink, {
                  to: "/admin/permissions",
                  class: "block p-4 text-sm hover:bg-canvas"
                }, {
                  default: withCtx(() => [
                    createTextVNode("Permission codes")
                  ]),
                  _: 1
                })
              ];
            }
          }),
          _: 1
        }, _parent));
        _push(ssrRenderComponent(_component_Can, { perm: "audit-logs.view" }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              _push2(ssrRenderComponent(_component_NuxtLink, {
                to: "/admin/audit",
                class: "block p-4 text-sm hover:bg-canvas"
              }, {
                default: withCtx((_2, _push3, _parent3, _scopeId2) => {
                  if (_push3) {
                    _push3(`Audit log`);
                  } else {
                    return [
                      createTextVNode("Audit log")
                    ];
                  }
                }),
                _: 1
              }, _parent2, _scopeId));
            } else {
              return [
                createVNode(_component_NuxtLink, {
                  to: "/admin/audit",
                  class: "block p-4 text-sm hover:bg-canvas"
                }, {
                  default: withCtx(() => [
                    createTextVNode("Audit log")
                  ]),
                  _: 1
                })
              ];
            }
          }),
          _: 1
        }, _parent));
        _push(`</div>`);
      } else {
        _push(`<!---->`);
      }
      _push(`<div class="bg-white rounded-xl border border-border p-4"><div class="text-xs font-semibold text-muted uppercase mb-2">Your permissions</div><div class="flex flex-wrap gap-1 max-h-32 overflow-y-auto"><!--[-->`);
      ssrRenderList(unref(permissions), (code) => {
        _push(`<span class="text-[10px] px-1.5 py-0.5 rounded bg-canvas border border-border font-mono">${ssrInterpolate(code)}</span>`);
      });
      _push(`<!--]-->`);
      if (unref(isSuperAdmin) && !unref(permissions).length) {
        _push(`<span class="text-xs text-muted">All (super admin)</span>`);
      } else {
        _push(`<!---->`);
      }
      if (!unref(isSuperAdmin) && !unref(permissions).length) {
        _push(`<span class="text-xs text-muted">None loaded</span>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</div></div><div class="bg-white rounded-xl border border-border"><button class="w-full text-left p-4 text-sm text-expense hover:bg-canvas"> Sign out </button></div></div>`);
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/more.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
export {
  _sfc_main as default
};
//# sourceMappingURL=more-DXlnPMw6.js.map
