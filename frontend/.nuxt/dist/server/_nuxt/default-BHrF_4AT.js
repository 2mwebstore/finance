import { _ as __nuxt_component_0 } from "./nuxt-link-Cf-wdoG-.js";
import { defineComponent, ref, computed, watch, mergeProps, withCtx, unref, createVNode, resolveDynamicComponent, openBlock, createBlock, toDisplayString, createCommentVNode, useSSRContext } from "vue";
import { ssrRenderAttrs, ssrRenderComponent, ssrRenderList, ssrRenderVNode, ssrInterpolate, ssrRenderAttr, ssrRenderSlot, ssrRenderClass } from "vue/server-renderer";
import { LayoutDashboard, ReceiptText, TrendingUp, TrendingDown, FileChartColumnIncreasing, Building2, ShieldCheck, KeyRound, UsersRound, Tags, ChartNoAxesCombined, ChevronRight, LogOut, Menu, Search, ChevronDown, X, UserRound } from "lucide-vue-next";
import { i as useRoute, f as useAuth, d as usePermissions, _ as _export_sfc } from "../server.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/ufo/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/defu/dist/defu.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/ofetch/dist/node.mjs";
import "#internal/nuxt/paths";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/hookable/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/unctx/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/h3/dist/index.mjs";
import "vue-router";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/klona/dist/index.mjs";
import "@vuetify/v0/utilities";
const _sfc_main = /* @__PURE__ */ defineComponent({
  __name: "default",
  __ssrInlineRender: true,
  setup(__props) {
    const route = useRoute();
    const { user } = useAuth();
    const {
      can,
      canViewDashboard,
      canViewTransactions,
      canViewReports,
      canViewBranches,
      canViewUsers,
      canViewRoles,
      canViewAudit,
      isSuperAdmin
    } = usePermissions();
    const searchTerm = ref("");
    const mobileMenuOpen = ref(false);
    const mainNav = computed(() => {
      const items = [];
      if (canViewDashboard.value) items.push({ label: "Dashboard", to: "/dashboard", icon: LayoutDashboard });
      if (canViewTransactions.value) items.push({ label: "Transactions", to: "/transactions", icon: ReceiptText });
      if (can("income.view")) items.push({ label: "Income", to: "/transactions?type=income", icon: TrendingUp, queryType: "income" });
      if (can("expense.view")) items.push({ label: "Expense", to: "/transactions?type=expense", icon: TrendingDown, queryType: "expense" });
      if (canViewReports.value) items.push({ label: "Reports", to: "/reports", icon: FileChartColumnIncreasing });
      return items;
    });
    const adminNav = computed(() => {
      const items = [];
      if (canViewBranches.value) items.push({ label: "Branches", to: "/admin/branches", icon: Building2 });
      if (canViewRoles.value) items.push({ label: "Roles & Permissions", to: "/admin/roles", icon: ShieldCheck });
      if (canViewRoles.value) items.push({ label: "Permission Codes", to: "/admin/permissions", icon: KeyRound });
      if (canViewUsers.value) items.push({ label: "Users", to: "/admin/users", icon: UsersRound });
      if (canViewAudit.value) items.push({ label: "Audit Log", to: "/admin/audit", icon: ReceiptText });
      if (can("categories.manage")) items.push({ label: "Categories", to: "/admin/categories", icon: Tags });
      return items;
    });
    const mobileMainNav = computed(() => {
      const items = [];
      if (canViewDashboard.value) items.push({ label: "Home", to: "/dashboard", icon: LayoutDashboard });
      if (canViewTransactions.value) items.push({ label: "Activity", to: "/transactions", icon: ReceiptText });
      if (can("income.view")) items.push({ label: "Income", to: "/transactions?type=income", icon: TrendingUp, queryType: "income" });
      if (can("expense.view")) items.push({ label: "Expense", to: "/transactions?type=expense", icon: TrendingDown, queryType: "expense" });
      if (canViewReports.value) items.push({ label: "Reports", to: "/reports", icon: FileChartColumnIncreasing });
      return items;
    });
    watch(() => route.fullPath, () => {
      mobileMenuOpen.value = false;
    });
    const initials = computed(() => (user.value?.full_name || user.value?.username || "A").split(/\s+/).filter(Boolean).slice(0, 2).map((part) => part[0]?.toUpperCase()).join(""));
    const profileRole = computed(() => isSuperAdmin.value ? "Super Administrator" : user.value?.roles?.[0] || "Administrator");
    function isActive(item) {
      const path = item.to.split("?")[0];
      if (item.queryType) return route.path === path && route.query.type === item.queryType;
      if (path === "/dashboard") return route.path === "/dashboard" || route.path === "/";
      if (path === "/transactions") return route.path.startsWith("/transactions") && route.path !== "/transactions/new" && route.query.type !== "income" && route.query.type !== "expense";
      if (path === "/transactions/new") return route.path === path;
      return route.path === path || route.path.startsWith(path + "/");
    }
    return (_ctx, _push, _parent, _attrs) => {
      const _component_NuxtLink = __nuxt_component_0;
      _push(`<div${ssrRenderAttrs(mergeProps({ class: "app-shell" }, _attrs))} data-v-f0e206c6><aside class="sidebar" data-v-f0e206c6>`);
      _push(ssrRenderComponent(_component_NuxtLink, {
        to: "/dashboard",
        class: "brand-lockup"
      }, {
        default: withCtx((_, _push2, _parent2, _scopeId) => {
          if (_push2) {
            _push2(`<span class="brand-mark" data-v-f0e206c6${_scopeId}>`);
            _push2(ssrRenderComponent(unref(ChartNoAxesCombined), {
              size: 23,
              "stroke-width": 2.4
            }, null, _parent2, _scopeId));
            _push2(`</span><span class="brand-copy" data-v-f0e206c6${_scopeId}><strong data-v-f0e206c6${_scopeId}>Income &amp; Expense</strong><small data-v-f0e206c6${_scopeId}>Manage your business</small></span>`);
          } else {
            return [
              createVNode("span", { class: "brand-mark" }, [
                createVNode(unref(ChartNoAxesCombined), {
                  size: 23,
                  "stroke-width": 2.4
                })
              ]),
              createVNode("span", { class: "brand-copy" }, [
                createVNode("strong", null, "Income & Expense"),
                createVNode("small", null, "Manage your business")
              ])
            ];
          }
        }),
        _: 1
      }, _parent));
      _push(`<nav class="sidebar-nav" aria-label="Main navigation" data-v-f0e206c6><div class="nav-section" data-v-f0e206c6><!--[-->`);
      ssrRenderList(unref(mainNav), (item) => {
        _push(ssrRenderComponent(_component_NuxtLink, {
          key: item.label,
          to: item.to,
          class: ["sidebar-link", isActive(item) ? "is-active" : ""]
        }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              ssrRenderVNode(_push2, createVNode(resolveDynamicComponent(item.icon), {
                size: 18,
                "stroke-width": 1.9
              }, null), _parent2, _scopeId);
              _push2(`<span data-v-f0e206c6${_scopeId}>${ssrInterpolate(item.label)}</span>`);
              if (item.label === "Branches" || item.label === "Roles & Permissions") {
                _push2(ssrRenderComponent(unref(ChevronRight), {
                  class: "nav-chevron",
                  size: 15
                }, null, _parent2, _scopeId));
              } else {
                _push2(`<!---->`);
              }
            } else {
              return [
                (openBlock(), createBlock(resolveDynamicComponent(item.icon), {
                  size: 18,
                  "stroke-width": 1.9
                })),
                createVNode("span", null, toDisplayString(item.label), 1),
                item.label === "Branches" || item.label === "Roles & Permissions" ? (openBlock(), createBlock(unref(ChevronRight), {
                  key: 0,
                  class: "nav-chevron",
                  size: 15
                })) : createCommentVNode("", true)
              ];
            }
          }),
          _: 2
        }, _parent));
      });
      _push(`<!--]--></div>`);
      if (unref(adminNav).length) {
        _push(`<div class="nav-divider" data-v-f0e206c6></div>`);
      } else {
        _push(`<!---->`);
      }
      if (unref(adminNav).length) {
        _push(`<div class="nav-section" data-v-f0e206c6><p class="nav-caption" data-v-f0e206c6>MANAGE</p><!--[-->`);
        ssrRenderList(unref(adminNav), (item) => {
          _push(ssrRenderComponent(_component_NuxtLink, {
            key: item.label,
            to: item.to,
            class: ["sidebar-link", isActive(item) ? "is-active" : ""]
          }, {
            default: withCtx((_, _push2, _parent2, _scopeId) => {
              if (_push2) {
                ssrRenderVNode(_push2, createVNode(resolveDynamicComponent(item.icon), {
                  size: 18,
                  "stroke-width": 1.9
                }, null), _parent2, _scopeId);
                _push2(`<span data-v-f0e206c6${_scopeId}>${ssrInterpolate(item.label)}</span>`);
                if (item.label === "Branches" || item.label === "Roles & Permissions") {
                  _push2(ssrRenderComponent(unref(ChevronRight), {
                    class: "nav-chevron",
                    size: 15
                  }, null, _parent2, _scopeId));
                } else {
                  _push2(`<!---->`);
                }
              } else {
                return [
                  (openBlock(), createBlock(resolveDynamicComponent(item.icon), {
                    size: 18,
                    "stroke-width": 1.9
                  })),
                  createVNode("span", null, toDisplayString(item.label), 1),
                  item.label === "Branches" || item.label === "Roles & Permissions" ? (openBlock(), createBlock(unref(ChevronRight), {
                    key: 0,
                    class: "nav-chevron",
                    size: 15
                  })) : createCommentVNode("", true)
                ];
              }
            }),
            _: 2
          }, _parent));
        });
        _push(`<!--]--></div>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</nav><div class="sidebar-profile" data-v-f0e206c6><div class="profile-avatar" data-v-f0e206c6>${ssrInterpolate(unref(initials))}</div><div class="sidebar-profile-copy" data-v-f0e206c6><strong data-v-f0e206c6>${ssrInterpolate(unref(user)?.full_name || "Account")}</strong><small data-v-f0e206c6>${ssrInterpolate(unref(profileRole))}</small></div><button class="logout-button" title="Sign out" aria-label="Sign out" data-v-f0e206c6>`);
      _push(ssrRenderComponent(unref(LogOut), { size: 16 }, null, _parent));
      _push(`</button></div></aside><div class="app-content" data-v-f0e206c6><header class="topbar" data-v-f0e206c6><button class="mobile-menu-button" type="button" aria-label="Open navigation"${ssrRenderAttr("aria-expanded", unref(mobileMenuOpen))} data-v-f0e206c6>`);
      _push(ssrRenderComponent(unref(Menu), { size: 20 }, null, _parent));
      _push(`</button><div class="mobile-brand" data-v-f0e206c6><span class="brand-mark small-brand-mark" data-v-f0e206c6>`);
      _push(ssrRenderComponent(unref(ChartNoAxesCombined), { size: 19 }, null, _parent));
      _push(`</span><strong data-v-f0e206c6>Income &amp; Expense</strong></div>`);
      if (unref(canViewTransactions)) {
        _push(`<form class="global-search" role="search" data-v-f0e206c6>`);
        _push(ssrRenderComponent(unref(Search), { size: 17 }, null, _parent));
        _push(`<input${ssrRenderAttr("value", unref(searchTerm))} type="search" placeholder="Search transactions…" aria-label="Search transactions" data-v-f0e206c6><kbd data-v-f0e206c6>↵</kbd></form>`);
      } else {
        _push(`<!---->`);
      }
      _push(`<div class="topbar-spacer" data-v-f0e206c6></div>`);
      _push(ssrRenderComponent(_component_NuxtLink, {
        to: "/profile",
        class: "topbar-user",
        "aria-label": "Open your profile"
      }, {
        default: withCtx((_, _push2, _parent2, _scopeId) => {
          if (_push2) {
            _push2(`<div class="profile-avatar topbar-avatar" data-v-f0e206c6${_scopeId}>${ssrInterpolate(unref(initials))}</div><div class="topbar-user-copy" data-v-f0e206c6${_scopeId}><strong data-v-f0e206c6${_scopeId}>${ssrInterpolate(unref(user)?.full_name || "Account")}</strong><small data-v-f0e206c6${_scopeId}>${ssrInterpolate(unref(profileRole))}</small></div>`);
            _push2(ssrRenderComponent(unref(ChevronDown), {
              size: 15,
              class: "topbar-chevron"
            }, null, _parent2, _scopeId));
          } else {
            return [
              createVNode("div", { class: "profile-avatar topbar-avatar" }, toDisplayString(unref(initials)), 1),
              createVNode("div", { class: "topbar-user-copy" }, [
                createVNode("strong", null, toDisplayString(unref(user)?.full_name || "Account"), 1),
                createVNode("small", null, toDisplayString(unref(profileRole)), 1)
              ]),
              createVNode(unref(ChevronDown), {
                size: 15,
                class: "topbar-chevron"
              })
            ];
          }
        }),
        _: 1
      }, _parent));
      _push(`<button class="mobile-logout" aria-label="Sign out" data-v-f0e206c6>`);
      _push(ssrRenderComponent(unref(LogOut), { size: 17 }, null, _parent));
      _push(`</button></header><main class="page-content" data-v-f0e206c6>`);
      ssrRenderSlot(_ctx.$slots, "default", {}, null, _push, _parent);
      _push(`</main></div>`);
      if (unref(mobileMenuOpen)) {
        _push(`<div class="mobile-menu-backdrop" data-v-f0e206c6></div>`);
      } else {
        _push(`<!---->`);
      }
      _push(`<aside class="${ssrRenderClass([{ "is-open": unref(mobileMenuOpen) }, "mobile-drawer"])}" aria-label="Mobile navigation" data-v-f0e206c6><div class="mobile-drawer-header" data-v-f0e206c6><div class="mobile-brand" data-v-f0e206c6><span class="brand-mark small-brand-mark" data-v-f0e206c6>`);
      _push(ssrRenderComponent(unref(ChartNoAxesCombined), { size: 19 }, null, _parent));
      _push(`</span><strong data-v-f0e206c6>Navigation</strong></div><button type="button" class="drawer-close" aria-label="Close navigation" data-v-f0e206c6>`);
      _push(ssrRenderComponent(unref(X), { size: 20 }, null, _parent));
      _push(`</button></div><nav class="mobile-drawer-nav" data-v-f0e206c6><!--[-->`);
      ssrRenderList(unref(mainNav), (item) => {
        _push(ssrRenderComponent(_component_NuxtLink, {
          key: `drawer-${item.label}`,
          to: item.to,
          class: isActive(item) ? "is-active" : "",
          onClick: ($event) => mobileMenuOpen.value = false
        }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              ssrRenderVNode(_push2, createVNode(resolveDynamicComponent(item.icon), { size: 18 }, null), _parent2, _scopeId);
              _push2(`<span data-v-f0e206c6${_scopeId}>${ssrInterpolate(item.label)}</span>`);
              _push2(ssrRenderComponent(unref(ChevronRight), { size: 15 }, null, _parent2, _scopeId));
            } else {
              return [
                (openBlock(), createBlock(resolveDynamicComponent(item.icon), { size: 18 })),
                createVNode("span", null, toDisplayString(item.label), 1),
                createVNode(unref(ChevronRight), { size: 15 })
              ];
            }
          }),
          _: 2
        }, _parent));
      });
      _push(`<!--]-->`);
      if (unref(adminNav).length) {
        _push(`<div class="drawer-caption" data-v-f0e206c6>MANAGE</div>`);
      } else {
        _push(`<!---->`);
      }
      _push(`<!--[-->`);
      ssrRenderList(unref(adminNav), (item) => {
        _push(ssrRenderComponent(_component_NuxtLink, {
          key: `drawer-admin-${item.label}`,
          to: item.to,
          class: isActive(item) ? "is-active" : "",
          onClick: ($event) => mobileMenuOpen.value = false
        }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              ssrRenderVNode(_push2, createVNode(resolveDynamicComponent(item.icon), { size: 18 }, null), _parent2, _scopeId);
              _push2(`<span data-v-f0e206c6${_scopeId}>${ssrInterpolate(item.label)}</span>`);
              _push2(ssrRenderComponent(unref(ChevronRight), { size: 15 }, null, _parent2, _scopeId));
            } else {
              return [
                (openBlock(), createBlock(resolveDynamicComponent(item.icon), { size: 18 })),
                createVNode("span", null, toDisplayString(item.label), 1),
                createVNode(unref(ChevronRight), { size: 15 })
              ];
            }
          }),
          _: 2
        }, _parent));
      });
      _push(`<!--]--></nav>`);
      _push(ssrRenderComponent(_component_NuxtLink, {
        to: "/profile",
        class: "drawer-profile",
        onClick: ($event) => mobileMenuOpen.value = false
      }, {
        default: withCtx((_, _push2, _parent2, _scopeId) => {
          if (_push2) {
            _push2(`<div class="profile-avatar" data-v-f0e206c6${_scopeId}>${ssrInterpolate(unref(initials))}</div><span data-v-f0e206c6${_scopeId}><strong data-v-f0e206c6${_scopeId}>${ssrInterpolate(unref(user)?.full_name || "Account")}</strong><small data-v-f0e206c6${_scopeId}>${ssrInterpolate(unref(profileRole))}</small></span>`);
            _push2(ssrRenderComponent(unref(ChevronRight), { size: 16 }, null, _parent2, _scopeId));
          } else {
            return [
              createVNode("div", { class: "profile-avatar" }, toDisplayString(unref(initials)), 1),
              createVNode("span", null, [
                createVNode("strong", null, toDisplayString(unref(user)?.full_name || "Account"), 1),
                createVNode("small", null, toDisplayString(unref(profileRole)), 1)
              ]),
              createVNode(unref(ChevronRight), { size: 16 })
            ];
          }
        }),
        _: 1
      }, _parent));
      _push(`</aside><nav class="mobile-nav" aria-label="Mobile navigation" data-v-f0e206c6><!--[-->`);
      ssrRenderList(unref(mobileMainNav).slice(0, 3), (item) => {
        _push(ssrRenderComponent(_component_NuxtLink, {
          key: item.label,
          to: item.to,
          class: isActive(item) ? "is-active" : ""
        }, {
          default: withCtx((_, _push2, _parent2, _scopeId) => {
            if (_push2) {
              ssrRenderVNode(_push2, createVNode(resolveDynamicComponent(item.icon), { size: 19 }, null), _parent2, _scopeId);
              _push2(`<span data-v-f0e206c6${_scopeId}>${ssrInterpolate(item.label)}</span>`);
            } else {
              return [
                (openBlock(), createBlock(resolveDynamicComponent(item.icon), { size: 19 })),
                createVNode("span", null, toDisplayString(item.label), 1)
              ];
            }
          }),
          _: 2
        }, _parent));
      });
      _push(`<!--]--><button type="button" class="mobile-nav-menu" data-v-f0e206c6>`);
      _push(ssrRenderComponent(unref(Menu), { size: 19 }, null, _parent));
      _push(`<span data-v-f0e206c6>Menu</span></button>`);
      _push(ssrRenderComponent(_component_NuxtLink, {
        to: "/profile",
        class: unref(route).path === "/profile" ? "is-active" : ""
      }, {
        default: withCtx((_, _push2, _parent2, _scopeId) => {
          if (_push2) {
            _push2(ssrRenderComponent(unref(UserRound), { size: 19 }, null, _parent2, _scopeId));
            _push2(`<span data-v-f0e206c6${_scopeId}>Profile</span>`);
          } else {
            return [
              createVNode(unref(UserRound), { size: 19 }),
              createVNode("span", null, "Profile")
            ];
          }
        }),
        _: 1
      }, _parent));
      _push(`</nav></div>`);
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("layouts/default.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
const _default = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-f0e206c6"]]);
export {
  _default as default
};
//# sourceMappingURL=default-BHrF_4AT.js.map
