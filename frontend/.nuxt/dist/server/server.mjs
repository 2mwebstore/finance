import { shallowReactive, reactive, effectScope, getCurrentScope, hasInjectionContext, getCurrentInstance as getCurrentInstance$1, inject as inject$1, toRef, ref, markRaw, isRef, isReactive, toRaw, onScopeDispose, watch, nextTick, toRefs, computed, readonly, shallowRef, isReadonly, isShallow, defineComponent as defineComponent$1, createElementBlock, provide, cloneVNode, h, warn, capitalize, Fragment, watchEffect, unref, camelize, createVNode, normalizeClass, mergeProps, createElementVNode, toValue, normalizeStyle, useId, Text, resolveDynamicComponent, withDirectives, toDisplayString, TransitionGroup, Transition, vShow, createTextVNode, defineAsyncComponent, Suspense, isVNode, createCommentVNode, withCtx, useSSRContext, onErrorCaptured, onServerPrefetch, createApp } from "vue";
import { $fetch as $fetch$1 } from "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/ofetch/dist/node.mjs";
import { baseURL } from "#internal/nuxt/paths";
import { createHooks } from "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/hookable/dist/index.mjs";
import { getContext, executeAsync } from "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/unctx/dist/index.mjs";
import { sanitizeStatusCode, createError as createError$1 } from "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/h3/dist/index.mjs";
import { START_LOCATION, createMemoryHistory, createRouter, useRoute as useRoute$2, RouterView } from "vue-router";
import { defu } from "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/defu/dist/defu.mjs";
import { hasProtocol, joinURL, parseURL, encodePath, decodePath, withQuery, isScriptProtocol } from "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/ufo/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/klona/dist/index.mjs";
import { isNullOrUndefined, isString, isUndefined, isNumber, isBoolean, isElement, isObject, range, isFunction, isNull } from "@vuetify/v0/utilities";
import { ssrRenderComponent, ssrRenderSuspense, ssrRenderVNode } from "vue/server-renderer";
if (!globalThis.$fetch) {
  globalThis.$fetch = $fetch$1.create({
    baseURL: baseURL()
  });
}
if (!("global" in globalThis)) {
  globalThis.global = globalThis;
}
const appLayoutTransition = false;
const nuxtLinkDefaults = { "componentName": "NuxtLink" };
const appId = "nuxt-app";
function getNuxtAppCtx(id = appId) {
  return getContext(id, {
    asyncContext: false
  });
}
const NuxtPluginIndicator = "__nuxt_plugin";
function createNuxtApp(options) {
  let hydratingCount = 0;
  const nuxtApp = {
    _id: options.id || appId || "nuxt-app",
    _scope: effectScope(),
    provide: void 0,
    globalName: "nuxt",
    versions: {
      get nuxt() {
        return "3.21.11";
      },
      get vue() {
        return nuxtApp.vueApp.version;
      }
    },
    payload: shallowReactive({
      ...options.ssrContext?.payload || {},
      data: shallowReactive({}),
      state: reactive({}),
      once: /* @__PURE__ */ new Set(),
      _errors: shallowReactive({})
    }),
    static: {
      data: {}
    },
    runWithContext(fn) {
      if (nuxtApp._scope.active && !getCurrentScope()) {
        return nuxtApp._scope.run(() => callWithNuxt(nuxtApp, fn));
      }
      return callWithNuxt(nuxtApp, fn);
    },
    isHydrating: false,
    deferHydration() {
      if (!nuxtApp.isHydrating) {
        return () => {
        };
      }
      hydratingCount++;
      let called = false;
      return () => {
        if (called) {
          return;
        }
        called = true;
        hydratingCount--;
        if (hydratingCount === 0) {
          nuxtApp.isHydrating = false;
          return nuxtApp.callHook("app:suspense:resolve");
        }
      };
    },
    _asyncDataPromises: {},
    _asyncData: shallowReactive({}),
    _payloadRevivers: {},
    ...options
  };
  {
    nuxtApp.payload.serverRendered = true;
  }
  if (nuxtApp.ssrContext) {
    nuxtApp.payload.path = nuxtApp.ssrContext.url;
    nuxtApp.ssrContext.nuxt = nuxtApp;
    nuxtApp.ssrContext.payload = nuxtApp.payload;
    nuxtApp.ssrContext.config = {
      public: nuxtApp.ssrContext.runtimeConfig.public,
      app: nuxtApp.ssrContext.runtimeConfig.app
    };
  }
  nuxtApp.hooks = createHooks();
  nuxtApp.hook = nuxtApp.hooks.hook;
  {
    const contextCaller = async function(hooks, args) {
      for (const hook of hooks) {
        await nuxtApp.runWithContext(() => hook(...args));
      }
    };
    nuxtApp.hooks.callHook = (name, ...args) => nuxtApp.hooks.callHookWith(contextCaller, name, ...args);
  }
  nuxtApp.callHook = nuxtApp.hooks.callHook;
  nuxtApp.provide = (name, value) => {
    const $name = "$" + name;
    defineGetter(nuxtApp, $name, value);
    defineGetter(nuxtApp.vueApp.config.globalProperties, $name, value);
  };
  defineGetter(nuxtApp.vueApp, "$nuxt", nuxtApp);
  defineGetter(nuxtApp.vueApp.config.globalProperties, "$nuxt", nuxtApp);
  const runtimeConfig = options.ssrContext.runtimeConfig;
  nuxtApp.provide("config", runtimeConfig);
  return nuxtApp;
}
function registerPluginHooks(nuxtApp, plugin2) {
  if (plugin2.hooks) {
    nuxtApp.hooks.addHooks(plugin2.hooks);
  }
}
async function applyPlugin(nuxtApp, plugin2) {
  if (typeof plugin2 === "function") {
    const { provide: provide2 } = await nuxtApp.runWithContext(() => plugin2(nuxtApp)) || {};
    if (provide2 && typeof provide2 === "object") {
      for (const key in provide2) {
        nuxtApp.provide(key, provide2[key]);
      }
    }
  }
}
async function applyPlugins(nuxtApp, plugins2) {
  const resolvedPlugins = /* @__PURE__ */ new Set();
  const unresolvedPlugins = [];
  const parallels = [];
  let error = void 0;
  let promiseDepth = 0;
  async function executePlugin(plugin2) {
    const unresolvedPluginsForThisPlugin = plugin2.dependsOn?.filter((name) => plugins2.some((p) => p._name === name) && !resolvedPlugins.has(name)) ?? [];
    if (unresolvedPluginsForThisPlugin.length > 0) {
      unresolvedPlugins.push([new Set(unresolvedPluginsForThisPlugin), plugin2]);
    } else {
      const promise = applyPlugin(nuxtApp, plugin2).then(async () => {
        if (plugin2._name) {
          resolvedPlugins.add(plugin2._name);
          await Promise.all(unresolvedPlugins.map(async ([dependsOn, unexecutedPlugin]) => {
            if (dependsOn.has(plugin2._name)) {
              dependsOn.delete(plugin2._name);
              if (dependsOn.size === 0) {
                promiseDepth++;
                await executePlugin(unexecutedPlugin);
              }
            }
          }));
        }
      }).catch((e) => {
        if (!plugin2.parallel && !nuxtApp.payload.error) {
          throw e;
        }
        error ||= e;
      });
      if (plugin2.parallel) {
        parallels.push(promise);
      } else {
        await promise;
      }
    }
  }
  for (const plugin2 of plugins2) {
    if (nuxtApp.ssrContext?.islandContext && plugin2.env?.islands === false) {
      continue;
    }
    registerPluginHooks(nuxtApp, plugin2);
  }
  for (const plugin2 of plugins2) {
    if (nuxtApp.ssrContext?.islandContext && plugin2.env?.islands === false) {
      continue;
    }
    await executePlugin(plugin2);
  }
  await Promise.all(parallels);
  if (promiseDepth) {
    for (let i = 0; i < promiseDepth; i++) {
      await Promise.all(parallels);
    }
  }
  if (error) {
    throw nuxtApp.payload.error || error;
  }
}
// @__NO_SIDE_EFFECTS__
function defineNuxtPlugin(plugin2) {
  if (typeof plugin2 === "function") {
    return plugin2;
  }
  const _name = plugin2._name || plugin2.name;
  delete plugin2.name;
  return Object.assign(plugin2.setup || (() => {
  }), plugin2, { [NuxtPluginIndicator]: true, _name });
}
function callWithNuxt(nuxt, setup, args) {
  const fn = () => setup();
  const nuxtAppCtx = getNuxtAppCtx(nuxt._id);
  {
    return nuxt.vueApp.runWithContext(() => nuxtAppCtx.callAsync(nuxt, fn));
  }
}
function tryUseNuxtApp(id) {
  let nuxtAppInstance;
  if (hasInjectionContext()) {
    nuxtAppInstance = getCurrentInstance$1()?.appContext.app.$nuxt;
  }
  nuxtAppInstance ||= getNuxtAppCtx(id).tryUse();
  return nuxtAppInstance || null;
}
function useNuxtApp(id) {
  const nuxtAppInstance = tryUseNuxtApp(id);
  if (!nuxtAppInstance) {
    {
      throw new Error("[nuxt] instance unavailable");
    }
  }
  return nuxtAppInstance;
}
// @__NO_SIDE_EFFECTS__
function useRuntimeConfig(_event) {
  return useNuxtApp().$config;
}
function defineGetter(obj, key, val) {
  Object.defineProperty(obj, key, { get: () => val });
}
const LayoutMetaSymbol = /* @__PURE__ */ Symbol("layout-meta");
const PageRouteSymbol = /* @__PURE__ */ Symbol("route");
import.meta.url.replace(/\/app\/.*$/, "/");
const useRouter = () => {
  return useNuxtApp()?.$router;
};
function isScopeWithinInstance(instance) {
  const instanceScope = instance.scope;
  let scope = getCurrentScope();
  while (scope) {
    if (scope === instanceScope) {
      return true;
    }
    scope = scope.parent;
  }
  return false;
}
const useRoute$1 = () => {
  if (hasInjectionContext()) {
    const instance = getCurrentInstance$1();
    if (!instance || isScopeWithinInstance(instance)) {
      return inject$1(PageRouteSymbol, useNuxtApp()._route);
    }
  }
  return useNuxtApp()._route;
};
// @__NO_SIDE_EFFECTS__
function defineNuxtRouteMiddleware(middleware) {
  return middleware;
}
const isProcessingMiddleware = () => {
  try {
    if (useNuxtApp()._processingMiddleware) {
      return true;
    }
  } catch {
    return false;
  }
  return false;
};
const HTML_ATTR_UNSAFE_RE = /[&"'<>]/g;
const HTML_ATTR_ENCODE_MAP = {
  "&": "%26",
  '"': "%22",
  "'": "%27",
  "<": "%3C",
  ">": "%3E"
};
function encodeForHtmlAttr(value) {
  return value.replace(HTML_ATTR_UNSAFE_RE, (c) => HTML_ATTR_ENCODE_MAP[c]);
}
const navigateTo = (to, options) => {
  to ||= "/";
  const toPath = typeof to === "string" ? to : "path" in to ? resolveRouteObject(to) : useRouter().resolve(to).href;
  const isExternalHost = hasProtocol(toPath, { acceptRelative: true });
  const isExternal = options?.external || isExternalHost;
  if (isExternal) {
    if (!options?.external) {
      throw new Error("Navigating to an external URL is not allowed by default. Use `navigateTo(url, { external: true })`.");
    }
    const { protocol } = new URL(toPath, "http://localhost");
    if (protocol && isScriptProtocol(protocol)) {
      throw new Error(`Cannot navigate to a URL with '${protocol}' protocol.`);
    }
  }
  const inMiddleware = isProcessingMiddleware();
  const router = useRouter();
  const nuxtApp = useNuxtApp();
  {
    if (nuxtApp.ssrContext) {
      const fullPath = typeof to === "string" || isExternal ? toPath : router.resolve(to).fullPath || "/";
      const location2 = isExternal ? toPath : joinURL((/* @__PURE__ */ useRuntimeConfig()).app.baseURL, fullPath);
      const redirect = async function(response) {
        await nuxtApp.callHook("app:redirected");
        const encodedHeader = encodeURL(location2, isExternalHost);
        const encodedLoc = encodeForHtmlAttr(encodedHeader);
        nuxtApp.ssrContext["~renderResponse"] = {
          statusCode: sanitizeStatusCode(options?.redirectCode || 302, 302),
          body: `<!DOCTYPE html><html><head><meta http-equiv="refresh" content="0; url=${encodedLoc}"></head></html>`,
          headers: { location: encodedHeader }
        };
        return response;
      };
      if (!isExternal && inMiddleware) {
        router.afterEach((final) => final.fullPath === fullPath ? redirect(false) : void 0);
        return to;
      }
      return redirect(!inMiddleware ? void 0 : (
        /* abort route navigation */
        false
      ));
    }
  }
  if (isExternal) {
    nuxtApp._scope.stop();
    if (options?.replace) {
      (void 0).replace(toPath);
    } else {
      (void 0).href = toPath;
    }
    if (inMiddleware) {
      if (!nuxtApp.isHydrating) {
        return false;
      }
      return new Promise(() => {
      });
    }
    return Promise.resolve();
  }
  const encodedTo = typeof to === "string" ? encodeRoutePath(to) : to;
  return options?.replace ? router.replace(encodedTo) : router.push(encodedTo);
};
function resolveRouteObject(to) {
  return withQuery(to.path || "", to.query || {}) + (to.hash || "");
}
function encodeURL(location2, isExternalHost = false) {
  const url = new URL(location2, "http://localhost");
  if (!isExternalHost) {
    const pathname = url.pathname.replace(/^\/{2,}/, "/");
    return pathname + url.search + url.hash;
  }
  if (location2.startsWith("//")) {
    return url.toString().replace(url.protocol, "");
  }
  return url.toString();
}
function encodeRoutePath(url) {
  const parsed = parseURL(url);
  return encodePath(decodePath(parsed.pathname)) + parsed.search + parsed.hash;
}
const NUXT_ERROR_SIGNATURE = "__nuxt_error";
const useError = /* @__NO_SIDE_EFFECTS__ */ () => toRef(useNuxtApp().payload, "error");
const showError = (error) => {
  const nuxtError = createError(error);
  try {
    const error2 = /* @__PURE__ */ useError();
    if (false) ;
    error2.value ||= nuxtError;
  } catch {
    throw nuxtError;
  }
  return nuxtError;
};
const isNuxtError = (error) => !!error && typeof error === "object" && NUXT_ERROR_SIGNATURE in error;
const createError = (error) => {
  if (typeof error !== "string" && error.statusText) {
    error.message ??= error.statusText;
  }
  const nuxtError = createError$1(error);
  Object.defineProperty(nuxtError, NUXT_ERROR_SIGNATURE, {
    value: true,
    configurable: false,
    writable: false
  });
  Object.defineProperty(nuxtError, "status", {
    // eslint-disable-next-line @typescript-eslint/no-deprecated
    get: () => nuxtError.statusCode,
    configurable: true
  });
  Object.defineProperty(nuxtError, "statusText", {
    // eslint-disable-next-line @typescript-eslint/no-deprecated
    get: () => nuxtError.statusMessage,
    configurable: true
  });
  return nuxtError;
};
function freezeHead(head) {
  const realPush = head.push;
  head.push = () => ({ dispose: () => {
  }, patch: () => {
  }, _poll: () => {
  } });
  return () => {
    head.push = realPush;
  };
}
const unhead_k2P3m_ZDyjlr2mMYnoDPwavjsDN8hBlk9cFai0bbopU = /* @__PURE__ */ defineNuxtPlugin({
  name: "nuxt:head",
  enforce: "pre",
  setup(nuxtApp) {
    const head = nuxtApp.ssrContext.head;
    if (nuxtApp.ssrContext.islandContext) {
      const unfreeze = freezeHead(head);
      nuxtApp.hooks.hookOnce("app:created", unfreeze);
    }
    nuxtApp.vueApp.use(head);
  }
});
const ROUTE_KEY_PARENTHESES_RE$1 = /(:\w+)\([^)]+\)/g;
const ROUTE_KEY_SYMBOLS_RE$1 = /(:\w+)[?+*]/g;
const ROUTE_KEY_NORMAL_RE$1 = /:\w+/g;
const interpolatePath = (route, match) => {
  return match.path.replace(ROUTE_KEY_PARENTHESES_RE$1, "$1").replace(ROUTE_KEY_SYMBOLS_RE$1, "$1").replace(ROUTE_KEY_NORMAL_RE$1, (r) => route.params[r.slice(1)]?.toString() || "");
};
const generateRouteKey$1 = (routeProps, override) => {
  const matchedRoute = routeProps.route.matched.find((m) => m.components?.default === routeProps.Component.type);
  const source = matchedRoute?.meta.key ?? (matchedRoute && interpolatePath(routeProps.route, matchedRoute));
  return typeof source === "function" ? source(routeProps.route) : source;
};
function toArray$1(value) {
  return Array.isArray(value) ? value : [value];
}
const _wrapInTransition = (props, children) => {
  return { default: () => children.default?.() };
};
const ROUTE_KEY_PARENTHESES_RE = /(:\w+)\([^)]+\)/g;
const ROUTE_KEY_SYMBOLS_RE = /(:\w+)[?+*]/g;
const ROUTE_KEY_NORMAL_RE = /:\w+/g;
function generateRouteKey(route) {
  const source = route?.meta.key ?? route.path.replace(ROUTE_KEY_PARENTHESES_RE, "$1").replace(ROUTE_KEY_SYMBOLS_RE, "$1").replace(ROUTE_KEY_NORMAL_RE, (r) => route.params[r.slice(1)]?.toString() || "");
  return typeof source === "function" ? source(route) : source;
}
function isChangingPage(to, from) {
  if (to === from || from === START_LOCATION) {
    return false;
  }
  if (generateRouteKey(to) !== generateRouteKey(from)) {
    return true;
  }
  const areComponentsSame = to.matched.every(
    (comp, index) => comp.components && comp.components.default === from.matched[index]?.components?.default
  );
  if (areComponentsSame) {
    return false;
  }
  return true;
}
const VALID_TAG_RE = /^[a-z][a-z0-9-]*$/i;
function sanitizeTag(tag, fallback) {
  return tag && VALID_TAG_RE.test(tag) ? tag : fallback;
}
function toArray(value) {
  return Array.isArray(value) ? value : [value];
}
function _mergeTransitionProps(routeProps) {
  const _props = [];
  for (const prop of routeProps) {
    if (!prop) {
      continue;
    }
    _props.push({
      ...prop,
      onAfterLeave: prop.onAfterLeave ? toArray(prop.onAfterLeave) : void 0,
      onBeforeLeave: prop.onBeforeLeave ? toArray(prop.onBeforeLeave) : void 0
    });
  }
  return defu(..._props);
}
const routerOptions0 = {
  scrollBehavior(to, from, savedPosition) {
    const nuxtApp = useNuxtApp();
    const router = useRouter();
    const hashScrollBehaviour = router.options?.scrollBehaviorType ?? "auto";
    if (to.path.replace(/\/$/, "") === from.path.replace(/\/$/, "")) {
      if (from.hash && !to.hash) {
        return { left: 0, top: 0 };
      }
      if (to.hash) {
        return { el: to.hash, top: _getHashElementScrollMarginTop(to.hash), behavior: hashScrollBehaviour };
      }
      return false;
    }
    const routeAllowsScrollToTop = typeof to.meta.scrollToTop === "function" ? to.meta.scrollToTop(to, from) : to.meta.scrollToTop;
    if (routeAllowsScrollToTop === false) {
      return false;
    }
    if (from === START_LOCATION) {
      return _calculatePosition(to, from, savedPosition, hashScrollBehaviour);
    }
    return new Promise((resolve) => {
      const doScroll = () => {
        requestAnimationFrame(() => {
          if (router.currentRoute.value.fullPath !== to.fullPath) {
            resolve(false);
            return;
          }
          resolve(_calculatePosition(to, from, savedPosition, hashScrollBehaviour));
        });
      };
      nuxtApp.hooks.hookOnce("page:loading:end", () => {
        const transitionPromise = nuxtApp["~transitionPromise"];
        if (transitionPromise) {
          transitionPromise.then(doScroll);
        } else {
          doScroll();
        }
      });
    });
  }
};
function _getHashElementScrollMarginTop(selector) {
  try {
    const elem = (void 0).querySelector(selector);
    if (elem) {
      return (Number.parseFloat(getComputedStyle(elem).scrollMarginTop) || 0) + (Number.parseFloat(getComputedStyle((void 0).documentElement).scrollPaddingTop) || 0);
    }
  } catch {
  }
  return 0;
}
function _calculatePosition(to, from, savedPosition, defaultHashScrollBehaviour) {
  if (savedPosition) {
    return savedPosition;
  }
  if (to.hash) {
    return {
      el: to.hash,
      top: _getHashElementScrollMarginTop(to.hash),
      behavior: isChangingPage(to, from) ? defaultHashScrollBehaviour : "instant"
    };
  }
  return {
    left: 0,
    top: 0
  };
}
const configRouterOptions = {
  hashMode: false,
  scrollBehaviorType: "auto"
};
const routerOptions = {
  ...configRouterOptions,
  ...routerOptions0
};
const sensitiveMatcher = (m, p) => {
  return [];
};
const foldedMatcher = sensitiveMatcher;
const decodeRoutePath = function decodeRoutePath2(path) {
  if (!path.includes("%")) return path;
  const queryIndex = path.indexOf("?");
  const pathname = queryIndex === -1 ? path : path.slice(0, queryIndex);
  try {
    return queryIndex === -1 ? decodeURI(pathname) : decodeURI(pathname) + path.slice(queryIndex);
  } catch {
    return path;
  }
};
const normalizePath = (path, fold) => {
  if (typeof path !== "string") {
    return path;
  }
  const decoded = decodeRoutePath(path);
  return fold ? decoded.toLowerCase() : decoded;
};
const _routeRulesMatcher = (path) => routerOptions.sensitive ? defu({}, ...sensitiveMatcher("", normalizePath(path, false)).map((r) => r.data).reverse()) : defu({}, ...foldedMatcher("", normalizePath(path, true)).map((r) => r.data).reverse());
const routeRulesMatcher$1 = _routeRulesMatcher;
function getRouteRules(arg) {
  const path = typeof arg === "string" ? arg : arg.path;
  try {
    return routeRulesMatcher$1(path);
  } catch (e) {
    console.error("[nuxt] Error matching route rules.", e);
    return {};
  }
}
const __nuxt_page_meta = { layout: "auth" };
const _routes = [
  {
    name: "more",
    path: "/more",
    component: () => import("./_nuxt/more-DXlnPMw6.js")
  },
  {
    name: "index",
    path: "/",
    component: () => import("./_nuxt/index-_CNWP3Gu.js")
  },
  {
    name: "login",
    path: "/login",
    meta: __nuxt_page_meta || {},
    component: () => import("./_nuxt/login-BLD2i4QZ.js")
  },
  {
    name: "manage",
    path: "/manage",
    component: () => import("./_nuxt/manage-DQ-4iHrd.js")
  },
  {
    name: "profile",
    path: "/profile",
    component: () => import("./_nuxt/profile-CaxEUmZX.js")
  },
  {
    name: "reports",
    path: "/reports",
    component: () => import("./_nuxt/reports-CI8BDXv_.js")
  },
  {
    name: "dashboard",
    path: "/dashboard",
    component: () => import("./_nuxt/dashboard-BqJ4RJYO.js")
  },
  {
    name: "admin-audit",
    path: "/admin/audit",
    component: () => import("./_nuxt/audit-Cyki4-1f.js")
  },
  {
    name: "admin-roles",
    path: "/admin/roles",
    component: () => import("./_nuxt/roles-D_XDrUlH.js")
  },
  {
    name: "admin-users",
    path: "/admin/users",
    component: () => import("./_nuxt/users-BnGad1Is.js")
  },
  {
    name: "access-denied",
    path: "/access-denied",
    component: () => import("./_nuxt/access-denied-D14M40Iv.js")
  },
  {
    name: "admin-branches",
    path: "/admin/branches",
    component: () => import("./_nuxt/branches-uzytKBXM.js")
  },
  {
    name: "admin-categories",
    path: "/admin/categories",
    component: () => import("./_nuxt/categories-dn42pZlQ.js")
  },
  {
    name: "transactions-new",
    path: "/transactions/new",
    component: () => import("./_nuxt/new-BHgMEuLn.js")
  },
  {
    name: "admin-permissions",
    path: "/admin/permissions",
    component: () => import("./_nuxt/permissions-BEoao-Fy.js")
  },
  {
    name: "transactions",
    path: "/transactions",
    component: () => import("./_nuxt/index-Ci5AUSnw.js")
  }
];
const validate = /* @__PURE__ */ defineNuxtRouteMiddleware(async (to) => {
  let __temp, __restore;
  if (!to.meta?.validate) {
    return;
  }
  const result = ([__temp, __restore] = executeAsync(() => Promise.resolve(to.meta.validate(to))), __temp = await __temp, __restore(), __temp);
  if (result === true) {
    return;
  }
  const error = createError({
    fatal: false,
    // eslint-disable-next-line @typescript-eslint/no-deprecated
    status: result && (result.status || result.statusCode) || 404,
    // eslint-disable-next-line @typescript-eslint/no-deprecated
    statusText: result && (result.statusText || result.statusMessage) || `Page Not Found: ${to.fullPath}`,
    data: {
      path: to.fullPath
    }
  });
  return error;
});
function useApiBase() {
  const config = /* @__PURE__ */ useRuntimeConfig();
  const configuredBase = String(config.public.apiBase || "").replace(/\/$/, "");
  if (configuredBase) return configuredBase;
  return "http://localhost:8080/api/v1";
}
let activePinia;
const setActivePinia = (pinia) => activePinia = pinia;
const piniaSymbol = (
  /* istanbul ignore next */
  /* @__PURE__ */ Symbol()
);
function isPlainObject$1(o) {
  return o && typeof o === "object" && Object.prototype.toString.call(o) === "[object Object]" && typeof o.toJSON !== "function";
}
var MutationType;
(function(MutationType2) {
  MutationType2["direct"] = "direct";
  MutationType2["patchObject"] = "patch object";
  MutationType2["patchFunction"] = "patch function";
})(MutationType || (MutationType = {}));
function createPinia() {
  const scope = effectScope(true);
  const state = scope.run(() => ref({}));
  let _p = [];
  let toBeInstalled = [];
  const pinia = markRaw({
    install(app) {
      setActivePinia(pinia);
      {
        pinia._a = app;
        app.provide(piniaSymbol, pinia);
        app.config.globalProperties.$pinia = pinia;
        toBeInstalled.forEach((plugin2) => _p.push(plugin2));
        toBeInstalled = [];
      }
    },
    use(plugin2) {
      if (!this._a && true) {
        toBeInstalled.push(plugin2);
      } else {
        _p.push(plugin2);
      }
      return this;
    },
    _p,
    // it's actually undefined here
    // @ts-expect-error
    _a: null,
    _e: scope,
    _s: /* @__PURE__ */ new Map(),
    state
  });
  return pinia;
}
const noop = () => {
};
function addSubscription(subscriptions, callback, detached, onCleanup = noop) {
  subscriptions.push(callback);
  const removeSubscription = () => {
    const idx = subscriptions.indexOf(callback);
    if (idx > -1) {
      subscriptions.splice(idx, 1);
      onCleanup();
    }
  };
  if (!detached && getCurrentScope()) {
    onScopeDispose(removeSubscription);
  }
  return removeSubscription;
}
function triggerSubscriptions(subscriptions, ...args) {
  subscriptions.slice().forEach((callback) => {
    callback(...args);
  });
}
const fallbackRunWithContext = (fn) => fn();
const ACTION_MARKER = /* @__PURE__ */ Symbol();
const ACTION_NAME = /* @__PURE__ */ Symbol();
function mergeReactiveObjects(target, patchToApply) {
  if (target instanceof Map && patchToApply instanceof Map) {
    patchToApply.forEach((value, key) => target.set(key, value));
  } else if (target instanceof Set && patchToApply instanceof Set) {
    patchToApply.forEach(target.add, target);
  }
  for (const key in patchToApply) {
    if (!patchToApply.hasOwnProperty(key))
      continue;
    const subPatch = patchToApply[key];
    const targetValue = target[key];
    if (isPlainObject$1(targetValue) && isPlainObject$1(subPatch) && target.hasOwnProperty(key) && !isRef(subPatch) && !isReactive(subPatch)) {
      target[key] = mergeReactiveObjects(targetValue, subPatch);
    } else {
      target[key] = subPatch;
    }
  }
  return target;
}
const skipHydrateSymbol = (
  /* istanbul ignore next */
  /* @__PURE__ */ Symbol()
);
function shouldHydrate(obj) {
  return !isPlainObject$1(obj) || !obj.hasOwnProperty(skipHydrateSymbol);
}
const { assign } = Object;
function isComputed(o) {
  return !!(isRef(o) && o.effect);
}
function createOptionsStore(id, options, pinia, hot) {
  const { state, actions, getters } = options;
  const initialState = pinia.state.value[id];
  let store;
  function setup() {
    if (!initialState && true) {
      {
        pinia.state.value[id] = state ? state() : {};
      }
    }
    const localState = toRefs(pinia.state.value[id]);
    return assign(localState, actions, Object.keys(getters || {}).reduce((computedGetters, name) => {
      computedGetters[name] = markRaw(computed(() => {
        setActivePinia(pinia);
        const store2 = pinia._s.get(id);
        return getters[name].call(store2, store2);
      }));
      return computedGetters;
    }, {}));
  }
  store = createSetupStore(id, setup, options, pinia, hot, true);
  return store;
}
function createSetupStore($id, setup, options = {}, pinia, hot, isOptionsStore) {
  let scope;
  const optionsForPlugin = assign({ actions: {} }, options);
  const $subscribeOptions = { deep: true };
  let isListening;
  let isSyncListening;
  let subscriptions = [];
  let actionSubscriptions = [];
  let debuggerEvents;
  const initialState = pinia.state.value[$id];
  if (!isOptionsStore && !initialState && true) {
    {
      pinia.state.value[$id] = {};
    }
  }
  ref({});
  let activeListener;
  function $patch(partialStateOrMutator) {
    let subscriptionMutation;
    isListening = isSyncListening = false;
    if (typeof partialStateOrMutator === "function") {
      partialStateOrMutator(pinia.state.value[$id]);
      subscriptionMutation = {
        type: MutationType.patchFunction,
        storeId: $id,
        events: debuggerEvents
      };
    } else {
      mergeReactiveObjects(pinia.state.value[$id], partialStateOrMutator);
      subscriptionMutation = {
        type: MutationType.patchObject,
        payload: partialStateOrMutator,
        storeId: $id,
        events: debuggerEvents
      };
    }
    const myListenerId = activeListener = /* @__PURE__ */ Symbol();
    nextTick().then(() => {
      if (activeListener === myListenerId) {
        isListening = true;
      }
    });
    isSyncListening = true;
    triggerSubscriptions(subscriptions, subscriptionMutation, pinia.state.value[$id]);
  }
  const $reset = isOptionsStore ? function $reset2() {
    const { state } = options;
    const newState = state ? state() : {};
    this.$patch(($state) => {
      assign($state, newState);
    });
  } : (
    /* istanbul ignore next */
    noop
  );
  function $dispose() {
    scope.stop();
    subscriptions = [];
    actionSubscriptions = [];
    pinia._s.delete($id);
  }
  const action = (fn, name = "") => {
    if (ACTION_MARKER in fn) {
      fn[ACTION_NAME] = name;
      return fn;
    }
    const wrappedAction = function() {
      setActivePinia(pinia);
      const args = Array.from(arguments);
      const afterCallbackList = [];
      const onErrorCallbackList = [];
      function after(callback) {
        afterCallbackList.push(callback);
      }
      function onError(callback) {
        onErrorCallbackList.push(callback);
      }
      triggerSubscriptions(actionSubscriptions, {
        args,
        name: wrappedAction[ACTION_NAME],
        store,
        after,
        onError
      });
      let ret;
      try {
        ret = fn.apply(this && this.$id === $id ? this : store, args);
      } catch (error) {
        triggerSubscriptions(onErrorCallbackList, error);
        throw error;
      }
      if (ret instanceof Promise) {
        return ret.then((value) => {
          triggerSubscriptions(afterCallbackList, value);
          return value;
        }).catch((error) => {
          triggerSubscriptions(onErrorCallbackList, error);
          return Promise.reject(error);
        });
      }
      triggerSubscriptions(afterCallbackList, ret);
      return ret;
    };
    wrappedAction[ACTION_MARKER] = true;
    wrappedAction[ACTION_NAME] = name;
    return wrappedAction;
  };
  const partialStore = {
    _p: pinia,
    // _s: scope,
    $id,
    $onAction: addSubscription.bind(null, actionSubscriptions),
    $patch,
    $reset,
    $subscribe(callback, options2 = {}) {
      const removeSubscription = addSubscription(subscriptions, callback, options2.detached, () => stopWatcher());
      const stopWatcher = scope.run(() => watch(() => pinia.state.value[$id], (state) => {
        if (options2.flush === "sync" ? isSyncListening : isListening) {
          callback({
            storeId: $id,
            type: MutationType.direct,
            events: debuggerEvents
          }, state);
        }
      }, assign({}, $subscribeOptions, options2)));
      return removeSubscription;
    },
    $dispose
  };
  const store = reactive(partialStore);
  pinia._s.set($id, store);
  const runWithContext = pinia._a && pinia._a.runWithContext || fallbackRunWithContext;
  const setupStore = runWithContext(() => pinia._e.run(() => (scope = effectScope()).run(() => setup({ action }))));
  for (const key in setupStore) {
    const prop = setupStore[key];
    if (isRef(prop) && !isComputed(prop) || isReactive(prop)) {
      if (!isOptionsStore) {
        if (initialState && shouldHydrate(prop)) {
          if (isRef(prop)) {
            prop.value = initialState[key];
          } else {
            mergeReactiveObjects(prop, initialState[key]);
          }
        }
        {
          pinia.state.value[$id][key] = prop;
        }
      }
    } else if (typeof prop === "function") {
      const actionValue = action(prop, key);
      {
        setupStore[key] = actionValue;
      }
      optionsForPlugin.actions[key] = prop;
    } else ;
  }
  {
    assign(store, setupStore);
    assign(toRaw(store), setupStore);
  }
  Object.defineProperty(store, "$state", {
    get: () => pinia.state.value[$id],
    set: (state) => {
      $patch(($state) => {
        assign($state, state);
      });
    }
  });
  pinia._p.forEach((extender) => {
    {
      assign(store, scope.run(() => extender({
        store,
        app: pinia._a,
        pinia,
        options: optionsForPlugin
      })));
    }
  });
  if (initialState && isOptionsStore && options.hydrate) {
    options.hydrate(store.$state, initialState);
  }
  isListening = true;
  isSyncListening = true;
  return store;
}
// @__NO_SIDE_EFFECTS__
function defineStore(idOrOptions, setup, setupOptions) {
  let id;
  let options;
  const isSetupStore = typeof setup === "function";
  {
    id = idOrOptions;
    options = isSetupStore ? setupOptions : setup;
  }
  function useStore(pinia, hot) {
    const hasContext = hasInjectionContext();
    pinia = // in test mode, ignore the argument provided as we can always retrieve a
    // pinia instance with getActivePinia()
    pinia || (hasContext ? inject$1(piniaSymbol, null) : null);
    if (pinia)
      setActivePinia(pinia);
    pinia = activePinia;
    if (!pinia._s.has(id)) {
      if (isSetupStore) {
        createSetupStore(id, setup, options, pinia);
      } else {
        createOptionsStore(id, options, pinia);
      }
    }
    const store = pinia._s.get(id);
    return store;
  }
  useStore.$id = id;
  return useStore;
}
const P = {
  DashboardView: "dashboard.view",
  IncomeView: "income.view",
  IncomeCreate: "income.create",
  IncomeEdit: "income.edit",
  IncomeVoid: "income.void",
  ExpenseView: "expense.view",
  ExpenseCreate: "expense.create",
  ExpenseEdit: "expense.edit",
  ExpenseVoid: "expense.void",
  TransactionsView: "transactions.view",
  TransactionsApprove: "transactions.approve",
  TransactionsReject: "transactions.reject",
  TransactionsReverse: "transactions.reverse",
  ReportsView: "reports.view",
  ReportsExport: "reports.export",
  BranchesView: "branches.view",
  BranchesCreate: "branches.create",
  BranchesEdit: "branches.edit",
  BranchesManage: "branches.manage",
  UsersView: "users.view",
  UsersCreate: "users.create",
  UsersEdit: "users.edit",
  UsersManage: "users.manage",
  RolesView: "roles.view",
  RolesManage: "roles.manage",
  PermissionsManage: "permissions.manage",
  CategoriesManage: "categories.manage",
  CurrencySettingsManage: "currency-settings.manage",
  AuditLogsView: "audit-logs.view",
  SettingsManage: "settings.manage"
};
const PERMISSION_CATALOG = [
  { code: P.DashboardView, name: "View Dashboard", module: "Dashboard", description: "Access the main dashboard" },
  { code: P.IncomeView, name: "View Income", module: "Income", description: "View income transactions" },
  { code: P.IncomeCreate, name: "Create Income", module: "Income", description: "Create income transactions" },
  { code: P.IncomeEdit, name: "Edit Income", module: "Income", description: "Edit income transactions" },
  { code: P.IncomeVoid, name: "Void Income", module: "Income", description: "Void income transactions" },
  { code: P.ExpenseView, name: "View Expenses", module: "Expenses", description: "View expense transactions" },
  { code: P.ExpenseCreate, name: "Create Expenses", module: "Expenses", description: "Create expense transactions" },
  { code: P.ExpenseEdit, name: "Edit Expenses", module: "Expenses", description: "Edit expense transactions" },
  { code: P.ExpenseVoid, name: "Void Expenses", module: "Expenses", description: "Void expense transactions" },
  { code: P.TransactionsView, name: "View Transactions", module: "Transactions", description: "View all transactions" },
  { code: P.TransactionsApprove, name: "Approve Transactions", module: "Transactions", description: "Approve pending transactions" },
  { code: P.TransactionsReject, name: "Reject Transactions", module: "Transactions", description: "Reject pending transactions" },
  { code: P.TransactionsReverse, name: "Reverse Transactions", module: "Transactions", description: "Reverse finalized transactions" },
  { code: P.ReportsView, name: "View Reports", module: "Reports", description: "View financial reports" },
  { code: P.ReportsExport, name: "Export Reports", module: "Reports", description: "Export reports to CSV/Excel/PDF" },
  { code: P.BranchesView, name: "View Branches", module: "Branches", description: "View branch list and details" },
  { code: P.BranchesCreate, name: "Create Branches", module: "Branches", description: "Create new branches" },
  { code: P.BranchesEdit, name: "Edit Branches", module: "Branches", description: "Edit branch details" },
  { code: P.BranchesManage, name: "Manage Branches", module: "Branches", description: "Full branch management including assignments" },
  { code: P.UsersView, name: "View Users", module: "Users", description: "View user list and profiles" },
  { code: P.UsersCreate, name: "Create Users", module: "Users", description: "Create new users" },
  { code: P.UsersEdit, name: "Edit Users", module: "Users", description: "Edit user profiles" },
  { code: P.UsersManage, name: "Manage Users", module: "Users", description: "Manage user roles, passwords, and account activation" },
  { code: P.RolesView, name: "View Roles", module: "Roles", description: "View roles and their permissions" },
  { code: P.RolesManage, name: "Manage Roles", module: "Roles", description: "Create, edit, and delete non-system roles" },
  { code: P.PermissionsManage, name: "Manage Permissions", module: "Roles", description: "Assign permissions to roles" },
  { code: P.CategoriesManage, name: "Manage Categories", module: "Settings", description: "Create, edit, and deactivate income/expense categories" },
  { code: P.CurrencySettingsManage, name: "Manage Currency Settings", module: "Settings", description: "Configure currency options" },
  { code: P.AuditLogsView, name: "View Audit Logs", module: "Audit", description: "View system audit logs" },
  { code: P.SettingsManage, name: "Manage Settings", module: "Settings", description: "Change application settings" }
];
const ROUTE_PERMISSIONS = {
  "/dashboard": [P.DashboardView],
  "/transactions": [P.TransactionsView, P.IncomeView, P.ExpenseView],
  "/transactions/new": [P.IncomeCreate, P.ExpenseCreate],
  "/reports": [P.ReportsView, P.DashboardView],
  "/admin/branches": [P.BranchesView, P.BranchesManage],
  "/admin/users": [P.UsersView, P.UsersManage],
  "/admin/roles": [P.RolesView, P.RolesManage],
  "/admin/audit": [P.AuditLogsView],
  "/admin/permissions": [P.RolesView],
  "/admin/categories": [P.CategoriesManage]
};
const usePermissionsStore = /* @__PURE__ */ defineStore("permissions", {
  state: () => ({
    /** Permission codes granted to the current user */
    userCodes: [],
    isSuperAdmin: false,
    /** Full system catalog (static seed, then API overwrite) */
    catalog: PERMISSION_CATALOG.map((c) => ({
      code: c.code,
      name: c.name,
      module: c.module,
      description: c.description
    })),
    catalogSynced: false,
    catalogSyncError: null,
    lastSyncedAt: null
  }),
  getters: {
    codeSet: (s) => new Set(s.userCodes),
    hasAny: (s) => {
      return (...codes) => {
        if (s.isSuperAdmin) return true;
        if (!codes.length) return true;
        return codes.some((c) => s.userCodes.includes(c));
      };
    },
    hasAll: (s) => {
      return (...codes) => {
        if (s.isSuperAdmin) return true;
        return codes.every((c) => s.userCodes.includes(c));
      };
    },
    modules: (s) => {
      const set = new Set(s.catalog.map((c) => c.module));
      return Array.from(set).sort();
    }
  },
  actions: {
    /** Apply user grants from /auth/me */
    setUserGrants(codes, isSuperAdmin) {
      this.userCodes = [...codes];
      this.isSuperAdmin = isSuperAdmin;
    },
    clearUser() {
      this.userCodes = [];
      this.isSuperAdmin = false;
    },
    /**
     * Fetch full permission catalog from API.
     * Requires roles.view (or super admin). Fails softly for other roles.
     */
    async syncCatalogFromApi(api) {
      try {
        const res = await api("/permissions");
        if (res.items?.length) {
          this.catalog = res.items.map((p) => ({
            id: p.id,
            code: p.code,
            name: p.name,
            module: p.module || "Other",
            description: p.description || ""
          }));
          this.catalogSynced = true;
          this.catalogSyncError = null;
          this.lastSyncedAt = (/* @__PURE__ */ new Date()).toISOString();
        }
      } catch (e) {
        this.catalogSyncError = e?.data?.error || e?.message || "sync failed";
        this.catalogSynced = false;
      }
    }
  }
});
const accessToken = ref(null);
const currentUser = ref(null);
const authReady = ref(false);
function useAuth() {
  const apiBase = useApiBase();
  async function login(identifier, password) {
    const res = await $fetch(`${apiBase}/auth/login`, {
      method: "POST",
      body: { identifier, password },
      credentials: "include"
    });
    accessToken.value = res.access_token;
    await fetchMe();
    return res;
  }
  async function fetchMe() {
    if (!accessToken.value) {
      currentUser.value = null;
      return null;
    }
    try {
      const me = await $fetch(`${apiBase}/auth/me`, {
        headers: { Authorization: `Bearer ${accessToken.value}` },
        credentials: "include"
      });
      currentUser.value = me;
      try {
        const store = usePermissionsStore();
        store.setUserGrants(me.permissions || [], !!me.is_super_admin);
      } catch {
      }
      return me;
    } catch {
      accessToken.value = null;
      currentUser.value = null;
      try {
        usePermissionsStore().clearUser();
      } catch {
      }
      return null;
    }
  }
  async function logout() {
    try {
      if (accessToken.value) {
        await $fetch(`${apiBase}/auth/logout`, {
          method: "POST",
          headers: { Authorization: `Bearer ${accessToken.value}` },
          credentials: "include"
        });
      }
    } catch {
    }
    accessToken.value = null;
    currentUser.value = null;
    try {
      usePermissionsStore().clearUser();
    } catch {
    }
  }
  async function tryRefresh() {
    try {
      const res = await $fetch(`${apiBase}/auth/refresh`, {
        method: "POST",
        credentials: "include"
      });
      accessToken.value = res.access_token;
      await fetchMe();
      return true;
    } catch {
      accessToken.value = null;
      currentUser.value = null;
      return false;
    }
  }
  function hasPermission(...codes) {
    if (!currentUser.value) return false;
    if (currentUser.value.is_super_admin) return true;
    const perms = currentUser.value.permissions || [];
    return codes.some((c) => perms.includes(c));
  }
  function getToken() {
    return accessToken.value;
  }
  async function initAuth() {
    if (authReady.value) return;
    if (!accessToken.value) {
      await tryRefresh();
    } else {
      await fetchMe();
    }
    authReady.value = true;
  }
  return {
    accessToken: readonly(accessToken),
    user: readonly(currentUser),
    authReady: readonly(authReady),
    login,
    logout,
    fetchMe,
    tryRefresh,
    hasPermission,
    getToken,
    initAuth
  };
}
const auth_45global = /* @__PURE__ */ defineNuxtRouteMiddleware(async (to) => {
  const publicPaths = ["/login", "/access-denied"];
  if (publicPaths.includes(to.path)) return;
  return;
});
function useApi() {
  const apiBase = useApiBase();
  const { getToken, tryRefresh, logout } = useAuth();
  async function api(path, opts = {}) {
    const headers = {};
    const token = getToken();
    if (token) headers.Authorization = `Bearer ${token}`;
    const query = opts.query ? "?" + Object.entries(opts.query).filter(([, v]) => v !== void 0 && v !== "").map(([k, v]) => `${encodeURIComponent(k)}=${encodeURIComponent(String(v))}`).join("&") : "";
    try {
      return await $fetch(`${apiBase}${path}${query}`, {
        method: opts.method || "GET",
        body: opts.body,
        headers,
        credentials: "include"
      });
    } catch (e) {
      if (e?.status === 401 || e?.statusCode === 401) {
        const ok = await tryRefresh();
        if (ok) {
          const t2 = getToken();
          if (t2) headers.Authorization = `Bearer ${t2}`;
          return await $fetch(`${apiBase}${path}${query}`, {
            method: opts.method || "GET",
            body: opts.body,
            headers,
            credentials: "include"
          });
        }
        await logout();
      }
      throw e;
    }
  }
  return { api, apiBase };
}
function usePermissions() {
  const store = usePermissionsStore();
  const { user, hasPermission } = useAuth();
  const isSuperAdmin = computed(
    () => store.isSuperAdmin || !!user.value?.is_super_admin
  );
  const permissions = computed(() => {
    if (store.userCodes.length) return store.userCodes;
    return user.value?.permissions || [];
  });
  const catalog = computed(() => store.catalog);
  const catalogSynced = computed(() => store.catalogSynced);
  const catalogSyncError = computed(() => store.catalogSyncError);
  function can(...codes) {
    if (!codes.length) return true;
    if (isSuperAdmin.value) return true;
    if (store.userCodes.length) {
      return codes.some((c) => store.userCodes.includes(c));
    }
    return hasPermission(...codes);
  }
  function canAll(...codes) {
    if (isSuperAdmin.value) return true;
    if (store.userCodes.length) {
      return codes.every((c) => store.userCodes.includes(c));
    }
    if (!user.value) return false;
    const set = new Set(user.value.permissions || []);
    return codes.every((c) => set.has(c));
  }
  function canAccessRoute(path) {
    const entry2 = Object.entries(ROUTE_PERMISSIONS).find(([route]) => {
      if (path === route) return true;
      if (route !== "/" && path.startsWith(route + "/")) return true;
      return false;
    });
    if (!entry2) return true;
    return can(...entry2[1]);
  }
  async function refreshCatalog() {
    const { api } = useApi();
    await store.syncCatalogFromApi(api);
  }
  const canViewDashboard = computed(() => can(P.DashboardView));
  const canViewTransactions = computed(() => can(P.TransactionsView, P.IncomeView, P.ExpenseView));
  const canCreateIncome = computed(() => can(P.IncomeCreate));
  const canCreateExpense = computed(() => can(P.ExpenseCreate));
  const canCreateTransaction = computed(() => can(P.IncomeCreate, P.ExpenseCreate));
  const canEditIncome = computed(() => can(P.IncomeEdit));
  const canEditExpense = computed(() => can(P.ExpenseEdit));
  const canVoidIncome = computed(() => can(P.IncomeVoid));
  const canVoidExpense = computed(() => can(P.ExpenseVoid));
  const canApprove = computed(() => can(P.TransactionsApprove));
  const canReject = computed(() => can(P.TransactionsReject));
  const canReverse = computed(() => can(P.TransactionsReverse));
  const canViewReports = computed(() => can(P.ReportsView, P.DashboardView));
  const canExportReports = computed(() => can(P.ReportsExport));
  const canManageBranches = computed(() => can(P.BranchesCreate, P.BranchesEdit, P.BranchesManage));
  const canCreateBranch = computed(() => can(P.BranchesCreate, P.BranchesManage));
  const canViewBranches = computed(() => can(P.BranchesView, P.BranchesManage));
  const canViewUsers = computed(() => can(P.UsersView, P.UsersManage));
  const canCreateUser = computed(() => can(P.UsersCreate, P.UsersManage));
  const canViewRoles = computed(() => can(P.RolesView, P.RolesManage));
  const canViewAudit = computed(() => can(P.AuditLogsView));
  const canManageSettings = computed(() => can(P.SettingsManage));
  return {
    P,
    catalog,
    catalogSynced,
    catalogSyncError,
    refreshCatalog,
    isSuperAdmin,
    permissions,
    can,
    canAll,
    canAccessRoute,
    canViewDashboard,
    canViewTransactions,
    canCreateIncome,
    canCreateExpense,
    canCreateTransaction,
    canEditIncome,
    canEditExpense,
    canVoidIncome,
    canVoidExpense,
    canApprove,
    canReject,
    canReverse,
    canViewReports,
    canExportReports,
    canManageBranches,
    canCreateBranch,
    canViewBranches,
    canViewUsers,
    canCreateUser,
    canViewRoles,
    canViewAudit,
    canManageSettings
  };
}
const permission_45global = /* @__PURE__ */ defineNuxtRouteMiddleware((to) => {
  if (to.path === "/login" || to.path === "/access-denied") return;
  const { user, authReady: authReady2 } = useAuth();
  if (!authReady2.value || !user.value) return;
  const { canAccessRoute } = usePermissions();
  const matched = Object.keys(ROUTE_PERMISSIONS).filter((r) => to.path === r || to.path.startsWith(r + "/")).sort((a, b) => b.length - a.length)[0];
  if (matched && !canAccessRoute(matched)) {
    return navigateTo("/access-denied");
  }
});
const manifest_45route_45rule = /* @__PURE__ */ defineNuxtRouteMiddleware((to) => {
  {
    return;
  }
});
const globalMiddleware = [
  validate,
  auth_45global,
  permission_45global,
  manifest_45route_45rule
];
const namedMiddleware = {};
Object.assign(/* @__PURE__ */ Object.create(null), {});
const pageIslandRoutes = Object.assign(/* @__PURE__ */ Object.create(null), {});
const plugin$1 = /* @__PURE__ */ defineNuxtPlugin({
  name: "nuxt:router",
  enforce: "pre",
  async setup(nuxtApp) {
    let __temp, __restore;
    let routerBase = (/* @__PURE__ */ useRuntimeConfig()).app.baseURL;
    const history = routerOptions.history?.(routerBase) ?? createMemoryHistory(routerBase);
    const routes = routerOptions.routes ? ([__temp, __restore] = executeAsync(() => routerOptions.routes(_routes)), __temp = await __temp, __restore(), __temp) ?? _routes : _routes;
    let startPosition;
    const router = createRouter({
      ...routerOptions,
      scrollBehavior: (to, from, savedPosition) => {
        if (from === START_LOCATION) {
          startPosition = savedPosition;
          return;
        }
        if (routerOptions.scrollBehavior) {
          router.options.scrollBehavior = routerOptions.scrollBehavior;
          if ("scrollRestoration" in (void 0).history) {
            const unsub = router.beforeEach(() => {
              unsub();
              (void 0).history.scrollRestoration = "manual";
            });
          }
          return routerOptions.scrollBehavior(to, START_LOCATION, startPosition || savedPosition);
        }
      },
      history,
      routes
    });
    nuxtApp.vueApp.use(router);
    const previousRoute = shallowRef(router.currentRoute.value);
    router.afterEach((_to, from) => {
      previousRoute.value = from;
    });
    Object.defineProperty(nuxtApp.vueApp.config.globalProperties, "previousRoute", {
      get: () => previousRoute.value
    });
    const initialURL = nuxtApp.ssrContext.url;
    const _route = shallowRef(router.currentRoute.value);
    const syncCurrentRoute = () => {
      _route.value = router.currentRoute.value;
    };
    router.afterEach((to, from) => {
      const lastTo = to.matched.at(-1)?.components?.default;
      const lastFrom = from.matched.at(-1)?.components?.default;
      if (lastTo === lastFrom) {
        const toKey = generateRouteKey$1({ route: to, Component: { type: lastTo } });
        const fromKey = generateRouteKey$1({ route: from, Component: { type: lastFrom } });
        if (toKey === fromKey) {
          syncCurrentRoute();
        }
        return;
      }
      if (to.matched.length < from.matched.length && to.matched.every((m, i) => m.components?.default === from.matched[i]?.components?.default)) {
        syncCurrentRoute();
      }
    });
    const route = { sync: syncCurrentRoute };
    for (const key in _route.value) {
      Object.defineProperty(route, key, {
        get: () => _route.value[key],
        enumerable: true
      });
    }
    nuxtApp._route = shallowReactive(route);
    nuxtApp._middleware ||= {
      global: [],
      named: {}
    };
    const error = /* @__PURE__ */ useError();
    const isServerPage = nuxtApp.ssrContext?.islandContext?.name?.startsWith("page_");
    if (!nuxtApp.ssrContext?.islandContext || isServerPage) {
      router.afterEach(async (to, _from, failure) => {
        delete nuxtApp._processingMiddleware;
        {
          delete nuxtApp._middlewareTo;
        }
        if (failure) {
          await nuxtApp.callHook("page:loading:end");
        }
        if (failure?.type === 4) {
          return;
        }
        if (to.redirectedFrom && to.fullPath !== initialURL) {
          await nuxtApp.runWithContext(() => navigateTo(to.fullPath || "/"));
        }
      });
    }
    try {
      if (true) {
        ;
        [__temp, __restore] = executeAsync(() => router.push(initialURL)), await __temp, __restore();
        ;
      }
      ;
      [__temp, __restore] = executeAsync(() => router.isReady()), await __temp, __restore();
      ;
    } catch (error2) {
      [__temp, __restore] = executeAsync(() => nuxtApp.runWithContext(() => showError(error2))), await __temp, __restore();
    }
    const resolvedInitialRoute = router.currentRoute.value;
    const hasDeferredRoute = false;
    syncCurrentRoute();
    if (nuxtApp.ssrContext?.islandContext && !isServerPage) {
      return { provide: { router } };
    }
    function pushErroredRoute(to) {
    }
    const initialLayout = nuxtApp.payload.state._layout;
    router.beforeEach(async (to, from) => {
      await nuxtApp.callHook("page:loading:start");
      to.meta = reactive(to.meta);
      if (nuxtApp.isHydrating && initialLayout && !isReadonly(to.meta.layout)) {
        to.meta.layout = initialLayout;
      }
      nuxtApp._processingMiddleware = true;
      {
        nuxtApp._middlewareTo = to;
      }
      if (!nuxtApp.ssrContext?.islandContext || isServerPage) {
        const middlewareEntries = /* @__PURE__ */ new Set([...globalMiddleware, ...nuxtApp._middleware.global]);
        for (const component of to.matched) {
          const componentMiddleware = component.meta.middleware;
          if (!componentMiddleware) {
            continue;
          }
          for (const entry2 of toArray$1(componentMiddleware)) {
            middlewareEntries.add(entry2);
          }
        }
        const routeRules = getRouteRules({ path: to.path });
        if (routeRules.appMiddleware) {
          for (const key in routeRules.appMiddleware) {
            if (routeRules.appMiddleware[key]) {
              middlewareEntries.add(key);
            } else {
              middlewareEntries.delete(key);
            }
          }
        }
        for (const entry2 of middlewareEntries) {
          const middleware = typeof entry2 === "string" ? nuxtApp._middleware.named[entry2] || await namedMiddleware[entry2]?.().then((r) => r.default || r) : entry2;
          if (!middleware) {
            throw new Error(`Unknown route middleware: '${entry2}'.`);
          }
          try {
            if (false) ;
            const result = await nuxtApp.runWithContext(() => middleware(to, from));
            if (true) {
              if (result === false || result instanceof Error) {
                const error2 = result || createError({
                  status: 404,
                  statusText: `Page Not Found: ${initialURL}`
                });
                await nuxtApp.runWithContext(() => showError(error2));
                return false;
              }
            }
            if (result === true) {
              continue;
            }
            if (result === false) {
              return result;
            }
            if (result) {
              if (isNuxtError(result) && result.fatal) {
                await nuxtApp.runWithContext(() => showError(result));
                pushErroredRoute(to);
              }
              return result;
            }
          } catch (err) {
            const error2 = createError(err);
            if (error2.fatal) {
              await nuxtApp.runWithContext(() => showError(error2));
            }
            return error2;
          }
        }
      }
    });
    if (isServerPage) {
      router.beforeResolve((to) => {
        const expected = pageIslandRoutes[nuxtApp.ssrContext.islandContext.name];
        const actual = to.matched.find((m) => m.components?.default?.__nuxt_island)?.components?.default;
        if (!expected || expected !== actual?.__nuxt_island) {
          nuxtApp.ssrContext["~renderResponse"] = {
            statusCode: 400,
            statusMessage: "Invalid island request path"
          };
          return false;
        }
      });
    }
    router.onError(async () => {
      delete nuxtApp._processingMiddleware;
      {
        delete nuxtApp._middlewareTo;
      }
      await nuxtApp.callHook("page:loading:end");
    });
    router.afterEach((to) => {
      if (to.matched.length === 0 && !error.value) {
        return nuxtApp.runWithContext(() => showError(createError({
          status: 404,
          fatal: false,
          statusText: `Page not found: ${to.fullPath}`,
          data: {
            path: to.fullPath
          }
        })));
      }
    });
    nuxtApp.hooks.hookOnce("app:created", async () => {
      try {
        if ("name" in resolvedInitialRoute) {
          resolvedInitialRoute.name = void 0;
        }
        const pluginNavigatedAway = false;
        if (pluginNavigatedAway) ;
        else if (hasDeferredRoute) ;
        else {
          await router.replace({
            ...resolvedInitialRoute,
            force: true
          });
        }
        router.options.scrollBehavior = routerOptions.scrollBehavior;
      } catch (error2) {
        await nuxtApp.runWithContext(() => showError(error2));
      }
    });
    return { provide: { router } };
  }
});
function definePayloadReducer(name, reduce) {
  {
    useNuxtApp().ssrContext["~payloadReducers"][name] = reduce;
  }
}
const reducers = [
  ["NuxtError", (data) => isNuxtError(data) && data.toJSON()],
  ["EmptyShallowRef", (data) => isRef(data) && isShallow(data) && !data.value && (typeof data.value === "bigint" ? "0n" : JSON.stringify(data.value) || "_")],
  ["EmptyRef", (data) => isRef(data) && !data.value && (typeof data.value === "bigint" ? "0n" : JSON.stringify(data.value) || "_")],
  ["ShallowRef", (data) => isRef(data) && isShallow(data) && data.value],
  ["ShallowReactive", (data) => isReactive(data) && isShallow(data) && toRaw(data)],
  ["Ref", (data) => isRef(data) && data.value],
  ["Reactive", (data) => isReactive(data) && toRaw(data)]
];
const revive_payload_server_MVtmlZaQpj6ApFmshWfUWl5PehCebzaBf2NuRMiIbms = /* @__PURE__ */ defineNuxtPlugin({
  name: "nuxt:revive-payload:server",
  setup() {
    for (const [reducer, fn] of reducers) {
      definePayloadReducer(reducer, fn);
    }
  }
});
defineComponent$1({
  name: "ServerPlaceholder",
  render() {
    return createElementBlock("div");
  }
});
const clientOnlySymbol = /* @__PURE__ */ Symbol.for("nuxt:client-only");
const __nuxt_component_0$1 = defineComponent$1({
  name: "ClientOnly",
  inheritAttrs: false,
  props: ["fallback", "placeholder", "placeholderTag", "fallbackTag"],
  ...false,
  setup(props, { slots, attrs }) {
    const mounted2 = shallowRef(false);
    const vm = getCurrentInstance$1();
    if (vm) {
      vm._nuxtClientOnly = true;
    }
    provide(clientOnlySymbol, true);
    return () => {
      if (mounted2.value) {
        const vnodes = slots.default?.();
        if (vnodes && vnodes.length === 1) {
          return [cloneVNode(vnodes[0], attrs)];
        }
        return vnodes;
      }
      const slot = slots.fallback || slots.placeholder;
      if (slot) {
        return h(slot);
      }
      const fallbackStr = props.fallback || props.placeholder || "";
      const fallbackTag = sanitizeTag(props.fallbackTag || props.placeholderTag, "span");
      return createElementBlock(fallbackTag, attrs, fallbackStr);
    };
  }
});
const plugin = /* @__PURE__ */ defineNuxtPlugin({
  name: "pinia",
  setup(nuxtApp) {
    const pinia = createPinia();
    nuxtApp.vueApp.use(pinia);
    setActivePinia(pinia);
    {
      nuxtApp.payload.pinia = pinia.state.value;
    }
    return {
      provide: {
        pinia
      }
    };
  }
});
const components_plugin_z4hgvsiddfKkfXTP6M8M4zG5Cb7sGnDhcryKVM45Di4 = /* @__PURE__ */ defineNuxtPlugin({
  name: "nuxt:global-components"
});
function consoleWarn(message) {
  warn(`Vuetify: ${message}`);
}
function consoleError(message) {
  warn(`Vuetify error: ${message}`);
}
function deprecate(original, replacement) {
  replacement = Array.isArray(replacement) ? replacement.slice(0, -1).map((s) => `'${s}'`).join(", ") + ` or '${replacement.at(-1)}'` : `'${replacement}'`;
  warn(`[Vuetify UPGRADE] '${original}' is deprecated, use ${replacement} instead.`);
}
const IN_BROWSER = false;
const SUPPORTS_TOUCH = IN_BROWSER;
const SUPPORTS_MATCH_MEDIA = IN_BROWSER;
const PREFERS_REDUCED_MOTION = () => SUPPORTS_MATCH_MEDIA;
function getNestedValue(obj, path, fallback) {
  const last = path.length - 1;
  if (last < 0) return isUndefined(obj) ? fallback : obj;
  for (let i = 0; i < last; i++) {
    if (isNullOrUndefined(obj)) {
      return fallback;
    }
    obj = obj[path[i]];
  }
  if (isNullOrUndefined(obj)) return fallback;
  return obj[path[last]] === void 0 ? fallback : obj[path[last]];
}
function getObjectValueByPath(obj, path, fallback) {
  if (isNullOrUndefined(obj) || !path || !isString(path)) return fallback;
  if (!isUndefined(obj[path])) return obj[path];
  path = path.replace(/\[(\w+)\]/g, ".$1");
  path = path.replace(/^\./, "");
  return getNestedValue(obj, path.split("."), fallback);
}
function convertToUnit(str, unit = "px") {
  if (isNullOrUndefined(str) || str === "") {
    return void 0;
  }
  const num = Number(str);
  if (isNaN(num)) {
    return String(str);
  } else if (!isFinite(num)) {
    return void 0;
  } else {
    return `${num}${unit}`;
  }
}
function isPlainObject(obj) {
  let proto;
  return !!obj && typeof obj === "object" && ((proto = Object.getPrototypeOf(obj)) === Object.prototype || !proto);
}
function refElement(obj) {
  if (obj && "$el" in obj) {
    const el = obj.$el;
    if (el?.nodeType === Node.TEXT_NODE) {
      return el.nextElementSibling;
    }
    return el;
  }
  return obj;
}
function has(obj, key) {
  return key.every((k) => obj.hasOwnProperty(k));
}
function pick(obj, paths) {
  const found = {};
  for (const key of paths) {
    if (Object.prototype.hasOwnProperty.call(obj, key)) {
      found[key] = obj[key];
    }
  }
  return found;
}
function pickWithRest(obj, paths, exclude) {
  const found = /* @__PURE__ */ Object.create(null);
  const rest = /* @__PURE__ */ Object.create(null);
  for (const key in obj) {
    if (paths.some((path) => path instanceof RegExp ? path.test(key) : path === key) && true) {
      found[key] = obj[key];
    } else {
      rest[key] = obj[key];
    }
  }
  return [found, rest];
}
function omit(obj, exclude) {
  const clone = {
    ...obj
  };
  exclude.forEach((prop) => delete clone[prop]);
  return clone;
}
function wrapInArray(v) {
  return isNullOrUndefined(v) ? [] : Array.isArray(v) ? v : [v];
}
function clamp(value, min = 0, max = 1) {
  return Math.max(min, Math.min(max, value));
}
function padEnd(str, length, char = "0") {
  return str + char.repeat(Math.max(0, length - str.length));
}
function padStart(str, length, char = "0") {
  return char.repeat(Math.max(0, length - str.length)) + str;
}
function chunk(str, size = 1) {
  const chunked = [];
  let index = 0;
  while (index < str.length) {
    chunked.push(str.substr(index, size));
    index += size;
  }
  return chunked;
}
function chunkArray(array, size = 1) {
  return Array.from({
    length: Math.ceil(array.length / size)
  }, (v, i) => array.slice(i * size, i * size + size));
}
function mergeDeep(source = {}, target = {}, arrayFn, targetCondition) {
  const out = {};
  for (const key in source) {
    out[key] = source[key];
  }
  for (const key in target) {
    const targetProperty = target[key];
    if (targetCondition && !targetCondition(key, targetProperty)) {
      continue;
    }
    const sourceProperty = source[key];
    if (isPlainObject(sourceProperty) && isPlainObject(targetProperty)) {
      out[key] = mergeDeep(sourceProperty, targetProperty, arrayFn, targetCondition);
      continue;
    }
    out[key] = targetProperty;
  }
  return out;
}
function flattenFragments(nodes) {
  return nodes.map((node) => {
    if (node.type === Fragment) {
      return flattenFragments(node.children);
    } else {
      return node;
    }
  }).flat();
}
function toKebabCase(str = "") {
  if (toKebabCase.cache.has(str)) return toKebabCase.cache.get(str);
  const kebab = str.replace(/[^a-z]/gi, "-").replace(/\B([A-Z])/g, "-$1").toLowerCase();
  toKebabCase.cache.set(str, kebab);
  return kebab;
}
toKebabCase.cache = /* @__PURE__ */ new Map();
function findChildrenWithProvide(key, vnode) {
  if (!vnode || typeof vnode !== "object") return [];
  if (Array.isArray(vnode)) {
    return vnode.map((child) => findChildrenWithProvide(key, child)).flat(1);
  } else if (vnode.suspense) {
    return findChildrenWithProvide(key, vnode.ssContent);
  } else if (Array.isArray(vnode.children)) {
    return vnode.children.map((child) => findChildrenWithProvide(key, child)).flat(1);
  } else if (vnode.component) {
    if (Object.getOwnPropertyDescriptor(vnode.component.provides, key)) {
      return [vnode.component];
    } else if (vnode.component.subTree) {
      return findChildrenWithProvide(key, vnode.component.subTree).flat(1);
    }
  }
  return [];
}
function destructComputed(getter) {
  const base = computed(getter);
  const obj = {};
  for (const key in base.value) {
    obj[key] = toRef(() => base.value[key]);
  }
  return obj;
}
function includes(arr, val) {
  return arr.includes(val);
}
const EventProp = () => [Function, Array];
function hasEvent(props, name) {
  name = "on" + capitalize(name);
  return !!(props[name] || props[`${name}Once`] || props[`${name}Capture`] || props[`${name}OnceCapture`] || props[`${name}CaptureOnce`]);
}
function templateRef() {
  const el = shallowRef();
  const fn = (target) => {
    el.value = target;
  };
  Object.defineProperty(fn, "value", {
    enumerable: true,
    get: () => el.value,
    set: (val) => el.value = val
  });
  Object.defineProperty(fn, "el", {
    enumerable: true,
    get: () => refElement(el.value)
  });
  return fn;
}
function isPrimitive(value) {
  return isString(value) || isNumber(value) || isBoolean(value) || typeof value === "bigint";
}
function onlyDefinedProps(props) {
  const booleanAttributes = ["checked", "disabled"];
  return Object.fromEntries(Object.entries(props).filter(([key, v]) => booleanAttributes.includes(key) ? !!v : !isUndefined(v)));
}
const block = ["top", "bottom"];
const inline = ["start", "end", "left", "right"];
function parseAnchor(anchor, isRtl) {
  let [side, align] = anchor.split(" ");
  if (!align) {
    align = includes(block, side) ? "start" : includes(inline, side) ? "top" : "center";
  }
  return {
    side: toPhysical(side, isRtl),
    align: toPhysical(align, isRtl)
  };
}
function toPhysical(str, isRtl) {
  if (str === "start") return isRtl ? "right" : "left";
  if (str === "end") return isRtl ? "left" : "right";
  return str;
}
class Box {
  constructor(args) {
    if (isElement(args)) {
      const pageScale = (void 0).body.currentCSSZoom ?? 1;
      const factor = 1 + (1 - pageScale) / pageScale;
      const {
        x,
        y,
        width,
        height
      } = args.getBoundingClientRect();
      this.x = x * factor;
      this.y = y * factor;
      this.width = width * factor;
      this.height = height * factor;
    } else {
      this.x = args.x;
      this.y = args.y;
      this.width = args.width;
      this.height = args.height;
    }
  }
  get top() {
    return this.y;
  }
  get bottom() {
    return this.y + this.height;
  }
  get left() {
    return this.x;
  }
  get right() {
    return this.x + this.width;
  }
}
function getTargetBox(target) {
  if (Array.isArray(target)) {
    const pageScale = (void 0).body.currentCSSZoom ?? 1;
    const factor = 1 + (1 - pageScale) / pageScale;
    return new Box({
      x: target[0] * factor,
      y: target[1] * factor,
      width: 0 * factor,
      height: 0 * factor
    });
  } else {
    return new Box(target);
  }
}
const mainTRC = 2.4;
const Rco = 0.2126729;
const Gco = 0.7151522;
const Bco = 0.072175;
const normBG = 0.55;
const normTXT = 0.58;
const revTXT = 0.57;
const revBG = 0.62;
const blkThrs = 0.03;
const blkClmp = 1.45;
const deltaYmin = 5e-4;
const scaleBoW = 1.25;
const scaleWoB = 1.25;
const loConThresh = 0.078;
const loConFactor = 12.82051282051282;
const loConOffset = 0.06;
const loClip = 1e-3;
function APCAcontrast(text, background) {
  const Rtxt = (text.r / 255) ** mainTRC;
  const Gtxt = (text.g / 255) ** mainTRC;
  const Btxt = (text.b / 255) ** mainTRC;
  const Rbg = (background.r / 255) ** mainTRC;
  const Gbg = (background.g / 255) ** mainTRC;
  const Bbg = (background.b / 255) ** mainTRC;
  let Ytxt = Rtxt * Rco + Gtxt * Gco + Btxt * Bco;
  let Ybg = Rbg * Rco + Gbg * Gco + Bbg * Bco;
  if (Ytxt <= blkThrs) Ytxt += (blkThrs - Ytxt) ** blkClmp;
  if (Ybg <= blkThrs) Ybg += (blkThrs - Ybg) ** blkClmp;
  if (Math.abs(Ybg - Ytxt) < deltaYmin) return 0;
  let outputContrast;
  if (Ybg > Ytxt) {
    const SAPC = (Ybg ** normBG - Ytxt ** normTXT) * scaleBoW;
    outputContrast = SAPC < loClip ? 0 : SAPC < loConThresh ? SAPC - SAPC * loConFactor * loConOffset : SAPC - loConOffset;
  } else {
    const SAPC = (Ybg ** revBG - Ytxt ** revTXT) * scaleWoB;
    outputContrast = SAPC > -loClip ? 0 : SAPC > -loConThresh ? SAPC - SAPC * loConFactor * loConOffset : SAPC + loConOffset;
  }
  return outputContrast * 100;
}
const delta = 0.20689655172413793;
const cielabForwardTransform = (t) => t > delta ** 3 ? Math.cbrt(t) : t / (3 * delta ** 2) + 4 / 29;
const cielabReverseTransform = (t) => t > delta ? t ** 3 : 3 * delta ** 2 * (t - 4 / 29);
function fromXYZ$1(xyz) {
  const transform2 = cielabForwardTransform;
  const transformedY = transform2(xyz[1]);
  return [116 * transformedY - 16, 500 * (transform2(xyz[0] / 0.95047) - transformedY), 200 * (transformedY - transform2(xyz[2] / 1.08883))];
}
function toXYZ$1(lab) {
  const transform2 = cielabReverseTransform;
  const Ln = (lab[0] + 16) / 116;
  return [transform2(Ln + lab[1] / 500) * 0.95047, transform2(Ln), transform2(Ln - lab[2] / 200) * 1.08883];
}
const srgbForwardMatrix = [[3.2406, -1.5372, -0.4986], [-0.9689, 1.8758, 0.0415], [0.0557, -0.204, 1.057]];
const srgbForwardTransform = (C) => C <= 31308e-7 ? C * 12.92 : 1.055 * C ** (1 / 2.4) - 0.055;
const srgbReverseMatrix = [[0.4124, 0.3576, 0.1805], [0.2126, 0.7152, 0.0722], [0.0193, 0.1192, 0.9505]];
const srgbReverseTransform = (C) => C <= 0.04045 ? C / 12.92 : ((C + 0.055) / 1.055) ** 2.4;
function fromXYZ(xyz) {
  const rgb = Array(3);
  const transform2 = srgbForwardTransform;
  const matrix = srgbForwardMatrix;
  for (let i = 0; i < 3; ++i) {
    rgb[i] = Math.round(clamp(transform2(matrix[i][0] * xyz[0] + matrix[i][1] * xyz[1] + matrix[i][2] * xyz[2])) * 255);
  }
  return {
    r: rgb[0],
    g: rgb[1],
    b: rgb[2]
  };
}
function toXYZ({
  r,
  g,
  b
}) {
  const xyz = [0, 0, 0];
  const transform2 = srgbReverseTransform;
  const matrix = srgbReverseMatrix;
  r = transform2(r / 255);
  g = transform2(g / 255);
  b = transform2(b / 255);
  for (let i = 0; i < 3; ++i) {
    xyz[i] = matrix[i][0] * r + matrix[i][1] * g + matrix[i][2] * b;
  }
  return xyz;
}
function isCssColor(color) {
  return !!color && /^(#|var\(--|(rgb|hsl)a?\()/.test(color);
}
function isParsableColor(color) {
  return isCssColor(color) && !/^((rgb|hsl)a?\()?var\(--/.test(color);
}
const cssColorRe = /^(?<fn>(?:rgb|hsl)a?)\((?<values>.+)\)/;
const mappers = {
  rgb: (r, g, b, a) => ({
    r,
    g,
    b,
    a
  }),
  rgba: (r, g, b, a) => ({
    r,
    g,
    b,
    a
  }),
  hsl: (h2, s, l, a) => HSLtoRGB({
    h: h2,
    s,
    l,
    a
  }),
  hsla: (h2, s, l, a) => HSLtoRGB({
    h: h2,
    s,
    l,
    a
  }),
  hsv: (h2, s, v, a) => HSVtoRGB({
    h: h2,
    s,
    v,
    a
  }),
  hsva: (h2, s, v, a) => HSVtoRGB({
    h: h2,
    s,
    v,
    a
  })
};
function parseColor(color) {
  if (isNumber(color)) {
    if (isNaN(color) || color < 0 || color > 16777215) {
      consoleWarn(`'${color}' is not a valid hex color`);
    }
    return {
      r: (color & 16711680) >> 16,
      g: (color & 65280) >> 8,
      b: color & 255
    };
  } else if (isString(color) && cssColorRe.test(color)) {
    const {
      groups
    } = color.match(cssColorRe);
    const {
      fn,
      values
    } = groups;
    const realValues = values.split(/,\s*|\s*\/\s*|\s+/).map((v, i) => {
      if (v.endsWith("%") || // unitless slv are %
      i > 0 && i < 3 && ["hsl", "hsla", "hsv", "hsva"].includes(fn)) {
        return parseFloat(v) / 100;
      } else {
        return parseFloat(v);
      }
    });
    return mappers[fn](...realValues);
  } else if (isString(color)) {
    let hex = color.startsWith("#") ? color.slice(1) : color;
    if ([3, 4].includes(hex.length)) {
      hex = hex.split("").map((char) => char + char).join("");
    } else if (![6, 8].includes(hex.length)) {
      consoleWarn(`'${color}' is not a valid hex(a) color`);
    }
    const int = parseInt(hex, 16);
    if (isNaN(int) || int < 0 || int > 4294967295) {
      consoleWarn(`'${color}' is not a valid hex(a) color`);
    }
    return HexToRGB(hex);
  } else if (isObject(color)) {
    if (has(color, ["r", "g", "b"])) {
      return color;
    } else if (has(color, ["h", "s", "l"])) {
      return HSVtoRGB(HSLtoHSV(color));
    } else if (has(color, ["h", "s", "v"])) {
      return HSVtoRGB(color);
    }
  }
  throw new TypeError(`Invalid color: ${isNullOrUndefined(color) ? color : String(color) || color.constructor.name}
Expected #hex, #hexa, rgb(), rgba(), hsl(), hsla(), object or number`);
}
function HSVtoRGB(hsva) {
  const {
    h: h2,
    s,
    v,
    a
  } = hsva;
  const f = (n) => {
    const k = (n + h2 / 60) % 6;
    return v - v * s * Math.max(Math.min(k, 4 - k, 1), 0);
  };
  const rgb = [f(5), f(3), f(1)].map((v2) => Math.round(v2 * 255));
  return {
    r: rgb[0],
    g: rgb[1],
    b: rgb[2],
    a
  };
}
function HSLtoRGB(hsla) {
  return HSVtoRGB(HSLtoHSV(hsla));
}
function HSLtoHSV(hsl) {
  const {
    h: h2,
    s,
    l,
    a
  } = hsl;
  const v = l + s * Math.min(l, 1 - l);
  const sprime = v === 0 ? 0 : 2 - 2 * l / v;
  return {
    h: h2,
    s: sprime,
    v,
    a
  };
}
function toHex(v) {
  const h2 = Math.round(v).toString(16);
  return ("00".substr(0, 2 - h2.length) + h2).toUpperCase();
}
function RGBtoHex({
  r,
  g,
  b,
  a
}) {
  return `#${[toHex(r), toHex(g), toHex(b), isNumber(a) ? toHex(Math.round(a * 255)) : ""].join("")}`;
}
function HexToRGB(hex) {
  hex = parseHex(hex);
  let [r, g, b, a] = chunk(hex, 2).map((c) => parseInt(c, 16));
  a = isNumber(a) ? a / 255 : a;
  return {
    r,
    g,
    b,
    a
  };
}
function parseHex(hex) {
  if (hex.startsWith("#")) {
    hex = hex.slice(1);
  }
  hex = hex.replace(/([^0-9a-f])/gi, "F");
  if (hex.length === 3 || hex.length === 4) {
    hex = hex.split("").map((x) => x + x).join("");
  }
  if (hex.length !== 6) {
    hex = padEnd(padEnd(hex, 6), 8, "F");
  }
  return hex;
}
function lighten(value, amount) {
  const lab = fromXYZ$1(toXYZ(value));
  lab[0] = lab[0] + amount * 10;
  return fromXYZ(toXYZ$1(lab));
}
function darken(value, amount) {
  const lab = fromXYZ$1(toXYZ(value));
  lab[0] = lab[0] - amount * 10;
  return fromXYZ(toXYZ$1(lab));
}
function getLuma(color) {
  const rgb = parseColor(color);
  return toXYZ(rgb)[1];
}
function hasLightForeground(color) {
  const blackContrast = Math.abs(APCAcontrast(parseColor(0), parseColor(color)));
  const whiteContrast = Math.abs(APCAcontrast(parseColor(16777215), parseColor(color)));
  return whiteContrast > Math.min(blackContrast, 50);
}
function propsFactory(props, source) {
  return (defaults) => {
    return Object.keys(props).reduce((obj, prop) => {
      const isObjectDefinition = isObject(props[prop]);
      const definition = isObjectDefinition ? props[prop] : {
        type: props[prop]
      };
      if (defaults && prop in defaults) {
        obj[prop] = {
          ...definition,
          default: defaults[prop]
        };
      } else {
        obj[prop] = definition;
      }
      if (source && !obj[prop].source) {
        obj[prop].source = source;
      }
      return obj;
    }, {});
  };
}
const makeComponentProps = propsFactory({
  class: [String, Array, Object],
  style: {
    type: [String, Array, Object],
    default: null
  }
}, "component");
function getCurrentInstance(name, message) {
  const vm = getCurrentInstance$1();
  if (!vm) {
    throw new Error(`[Vuetify] ${name} ${"must be called from inside a setup function"}`);
  }
  return vm;
}
function getCurrentInstanceName(name = "composables") {
  const vm = getCurrentInstance(name).type;
  return toKebabCase(vm?.aliasName || vm?.name);
}
function injectSelf(key, vm = getCurrentInstance("injectSelf")) {
  const {
    provides
  } = vm;
  if (provides && key in provides) {
    return provides[key];
  }
  return void 0;
}
const DefaultsSymbol = /* @__PURE__ */ Symbol.for("vuetify:defaults");
const RootDefaultsSymbol = /* @__PURE__ */ Symbol.for("vuetify:defaults:root");
function createDefaults(options) {
  return ref(options);
}
function injectDefaults() {
  const defaults = inject$1(DefaultsSymbol);
  if (!defaults) throw new Error("[Vuetify] Could not find defaults instance");
  return defaults;
}
function provideDefaults(defaults, options) {
  const injectedDefaults = injectDefaults();
  const injectedRoot = inject$1(RootDefaultsSymbol, null);
  const providedDefaults = ref(defaults);
  const newDefaults = computed(() => {
    const disabled = unref(options?.disabled);
    if (disabled) return injectedDefaults.value;
    const scoped = unref(options?.scoped);
    const reset = unref(options?.reset);
    const root = unref(options?.root);
    if (!providedDefaults.value && !(scoped || reset || root)) return injectedDefaults.value;
    let properties = mergeDeep(providedDefaults.value, {
      prev: injectedDefaults.value
    });
    if (scoped) return properties;
    if (reset || root) {
      const len = Number(reset || Infinity);
      const rootDefaults = isString(root) ? properties.prev?.[root] : void 0;
      if (root && injectedRoot?.value) {
        properties = injectedRoot.value;
      }
      for (let i = 0; i <= len; i++) {
        if (!properties || !("prev" in properties)) {
          break;
        }
        properties = properties.prev;
      }
      if (properties && rootDefaults) {
        properties = mergeDeep(mergeDeep(properties, {
          prev: properties
        }), rootDefaults);
      }
      return properties;
    }
    return properties.prev ? mergeDeep(properties.prev, properties, void 0, (_, v) => v !== void 0) : properties;
  });
  provide(DefaultsSymbol, newDefaults);
  return newDefaults;
}
function propIsDefined(vnode, prop) {
  return vnode.props && (typeof vnode.props[prop] !== "undefined" || typeof vnode.props[toKebabCase(prop)] !== "undefined");
}
function internalUseDefaults(props = {}, name, defaults = injectDefaults()) {
  const vm = getCurrentInstance("useDefaults");
  name = name ?? vm.type.name ?? vm.type.__name;
  if (!name) {
    throw new Error("[Vuetify] Could not determine component name");
  }
  const componentDefaults = computed(() => defaults.value?.[props._as ?? name]);
  const _props = new Proxy(props, {
    get(target, prop) {
      const propValue = Reflect.get(target, prop);
      if (prop === "class" || prop === "style") {
        return [componentDefaults.value?.[prop], propValue].filter((v) => v != null);
      }
      if (propIsDefined(vm.vnode, prop)) return propValue;
      const _componentDefault = componentDefaults.value?.[prop];
      if (_componentDefault !== void 0) return _componentDefault;
      const _globalDefault = defaults.value?.global?.[prop];
      if (_globalDefault !== void 0) return _globalDefault;
      return propValue;
    }
  });
  const _subcomponentDefaults = shallowRef();
  watchEffect(() => {
    if (componentDefaults.value) {
      const subComponents = Object.entries(componentDefaults.value).filter(([key]) => key.startsWith(key[0].toUpperCase()));
      _subcomponentDefaults.value = subComponents.length ? Object.fromEntries(subComponents) : void 0;
    } else {
      _subcomponentDefaults.value = void 0;
    }
  });
  function provideSubDefaults() {
    const injected = injectSelf(DefaultsSymbol, vm);
    provide(DefaultsSymbol, computed(() => {
      return _subcomponentDefaults.value ? mergeDeep(injected?.value ?? {}, _subcomponentDefaults.value) : injected?.value;
    }));
  }
  return {
    props: _props,
    provideSubDefaults
  };
}
function defineComponent(options) {
  options._setup = options._setup ?? options.setup;
  if (!options.name) {
    consoleWarn("The component is missing an explicit name, unable to generate default prop value");
    return options;
  }
  if (options._setup) {
    options.props = propsFactory(options.props ?? {}, options.name)();
    const propKeys = Object.keys(options.props).filter((key) => key !== "class" && key !== "style");
    options.filterProps = function filterProps(props) {
      return pick(props, propKeys);
    };
    options.props._as = String;
    options.setup = function setup(props, ctx) {
      const defaults = injectDefaults();
      if (!defaults.value) return options._setup(props, ctx);
      const {
        props: _props,
        provideSubDefaults
      } = internalUseDefaults(props, props._as ?? options.name, defaults);
      const setupBindings = options._setup(_props, ctx);
      provideSubDefaults();
      return setupBindings;
    };
  }
  return options;
}
function genericComponent(exposeDefaults = true) {
  return (options) => (exposeDefaults ? defineComponent : defineComponent$1)(options);
}
function createSimpleFunctional(klass, tag = "div", name) {
  return genericComponent()({
    name: name ?? capitalize(camelize(klass.replace(/__/g, "-"))),
    props: {
      tag: {
        type: String,
        default: tag
      },
      ...makeComponentProps()
    },
    setup(props, {
      slots
    }) {
      return () => {
        return h(props.tag, {
          class: [klass, props.class],
          style: props.style
        }, slots.default?.());
      };
    }
  });
}
function updateRecursionCache(a, b, cache, result) {
  if (!cache || isPrimitive(a) || isPrimitive(b)) return;
  const visitedObject = cache.get(a);
  if (visitedObject) {
    visitedObject.set(b, result);
  } else {
    const newCacheItem = /* @__PURE__ */ new WeakMap();
    newCacheItem.set(b, result);
    cache.set(a, newCacheItem);
  }
}
function findCachedComparison(a, b, cache) {
  if (!cache || isPrimitive(a) || isPrimitive(b)) return null;
  const r1 = cache.get(a)?.get(b);
  if (isBoolean(r1)) return r1;
  const r2 = cache.get(b)?.get(a);
  if (isBoolean(r2)) return r2;
  return null;
}
function deepEqual(a, b, recursionCache = /* @__PURE__ */ new WeakMap()) {
  if (a === b) return true;
  if (a instanceof Date && b instanceof Date && a.getTime() !== b.getTime()) {
    return false;
  }
  if (a !== Object(a) || b !== Object(b)) {
    return false;
  }
  const props = Object.keys(a);
  if (props.length !== Object.keys(b).length) {
    return false;
  }
  const cachedComparisonResult = findCachedComparison(a, b, recursionCache);
  if (cachedComparisonResult) {
    return cachedComparisonResult;
  }
  updateRecursionCache(a, b, recursionCache, true);
  return props.every((p) => deepEqual(a[p], b[p], recursionCache));
}
const easingPatterns = {
  linear: (t) => t,
  easeInQuad: (t) => t ** 2,
  easeOutQuad: (t) => t * (2 - t),
  easeInOutQuad: (t) => t < 0.5 ? 2 * t ** 2 : -1 + (4 - 2 * t) * t,
  easeInCubic: (t) => t ** 3,
  easeOutCubic: (t) => --t ** 3 + 1,
  easeInOutCubic: (t) => t < 0.5 ? 4 * t ** 3 : (t - 1) * (2 * t - 2) * (2 * t - 2) + 1,
  easeInQuart: (t) => t ** 4,
  easeOutQuart: (t) => 1 - --t ** 4,
  easeInOutQuart: (t) => t < 0.5 ? 8 * t ** 4 : 1 - 8 * --t ** 4,
  easeInQuint: (t) => t ** 5,
  easeOutQuint: (t) => 1 + --t ** 5,
  easeInOutQuint: (t) => t < 0.5 ? 16 * t ** 5 : 1 + 16 * --t ** 5,
  instant: (t) => 1
};
function useRender(render) {
  const vm = getCurrentInstance("useRender");
  vm.render = render;
}
const IconValue = [String, Function, Object, Array];
const IconSymbol = /* @__PURE__ */ Symbol.for("vuetify:icons");
const makeIconProps = propsFactory({
  icon: {
    type: IconValue
  },
  // Could not remove this and use makeTagProps, types complained because it is not required
  tag: {
    type: [String, Object, Function],
    required: true
  }
}, "icon");
const VComponentIcon = genericComponent()({
  name: "VComponentIcon",
  props: makeIconProps(),
  setup(props, {
    slots
  }) {
    return () => {
      const Icon = props.icon;
      return createVNode(props.tag, null, {
        default: () => [props.icon ? createVNode(Icon, null, null) : slots.default?.()]
      });
    };
  }
});
const VSvgIcon = defineComponent({
  name: "VSvgIcon",
  inheritAttrs: false,
  props: makeIconProps(),
  setup(props, {
    attrs
  }) {
    return () => {
      return createVNode(props.tag, mergeProps(attrs, {
        "style": null
      }), {
        default: () => [createElementVNode("svg", {
          "class": "v-icon__svg",
          "xmlns": "http://www.w3.org/2000/svg",
          "viewBox": "0 0 24 24",
          "role": "img",
          "aria-hidden": "true"
        }, [Array.isArray(props.icon) ? props.icon.map((path) => Array.isArray(path) ? createElementVNode("path", {
          "d": path[0],
          "fill-opacity": path[1]
        }, null) : createElementVNode("path", {
          "d": path
        }, null)) : createElementVNode("path", {
          "d": props.icon
        }, null)])]
      });
    };
  }
});
defineComponent({
  name: "VLigatureIcon",
  props: makeIconProps(),
  setup(props) {
    return () => {
      return createVNode(props.tag, null, {
        default: () => [props.icon]
      });
    };
  }
});
const VClassIcon = defineComponent({
  name: "VClassIcon",
  props: makeIconProps(),
  setup(props) {
    return () => {
      return createVNode(props.tag, {
        "class": normalizeClass(props.icon)
      }, null);
    };
  }
});
const useIcon = (props) => {
  const icons = inject$1(IconSymbol);
  if (!icons) throw new Error("Missing Vuetify Icons provide!");
  const iconData = computed(() => {
    const iconAlias = toValue(props);
    if (!iconAlias) return {
      component: VComponentIcon
    };
    let icon = iconAlias;
    if (isString(icon)) {
      icon = icon.trim();
      if (icon.startsWith("$")) {
        icon = icons.aliases?.[icon.slice(1)];
      }
    }
    if (!icon) consoleWarn(`Could not find aliased icon "${iconAlias}"`);
    if (Array.isArray(icon)) {
      return {
        component: VSvgIcon,
        icon
      };
    } else if (!isString(icon)) {
      return {
        component: VComponentIcon,
        icon
      };
    }
    const iconSetName = Object.keys(icons.sets).find((setName) => isString(icon) && icon.startsWith(`${setName}:`));
    const iconName = iconSetName ? icon.slice(iconSetName.length + 1) : icon;
    const iconSet = icons.sets[iconSetName ?? icons.defaultSet];
    return {
      component: iconSet.component,
      icon: iconName
    };
  });
  return {
    iconData
  };
};
const aliases = {
  collapse: "mdi-chevron-up",
  complete: "mdi-check",
  cancel: "mdi-close-circle",
  close: "mdi-close",
  delete: "mdi-close-circle",
  // delete (e.g. v-chip close)
  clear: "mdi-close-circle",
  success: "mdi-check-circle",
  info: "mdi-information",
  warning: "mdi-alert-circle",
  error: "mdi-close-circle",
  prev: "mdi-chevron-left",
  next: "mdi-chevron-right",
  checkboxOn: "mdi-checkbox-marked",
  checkboxOff: "mdi-checkbox-blank-outline",
  checkboxIndeterminate: "mdi-minus-box",
  delimiter: "mdi-circle",
  // for carousel
  sortAsc: "mdi-arrow-up",
  sortDesc: "mdi-arrow-down",
  expand: "mdi-chevron-down",
  menu: "mdi-menu",
  subgroup: "mdi-menu-down",
  dropdown: "mdi-menu-down",
  radioOn: "mdi-radiobox-marked",
  radioOff: "mdi-radiobox-blank",
  edit: "mdi-pencil",
  ratingEmpty: "mdi-star-outline",
  ratingFull: "mdi-star",
  ratingHalf: "mdi-star-half-full",
  loading: "mdi-cached",
  first: "mdi-page-first",
  last: "mdi-page-last",
  unfold: "mdi-unfold-more-horizontal",
  file: "mdi-paperclip",
  plus: "mdi-plus",
  minus: "mdi-minus",
  calendar: "mdi-calendar",
  treeviewCollapse: "mdi-menu-down",
  treeviewExpand: "mdi-menu-right",
  tableGroupCollapse: "mdi-chevron-down",
  tableGroupExpand: "mdi-chevron-right",
  eyeDropper: "mdi-eyedropper",
  upload: "mdi-cloud-upload",
  color: "mdi-palette",
  command: "mdi-apple-keyboard-command",
  ctrl: "mdi-apple-keyboard-control",
  space: "mdi-keyboard-space",
  shift: "mdi-apple-keyboard-shift",
  alt: "mdi-apple-keyboard-option",
  enter: "mdi-keyboard-return",
  arrowup: "mdi-arrow-up",
  arrowdown: "mdi-arrow-down",
  arrowleft: "mdi-arrow-left",
  arrowright: "mdi-arrow-right",
  backspace: "mdi-backspace",
  play: "mdi-play",
  pause: "mdi-pause",
  fullscreen: "mdi-fullscreen",
  fullscreenExit: "mdi-fullscreen-exit",
  volumeHigh: "mdi-volume-high",
  volumeMedium: "mdi-volume-medium",
  volumeLow: "mdi-volume-low",
  volumeOff: "mdi-volume-variant-off",
  search: "mdi-magnify"
};
const mdi = {
  // Not using mergeProps here, functional components merge props by default (?)
  component: (props) => h(VClassIcon, {
    ...props,
    class: "mdi"
  })
};
function genDefaults$3() {
  return {
    svg: {
      component: VSvgIcon
    },
    class: {
      component: VClassIcon
    }
  };
}
function createIcons(options) {
  const sets = genDefaults$3();
  const defaultSet = options?.defaultSet ?? "mdi";
  if (defaultSet === "mdi" && !sets.mdi) {
    sets.mdi = mdi;
  }
  return mergeDeep({
    defaultSet,
    sets,
    aliases: {
      ...aliases,
      /* eslint-disable max-len */
      vuetify: ["M8.2241 14.2009L12 21L22 3H14.4459L8.2241 14.2009Z", ["M7.26303 12.4733L7.00113 12L2 3H12.5261C12.5261 3 12.5261 3 12.5261 3L7.26303 12.4733Z", 0.6]],
      "vuetify-outline": "svg:M7.26 12.47 12.53 3H2L7.26 12.47ZM14.45 3 8.22 14.2 12 21 22 3H14.45ZM18.6 5 12 16.88 10.51 14.2 15.62 5ZM7.26 8.35 5.4 5H9.13L7.26 8.35Z",
      "vuetify-play": ["m6.376 13.184-4.11-7.192C1.505 4.66 2.467 3 4.003 3h8.532l-.953 1.576-.006.01-.396.677c-.429.732-.214 1.507.194 2.015.404.503 1.092.878 1.869.806a3.72 3.72 0 0 1 1.005.022c.276.053.434.143.523.237.138.146.38.635-.25 2.09-.893 1.63-1.553 1.722-1.847 1.677-.213-.033-.468-.158-.756-.406a4.95 4.95 0 0 1-.8-.927c-.39-.564-1.04-.84-1.66-.846-.625-.006-1.316.27-1.693.921l-.478.826-.911 1.506Z", ["M9.093 11.552c.046-.079.144-.15.32-.148a.53.53 0 0 1 .43.207c.285.414.636.847 1.046 1.2.405.35.914.662 1.516.754 1.334.205 2.502-.698 3.48-2.495l.014-.028.013-.03c.687-1.574.774-2.852-.005-3.675-.37-.391-.861-.586-1.333-.676a5.243 5.243 0 0 0-1.447-.044c-.173.016-.393-.073-.54-.257-.145-.18-.127-.316-.082-.392l.393-.672L14.287 3h5.71c1.536 0 2.499 1.659 1.737 2.992l-7.997 13.996c-.768 1.344-2.706 1.344-3.473 0l-3.037-5.314 1.377-2.278.004-.006.004-.007.481-.831Z", 0.6]]
      /* eslint-enable max-len */
    }
  }, options);
}
function useToggleScope(source, fn) {
  let scope;
  function start() {
    scope = effectScope();
    scope.run(() => fn.length ? fn(() => {
      scope?.stop();
      start();
    }) : fn());
  }
  watch(source, (active) => {
    if (active && !scope) {
      start();
    } else if (!active) {
      scope?.stop();
      scope = void 0;
    }
  }, {
    immediate: true
  });
  onScopeDispose(() => {
    scope?.stop();
  });
}
function useProxiedModel(props, prop, defaultValue, transformIn = (v) => v, transformOut = (v) => v) {
  const vm = getCurrentInstance("useProxiedModel");
  const internal = ref(!isUndefined(props[prop]) ? props[prop] : defaultValue);
  const kebabProp = toKebabCase(prop);
  const checkKebab = kebabProp !== prop;
  const isControlled = checkKebab ? computed(() => {
    void props[prop];
    return !!((vm.vnode.props?.hasOwnProperty(prop) || vm.vnode.props?.hasOwnProperty(kebabProp)) && (vm.vnode.props?.hasOwnProperty(`onUpdate:${prop}`) || vm.vnode.props?.hasOwnProperty(`onUpdate:${kebabProp}`)));
  }) : computed(() => {
    void props[prop];
    return !!(vm.vnode.props?.hasOwnProperty(prop) && vm.vnode.props?.hasOwnProperty(`onUpdate:${prop}`));
  });
  useToggleScope(() => !isControlled.value, () => {
    watch(() => props[prop], (val) => {
      internal.value = val;
    });
  });
  const model = computed({
    get() {
      const externalValue = props[prop];
      return transformIn(isControlled.value ? externalValue : internal.value);
    },
    set(internalValue) {
      const newValue = transformOut(internalValue);
      const value = toRaw(isControlled.value ? props[prop] : internal.value);
      if (value === newValue || transformIn(value) === internalValue) {
        return;
      }
      internal.value = newValue;
      vm?.emit(`update:${prop}`, newValue);
    }
  });
  Object.defineProperty(model, "externalValue", {
    get: () => isControlled.value ? props[prop] : internal.value
  });
  return model;
}
const en = {
  badge: "Badge",
  open: "Open",
  close: "Close",
  dismiss: "Dismiss",
  confirmEdit: {
    ok: "OK",
    cancel: "Cancel"
  },
  dataIterator: {
    noResultsText: "No matching records found",
    loadingText: "Loading items..."
  },
  dataTable: {
    itemsPerPageText: "Rows per page:",
    ariaLabel: {
      sortDescending: "Sorted descending.",
      sortAscending: "Sorted ascending.",
      sortNone: "Not sorted.",
      activateNone: "Activate to remove sorting.",
      activateDescending: "Activate to sort descending.",
      activateAscending: "Activate to sort ascending.",
      selectRow: "Select row",
      selectAll: "Select all",
      selectGroup: "Select group"
    },
    sortBy: "Sort by"
  },
  dataFooter: {
    itemsPerPageText: "Items per page:",
    itemsPerPageAll: "All",
    nextPage: "Next page",
    prevPage: "Previous page",
    firstPage: "First page",
    lastPage: "Last page",
    pageText: "{0}-{1} of {2}"
  },
  dateRangeInput: {
    divider: "to"
  },
  monthPicker: {
    title: "Select month",
    itemsSelected: "{0} selected",
    header: "Enter month",
    range: {
      title: "Enter months"
    },
    ariaLabel: {
      previousYear: "Previous year",
      nextYear: "Next year",
      selectYear: "Select year",
      currentMonth: "Current month, {0}"
    }
  },
  datePicker: {
    itemsSelected: "{0} selected",
    range: {
      title: "Select dates",
      header: "Enter dates"
    },
    title: "Select date",
    header: "Enter date",
    input: {
      placeholder: "Enter date"
    },
    ariaLabel: {
      previousMonth: "Previous month",
      nextMonth: "Next month",
      selectYear: "Select year",
      previousYear: "Previous year",
      nextYear: "Next year",
      selectMonth: "Select month",
      selectDate: "{0}",
      // Full date format
      currentDate: "Today, {0}"
    }
  },
  noDataText: "No data available",
  carousel: {
    prev: "Previous visual",
    next: "Next visual",
    ariaLabel: {
      delimiter: "Carousel slide {0} of {1}"
    }
  },
  calendar: {
    moreEvents: "{0} more",
    today: "Today"
  },
  heatmap: {
    less: "Less",
    more: "More"
  },
  input: {
    clear: "Clear {0}",
    prependAction: "{0} prepended action",
    appendAction: "{0} appended action",
    otp: "Enter verification code"
  },
  fileInput: {
    counter: "{0} files",
    counterSize: "{0} files ({1} in total)"
  },
  fileUpload: {
    title: "Drag and drop files here",
    divider: "or",
    browse: "Browse Files"
  },
  timePicker: {
    am: "AM",
    pm: "PM",
    title: "Select Time",
    hour: "Hour",
    minute: "Minute",
    second: "Second",
    notAllowed: "Value is not allowed"
  },
  pagination: {
    ariaLabel: {
      root: "Pagination Navigation",
      next: "Next page",
      previous: "Previous page",
      page: "Go to page {0}",
      currentPage: "Page {0}, Current page",
      first: "First page",
      last: "Last page"
    }
  },
  stepper: {
    next: "Next",
    prev: "Previous"
  },
  rating: {
    ariaLabel: {
      item: "Rating {0} of {1}"
    }
  },
  loading: "Loading...",
  infiniteScroll: {
    loadMore: "Load more",
    empty: "No more"
  },
  rules: {
    required: "This field is required",
    email: "Please enter a valid email",
    number: "This field can only contain numbers",
    integer: "This field can only contain integer values",
    capital: "This field can only contain uppercase letters",
    maxLength: "You must enter a maximum of {0} characters",
    minLength: "You must enter a minimum of {0} characters",
    strictLength: "The length of the entered field is invalid",
    exclude: "The {0} character is not allowed",
    notEmpty: "Please choose at least one value",
    pattern: "Invalid format"
  },
  command: {
    search: "Type a command or search..."
  },
  hotkey: {
    then: "then",
    ctrl: "Ctrl",
    command: "Command",
    space: "Space",
    shift: "Shift",
    alt: "Alt",
    enter: "Enter",
    escape: "Escape",
    upArrow: "Up Arrow",
    downArrow: "Down Arrow",
    leftArrow: "Left Arrow",
    rightArrow: "Right Arrow",
    backspace: "Backspace",
    option: "Option",
    plus: "plus",
    shortcut: "Keyboard shortcut: {0}",
    or: "or"
  },
  video: {
    play: "Play",
    pause: "Pause",
    seek: "Seek",
    volume: "Volume",
    showVolume: "Show volume control",
    mute: "Mute",
    unmute: "Unmute",
    enterFullscreen: "Full screen",
    exitFullscreen: "Exit full screen"
  },
  colorPicker: {
    ariaLabel: {
      eyedropper: "Select color with eyedropper",
      hueSlider: "Hue",
      alphaSlider: "Alpha",
      redInput: "Red value",
      greenInput: "Green value",
      blueInput: "Blue value",
      alphaInput: "Alpha value",
      hueInput: "Hue value",
      saturationInput: "Saturation value",
      lightnessInput: "Lightness value",
      hexInput: "HEX value",
      hexaInput: "HEX with alpha value",
      changeFormat: "Change color format"
    }
  }
};
const LANG_PREFIX = "$vuetify.";
const replace = (str, params) => {
  return str.replace(/\{(\d+)\}/g, (match, index) => {
    return String(params[Number(index)]);
  });
};
const createTranslateFunction = (current, fallback, messages) => {
  return (key, ...params) => {
    if (!key.startsWith(LANG_PREFIX)) {
      return replace(key, params);
    }
    const shortKey = key.replace(LANG_PREFIX, "");
    const currentLocale = current.value && messages.value[current.value];
    const fallbackLocale = fallback.value && messages.value[fallback.value];
    let str = getObjectValueByPath(currentLocale, shortKey, null);
    if (!str) {
      consoleWarn(`Translation key "${key}" not found in "${current.value}", trying fallback locale`);
      str = getObjectValueByPath(fallbackLocale, shortKey, null);
    }
    if (!str) {
      consoleError(`Translation key "${key}" not found in fallback`);
      str = key;
    }
    if (!isString(str)) {
      consoleError(`Translation key "${key}" has a non-string value`);
      str = key;
    }
    return replace(str, params);
  };
};
function createNumberFunction(current, fallback) {
  return (value, options) => {
    const numberFormat = new Intl.NumberFormat([current.value, fallback.value], options);
    return numberFormat.format(value);
  };
}
function inferDecimalSeparator(current, fallback) {
  const format2 = createNumberFunction(current, fallback);
  return format2(0.1).includes(",") ? "," : ".";
}
function inferNumericGroupSeparator(current, fallback) {
  return new Intl.NumberFormat([current.value, fallback.value], {
    useGrouping: true
  }).formatToParts(1e4).find((p) => p.type === "group")?.value ?? " ";
}
function useProvided(props, prop, provided) {
  const internal = useProxiedModel(props, prop, props[prop] ?? provided.value);
  internal.value = props[prop] ?? provided.value;
  watch(provided, (v) => {
    if (props[prop] == null) {
      internal.value = provided.value;
    }
  });
  return internal;
}
function createProvideFunction(state) {
  return (props) => {
    const current = useProvided(props, "locale", state.current);
    const fallback = useProvided(props, "fallback", state.fallback);
    const messages = useProvided(props, "messages", state.messages);
    return {
      name: "vuetify",
      current,
      fallback,
      messages,
      decimalSeparator: toRef(() => inferDecimalSeparator(current, fallback)),
      numericGroupSeparator: toRef(() => inferNumericGroupSeparator(current, fallback)),
      t: createTranslateFunction(current, fallback, messages),
      n: createNumberFunction(current, fallback),
      provide: createProvideFunction({
        current,
        fallback,
        messages
      })
    };
  };
}
function createVuetifyAdapter(options) {
  const current = shallowRef(options?.locale ?? "en");
  const fallback = shallowRef(options?.fallback ?? "en");
  const messages = ref({
    en,
    ...options?.messages
  });
  return {
    name: "vuetify",
    current,
    fallback,
    messages,
    decimalSeparator: toRef(() => options?.decimalSeparator ?? inferDecimalSeparator(current, fallback)),
    numericGroupSeparator: toRef(() => options?.numericGroupSeparator ?? inferNumericGroupSeparator(current, fallback)),
    t: createTranslateFunction(current, fallback, messages),
    n: createNumberFunction(current, fallback),
    provide: createProvideFunction({
      current,
      fallback,
      messages
    })
  };
}
const LocaleSymbol = /* @__PURE__ */ Symbol.for("vuetify:locale");
function createLocale(options) {
  const i18n = options?.adapter?.name ? options.adapter : createVuetifyAdapter(options);
  const rtl = createRtl(i18n, options);
  return {
    ...i18n,
    ...rtl
  };
}
function useLocale() {
  const locale = inject$1(LocaleSymbol);
  if (!locale) throw new Error("[Vuetify] Could not find injected locale instance");
  return locale;
}
function genDefaults$2() {
  return {
    af: false,
    ar: true,
    bg: false,
    ca: false,
    ckb: true,
    cs: false,
    de: false,
    el: false,
    en: false,
    es: false,
    et: false,
    fa: true,
    fi: false,
    fr: false,
    hr: false,
    hu: false,
    he: true,
    id: false,
    it: false,
    ja: false,
    km: false,
    ko: false,
    lv: false,
    lt: false,
    nl: false,
    no: false,
    pl: false,
    pt: false,
    ro: false,
    ru: false,
    sk: false,
    sl: false,
    srCyrl: false,
    srLatn: false,
    sv: false,
    th: false,
    tr: false,
    az: false,
    uk: false,
    vi: false,
    zhHans: false,
    zhHant: false
  };
}
function createRtl(i18n, options) {
  const rtl = ref(options?.rtl ?? genDefaults$2());
  const isRtl = computed(() => rtl.value[i18n.current.value] ?? false);
  return {
    isRtl,
    rtl,
    rtlClasses: toRef(() => `v-locale--is-${isRtl.value ? "rtl" : "ltr"}`)
  };
}
function useRtl() {
  const locale = inject$1(LocaleSymbol);
  if (!locale) throw new Error("[Vuetify] Could not find injected rtl instance");
  return {
    isRtl: locale.isRtl,
    rtlClasses: locale.rtlClasses
  };
}
function weekInfo(locale) {
  const code = locale.slice(-2).toUpperCase();
  switch (true) {
    case locale === "GB-alt-variant": {
      return {
        firstDay: 0,
        firstWeekSize: 4
      };
    }
    case locale === "001": {
      return {
        firstDay: 1,
        firstWeekSize: 1
      };
    }
    case `AG AS BD BR BS BT BW BZ CA CO DM DO ET GT GU HK HN ID IL IN JM JP KE
    KH KR LA MH MM MO MT MX MZ NI NP PA PE PH PK PR PY SA SG SV TH TT TW UM US
    VE VI WS YE ZA ZW`.includes(code): {
      return {
        firstDay: 0,
        firstWeekSize: 1
      };
    }
    case `AI AL AM AR AU AZ BA BM BN BY CL CM CN CR CY EC GE HR KG KZ LB LK LV
    MD ME MK MN MY NZ RO RS SI TJ TM TR UA UY UZ VN XK`.includes(code): {
      return {
        firstDay: 1,
        firstWeekSize: 1
      };
    }
    case `AD AN AT AX BE BG CH CZ DE DK EE ES FI FJ FO FR GB GF GP GR HU IE IS
    IT LI LT LU MC MQ NL NO PL RE RU SE SK SM VA`.includes(code): {
      return {
        firstDay: 1,
        firstWeekSize: 4
      };
    }
    case `AE AF BH DJ DZ EG IQ IR JO KW LY OM QA SD SY`.includes(code): {
      return {
        firstDay: 6,
        firstWeekSize: 1
      };
    }
    case code === "MV": {
      return {
        firstDay: 5,
        firstWeekSize: 1
      };
    }
    case code === "PT": {
      return {
        firstDay: 0,
        firstWeekSize: 4
      };
    }
    default:
      return null;
  }
}
function getWeekArray(date2, locale, firstDayOfWeek) {
  const weeks = [];
  let currentWeek = [];
  const firstDayOfMonth = startOfMonth(date2);
  const lastDayOfMonth = endOfMonth(date2);
  const first = firstDayOfWeek ?? weekInfo(locale)?.firstDay ?? 0;
  const firstDayWeekIndex = (firstDayOfMonth.getDay() - first + 7) % 7;
  const lastDayWeekIndex = (lastDayOfMonth.getDay() - first + 7) % 7;
  for (let i = 0; i < firstDayWeekIndex; i++) {
    const adjacentDay = new Date(firstDayOfMonth);
    adjacentDay.setDate(adjacentDay.getDate() - (firstDayWeekIndex - i));
    currentWeek.push(adjacentDay);
  }
  for (let i = 1; i <= lastDayOfMonth.getDate(); i++) {
    const day = new Date(date2.getFullYear(), date2.getMonth(), i);
    currentWeek.push(day);
    if (currentWeek.length === 7) {
      weeks.push(currentWeek);
      currentWeek = [];
    }
  }
  for (let i = 1; i < 7 - lastDayWeekIndex; i++) {
    const adjacentDay = new Date(lastDayOfMonth);
    adjacentDay.setDate(adjacentDay.getDate() + i);
    currentWeek.push(adjacentDay);
  }
  if (currentWeek.length > 0) {
    weeks.push(currentWeek);
  }
  return weeks;
}
function startOfWeek(date2, locale, firstDayOfWeek) {
  let day = (firstDayOfWeek ?? weekInfo(locale)?.firstDay ?? 0) % 7;
  if (![0, 1, 2, 3, 4, 5, 6].includes(day)) {
    consoleWarn("Invalid firstDayOfWeek, expected discrete number in range [0-6]");
    day = 0;
  }
  const d = new Date(date2);
  while (d.getDay() !== day) {
    d.setDate(d.getDate() - 1);
  }
  return d;
}
function endOfWeek(date2, locale) {
  const d = new Date(date2);
  const lastDay = ((weekInfo(locale)?.firstDay ?? 0) + 6) % 7;
  while (d.getDay() !== lastDay) {
    d.setDate(d.getDate() + 1);
  }
  return d;
}
function startOfMonth(date2) {
  return new Date(date2.getFullYear(), date2.getMonth(), 1);
}
function endOfMonth(date2) {
  return new Date(date2.getFullYear(), date2.getMonth() + 1, 0);
}
function parseLocalDate(value) {
  const parts = value.split("-").map(Number);
  return new Date(parts[0], parts[1] - 1, parts[2]);
}
const _YYYMMDD = /^([12]\d{3}-([1-9]|0[1-9]|1[0-2])-([1-9]|0[1-9]|[12]\d|3[01]))$/;
function date(value) {
  if (value == null) return /* @__PURE__ */ new Date();
  if (value instanceof Date) return value;
  if (typeof value === "string") {
    let parsed;
    if (_YYYMMDD.test(value)) {
      return parseLocalDate(value);
    } else {
      parsed = Date.parse(value);
    }
    if (!isNaN(parsed)) return new Date(parsed);
  }
  return null;
}
const sundayJanuarySecond2000 = new Date(2e3, 0, 2);
function getWeekdays(locale, firstDayOfWeek, weekdayFormat) {
  const daysFromSunday = firstDayOfWeek ?? weekInfo(locale)?.firstDay ?? 0;
  return range(7).map((i) => {
    const weekday = new Date(sundayJanuarySecond2000);
    weekday.setDate(sundayJanuarySecond2000.getDate() + daysFromSunday + i);
    return new Intl.DateTimeFormat(locale, {
      weekday: weekdayFormat ?? "narrow"
    }).format(weekday);
  });
}
function format(value, formatString, locale, formats) {
  const newDate = date(value) ?? /* @__PURE__ */ new Date();
  const customFormat = formats?.[formatString];
  if (typeof customFormat === "function") {
    return customFormat(newDate, formatString, locale);
  }
  let options = {};
  switch (formatString) {
    case "fullDate":
      options = {
        year: "numeric",
        month: "short",
        day: "numeric"
      };
      break;
    case "fullDateWithWeekday":
      options = {
        weekday: "long",
        year: "numeric",
        month: "long",
        day: "numeric"
      };
      break;
    case "normalDate":
      const day = newDate.getDate();
      const month = new Intl.DateTimeFormat(locale, {
        month: "long"
      }).format(newDate);
      return `${day} ${month}`;
    case "normalDateWithWeekday":
      options = {
        weekday: "short",
        day: "numeric",
        month: "short"
      };
      break;
    case "shortDate":
      options = {
        month: "short",
        day: "numeric"
      };
      break;
    case "year":
      options = {
        year: "numeric"
      };
      break;
    case "month":
      options = {
        month: "long"
      };
      break;
    case "monthShort":
      options = {
        month: "short"
      };
      break;
    case "monthAndYear":
      options = {
        month: "long",
        year: "numeric"
      };
      break;
    case "monthAndDate":
      options = {
        month: "long",
        day: "numeric"
      };
      break;
    case "weekday":
      options = {
        weekday: "long"
      };
      break;
    case "weekdayShort":
      options = {
        weekday: "short"
      };
      break;
    case "dayOfMonth":
      return new Intl.NumberFormat(locale).format(newDate.getDate());
    case "hours12h":
      options = {
        hour: "numeric",
        hour12: true
      };
      break;
    case "hours24h":
      options = {
        hour: "numeric",
        hour12: false
      };
      break;
    case "minutes":
      options = {
        minute: "numeric"
      };
      break;
    case "seconds":
      options = {
        second: "numeric"
      };
      break;
    case "fullTime":
      options = {
        hour: "numeric",
        minute: "numeric"
      };
      break;
    case "fullTime12h":
      options = {
        hour: "numeric",
        minute: "numeric",
        hour12: true
      };
      break;
    case "fullTime24h":
      options = {
        hour: "numeric",
        minute: "numeric",
        hour12: false
      };
      break;
    case "fullDateTime":
      options = {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "numeric",
        minute: "numeric"
      };
      break;
    case "fullDateTime12h":
      options = {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "numeric",
        minute: "numeric",
        hour12: true
      };
      break;
    case "fullDateTime24h":
      options = {
        year: "numeric",
        month: "short",
        day: "numeric",
        hour: "numeric",
        minute: "numeric",
        hour12: false
      };
      break;
    case "keyboardDate":
      options = {
        year: "numeric",
        month: "2-digit",
        day: "2-digit"
      };
      break;
    case "keyboardDateTime":
      options = {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "numeric",
        minute: "numeric"
      };
      return new Intl.DateTimeFormat(locale, options).format(newDate).replace(/, /g, " ");
    case "keyboardDateTime12h":
      options = {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "numeric",
        minute: "numeric",
        hour12: true
      };
      return new Intl.DateTimeFormat(locale, options).format(newDate).replace(/, /g, " ");
    case "keyboardDateTime24h":
      options = {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "numeric",
        minute: "numeric",
        hour12: false
      };
      return new Intl.DateTimeFormat(locale, options).format(newDate).replace(/, /g, " ");
    default:
      options = customFormat ?? {
        timeZone: "UTC",
        timeZoneName: "short"
      };
  }
  return new Intl.DateTimeFormat(locale, options).format(newDate);
}
function toISO(adapter, value) {
  const date2 = adapter.toJsDate(value);
  const year = date2.getFullYear();
  const month = padStart(String(date2.getMonth() + 1), 2, "0");
  const day = padStart(String(date2.getDate()), 2, "0");
  return `${year}-${month}-${day}`;
}
function parseISO(value) {
  const [year, month, day] = value.split("-").map(Number);
  return new Date(year, month - 1, day);
}
function addMinutes(date2, amount) {
  const d = new Date(date2);
  d.setMinutes(d.getMinutes() + amount);
  return d;
}
function addHours(date2, amount) {
  const d = new Date(date2);
  d.setHours(d.getHours() + amount);
  return d;
}
function addDays(date2, amount) {
  const d = new Date(date2);
  d.setDate(d.getDate() + amount);
  return d;
}
function addWeeks(date2, amount) {
  const d = new Date(date2);
  d.setDate(d.getDate() + amount * 7);
  return d;
}
function addMonths(date2, amount) {
  const d = new Date(date2);
  d.setDate(1);
  d.setMonth(d.getMonth() + amount);
  return d;
}
function getYear(date2) {
  return date2.getFullYear();
}
function getMonth(date2) {
  return date2.getMonth();
}
function getWeek(date2, locale, firstDayOfWeek, firstDayOfYear) {
  const weekInfoFromLocale = weekInfo(locale);
  const weekStart = firstDayOfWeek ?? weekInfoFromLocale?.firstDay ?? 0;
  const minWeekSize = weekInfoFromLocale?.firstWeekSize ?? 1;
  return firstDayOfYear !== void 0 ? calculateWeekWithFirstDayOfYear(date2, locale, weekStart, firstDayOfYear) : calculateWeekWithMinWeekSize(date2, locale, weekStart, minWeekSize);
}
function calculateWeekWithFirstDayOfYear(date2, locale, weekStart, firstDayOfYear) {
  const firstDayOfYearOffset = (7 + firstDayOfYear - weekStart) % 7;
  const currentWeekStart = startOfWeek(date2, locale, weekStart);
  const currentWeekEnd = addDays(currentWeekStart, 6);
  function yearStartWeekdayOffset(year2) {
    return (7 + new Date(year2, 0, 1).getDay() - weekStart) % 7;
  }
  let year = getYear(currentWeekStart);
  if (year < getYear(currentWeekEnd) && yearStartWeekdayOffset(year + 1) <= firstDayOfYearOffset) {
    year++;
  }
  const yearStart = new Date(year, 0, 1);
  const offset = yearStartWeekdayOffset(year);
  const d1w1 = offset <= firstDayOfYearOffset ? addDays(yearStart, -offset) : addDays(yearStart, 7 - offset);
  return 1 + getDiff(endOfDay(currentWeekStart), startOfDay(d1w1), "weeks");
}
function calculateWeekWithMinWeekSize(date2, locale, weekStart, minWeekSize) {
  const currentWeekStart = startOfWeek(date2, locale, weekStart);
  const currentWeekEnd = addDays(startOfWeek(date2, locale, weekStart), 6);
  function firstWeekSize(year2) {
    const yearStart2 = new Date(year2, 0, 1);
    return 7 - getDiff(yearStart2, startOfWeek(yearStart2, locale, weekStart), "days");
  }
  let year = getYear(currentWeekStart);
  if (year < getYear(currentWeekEnd) && firstWeekSize(year + 1) >= minWeekSize) {
    year++;
  }
  const yearStart = new Date(year, 0, 1);
  const size = firstWeekSize(year);
  const d1w1 = size >= minWeekSize ? addDays(yearStart, size - 7) : addDays(yearStart, size);
  return 1 + getDiff(endOfDay(currentWeekStart), startOfDay(d1w1), "weeks");
}
function getDate(date2) {
  return date2.getDate();
}
function getNextMonth(date2) {
  return new Date(date2.getFullYear(), date2.getMonth() + 1, 1);
}
function getPreviousMonth(date2) {
  return new Date(date2.getFullYear(), date2.getMonth() - 1, 1);
}
function getHours(date2) {
  return date2.getHours();
}
function getMinutes(date2) {
  return date2.getMinutes();
}
function startOfYear(date2) {
  return new Date(date2.getFullYear(), 0, 1);
}
function endOfYear(date2) {
  return new Date(date2.getFullYear(), 11, 31);
}
function isWithinRange(date2, range2) {
  return isEqual(date2, range2[0]) || isEqual(date2, range2[1]) || isAfter(date2, range2[0]) && isBefore(date2, range2[1]);
}
function isValid(date2) {
  const d = new Date(date2);
  return d instanceof Date && !isNaN(d.getTime());
}
function isAfter(date2, comparing) {
  return date2.getTime() > comparing.getTime();
}
function isAfterDay(date2, comparing) {
  return isAfter(startOfDay(date2), startOfDay(comparing));
}
function isBefore(date2, comparing) {
  return date2.getTime() < comparing.getTime();
}
function isEqual(date2, comparing) {
  return date2.getTime() === comparing.getTime();
}
function isSameDay(date2, comparing) {
  return date2.getDate() === comparing.getDate() && date2.getMonth() === comparing.getMonth() && date2.getFullYear() === comparing.getFullYear();
}
function isSameMonth(date2, comparing) {
  return date2.getMonth() === comparing.getMonth() && date2.getFullYear() === comparing.getFullYear();
}
function isSameYear(date2, comparing) {
  return date2.getFullYear() === comparing.getFullYear();
}
function getDiff(date2, comparing, unit) {
  const d = new Date(date2);
  const c = new Date(comparing);
  switch (unit) {
    case "years":
      return d.getFullYear() - c.getFullYear();
    case "quarters":
      return Math.floor((d.getMonth() - c.getMonth() + (d.getFullYear() - c.getFullYear()) * 12) / 4);
    case "months":
      return d.getMonth() - c.getMonth() + (d.getFullYear() - c.getFullYear()) * 12;
    case "weeks":
      return Math.floor((d.getTime() - c.getTime()) / (1e3 * 60 * 60 * 24 * 7));
    case "days":
      return Math.floor((d.getTime() - c.getTime()) / (1e3 * 60 * 60 * 24));
    case "hours":
      return Math.floor((d.getTime() - c.getTime()) / (1e3 * 60 * 60));
    case "minutes":
      return Math.floor((d.getTime() - c.getTime()) / (1e3 * 60));
    case "seconds":
      return Math.floor((d.getTime() - c.getTime()) / 1e3);
    default: {
      return d.getTime() - c.getTime();
    }
  }
}
function setHours(date2, count) {
  const d = new Date(date2);
  d.setHours(count);
  return d;
}
function setMinutes(date2, count) {
  const d = new Date(date2);
  d.setMinutes(count);
  return d;
}
function setMonth(date2, count) {
  const d = new Date(date2);
  d.setMonth(count);
  return d;
}
function setDate(date2, day) {
  const d = new Date(date2);
  d.setDate(day);
  return d;
}
function setYear(date2, year) {
  const d = new Date(date2);
  d.setFullYear(year);
  return d;
}
function startOfDay(date2) {
  return new Date(date2.getFullYear(), date2.getMonth(), date2.getDate(), 0, 0, 0, 0);
}
function endOfDay(date2) {
  return new Date(date2.getFullYear(), date2.getMonth(), date2.getDate(), 23, 59, 59, 999);
}
class VuetifyDateAdapter {
  constructor(options) {
    this.locale = options.locale;
    this.formats = options.formats;
  }
  date(value) {
    return date(value);
  }
  toJsDate(date2) {
    return date2;
  }
  toISO(date2) {
    return toISO(this, date2);
  }
  parseISO(date2) {
    return parseISO(date2);
  }
  addMinutes(date2, amount) {
    return addMinutes(date2, amount);
  }
  addHours(date2, amount) {
    return addHours(date2, amount);
  }
  addDays(date2, amount) {
    return addDays(date2, amount);
  }
  addWeeks(date2, amount) {
    return addWeeks(date2, amount);
  }
  addMonths(date2, amount) {
    return addMonths(date2, amount);
  }
  getWeekArray(date2, firstDayOfWeek) {
    const firstDay = firstDayOfWeek !== void 0 ? Number(firstDayOfWeek) : void 0;
    return getWeekArray(date2, this.locale, firstDay);
  }
  startOfWeek(date2, firstDayOfWeek) {
    const firstDay = firstDayOfWeek !== void 0 ? Number(firstDayOfWeek) : void 0;
    return startOfWeek(date2, this.locale, firstDay);
  }
  endOfWeek(date2) {
    return endOfWeek(date2, this.locale);
  }
  startOfMonth(date2) {
    return startOfMonth(date2);
  }
  endOfMonth(date2) {
    return endOfMonth(date2);
  }
  format(date2, formatString) {
    return format(date2, formatString, this.locale, this.formats);
  }
  isEqual(date2, comparing) {
    return isEqual(date2, comparing);
  }
  isValid(date2) {
    return isValid(date2);
  }
  isWithinRange(date2, range2) {
    return isWithinRange(date2, range2);
  }
  isAfter(date2, comparing) {
    return isAfter(date2, comparing);
  }
  isAfterDay(date2, comparing) {
    return isAfterDay(date2, comparing);
  }
  isBefore(date2, comparing) {
    return !isAfter(date2, comparing) && !isEqual(date2, comparing);
  }
  isSameDay(date2, comparing) {
    return isSameDay(date2, comparing);
  }
  isSameMonth(date2, comparing) {
    return isSameMonth(date2, comparing);
  }
  isSameYear(date2, comparing) {
    return isSameYear(date2, comparing);
  }
  setMinutes(date2, count) {
    return setMinutes(date2, count);
  }
  setHours(date2, count) {
    return setHours(date2, count);
  }
  setMonth(date2, count) {
    return setMonth(date2, count);
  }
  setDate(date2, day) {
    return setDate(date2, day);
  }
  setYear(date2, year) {
    return setYear(date2, year);
  }
  getDiff(date2, comparing, unit) {
    return getDiff(date2, comparing, unit);
  }
  getWeekdays(firstDayOfWeek, weekdayFormat) {
    const firstDay = firstDayOfWeek !== void 0 ? Number(firstDayOfWeek) : void 0;
    return getWeekdays(this.locale, firstDay, weekdayFormat);
  }
  getYear(date2) {
    return getYear(date2);
  }
  getMonth(date2) {
    return getMonth(date2);
  }
  getWeek(date2, firstDayOfWeek, firstDayOfYear) {
    const firstDay = firstDayOfWeek !== void 0 ? Number(firstDayOfWeek) : void 0;
    const firstWeekStart = firstDayOfYear !== void 0 ? Number(firstDayOfYear) : void 0;
    return getWeek(date2, this.locale, firstDay, firstWeekStart);
  }
  getDate(date2) {
    return getDate(date2);
  }
  getNextMonth(date2) {
    return getNextMonth(date2);
  }
  getPreviousMonth(date2) {
    return getPreviousMonth(date2);
  }
  getHours(date2) {
    return getHours(date2);
  }
  getMinutes(date2) {
    return getMinutes(date2);
  }
  startOfDay(date2) {
    return startOfDay(date2);
  }
  endOfDay(date2) {
    return endOfDay(date2);
  }
  startOfYear(date2) {
    return startOfYear(date2);
  }
  endOfYear(date2) {
    return endOfYear(date2);
  }
}
const DateOptionsSymbol = /* @__PURE__ */ Symbol.for("vuetify:date-options");
const DateAdapterSymbol = /* @__PURE__ */ Symbol.for("vuetify:date-adapter");
function createDate(options, locale) {
  const _options = mergeDeep({
    adapter: VuetifyDateAdapter,
    locale: {
      af: "af-ZA",
      ar: "ar-DZ",
      // bare 'ar' would resolve to Argentina in ../adapters/vuetify.ts:25
      bg: "bg-BG",
      ca: "ca-ES",
      ckb: "ckb-IQ",
      cs: "cs-CZ",
      da: "da-DK",
      de: "de-DE",
      el: "el-GR",
      en: "en-US",
      // es: '', # not the same value for all variants
      et: "et-EE",
      fa: "fa-IR",
      fi: "fi-FI",
      // fr: '', #not the same value for all variants
      hr: "hr-HR",
      hu: "hu-HU",
      he: "he-IL",
      id: "id-ID",
      it: "it-IT",
      ja: "ja-JP",
      km: "km-KH",
      ko: "ko-KR",
      lv: "lv-LV",
      lt: "lt-LT",
      nl: "nl-NL",
      no: "no-NO",
      pl: "pl-PL",
      pt: "pt-PT",
      ro: "ro-RO",
      ru: "ru-RU",
      sk: "sk-SK",
      sl: "sl-SI",
      srCyrl: "sr-Cyrl-RS",
      srLatn: "sr-Latn-RS",
      sv: "sv-SE",
      th: "th-TH",
      tr: "tr-TR",
      az: "az-AZ",
      uk: "uk-UA",
      vi: "vi-VN",
      zhHans: "zh-CN",
      zhHant: "zh-TW"
    }
  }, options);
  return {
    options: _options,
    instance: createInstance(_options, locale)
  };
}
function daysDiff(adapter, start, stop) {
  const iso = [`${adapter.toISO(stop ?? start).split("T")[0]}T00:00:00Z`, `${adapter.toISO(start).split("T")[0]}T00:00:00Z`];
  return isString(adapter.date()) ? adapter.getDiff(iso[0], iso[1], "days") : adapter.getDiff(adapter.date(iso[0]), adapter.date(iso[1]), "days");
}
function createInstance(options, locale) {
  const instance = reactive(isFunction(options.adapter) ? new options.adapter({
    locale: options.locale[locale.current.value] ?? locale.current.value,
    formats: options.formats
  }) : options.adapter);
  watch(locale.current, (value) => {
    instance.locale = options.locale[value] ?? value ?? instance.locale;
  });
  return instance;
}
function useDate() {
  const options = inject$1(DateOptionsSymbol);
  if (!options) throw new Error("[Vuetify] Could not find injected date options");
  const locale = useLocale();
  return createInstance(options, locale);
}
const DisplaySymbol = /* @__PURE__ */ Symbol.for("vuetify:display");
const defaultDisplayOptions = {
  mobileBreakpoint: "lg",
  thresholds: {
    xs: 0,
    sm: 600,
    md: 840,
    lg: 1145,
    xl: 1545,
    xxl: 2138
  }
};
const parseDisplayOptions = (options = defaultDisplayOptions) => {
  return mergeDeep(defaultDisplayOptions, options);
};
function getClientWidth(ssr) {
  return isObject(ssr) && ssr.clientWidth || 0;
}
function getClientHeight(ssr) {
  return isObject(ssr) && ssr.clientHeight || 0;
}
function getPlatform(ssr) {
  const userAgent = "ssr";
  function match(regexp) {
    return Boolean(userAgent.match(regexp));
  }
  const android = match(/android/i);
  const ios = match(/iphone|ipad|ipod/i);
  const cordova = match(/cordova/i);
  const electron = match(/electron/i);
  const chrome = match(/chrome/i);
  const edge = match(/edge/i);
  const firefox = match(/firefox/i);
  const opera = match(/opera/i);
  const win = match(/win/i);
  const mac = match(/mac/i);
  const linux = match(/linux/i);
  return {
    android,
    ios,
    cordova,
    electron,
    chrome,
    edge,
    firefox,
    opera,
    win,
    mac,
    linux,
    touch: SUPPORTS_TOUCH,
    ssr: userAgent === "ssr"
  };
}
function createDisplay(options, ssr) {
  const {
    thresholds,
    mobileBreakpoint
  } = parseDisplayOptions(options);
  const height = shallowRef(getClientHeight(ssr));
  const platform = shallowRef(getPlatform());
  const state = reactive({});
  const width = shallowRef(getClientWidth(ssr));
  function updateSize() {
    height.value = getClientHeight();
    width.value = getClientWidth();
  }
  function update() {
    updateSize();
    platform.value = getPlatform();
  }
  watchEffect(() => {
    const xs = width.value < thresholds.sm;
    const sm = width.value < thresholds.md && !xs;
    const md = width.value < thresholds.lg && !(sm || xs);
    const lg = width.value < thresholds.xl && !(md || sm || xs);
    const xl = width.value < thresholds.xxl && !(lg || md || sm || xs);
    const xxl = width.value >= thresholds.xxl;
    const name = xs ? "xs" : sm ? "sm" : md ? "md" : lg ? "lg" : xl ? "xl" : "xxl";
    const breakpointValue = isNumber(mobileBreakpoint) ? mobileBreakpoint : thresholds[mobileBreakpoint];
    const mobile = width.value < breakpointValue;
    state.xs = xs;
    state.sm = sm;
    state.md = md;
    state.lg = lg;
    state.xl = xl;
    state.xxl = xxl;
    state.smAndUp = !xs;
    state.mdAndUp = !(xs || sm);
    state.lgAndUp = !(xs || sm || md);
    state.xlAndUp = !(xs || sm || md || lg);
    state.smAndDown = !(md || lg || xl || xxl);
    state.mdAndDown = !(lg || xl || xxl);
    state.lgAndDown = !(xl || xxl);
    state.xlAndDown = !xxl;
    state.name = name;
    state.height = height.value;
    state.width = width.value;
    state.mobile = mobile;
    state.mobileBreakpoint = mobileBreakpoint;
    state.platform = platform.value;
    state.thresholds = thresholds;
  });
  return {
    ...toRefs(state),
    update,
    ssr: !!ssr
  };
}
const GoToSymbol = /* @__PURE__ */ Symbol.for("vuetify:goto");
function genDefaults$1() {
  return {
    container: void 0,
    duration: 300,
    layout: false,
    offset: 0,
    easing: "easeInOutCubic",
    patterns: easingPatterns
  };
}
function createGoTo(options, locale) {
  return {
    rtl: locale.isRtl,
    options: mergeDeep(genDefaults$1(), options)
  };
}
const ThemeSymbol = /* @__PURE__ */ Symbol.for("vuetify:theme");
const makeThemeProps = propsFactory({
  theme: String
}, "theme");
function genDefaults() {
  return {
    defaultTheme: "system",
    prefix: "v-",
    variations: {
      colors: [],
      lighten: 0,
      darken: 0
    },
    themes: {
      light: {
        dark: false,
        colors: {
          background: "#FFFFFF",
          surface: "#FFFFFF",
          "surface-bright": "#FFFFFF",
          "surface-light": "#EEEEEE",
          "surface-variant": "#424242",
          "on-surface-variant": "#EEEEEE",
          primary: "#1867C0",
          "primary-darken-1": "#1F5592",
          secondary: "#48A9A6",
          "secondary-darken-1": "#018786",
          error: "#B00020",
          info: "#2196F3",
          success: "#4CAF50",
          warning: "#FB8C00"
        },
        variables: {
          "border-color": "#000000",
          "border-opacity": 0.12,
          "shadow-color": "#000000",
          "high-emphasis-opacity": 0.87,
          "medium-emphasis-opacity": 0.6,
          "disabled-opacity": 0.38,
          "idle-opacity": 0.04,
          "hover-opacity": 0.04,
          "focus-opacity": 0.12,
          "selected-opacity": 0.08,
          "activated-opacity": 0.12,
          "pressed-opacity": 0.12,
          "dragged-opacity": 0.08,
          "theme-kbd": "#EEEEEE",
          "theme-on-kbd": "#000000",
          "theme-code": "#F5F5F5",
          "theme-on-code": "#000000",
          "theme-on-dark": "#FFF",
          "theme-on-light": "#000",
          "elevation-overlay-color": "black",
          "elevation-overlay-opacity-step": "2%",
          "highlight-opacity": "8%"
        }
      },
      dark: {
        dark: true,
        colors: {
          background: "#121212",
          surface: "#212121",
          "surface-bright": "#ccbfd6",
          "surface-light": "#424242",
          "surface-variant": "#c8c8c8",
          "on-surface-variant": "#000000",
          primary: "#2196F3",
          "primary-darken-1": "#277CC1",
          secondary: "#54B6B2",
          "secondary-darken-1": "#48A9A6",
          error: "#CF6679",
          info: "#2196F3",
          success: "#4CAF50",
          warning: "#FB8C00"
        },
        variables: {
          "border-color": "#FFFFFF",
          "border-opacity": 0.12,
          "shadow-color": "#000000",
          "high-emphasis-opacity": 1,
          "medium-emphasis-opacity": 0.7,
          "disabled-opacity": 0.5,
          "idle-opacity": 0.1,
          "hover-opacity": 0.04,
          "focus-opacity": 0.12,
          "selected-opacity": 0.08,
          "activated-opacity": 0.12,
          "pressed-opacity": 0.16,
          "dragged-opacity": 0.08,
          "theme-kbd": "#424242",
          "theme-on-kbd": "#FFFFFF",
          "theme-code": "#343434",
          "theme-on-code": "#CCCCCC",
          "theme-on-dark": "#FFF",
          "theme-on-light": "#000",
          "elevation-overlay-color": "white",
          "elevation-overlay-opacity-step": "2%",
          "highlight-opacity": "20%"
        }
      }
    },
    stylesheetId: "vuetify-theme-stylesheet",
    scoped: false,
    utilities: true,
    transition: false
  };
}
function parseThemeOptions(options = genDefaults()) {
  const defaults = genDefaults();
  if (!options) return {
    ...defaults,
    isDisabled: true
  };
  return mergeDeep(defaults, options);
}
function createCssClass(lines, selector, content, scope) {
  lines.push(`${getScopedSelector(selector, scope)} {
`, ...content.map((line) => `  ${line};
`), "}\n");
}
function genCssVariables(theme, prefix) {
  const lightOverlay = theme.dark ? 2 : 1;
  const darkOverlay = theme.dark ? 1 : 2;
  const variables = [];
  for (const [key, value] of Object.entries(theme.colors)) {
    const rgb = parseColor(value);
    variables.push(`--${prefix}theme-${key}: ${rgb.r},${rgb.g},${rgb.b}` + (isNumber(rgb.a) ? `,${rgb.a}` : ""));
    if (!key.startsWith("on-")) {
      variables.push(`--${prefix}theme-${key}-overlay-multiplier: ${getLuma(value) > 0.18 ? lightOverlay : darkOverlay}`);
    }
  }
  for (const [key, value] of Object.entries(theme.variables)) {
    const color = isString(value) && value.startsWith("#") ? parseColor(value) : void 0;
    const rgb = color ? `${color.r}, ${color.g}, ${color.b}` + (isNumber(color.a) ? `, ${color.a}` : "") : void 0;
    variables.push(`--${prefix}${key}: ${rgb ?? value}`);
  }
  return variables;
}
function genVariation(name, color, variations) {
  const object = {};
  if (variations) {
    for (const variation of ["lighten", "darken"]) {
      const fn = variation === "lighten" ? lighten : darken;
      for (const amount of range(variations[variation], 1)) {
        object[`${name}-${variation}-${amount}`] = RGBtoHex(fn(parseColor(color), amount));
      }
    }
  }
  return object;
}
function genVariations(colors, variations) {
  if (!variations) return {};
  let variationColors = {};
  for (const name of variations.colors) {
    const color = colors[name];
    if (!color) continue;
    variationColors = {
      ...variationColors,
      ...genVariation(name, color, variations)
    };
  }
  return variationColors;
}
function genOnColors(colors, variables) {
  const onColors = {};
  for (const color of Object.keys(colors)) {
    if (color.startsWith("on-") || colors[`on-${color}`]) continue;
    const onColor = `on-${color}`;
    const colorVal = parseColor(colors[color]);
    onColors[onColor] = hasLightForeground(colorVal) ? variables["theme-on-dark"] : variables["theme-on-light"];
  }
  return onColors;
}
function getScopedSelector(selector, scope) {
  if (!scope) return selector;
  const scopeSelector = `:where(${scope})`;
  return selector === ":root" ? scopeSelector : `${scopeSelector} ${selector}`;
}
function upsertStyles(id, cspNonce, styles) {
  const styleEl = getOrCreateStyleElement();
  if (!styleEl) return;
  styleEl.innerHTML = styles;
}
function getOrCreateStyleElement(id, cspNonce) {
  return null;
}
function createTheme(options) {
  const parsedOptions = parseThemeOptions(options);
  const _name = shallowRef(parsedOptions.defaultTheme);
  const themes = ref(parsedOptions.themes);
  const systemName = shallowRef("light");
  const name = computed({
    get() {
      return _name.value === "system" ? systemName.value : _name.value;
    },
    set(val) {
      _name.value = val;
    }
  });
  const computedThemes = computed(() => {
    const acc = {};
    for (const [name2, original] of Object.entries(themes.value)) {
      const defaultTheme = original.dark || name2 === "dark" ? themes.value.dark : themes.value.light;
      const merged = mergeDeep(defaultTheme, original);
      const colors = {
        ...merged.colors,
        ...genVariations(merged.colors, parsedOptions.variations)
      };
      acc[name2] = {
        ...merged,
        colors: {
          ...colors,
          ...genOnColors(colors, merged.variables)
        }
      };
    }
    return acc;
  });
  const current = toRef(() => computedThemes.value[name.value]);
  const isSystem = toRef(() => _name.value === "system");
  const styles = computed(() => {
    const lines = [];
    const scoped = parsedOptions.scoped ? parsedOptions.prefix : "";
    lines.push("@layer theme-base {\n");
    createCssClass(lines, ":root", [`color-scheme: ${current.value?.dark ? "dark" : "light"}`], parsedOptions.scope);
    createCssClass(lines, ":root", genCssVariables(current.value, parsedOptions.prefix), parsedOptions.scope);
    for (const [themeName, theme] of Object.entries(computedThemes.value)) {
      createCssClass(lines, `.${parsedOptions.prefix}theme--${themeName}`, [`color-scheme: ${theme.dark ? "dark" : "light"}`, ...genCssVariables(theme, parsedOptions.prefix)], parsedOptions.scope);
    }
    lines.push("}\n");
    if (parsedOptions.utilities) {
      const bgLines = [];
      const fgLines = [];
      const colors = new Set(Object.values(computedThemes.value).flatMap((theme) => Object.keys(theme.colors)));
      for (const key of colors) {
        if (key.startsWith("on-")) {
          createCssClass(fgLines, `.${key}`, [`color: rgb(var(--${parsedOptions.prefix}theme-${key}))`], parsedOptions.scope);
        } else {
          createCssClass(bgLines, `.${scoped}bg-${key}`, [`--${parsedOptions.prefix}theme-overlay-multiplier: var(--${parsedOptions.prefix}theme-${key}-overlay-multiplier)`, `background-color: rgb(var(--${parsedOptions.prefix}theme-${key}))`, `color: rgb(var(--${parsedOptions.prefix}theme-on-${key}))`], parsedOptions.scope);
          createCssClass(fgLines, `.${scoped}text-${key}`, [`color: rgb(var(--${parsedOptions.prefix}theme-${key}))`], parsedOptions.scope);
          createCssClass(fgLines, `.${scoped}border-${key}`, [`--${parsedOptions.prefix}border-color: var(--${parsedOptions.prefix}theme-${key})`], parsedOptions.scope);
        }
      }
      lines.push("@layer theme-background {\n", ...bgLines.map((v) => `  ${v}`), "}\n", "@layer theme-foreground {\n", ...fgLines.map((v) => `  ${v}`), "}\n");
    }
    return "@layer vuetify-utilities {\n" + lines.map((v) => `  ${v}`).join("") + "\n}";
  });
  const themeClasses = toRef(() => parsedOptions.isDisabled ? void 0 : `${parsedOptions.prefix}theme--${name.value}`);
  const themeNames = toRef(() => Object.keys(computedThemes.value));
  function install(app) {
    if (parsedOptions.isDisabled) return;
    const head = app._context.provides.usehead;
    if (head) {
      let getHead = function() {
        return {
          style: [{
            textContent: styles.value,
            id: parsedOptions.stylesheetId,
            nonce: parsedOptions.cspNonce || false,
            tagPosition: "bodyOpen"
          }]
        };
      };
      if (head.push) {
        head.push(getHead);
      } else {
        {
          head.addHeadObjs(getHead());
        }
      }
    } else {
      let updateStyles = function() {
        upsertStyles(parsedOptions.stylesheetId, parsedOptions.cspNonce, styles.value);
      };
      {
        updateStyles();
      }
    }
  }
  let _transitionOrigin;
  function setTransitionOrigin(e) {
    if (!e) {
      _transitionOrigin = null;
      return;
    }
    let x;
    let y;
    if (isElement(e)) {
      const box = new Box(e);
      x = box.left + box.width / 2;
      y = box.top + box.height / 2;
    } else {
      x = e.clientX;
      y = e.clientY;
    }
    const originX = Math.min(98.9, 100 * x / (void 0).innerWidth);
    const originY = 100 * y / (void 0).innerHeight;
    _transitionOrigin = `${originX.toFixed(2)}% ${originY.toFixed(2)}%`;
  }
  function resolveTransitionOptions(transition) {
    const opt = transition ?? parsedOptions.transition;
    if (!opt && !_transitionOrigin) return false;
    const global = isObject(parsedOptions.transition) ? parsedOptions.transition : {};
    const local = isObject(transition) ? transition : {};
    return {
      origin: local.origin ?? _transitionOrigin ?? global.origin,
      duration: local.duration ?? global.duration
    };
  }
  function withPageTransition(callback, options2) {
    {
      callback();
      return Promise.resolve();
    }
  }
  function change(themeName, transition) {
    if (themeName !== "system" && !themeNames.value.includes(themeName)) {
      consoleWarn(`Theme "${themeName}" not found on the Vuetify theme instance`);
      return Promise.resolve();
    }
    const apply = () => {
      name.value = themeName;
    };
    const transitionOptions = resolveTransitionOptions(transition);
    if (transitionOptions) {
      return withPageTransition(apply);
    }
    apply();
    return Promise.resolve();
  }
  function cycle(themeArray = themeNames.value, transition) {
    const currentIndex = themeArray.indexOf(name.value);
    const nextIndex = currentIndex === -1 ? 0 : (currentIndex + 1) % themeArray.length;
    return change(themeArray[nextIndex], transition);
  }
  function toggle(themeArray = ["light", "dark"], transition) {
    return cycle(themeArray, transition);
  }
  const globalName = new Proxy(name, {
    get(target, prop) {
      return Reflect.get(target, prop);
    },
    set(target, prop, val) {
      if (prop === "value") {
        deprecate(`theme.global.name.value = ${val}`, `theme.change('${val}')`);
      }
      return Reflect.set(target, prop, val);
    }
  });
  return {
    install,
    change,
    cycle,
    toggle,
    setTransitionOrigin,
    isDisabled: parsedOptions.isDisabled,
    isSystem,
    name,
    themes,
    current,
    computedThemes,
    prefix: parsedOptions.prefix,
    themeClasses,
    styles,
    global: {
      name: globalName,
      current
    }
  };
}
function provideTheme(props) {
  getCurrentInstance("provideTheme");
  const theme = inject$1(ThemeSymbol, null);
  if (!theme) throw new Error("Could not find Vuetify theme injection");
  const name = toRef(() => props.theme ?? theme.name.value);
  const current = toRef(() => theme.computedThemes.value[name.value]);
  const themeClasses = toRef(() => theme.isDisabled ? void 0 : `${theme.prefix}theme--${name.value}`);
  const newTheme = {
    ...theme,
    name,
    current,
    themeClasses
  };
  provide(ThemeSymbol, newTheme);
  return newTheme;
}
function useTheme() {
  getCurrentInstance("useTheme");
  const theme = inject$1(ThemeSymbol, null);
  if (!theme) throw new Error("Could not find Vuetify theme injection");
  return theme;
}
function useResizeObserver(callback, box = "content") {
  const resizeRef = templateRef();
  const contentRect = ref();
  return {
    resizeRef,
    contentRect: readonly(contentRect)
  };
}
function createVuetify(vuetify = {}) {
  const {
    blueprint,
    ...rest
  } = vuetify;
  const options = mergeDeep(blueprint, rest);
  const {
    aliases: aliases2 = {},
    components = {},
    directives = {}
  } = options;
  const scope = effectScope();
  return scope.run(() => {
    const defaults = createDefaults(options.defaults);
    const display = createDisplay(options.display, options.ssr);
    const theme = createTheme(options.theme);
    const icons = createIcons(options.icons);
    const locale = createLocale(options.locale);
    const date2 = createDate(options.date, locale);
    const goTo = createGoTo(options.goTo, locale);
    function install(app) {
      for (const key in directives) {
        app.directive(key, directives[key]);
      }
      for (const key in components) {
        app.component(key, components[key]);
      }
      for (const key in aliases2) {
        app.component(key, defineComponent({
          ...aliases2[key],
          name: key,
          aliasName: aliases2[key].name
        }));
      }
      const appScope = effectScope();
      appScope.run(() => {
        theme.install(app);
      });
      app.onUnmount(() => appScope.stop());
      app.provide(DefaultsSymbol, defaults);
      app.provide(RootDefaultsSymbol, defaults);
      app.provide(DisplaySymbol, display);
      app.provide(ThemeSymbol, theme);
      app.provide(IconSymbol, icons);
      app.provide(LocaleSymbol, locale);
      app.provide(DateOptionsSymbol, date2.options);
      app.provide(DateAdapterSymbol, date2.instance);
      app.provide(GoToSymbol, goTo);
      {
        app.mixin({
          computed: {
            $vuetify() {
              return reactive({
                defaults: inject.call(this, DefaultsSymbol),
                display: inject.call(this, DisplaySymbol),
                theme: inject.call(this, ThemeSymbol),
                icons: inject.call(this, IconSymbol),
                locale: inject.call(this, LocaleSymbol),
                date: inject.call(this, DateAdapterSymbol)
              });
            }
          }
        });
      }
    }
    function unmount() {
      scope.stop();
    }
    return {
      install,
      unmount,
      defaults,
      display,
      theme,
      icons,
      locale,
      date: date2,
      goTo
    };
  });
}
const version = "4.2.6";
createVuetify.version = version;
function inject(key) {
  const vm = this.$;
  const provides = vm.parent?.provides ?? vm.vnode.appContext?.provides;
  if (provides && key in provides) {
    return provides[key];
  }
}
const makeBorderProps = propsFactory({
  border: [Boolean, Number, String]
}, "border");
function useBorder(props, name = getCurrentInstanceName()) {
  const borderClasses = computed(() => {
    const border = props.border;
    if (border === true || border === "") {
      return `${name}--border`;
    } else if (isString(border) || border === 0) {
      return String(border).split(" ").map((v) => `border-${v}`);
    }
    return [];
  });
  return {
    borderClasses
  };
}
const allowedDensities = [null, "default", "comfortable", "compact"];
const makeDensityProps = propsFactory({
  density: {
    type: String,
    default: "default",
    validator: (v) => allowedDensities.includes(v)
  }
}, "density");
function useDensity(props, name = getCurrentInstanceName()) {
  const densityClasses = toRef(() => {
    return `${name}--density-${props.density}`;
  });
  return {
    densityClasses
  };
}
const makeElevationProps = propsFactory({
  elevation: {
    type: [Number, String],
    // no limit to allow both 0-6 (MD3) and legacy 0-24 (MD2)
    validator: (value) => parseInt(value) >= 0
  },
  hoverElevation: {
    type: [Number, String],
    validator: (value) => parseInt(value) >= 0
  }
}, "elevation");
function useElevation(props) {
  const elevationClasses = toRef(() => {
    const elevation = isRef(props) ? props.value : props.elevation;
    const hoverElevation = isRef(props) ? null : props.hoverElevation;
    return [...isNullOrUndefined(elevation) ? [] : [`elevation-${parseInt(elevation)}`], ...isNullOrUndefined(hoverElevation) ? [] : [`hover-elevation-${parseInt(hoverElevation)}`]];
  });
  return {
    elevationClasses
  };
}
const CSS_LENGTH = /^-?\d*\.?\d+([a-z%]{0,4})$/i;
function splitTokens(v) {
  return String(v).trim().split(/\s+/);
}
function isCssLength(token) {
  return token.toLowerCase().match(CSS_LENGTH)?.at(1)?.startsWith("x") === false;
}
function isCssValue(v) {
  return typeof v === "string" && !!v && (v.includes("(") || splitTokens(v).every(isCssLength));
}
const makeRoundedProps = propsFactory({
  rounded: {
    type: [Boolean, Number, String],
    default: void 0
  },
  tile: Boolean
}, "rounded");
function useRounded(props, name = getCurrentInstanceName()) {
  const roundedClasses = computed(() => {
    const rounded = isRef(props) ? props.value : props.rounded;
    const tile = isRef(props) ? false : props.tile;
    const classes = [];
    if (tile || rounded === false || String(rounded) === "0") {
      classes.push("rounded-0");
    } else if (rounded === true || rounded === "") {
      classes.push(`${name}--rounded`);
    } else if (isString(rounded) && !isCssValue(rounded)) {
      for (const value of splitTokens(rounded)) {
        classes.push(`rounded-${value}`);
      }
    }
    return classes;
  });
  const roundedStyles = computed(() => {
    const rounded = isRef(props) ? props.value : props.rounded;
    if ((!isCssValue(rounded) || String(rounded) === "0") && !(typeof rounded === "number" && rounded !== 0)) {
      return {};
    }
    return {
      borderRadius: convertToUnit(rounded)
    };
  });
  return {
    roundedClasses,
    roundedStyles
  };
}
const predefinedSizes = ["x-small", "small", "default", "large", "x-large"];
const makeSizeProps = propsFactory({
  size: {
    type: [String, Number],
    default: "default"
  }
}, "size");
function useSize(props, name = getCurrentInstanceName()) {
  return destructComputed(() => {
    const size = props.size;
    let sizeClasses;
    let sizeStyles;
    if (includes(predefinedSizes, size)) {
      sizeClasses = `${name}--size-${size}`;
    } else if (size) {
      sizeStyles = {
        width: convertToUnit(size),
        height: convertToUnit(size)
      };
    }
    return {
      sizeClasses,
      sizeStyles
    };
  });
}
const makeTagProps = propsFactory({
  tag: {
    type: [String, Object, Function],
    default: "div"
  }
}, "tag");
function useColor(colors) {
  return destructComputed(() => {
    const {
      class: colorClasses,
      style: colorStyles
    } = computeColor(colors);
    return {
      colorClasses,
      colorStyles
    };
  });
}
function useTextColor(color) {
  const {
    colorClasses: textColorClasses,
    colorStyles: textColorStyles
  } = useColor(() => ({
    text: toValue(color)
  }));
  return {
    textColorClasses,
    textColorStyles
  };
}
function useBackgroundColor(color) {
  const {
    colorClasses: backgroundColorClasses,
    colorStyles: backgroundColorStyles
  } = useColor(() => ({
    background: toValue(color)
  }));
  return {
    backgroundColorClasses,
    backgroundColorStyles
  };
}
function normalizeColors(colors) {
  return {
    text: isString(colors.text) ? colors.text.replace(/^text-/, "") : colors.text,
    background: isString(colors.background) ? colors.background.replace(/^bg-/, "") : colors.background
  };
}
function computeColor(colors) {
  const _colors = normalizeColors(toValue(colors));
  const classes = [];
  const styles = {};
  if (_colors.background) {
    if (isCssColor(_colors.background)) {
      styles.backgroundColor = _colors.background;
      if (!_colors.text && isParsableColor(_colors.background)) {
        const backgroundColor = parseColor(_colors.background);
        if (!isNumber(backgroundColor.a) || backgroundColor.a === 1) {
          classes.push(hasLightForeground(backgroundColor) ? "v-theme-on-dark" : "v-theme-on-light");
        }
      }
    } else {
      classes.push(`bg-${_colors.background}`);
    }
  }
  if (_colors.text) {
    if (isCssColor(_colors.text)) {
      styles.color = _colors.text;
      styles.caretColor = _colors.text;
    } else {
      classes.push(`text-${_colors.text}`);
    }
  }
  return {
    class: classes,
    style: styles
  };
}
const allowedVariants = ["elevated", "flat", "tonal", "outlined", "text", "plain"];
function genOverlays(isClickable, name) {
  return createElementVNode(Fragment, null, [createElementVNode("span", {
    "key": "overlay",
    "class": normalizeClass(`${name}__overlay`)
  }, null), createElementVNode("span", {
    "key": "underlay",
    "class": normalizeClass(`${name}__underlay`)
  }, null)]);
}
const makeVariantProps = propsFactory({
  color: String,
  variant: {
    type: String,
    default: "elevated",
    validator: (v) => allowedVariants.includes(v)
  }
}, "variant");
function useVariant(props, name = getCurrentInstanceName()) {
  const variantClasses = toRef(() => {
    const {
      variant
    } = toValue(props);
    return `${name}--variant-${variant}`;
  });
  const {
    colorClasses,
    colorStyles
  } = useColor(() => {
    const {
      variant,
      color
    } = toValue(props);
    return {
      [["elevated", "flat"].includes(variant) ? "background" : "text"]: color
    };
  });
  return {
    colorClasses,
    colorStyles,
    variantClasses
  };
}
const makeVBtnGroupProps = propsFactory({
  baseColor: String,
  divided: Boolean,
  direction: {
    type: String,
    default: "horizontal"
  },
  ...makeBorderProps(),
  ...makeComponentProps(),
  ...makeDensityProps(),
  ...makeElevationProps(),
  ...makeRoundedProps(),
  ...makeSizeProps({
    size: void 0
  }),
  ...makeTagProps(),
  ...makeThemeProps(),
  ...makeVariantProps()
}, "VBtnGroup");
const VBtnGroup = genericComponent()({
  name: "VBtnGroup",
  props: makeVBtnGroupProps(),
  setup(props, {
    slots
  }) {
    const {
      themeClasses
    } = provideTheme(props);
    const {
      densityClasses
    } = useDensity(props);
    const {
      borderClasses
    } = useBorder(props);
    const {
      elevationClasses
    } = useElevation(props);
    const {
      roundedClasses,
      roundedStyles
    } = useRounded(props);
    provideDefaults({
      VBtn: {
        height: toRef(() => props.direction === "horizontal" && props.size == null ? "auto" : null),
        baseColor: toRef(() => props.baseColor),
        color: toRef(() => props.color),
        density: toRef(() => props.density),
        flat: true,
        size: toRef(() => props.size),
        variant: toRef(() => props.variant)
      }
    });
    useRender(() => {
      return createVNode(props.tag, {
        "class": normalizeClass(["v-btn-group", `v-btn-group--${props.direction}`, {
          "v-btn-group--divided": props.divided,
          "v-btn-group--has-size": props.size != null
        }, themeClasses.value, borderClasses.value, densityClasses.value, elevationClasses.value, roundedClasses.value, props.class]),
        "style": normalizeStyle([roundedStyles.value, props.style])
      }, slots);
    });
  }
});
const makeGroupProps = propsFactory({
  modelValue: {
    type: null,
    default: void 0
  },
  multiple: Boolean,
  mandatory: [Boolean, String],
  max: Number,
  selectedClass: String,
  disabled: Boolean
}, "group");
const makeGroupItemProps = propsFactory({
  value: null,
  disabled: Boolean,
  selectedClass: String
}, "group-item");
function useGroupItem(props, injectKey, required = true) {
  const vm = getCurrentInstance("useGroupItem");
  if (!vm) {
    throw new Error("[Vuetify] useGroupItem composable must be used inside a component setup function");
  }
  const id = useId();
  provide(/* @__PURE__ */ Symbol.for(`${injectKey.description}:id`), id);
  const group = inject$1(injectKey, null);
  if (!group) {
    if (!required) return group;
    throw new Error(`[Vuetify] Could not find useGroup injection with symbol ${injectKey.description}`);
  }
  const value = toRef(() => props.value);
  const disabled = computed(() => !!(group.disabled.value || props.disabled));
  function register() {
    group?.register({
      id,
      value,
      disabled
    }, vm);
  }
  function unregister() {
    group?.unregister(id);
  }
  register();
  const isSelected = computed(() => {
    return group.isSelected(id);
  });
  const isFirst = computed(() => {
    return group.items.value[0].id === id;
  });
  const isLast = computed(() => {
    return group.items.value[group.items.value.length - 1].id === id;
  });
  const selectedClass = computed(() => isSelected.value && [group.selectedClass.value, props.selectedClass]);
  watch(isSelected, (value2) => {
    vm.emit("group:selected", {
      value: value2
    });
  }, {
    flush: "sync"
  });
  return {
    id,
    isSelected,
    isFirst,
    isLast,
    toggle: () => group.select(id, !isSelected.value),
    select: (value2) => group.select(id, value2),
    selectedClass,
    value,
    disabled,
    group,
    register,
    unregister
  };
}
function useGroup(props, injectKey) {
  const items = reactive([]);
  const selected = useProxiedModel(props, "modelValue", [], (v) => {
    if (isUndefined(v)) return [];
    return getIds(items, isNull(v) ? [null] : wrapInArray(v));
  }, (v) => {
    const arr = getValues(items, v);
    return props.multiple ? arr : arr[0];
  });
  const groupVm = getCurrentInstance("useGroup");
  function register(item, vm) {
    const unwrapped = item;
    const key = /* @__PURE__ */ Symbol.for(`${injectKey.description}:id`);
    const children = findChildrenWithProvide(key, groupVm?.vnode);
    const index = children.indexOf(vm);
    if (isUndefined(unref(unwrapped.value))) {
      unwrapped.value = index;
      unwrapped.useIndexAsValue = true;
    }
    if (index > -1) {
      items.splice(index, 0, unwrapped);
    } else {
      items.push(unwrapped);
    }
  }
  function unregister(id) {
    forceMandatoryValue();
    const index = items.findIndex((item) => item.id === id);
    items.splice(index, 1);
  }
  function forceMandatoryValue() {
    const item = items.find((item2) => !item2.disabled);
    if (item && props.mandatory === "force" && !selected.value.length) {
      selected.value = [item.id];
    }
  }
  function select(id, value) {
    const item = items.find((item2) => item2.id === id);
    if (value && item?.disabled) return;
    if (props.multiple) {
      const internalValue = selected.value.slice();
      const index = internalValue.findIndex((v) => v === id);
      const isSelected = ~index;
      value = value ?? !isSelected;
      if (isSelected && props.mandatory && internalValue.length <= 1) return;
      if (!isSelected && isNumber(props.max) && internalValue.length + 1 > props.max) return;
      if (index < 0 && value) internalValue.push(id);
      else if (index >= 0 && !value) internalValue.splice(index, 1);
      selected.value = internalValue;
    } else {
      const isSelected = selected.value.includes(id);
      if (props.mandatory && isSelected) return;
      if (!isSelected && !value) return;
      selected.value = value ?? !isSelected ? [id] : [];
    }
  }
  function step(offset) {
    if (props.multiple) consoleWarn('This method is not supported when using "multiple" prop');
    if (!selected.value.length) {
      const item = items.find((item2) => !item2.disabled);
      item && (selected.value = [item.id]);
    } else {
      const currentId = selected.value[0];
      const currentIndex = items.findIndex((i) => i.id === currentId);
      let newIndex = (currentIndex + offset) % items.length;
      let newItem = items[newIndex];
      while (newItem.disabled && newIndex !== currentIndex) {
        newIndex = (newIndex + offset) % items.length;
        newItem = items[newIndex];
      }
      if (newItem.disabled) return;
      selected.value = [items[newIndex].id];
    }
  }
  const state = {
    register,
    unregister,
    selected,
    select,
    disabled: toRef(() => props.disabled),
    prev: () => step(items.length - 1),
    next: () => step(1),
    isSelected: (id) => selected.value.includes(id),
    selectedClass: toRef(() => props.selectedClass),
    items: toRef(() => items),
    getItemIndex: (value) => getItemIndex(items, value)
  };
  provide(injectKey, state);
  return state;
}
function getItemIndex(items, value) {
  const ids = getIds(items, [value]);
  if (!ids.length) return -1;
  return items.findIndex((item) => item.id === ids[0]);
}
function getIds(items, modelValue) {
  const ids = [];
  modelValue.forEach((value) => {
    const item = items.find((item2) => deepEqual(value, item2.value));
    const itemByIndex = items[value];
    if (!isUndefined(item?.value)) {
      ids.push(item.id);
    } else if (itemByIndex?.useIndexAsValue) {
      ids.push(itemByIndex.id);
    }
  });
  return ids;
}
function getValues(items, ids) {
  const values = [];
  ids.forEach((id) => {
    const itemIndex = items.findIndex((item) => item.id === id);
    if (~itemIndex) {
      const item = items[itemIndex];
      values.push(!isUndefined(item.value) ? item.value : itemIndex);
    }
  });
  return values;
}
const VBtnToggleSymbol = /* @__PURE__ */ Symbol.for("vuetify:v-btn-toggle");
const makeVBtnToggleProps = propsFactory({
  ...makeVBtnGroupProps(),
  ...makeGroupProps()
}, "VBtnToggle");
genericComponent()({
  name: "VBtnToggle",
  props: makeVBtnToggleProps(),
  emits: {
    "update:modelValue": (value) => true
  },
  setup(props, {
    slots
  }) {
    const {
      isSelected,
      next,
      prev,
      select,
      selected
    } = useGroup(props, VBtnToggleSymbol);
    useRender(() => {
      const btnGroupProps = VBtnGroup.filterProps(props);
      return createVNode(VBtnGroup, mergeProps({
        "class": ["v-btn-toggle", {
          "v-btn-toggle--has-color": !!props.color
        }, props.class]
      }, btnGroupProps, {
        "style": props.style
      }), {
        default: () => [slots.default?.({
          isSelected,
          next,
          prev,
          select,
          selected
        })]
      });
    });
    return {
      next,
      prev,
      select
    };
  }
});
const makeVDefaultsProviderProps = propsFactory({
  defaults: Object,
  disabled: Boolean,
  reset: [Number, String],
  root: [Boolean, String],
  scoped: Boolean
}, "VDefaultsProvider");
const VDefaultsProvider = genericComponent(false)({
  name: "VDefaultsProvider",
  props: makeVDefaultsProviderProps(),
  setup(props, {
    slots
  }) {
    const {
      defaults,
      disabled,
      reset,
      root,
      scoped
    } = toRefs(props);
    provideDefaults(defaults, {
      reset,
      root,
      scoped,
      disabled
    });
    return () => slots.default?.();
  }
});
const makeVIconProps = propsFactory({
  color: String,
  disabled: Boolean,
  start: Boolean,
  end: Boolean,
  icon: IconValue,
  opacity: [String, Number],
  ...makeComponentProps(),
  ...makeSizeProps(),
  ...makeTagProps({
    tag: "i"
  }),
  ...makeThemeProps()
}, "VIcon");
const VIcon = genericComponent()({
  name: "VIcon",
  props: makeVIconProps(),
  setup(props, {
    attrs,
    slots
  }) {
    const slotIcon = shallowRef();
    const {
      themeClasses
    } = provideTheme(props);
    const {
      iconData
    } = useIcon(() => slotIcon.value || props.icon);
    const {
      sizeClasses
    } = useSize(props);
    const {
      textColorClasses,
      textColorStyles
    } = useTextColor(() => props.color);
    useRender(() => {
      const slotValue = slots.default?.();
      if (slotValue) {
        slotIcon.value = flattenFragments(slotValue).filter((node) => node.type === Text && node.children && isString(node.children))[0]?.children;
      }
      const hasClick = !!(attrs.onClick || attrs.onClickOnce);
      return createVNode(iconData.value.component, {
        "tag": props.tag,
        "icon": iconData.value.icon,
        "class": normalizeClass(["v-icon", "notranslate", themeClasses.value, sizeClasses.value, textColorClasses.value, {
          "v-icon--clickable": hasClick,
          "v-icon--disabled": props.disabled,
          "v-icon--start": props.start,
          "v-icon--end": props.end
        }, props.class]),
        "style": normalizeStyle([{
          "--v-icon-opacity": props.opacity
        }, !sizeClasses.value ? {
          fontSize: convertToUnit(props.size),
          height: convertToUnit(props.size),
          width: convertToUnit(props.size)
        } : void 0, textColorStyles.value, props.style]),
        "role": hasClick ? "button" : void 0,
        "aria-hidden": !hasClick,
        "tabindex": hasClick ? props.disabled ? -1 : 0 : void 0
      }, {
        default: () => [slotValue]
      });
    });
    return {};
  }
});
function useIntersectionObserver(callback, options) {
  const intersectionRef = ref();
  const isIntersecting = shallowRef(false);
  return {
    intersectionRef,
    isIntersecting
  };
}
const makeRevealProps = propsFactory({
  reveal: {
    type: [Boolean, Object],
    default: false
  }
}, "reveal");
function useReveal(props) {
  const defaultDuration = 900;
  const duration = toRef(() => isObject(props.reveal) ? Math.max(0, Number(props.reveal.duration ?? defaultDuration)) : defaultDuration);
  const state = shallowRef(props.reveal ? "initial" : "disabled");
  return {
    duration,
    state
  };
}
const makeVProgressCircularProps = propsFactory({
  bgColor: String,
  color: String,
  indeterminate: [Boolean, String],
  rounded: Boolean,
  modelValue: {
    type: [Number, String],
    default: 0
  },
  rotate: {
    type: [Number, String],
    default: 0
  },
  transition: {
    type: [Boolean, Object],
    default: void 0
  },
  width: {
    type: [Number, String],
    default: 4
  },
  ...makeComponentProps(),
  ...makeRevealProps(),
  ...makeSizeProps(),
  ...makeTagProps({
    tag: "div"
  }),
  ...makeThemeProps()
}, "VProgressCircular");
const VProgressCircular = genericComponent()({
  name: "VProgressCircular",
  props: makeVProgressCircularProps(),
  setup(props, {
    slots
  }) {
    const MAGIC_RADIUS_CONSTANT = 20;
    const CIRCUMFERENCE = 2 * Math.PI * MAGIC_RADIUS_CONSTANT;
    const root = ref();
    const {
      themeClasses
    } = provideTheme(props);
    const {
      sizeClasses,
      sizeStyles
    } = useSize(props);
    const {
      textColorClasses,
      textColorStyles
    } = useTextColor(() => props.color);
    const {
      textColorClasses: underlayColorClasses,
      textColorStyles: underlayColorStyles
    } = useTextColor(() => props.bgColor);
    const {
      intersectionRef,
      isIntersecting
    } = useIntersectionObserver();
    const {
      resizeRef,
      contentRect
    } = useResizeObserver();
    const {
      state: revealState,
      duration: revealDuration
    } = useReveal(props);
    const normalizedValue = toRef(() => revealState.value === "initial" ? 0 : clamp(parseFloat(props.modelValue), 0, 100));
    const width = toRef(() => Number(props.width));
    const transitionDuration = toRef(() => props.transition === false ? void 0 : convertToUnit(isObject(props.transition) ? props.transition.duration : void 0, "ms"));
    const size = toRef(() => {
      return sizeStyles.value ? Number(props.size) : contentRect.value ? contentRect.value.width : Math.max(width.value, 32);
    });
    const diameter = toRef(() => MAGIC_RADIUS_CONSTANT / (1 - width.value / size.value) * 2);
    const strokeWidth = toRef(() => width.value / size.value * diameter.value);
    const strokeDashOffset = toRef(() => {
      const baseLength = (100 - normalizedValue.value) / 100 * CIRCUMFERENCE;
      return props.rounded && normalizedValue.value > 0 && normalizedValue.value < 100 ? convertToUnit(Math.min(CIRCUMFERENCE - 0.01, baseLength + strokeWidth.value)) : convertToUnit(baseLength);
    });
    const startAngle = computed(() => {
      const baseAngle = Number(props.rotate);
      return props.rounded ? baseAngle + strokeWidth.value / 2 / CIRCUMFERENCE * 360 : baseAngle;
    });
    watchEffect(() => {
      intersectionRef.value = root.value;
      resizeRef.value = root.value;
    });
    useRender(() => createVNode(props.tag, {
      "ref": root,
      "class": normalizeClass(["v-progress-circular", {
        "v-progress-circular--indeterminate": !!props.indeterminate,
        "v-progress-circular--visible": isIntersecting.value,
        "v-progress-circular--disable-shrink": props.indeterminate && (props.indeterminate === "disable-shrink" || PREFERS_REDUCED_MOTION()),
        "v-progress-circular--revealing": ["initial", "pending"].includes(revealState.value),
        "v-progress-circular--no-transition": props.transition === false
      }, themeClasses.value, sizeClasses.value, textColorClasses.value, props.class]),
      "style": normalizeStyle([sizeStyles.value, textColorStyles.value, {
        "--v-progress-reveal-duration": `${revealDuration.value}ms`,
        "--v-progress-circular-transition-duration": transitionDuration.value
      }, props.style]),
      "role": "progressbar",
      "aria-valuemin": "0",
      "aria-valuemax": "100",
      "aria-valuenow": props.indeterminate ? void 0 : normalizedValue.value
    }, {
      default: () => [createElementVNode("svg", {
        "style": {
          transform: `rotate(calc(-90deg + ${startAngle.value}deg))`
        },
        "xmlns": "http://www.w3.org/2000/svg",
        "viewBox": `0 0 ${diameter.value} ${diameter.value}`
      }, [createElementVNode("circle", {
        "class": normalizeClass(["v-progress-circular__underlay", underlayColorClasses.value]),
        "style": normalizeStyle(underlayColorStyles.value),
        "fill": "transparent",
        "cx": "50%",
        "cy": "50%",
        "r": MAGIC_RADIUS_CONSTANT,
        "stroke-width": strokeWidth.value,
        "stroke-dasharray": CIRCUMFERENCE,
        "stroke-dashoffset": 0
      }, null), createElementVNode("circle", {
        "class": "v-progress-circular__overlay",
        "fill": "transparent",
        "cx": "50%",
        "cy": "50%",
        "r": MAGIC_RADIUS_CONSTANT,
        "stroke-width": strokeWidth.value,
        "stroke-dasharray": CIRCUMFERENCE,
        "stroke-dashoffset": strokeDashOffset.value,
        "stroke-linecap": props.rounded ? "round" : void 0
      }, null)]), slots.default && createElementVNode("div", {
        "class": "v-progress-circular__content"
      }, [slots.default({
        value: normalizedValue.value
      })])]
    }));
    return {};
  }
});
const makeDimensionProps = propsFactory({
  height: [Number, String],
  maxHeight: [Number, String],
  maxWidth: [Number, String],
  minHeight: [Number, String],
  minWidth: [Number, String],
  width: [Number, String]
}, "dimension");
function useDimension(props) {
  const dimensionStyles = computed(() => {
    const styles = {};
    const height = convertToUnit(props.height);
    const maxHeight = convertToUnit(props.maxHeight);
    const maxWidth = convertToUnit(props.maxWidth);
    const minHeight = convertToUnit(props.minHeight);
    const minWidth = convertToUnit(props.minWidth);
    const width = convertToUnit(props.width);
    if (height) styles.height = height;
    if (maxHeight) styles.maxHeight = maxHeight;
    if (maxWidth) styles.maxWidth = maxWidth;
    if (minHeight) styles.minHeight = minHeight;
    if (minWidth) styles.minWidth = minWidth;
    if (width) styles.width = width;
    return styles;
  });
  return {
    dimensionStyles
  };
}
const oppositeMap = {
  center: "center",
  top: "bottom",
  bottom: "top",
  left: "right",
  right: "left"
};
const makeLocationProps = propsFactory({
  location: String
}, "location");
function useLocation(props, opposite = false, offset) {
  const {
    isRtl
  } = useRtl();
  const locationStyles = computed(() => {
    if (!props.location) return {};
    const {
      side,
      align
    } = parseAnchor(props.location.split(" ").length > 1 ? props.location : `${props.location} center`, isRtl.value);
    function getOffset(side2) {
      return offset ? offset(side2) : 0;
    }
    const styles = {};
    if (side !== "center") {
      if (opposite) styles[oppositeMap[side]] = `calc(100% - ${getOffset(side)}px)`;
      else styles[side] = 0;
    }
    if (align !== "center") {
      if (opposite) styles[oppositeMap[align]] = `calc(100% - ${getOffset(align)}px)`;
      else styles[align] = 0;
    } else {
      if (side === "center") styles.top = styles.left = "50%";
      else {
        styles[{
          top: "left",
          bottom: "left",
          left: "top",
          right: "top"
        }[side]] = "50%";
      }
      styles.transform = {
        top: "translateX(-50%)",
        bottom: "translateX(-50%)",
        left: "translateY(-50%)",
        right: "translateY(-50%)",
        center: "translate(-50%, -50%)"
      }[side];
    }
    return styles;
  });
  return {
    locationStyles
  };
}
const makeLoaderProps = propsFactory({
  loading: [Boolean, String]
}, "loader");
function useLoader(props, name = getCurrentInstanceName()) {
  const loaderClasses = toRef(() => ({
    [`${name}--loading`]: props.loading
  }));
  return {
    loaderClasses
  };
}
const positionValues = ["static", "relative", "fixed", "absolute", "sticky"];
const makePositionProps = propsFactory({
  position: {
    type: String,
    validator: (
      /* istanbul ignore next */
      (v) => positionValues.includes(v)
    )
  }
}, "position");
function usePosition(props, name = getCurrentInstanceName()) {
  const positionClasses = toRef(() => {
    return props.position ? `${name}--${props.position}` : void 0;
  });
  return {
    positionClasses
  };
}
function useRoute() {
  const vm = getCurrentInstance("useRoute");
  return computed(() => vm?.proxy?.$route);
}
function useLink(props, attrs) {
  const RouterLink = resolveDynamicComponent("RouterLink");
  const isLink = toRef(() => !!(props.href || props.to));
  const isClickable = computed(() => {
    return isLink?.value || hasEvent(attrs, "click") || hasEvent(props, "click");
  });
  if (isString(RouterLink) || !("useLink" in RouterLink)) {
    const href2 = toRef(() => props.href);
    return {
      isLink,
      isRouterLink: toRef(() => false),
      isClickable,
      href: href2,
      linkProps: reactive({
        href: href2
      }),
      route: toRef(() => void 0),
      navigate: toRef(() => void 0)
    };
  }
  const routerLink = RouterLink.useLink({
    to: toRef(() => props.to || ""),
    replace: toRef(() => props.replace)
  });
  const link = computed(() => props.to ? routerLink : void 0);
  const route = useRoute();
  const isActive = computed(() => {
    if (!link.value) return false;
    if (route.value && route.value.matched.length === 0 && route.value.name == null) {
      return link.value.isExactActive?.value ?? false;
    }
    if (!props.exact) return link.value.isActive?.value ?? false;
    if (!route.value) return link.value.isExactActive?.value ?? false;
    return link.value.isExactActive?.value && deepEqual(link.value.route.value.query, route.value.query);
  });
  const href = computed(() => props.to ? link.value?.route.value.href : props.href);
  const isRouterLink = toRef(() => !!props.to);
  return {
    isLink,
    isRouterLink,
    isClickable,
    isActive,
    route: toRef(() => link.value?.route.value),
    navigate: toRef(() => link.value?.navigate),
    href,
    linkProps: reactive({
      href,
      "aria-current": toRef(() => isActive.value ? "page" : void 0),
      "aria-disabled": toRef(() => props.disabled && isLink.value ? "true" : void 0),
      tabindex: toRef(() => props.disabled && isLink.value ? "-1" : void 0)
    })
  };
}
const makeRouterProps = propsFactory({
  href: String,
  replace: Boolean,
  to: [String, Object],
  exact: Boolean
}, "router");
function useSelectLink(link, select) {
  watch(() => link.isActive?.value, (isActive) => {
    if (link.isLink.value && isBoolean(isActive) && select) {
      nextTick(() => {
        select(isActive);
      });
    }
  }, {
    immediate: true
  });
}
const stopSymbol = /* @__PURE__ */ Symbol("rippleStop");
const DELAY_RIPPLE = 80;
function transform(el, value) {
  el.style.transform = value;
  el.style.webkitTransform = value;
}
function isTouchEvent(e) {
  return e.constructor.name === "TouchEvent";
}
function isKeyboardEvent(e) {
  return e.constructor.name === "KeyboardEvent";
}
const calculate = (e, el, value = {}) => {
  let localX = 0;
  let localY = 0;
  if (!isKeyboardEvent(e)) {
    const offset = new Box(el);
    const target = isTouchEvent(e) ? e.touches[e.touches.length - 1] : e;
    const point = getTargetBox([target.clientX, target.clientY]);
    localX = point.x - offset.left;
    localY = point.y - offset.top;
  }
  let radius = 0;
  let scale = 0.3;
  if (el._ripple?.circle) {
    scale = 0.15;
    radius = el.clientWidth / 2;
    radius = value.center ? radius : radius + Math.sqrt((localX - radius) ** 2 + (localY - radius) ** 2) / 4;
  } else {
    radius = Math.sqrt(el.clientWidth ** 2 + el.clientHeight ** 2) / 2;
  }
  const centerX = `${(el.clientWidth - radius * 2) / 2}px`;
  const centerY = `${(el.clientHeight - radius * 2) / 2}px`;
  const x = value.center ? centerX : `${localX - radius}px`;
  const y = value.center ? centerY : `${localY - radius}px`;
  return {
    radius,
    scale,
    x,
    y,
    centerX,
    centerY
  };
};
const ripples = {
  show(e, el, value = {}) {
    if (!el?._ripple?.enabled) {
      return;
    }
    const container = (void 0).createElement("span");
    const animation = (void 0).createElement("span");
    container.appendChild(animation);
    container.className = "v-ripple__container";
    if (value.class) {
      container.className += ` ${value.class}`;
    }
    const {
      radius,
      scale,
      x,
      y,
      centerX,
      centerY
    } = calculate(e, el, value);
    const size = `${radius * 2}px`;
    animation.className = "v-ripple__animation";
    animation.style.width = size;
    animation.style.height = size;
    el.appendChild(container);
    const computed2 = (void 0).getComputedStyle(el);
    if (computed2 && computed2.position === "static") {
      el.style.position = "relative";
      el.dataset.previousPosition = "static";
    }
    animation.classList.add("v-ripple__animation--enter");
    animation.classList.add("v-ripple__animation--visible");
    transform(animation, `translate(${x}, ${y}) scale3d(${scale},${scale},${scale})`);
    animation.dataset.activated = String(performance.now());
    requestAnimationFrame(() => {
      requestAnimationFrame(() => {
        animation.classList.remove("v-ripple__animation--enter");
        animation.classList.add("v-ripple__animation--in");
        transform(animation, `translate(${centerX}, ${centerY}) scale3d(1,1,1)`);
      });
    });
  },
  hide(el) {
    if (!el?._ripple?.enabled) return;
    const ripples2 = el.getElementsByClassName("v-ripple__animation");
    if (ripples2.length === 0) return;
    const animation = Array.from(ripples2).findLast((ripple) => !ripple.dataset.isHiding);
    if (!animation) return;
    else animation.dataset.isHiding = "true";
    const diff = performance.now() - Number(animation.dataset.activated);
    const delay = Math.max(250 - diff, 0);
    setTimeout(() => {
      animation.classList.remove("v-ripple__animation--in");
      animation.classList.add("v-ripple__animation--out");
      setTimeout(() => {
        const ripples3 = el.getElementsByClassName("v-ripple__animation");
        if (ripples3.length === 1 && el.dataset.previousPosition) {
          el.style.position = el.dataset.previousPosition;
          delete el.dataset.previousPosition;
        }
        if (animation.parentNode?.parentNode === el) el.removeChild(animation.parentNode);
      }, 300);
    }, delay);
  }
};
function isRippleEnabled(value) {
  return typeof value === "undefined" || !!value;
}
function rippleShow(e) {
  const value = {};
  const element = e.currentTarget;
  if (!element?._ripple || element._ripple.touched || e[stopSymbol]) return;
  e[stopSymbol] = true;
  if (isTouchEvent(e)) {
    element._ripple.touched = true;
    element._ripple.isTouch = true;
  } else {
    if (element._ripple.isTouch) return;
  }
  value.center = element._ripple.centered || isKeyboardEvent(e);
  if (element._ripple.class) {
    value.class = element._ripple.class;
  }
  if (isTouchEvent(e)) {
    if (element._ripple.showTimerCommit) return;
    element._ripple.showTimerCommit = () => {
      ripples.show(e, element, value);
    };
    element._ripple.showTimer = (void 0).setTimeout(() => {
      if (element?._ripple?.showTimerCommit) {
        element._ripple.showTimerCommit();
        element._ripple.showTimerCommit = null;
      }
    }, DELAY_RIPPLE);
  } else {
    ripples.show(e, element, value);
  }
}
function rippleStop(e) {
  e[stopSymbol] = true;
}
function rippleHide(e) {
  const element = e.currentTarget;
  if (!element?._ripple) return;
  (void 0).clearTimeout(element._ripple.showTimer);
  if (e.type === "touchend" && element._ripple.showTimerCommit) {
    element._ripple.showTimerCommit();
    element._ripple.showTimerCommit = null;
    element._ripple.showTimer = (void 0).setTimeout(() => {
      rippleHide(e);
    });
    return;
  }
  (void 0).setTimeout(() => {
    if (element._ripple) {
      element._ripple.touched = false;
    }
  });
  ripples.hide(element);
}
function rippleCancelShow(e) {
  const element = e.currentTarget;
  if (!element?._ripple) return;
  if (element._ripple.showTimerCommit) {
    element._ripple.showTimerCommit = null;
  }
  (void 0).clearTimeout(element._ripple.showTimer);
}
let keyboardRipple = false;
function keyboardRippleShow(e, keys) {
  if (!keyboardRipple && keys.includes(e.key)) {
    keyboardRipple = true;
    rippleShow(e);
  }
}
function keyboardRippleHide(e) {
  keyboardRipple = false;
  rippleHide(e);
}
function focusRippleHide(e) {
  if (keyboardRipple) {
    keyboardRipple = false;
    rippleHide(e);
  }
}
function updateRipple(el, binding, wasEnabled) {
  const {
    value,
    modifiers
  } = binding;
  const enabled = isRippleEnabled(value);
  if (!enabled) {
    ripples.hide(el);
  }
  el._ripple = el._ripple ?? {};
  el._ripple.enabled = enabled;
  el._ripple.centered = modifiers.center;
  el._ripple.circle = modifiers.circle;
  const bindingValue = isObject(value) ? value : {};
  if (bindingValue.class) {
    el._ripple.class = bindingValue.class;
  }
  const allowedKeys = bindingValue.keys ?? ["Enter", "Space"];
  el._ripple.keyDownHandler = (e) => keyboardRippleShow(e, allowedKeys);
  if (enabled && !wasEnabled) {
    if (modifiers.stop) {
      el.addEventListener("touchstart", rippleStop, {
        passive: true
      });
      el.addEventListener("mousedown", rippleStop);
      return;
    }
    el.addEventListener("touchstart", rippleShow, {
      passive: true
    });
    el.addEventListener("touchend", rippleHide, {
      passive: true
    });
    el.addEventListener("touchmove", rippleCancelShow, {
      passive: true
    });
    el.addEventListener("touchcancel", rippleHide);
    el.addEventListener("mousedown", rippleShow);
    el.addEventListener("mouseup", rippleHide);
    el.addEventListener("mouseleave", rippleHide);
    el.addEventListener("keydown", el._ripple.keyDownHandler);
    el.addEventListener("keyup", keyboardRippleHide);
    el.addEventListener("blur", focusRippleHide);
    el.addEventListener("dragstart", rippleHide, {
      passive: true
    });
  } else if (!enabled && wasEnabled) {
    removeListeners(el);
  }
}
function removeListeners(el) {
  el.removeEventListener("touchstart", rippleStop);
  el.removeEventListener("mousedown", rippleStop);
  el.removeEventListener("touchstart", rippleShow);
  el.removeEventListener("touchend", rippleHide);
  el.removeEventListener("touchmove", rippleCancelShow);
  el.removeEventListener("touchcancel", rippleHide);
  el.removeEventListener("mousedown", rippleShow);
  el.removeEventListener("mouseup", rippleHide);
  el.removeEventListener("mouseleave", rippleHide);
  if (el._ripple?.keyDownHandler) {
    el.removeEventListener("keydown", el._ripple.keyDownHandler);
  }
  el.removeEventListener("keyup", keyboardRippleHide);
  el.removeEventListener("blur", focusRippleHide);
  el.removeEventListener("dragstart", rippleHide);
}
function mounted$1(el, binding) {
  updateRipple(el, binding, false);
}
function unmounted$1(el) {
  removeListeners(el);
  delete el._ripple;
}
function updated(el, binding) {
  if (binding.value === binding.oldValue) {
    return;
  }
  const wasEnabled = isRippleEnabled(binding.oldValue);
  updateRipple(el, binding, wasEnabled);
}
const Ripple = {
  mounted: mounted$1,
  unmounted: unmounted$1,
  updated
};
const makeVBtnProps = propsFactory({
  active: {
    type: Boolean,
    default: void 0
  },
  activeColor: String,
  baseColor: String,
  symbol: {
    type: null,
    default: VBtnToggleSymbol
  },
  flat: Boolean,
  icon: [Boolean, String, Function, Object],
  prependIcon: IconValue,
  appendIcon: IconValue,
  block: Boolean,
  readonly: Boolean,
  slim: Boolean,
  stacked: Boolean,
  spaced: String,
  ripple: {
    type: [Boolean, Object],
    default: true
  },
  text: {
    type: [String, Number, Boolean],
    default: void 0
  },
  ...makeBorderProps(),
  ...makeComponentProps(),
  ...makeDensityProps(),
  ...makeDimensionProps(),
  ...makeElevationProps(),
  ...makeGroupItemProps(),
  ...makeLoaderProps(),
  ...makeLocationProps(),
  ...makePositionProps(),
  ...makeRoundedProps(),
  ...makeRouterProps(),
  ...makeSizeProps(),
  ...makeTagProps({
    tag: "button"
  }),
  ...makeThemeProps(),
  ...makeVariantProps({
    variant: "elevated"
  })
}, "VBtn");
const VBtn = genericComponent()({
  name: "VBtn",
  props: makeVBtnProps(),
  emits: {
    "group:selected": (val) => true
  },
  setup(props, {
    attrs,
    slots
  }) {
    const {
      themeClasses
    } = provideTheme(props);
    const {
      borderClasses
    } = useBorder(props);
    const {
      densityClasses
    } = useDensity(props);
    const {
      dimensionStyles
    } = useDimension(props);
    const {
      elevationClasses
    } = useElevation(props);
    const {
      loaderClasses
    } = useLoader(props);
    const {
      locationStyles
    } = useLocation(props);
    const {
      positionClasses
    } = usePosition(props);
    const {
      roundedClasses,
      roundedStyles
    } = useRounded(props);
    const {
      sizeClasses,
      sizeStyles
    } = useSize(props);
    const group = useGroupItem(props, props.symbol, false);
    const link = useLink(props, attrs);
    const isActive = computed(() => {
      if (props.active !== void 0) {
        return props.active;
      }
      if (link.isRouterLink.value) {
        return link.isActive?.value;
      }
      return group?.isSelected.value;
    });
    const color = toRef(() => isActive.value ? props.activeColor ?? props.color : props.color);
    const variantProps = computed(() => {
      const showColor = group?.isSelected.value && (!link.isLink.value || link.isActive?.value) || !group || link.isActive?.value;
      return {
        color: showColor ? color.value ?? props.baseColor : props.baseColor,
        variant: props.variant
      };
    });
    const {
      colorClasses,
      colorStyles,
      variantClasses
    } = useVariant(variantProps);
    const isDisabled = computed(() => group?.disabled.value || props.disabled);
    const isElevated = toRef(() => {
      return props.variant === "elevated" && !(props.disabled || props.flat || props.border);
    });
    const valueAttr = computed(() => {
      if (props.value === void 0 || typeof props.value === "symbol") return void 0;
      return Object(props.value) === props.value ? JSON.stringify(props.value, null, 0) : props.value;
    });
    function onClick(e) {
      if (isDisabled.value || link.isLink.value && (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0 || attrs.target === "_blank")) return;
      if (link.isRouterLink.value) {
        link.navigate.value?.(e);
      } else {
        group?.toggle();
      }
    }
    useSelectLink(link, group?.select);
    useRender(() => {
      const Tag = link.isLink.value ? "a" : props.tag;
      const hasPrepend = !!(props.prependIcon || slots.prepend);
      const hasAppend = !!(props.appendIcon || slots.append);
      const hasIcon = !!(props.icon && props.icon !== true);
      return withDirectives(createVNode(Tag, mergeProps(link.linkProps, {
        "type": Tag === "a" ? void 0 : "button",
        "class": ["v-btn", group?.selectedClass.value, {
          "v-btn--active": isActive.value,
          "v-btn--block": props.block,
          "v-btn--disabled": isDisabled.value,
          "v-btn--elevated": isElevated.value,
          "v-btn--flat": props.flat,
          "v-btn--icon": !!props.icon,
          "v-btn--loading": props.loading,
          "v-btn--readonly": props.readonly,
          "v-btn--slim": props.slim,
          "v-btn--stacked": props.stacked
        }, props.spaced ? ["v-btn--spaced", `v-btn--spaced-${props.spaced}`] : [], themeClasses.value, borderClasses.value, colorClasses.value, densityClasses.value, elevationClasses.value, loaderClasses.value, positionClasses.value, roundedClasses.value, sizeClasses.value, variantClasses.value, props.class],
        "style": [colorStyles.value, dimensionStyles.value, locationStyles.value, sizeStyles.value, roundedStyles.value, props.style],
        "aria-busy": props.loading ? true : void 0,
        "disabled": isDisabled.value && Tag !== "a" || void 0,
        "tabindex": props.loading || props.readonly ? -1 : void 0,
        "onClick": onClick,
        "value": Tag !== "a" ? valueAttr.value : void 0
      }), {
        default: () => [genOverlays(true, "v-btn"), !props.icon && hasPrepend && createElementVNode("span", {
          "key": "prepend",
          "class": "v-btn__prepend"
        }, [!slots.prepend ? createVNode(VIcon, {
          "key": "prepend-icon",
          "icon": props.prependIcon
        }, null) : createVNode(VDefaultsProvider, {
          "key": "prepend-defaults",
          "disabled": !props.prependIcon,
          "defaults": {
            VIcon: {
              icon: props.prependIcon
            }
          }
        }, slots.prepend)]), createElementVNode("span", {
          "class": "v-btn__content",
          "data-no-activator": ""
        }, [!slots.default && hasIcon ? createVNode(VIcon, {
          "key": "content-icon",
          "icon": props.icon
        }, null) : createVNode(VDefaultsProvider, {
          "key": "content-defaults",
          "disabled": !hasIcon,
          "defaults": {
            VIcon: {
              icon: props.icon
            }
          }
        }, {
          default: () => [slots.default?.() ?? toDisplayString(props.text)]
        })]), !props.icon && hasAppend && createElementVNode("span", {
          "key": "append",
          "class": "v-btn__append"
        }, [!slots.append ? createVNode(VIcon, {
          "key": "append-icon",
          "icon": props.appendIcon
        }, null) : createVNode(VDefaultsProvider, {
          "key": "append-defaults",
          "disabled": !props.appendIcon,
          "defaults": {
            VIcon: {
              icon: props.appendIcon
            }
          }
        }, slots.append)]), !!props.loading && createElementVNode("span", {
          "key": "loader",
          "class": "v-btn__loader"
        }, [slots.loader?.() ?? createVNode(VProgressCircular, {
          "color": isBoolean(props.loading) ? void 0 : props.loading,
          "indeterminate": true,
          "width": "2"
        }, null)])]
      }), [[Ripple, !isDisabled.value && props.ripple, "", {
        center: !!props.icon
      }]]);
    });
    return {
      group
    };
  }
});
const VSpacer = createSimpleFunctional("v-spacer", "div", "VSpacer");
const makeVDatePickerControlsProps = propsFactory({
  active: {
    type: [String, Array],
    default: void 0
  },
  controlHeight: [Number, String],
  controlVariant: {
    type: String,
    default: "docked"
  },
  noMonthPicker: Boolean,
  disabled: {
    type: [Boolean, String, Array],
    default: null
  },
  nextIcon: {
    type: IconValue,
    default: "$next"
  },
  prevIcon: {
    type: IconValue,
    default: "$prev"
  },
  modeIcon: {
    type: IconValue,
    default: "$subgroup"
  },
  text: String,
  monthText: String,
  yearText: String,
  viewMode: {
    type: String,
    default: "month"
  }
}, "VDatePickerControls");
const VDatePickerControls = genericComponent()({
  name: "VDatePickerControls",
  props: makeVDatePickerControlsProps(),
  emits: {
    "click:year": () => true,
    "click:month": () => true,
    "click:prev": () => true,
    "click:next": () => true,
    "click:prev-year": () => true,
    "click:next-year": () => true
  },
  setup(props, {
    emit,
    slots
  }) {
    const {
      t
    } = useLocale();
    const disableMonth = computed(() => {
      return Array.isArray(props.disabled) ? props.disabled.includes("text") : !!props.disabled;
    });
    const disableYear = computed(() => {
      return Array.isArray(props.disabled) ? props.disabled.includes("mode") : !!props.disabled;
    });
    const disablePrevMonth = computed(() => {
      return Array.isArray(props.disabled) ? props.disabled.includes("prev-month") : !!props.disabled;
    });
    const disableNextMonth = computed(() => {
      return Array.isArray(props.disabled) ? props.disabled.includes("next-month") : !!props.disabled;
    });
    const disablePrevYear = computed(() => {
      return Array.isArray(props.disabled) ? props.disabled.includes("prev-year") : !!props.disabled;
    });
    const disableNextYear = computed(() => {
      return Array.isArray(props.disabled) ? props.disabled.includes("next-year") : !!props.disabled;
    });
    function onClickPrevMonth() {
      emit("click:prev");
    }
    function onClickNextMonth() {
      emit("click:next");
    }
    function onClickPrevYear() {
      emit("click:prev-year");
    }
    function onClickNextYear() {
      emit("click:next-year");
    }
    function onClickYear() {
      emit("click:year");
    }
    function onClickMonth() {
      emit("click:month");
    }
    useRender(() => {
      const innerDefaults = {
        VBtn: {
          density: "comfortable",
          variant: "text"
        }
      };
      const prevMonth = createVNode(VBtn, {
        "data-testid": "prev-month",
        "disabled": disablePrevMonth.value,
        "icon": props.prevIcon,
        "aria-label": t("$vuetify.datePicker.ariaLabel.previousMonth"),
        "onClick": onClickPrevMonth
      }, null);
      const nextMonth = createVNode(VBtn, {
        "data-testid": "next-month",
        "disabled": disableNextMonth.value,
        "icon": props.nextIcon,
        "aria-label": t("$vuetify.datePicker.ariaLabel.nextMonth"),
        "onClick": onClickNextMonth
      }, null);
      const prevYear = createVNode(VBtn, {
        "data-testid": "prev-year",
        "disabled": disablePrevYear.value,
        "icon": props.prevIcon,
        "aria-label": t("$vuetify.datePicker.ariaLabel.previousYear"),
        "onClick": onClickPrevYear
      }, null);
      const nextYear = createVNode(VBtn, {
        "data-testid": "next-year",
        "disabled": disableNextYear.value,
        "icon": props.nextIcon,
        "aria-label": t("$vuetify.datePicker.ariaLabel.nextYear"),
        "onClick": onClickNextYear
      }, null);
      const onlyMonthBtn = createVNode(VBtn, {
        "class": "v-date-picker-controls__only-month-btn",
        "data-testid": "month-btn",
        "density": "default",
        "disabled": disableMonth.value,
        "text": props.monthText,
        "appendIcon": props.modeIcon,
        "rounded": true,
        "aria-label": t("$vuetify.datePicker.ariaLabel.selectMonth"),
        "onClick": onClickMonth
      }, null);
      const onlyYearBtn = createVNode(VBtn, {
        "class": "v-date-picker-controls__only-year-btn",
        "data-testid": "year-btn",
        "density": "default",
        "disabled": disableYear.value,
        "text": props.yearText,
        "appendIcon": props.modeIcon,
        "rounded": true,
        "aria-label": t("$vuetify.datePicker.ariaLabel.selectYear"),
        "onClick": onClickYear
      }, null);
      const monthYearBtn = createVNode(VBtn, {
        "class": "v-date-picker-controls__year-btn",
        "data-testid": "year-btn",
        "density": "default",
        "disabled": disableYear.value,
        "text": props.text,
        "appendIcon": props.modeIcon,
        "rounded": true,
        "aria-label": t("$vuetify.datePicker.ariaLabel.selectYear"),
        "onClick": onClickYear
      }, null);
      const monthYearSplit = createElementVNode(Fragment, null, [createVNode(VBtn, {
        "class": "v-date-picker-controls__month-btn",
        "data-testid": "month-btn",
        "height": "36",
        "disabled": disableMonth.value,
        "text": props.text,
        "rounded": true,
        "aria-label": t("$vuetify.datePicker.ariaLabel.selectMonth"),
        "onClick": onClickMonth
      }, null), createVNode(VBtn, {
        "class": "v-date-picker-controls__mode-btn",
        "data-testid": "year-btn",
        "disabled": disableYear.value,
        "icon": props.modeIcon,
        "aria-label": t("$vuetify.datePicker.ariaLabel.selectYear"),
        "onClick": onClickYear
      }, null)]);
      const slotProps = {
        viewMode: props.viewMode,
        disabled: Array.isArray(props.disabled) ? props.disabled : [],
        monthYearText: props.text ?? "",
        monthText: props.monthText ?? "",
        yearText: props.yearText ?? "",
        openMonths: onClickMonth,
        openYears: onClickYear,
        prevMonth: onClickPrevMonth,
        nextMonth: onClickNextMonth,
        prevYear: onClickPrevYear,
        nextYear: onClickNextYear
      };
      const modalControls = createElementVNode(Fragment, null, [props.noMonthPicker ? monthYearBtn : monthYearSplit, createVNode(VSpacer, null, null), createElementVNode("div", {
        "class": "v-date-picker-controls__month"
      }, [prevMonth, nextMonth])]);
      const dockedControls = createElementVNode(Fragment, null, [createElementVNode("div", {
        "class": "v-date-picker-controls__month"
      }, [prevMonth, onlyMonthBtn, nextMonth]), createVNode(VSpacer, null, null), createElementVNode("div", {
        "class": "v-date-picker-controls__year"
      }, [prevYear, onlyYearBtn, nextYear])]);
      return createVNode(VDefaultsProvider, {
        "defaults": innerDefaults
      }, {
        default: () => [createElementVNode("div", {
          "class": normalizeClass(["v-date-picker-controls", `v-date-picker-controls--variant-${props.controlVariant}`]),
          "style": {
            "--v-date-picker-controls-height": convertToUnit(props.controlHeight)
          }
        }, [slots.default?.(slotProps) ?? createElementVNode(Fragment, null, [props.controlVariant === "modal" && modalControls, props.controlVariant === "docked" && dockedControls])])]
      });
    });
    return {};
  }
});
const makeTransitionProps$1 = propsFactory({
  transition: {
    type: null,
    default: "fade-transition",
    validator: (val) => val !== true
  }
}, "transition");
const MaybeTransition = (props, {
  slots
}) => {
  const {
    transition,
    disabled,
    group,
    target,
    ...rest
  } = props;
  const {
    component = group ? TransitionGroup : Transition,
    ...customProps
  } = isObject(transition) ? transition : {};
  let transitionProps;
  if (isObject(transition)) {
    transitionProps = mergeProps(customProps, onlyDefinedProps({
      disabled,
      group,
      target
    }), rest);
  } else {
    transitionProps = mergeProps({
      name: disabled || !transition ? "" : transition
    }, rest);
  }
  return h(component, transitionProps, slots);
};
const makeVDatePickerHeaderProps = propsFactory({
  appendIcon: IconValue,
  color: String,
  header: String,
  transition: String,
  onClick: EventProp(),
  onClickOnce: EventProp()
}, "VDatePickerHeader");
const VDatePickerHeader = genericComponent()({
  name: "VDatePickerHeader",
  props: makeVDatePickerHeaderProps(),
  emits: {
    click: () => true,
    "click:append": () => true
  },
  setup(props, {
    emit,
    slots
  }) {
    const {
      backgroundColorClasses,
      backgroundColorStyles
    } = useBackgroundColor(() => props.color);
    function onClick() {
      emit("click");
    }
    function onClickAppend() {
      emit("click:append");
    }
    useRender(() => {
      const hasContent = !!(slots.default || props.header);
      const hasAppend = !!(slots.append || props.appendIcon);
      return createElementVNode("div", {
        "class": normalizeClass(["v-date-picker-header", {
          "v-date-picker-header--clickable": !!(props.onClick || props.onClickOnce)
        }, backgroundColorClasses.value]),
        "style": normalizeStyle(backgroundColorStyles.value),
        "onClick": (props.onClick || props.onClickOnce) && onClick
      }, [slots.prepend && createElementVNode("div", {
        "key": "prepend",
        "class": "v-date-picker-header__prepend"
      }, [slots.prepend()]), hasContent && createVNode(MaybeTransition, {
        "key": "content",
        "name": props.transition
      }, {
        default: () => [createElementVNode("div", {
          "key": props.header,
          "class": "v-date-picker-header__content"
        }, [slots.default?.() ?? props.header])]
      }), hasAppend && createElementVNode("div", {
        "class": "v-date-picker-header__append"
      }, [!slots.append ? createVNode(VBtn, {
        "key": "append-btn",
        "icon": props.appendIcon,
        "variant": "text",
        "onClick": onClickAppend
      }, null) : createVNode(VDefaultsProvider, {
        "key": "append-defaults",
        "disabled": !props.appendIcon,
        "defaults": {
          VBtn: {
            icon: props.appendIcon,
            variant: "text"
          }
        }
      }, {
        default: () => [slots.append?.()]
      })])]);
    });
    return {};
  }
});
const makeVThemeProviderProps = propsFactory({
  withBackground: Boolean,
  ...makeComponentProps(),
  ...makeThemeProps(),
  ...makeTagProps()
}, "VThemeProvider");
const VThemeProvider = genericComponent()({
  name: "VThemeProvider",
  props: makeVThemeProviderProps(),
  setup(props, {
    slots
  }) {
    const {
      themeClasses
    } = provideTheme(props);
    return () => {
      if (!props.withBackground) return slots.default?.();
      return createVNode(props.tag, {
        "class": normalizeClass(["v-theme-provider", themeClasses.value, props.class]),
        "style": normalizeStyle(props.style)
      }, {
        default: () => [slots.default?.()]
      });
    };
  }
});
const makeVBadgeProps = propsFactory({
  bordered: Boolean,
  color: String,
  content: [Number, String],
  dot: Boolean,
  dotSize: [Number, String],
  floating: Boolean,
  icon: IconValue,
  inline: Boolean,
  label: {
    type: String,
    default: "$vuetify.badge"
  },
  max: [Number, String],
  modelValue: {
    type: Boolean,
    default: true
  },
  offsetX: [Number, String],
  offsetY: [Number, String],
  textColor: String,
  ...makeComponentProps(),
  ...makeLocationProps({
    location: "top end"
  }),
  ...makeRoundedProps(),
  ...makeTagProps(),
  ...makeThemeProps(),
  ...makeTransitionProps$1({
    transition: "scale-rotate-transition"
  }),
  ...makeDimensionProps()
}, "VBadge");
const VBadge = genericComponent()({
  name: "VBadge",
  inheritAttrs: false,
  props: makeVBadgeProps(),
  setup(props, ctx) {
    const {
      backgroundColorClasses,
      backgroundColorStyles
    } = useBackgroundColor(() => props.color);
    const {
      roundedClasses,
      roundedStyles
    } = useRounded(props);
    const {
      t
    } = useLocale();
    const {
      textColorClasses,
      textColorStyles
    } = useTextColor(() => props.textColor);
    const theme = useTheme();
    const themeClasses = toRef(() => theme.isDisabled ? void 0 : `${theme.prefix}theme--${props.theme ?? theme.name.value}`);
    const {
      locationStyles
    } = useLocation(props, true, (side) => {
      const base = props.floating ? props.dot ? 2 : 4 : props.dot ? Number(props.dotSize ?? 8) : 12;
      return base + (["top", "bottom"].includes(side) ? Number(props.offsetY ?? 0) : ["left", "right"].includes(side) ? Number(props.offsetX ?? 0) : 0);
    });
    const {
      dimensionStyles
    } = useDimension(props);
    useRender(() => {
      const value = Number(props.content);
      const content = !props.max || isNaN(value) ? props.content : value <= Number(props.max) ? value : `${props.max}+`;
      const [badgeAttrs, attrs] = pickWithRest(ctx.attrs, ["aria-atomic", "aria-label", "aria-live", "role", "title"]);
      return createVNode(props.tag, mergeProps({
        "class": ["v-badge", {
          "v-badge--bordered": props.bordered,
          "v-badge--dot": props.dot,
          "v-badge--floating": props.floating,
          "v-badge--inline": props.inline
        }, props.class]
      }, attrs, {
        "style": props.style
      }), {
        default: () => [createElementVNode("div", {
          "class": "v-badge__wrapper"
        }, [ctx.slots.default?.(), createVNode(MaybeTransition, {
          "transition": props.transition
        }, {
          default: () => [withDirectives(createElementVNode("span", mergeProps({
            "class": ["v-badge__badge", themeClasses.value, backgroundColorClasses.value, roundedClasses.value, textColorClasses.value],
            "style": [backgroundColorStyles.value, textColorStyles.value, dimensionStyles.value, props.inline ? {} : locationStyles.value, props.dot && props.dotSize ? {
              width: convertToUnit(props.dotSize),
              height: convertToUnit(props.dotSize)
            } : {}, roundedStyles.value],
            "aria-atomic": "true",
            "aria-label": t(props.label, value),
            "aria-live": "polite",
            "role": "status"
          }, badgeAttrs), [createVNode(VThemeProvider, {
            "theme": props.theme
          }, {
            default: () => [props.dot ? void 0 : ctx.slots.badge ? ctx.slots.badge?.() : props.icon ? createVNode(VIcon, {
              "icon": props.icon
            }, null) : content]
          })]), [[vShow, props.modelValue]])]
        })])]
      });
    });
    return {};
  }
});
const makeCalendarProps = propsFactory({
  allowedDates: [Array, Function],
  disabled: {
    type: Boolean,
    default: null
  },
  displayValue: null,
  modelValue: Array,
  month: [Number, String],
  max: null,
  min: null,
  showAdjacentMonths: Boolean,
  year: [Number, String],
  weekdays: {
    type: Array,
    default: () => [0, 1, 2, 3, 4, 5, 6]
  },
  weeksInMonth: {
    type: String,
    default: "dynamic"
  },
  firstDayOfWeek: {
    type: [Number, String],
    default: void 0
  },
  firstDayOfYear: {
    type: [Number, String],
    default: void 0
  },
  weekdayFormat: String
}, "calendar");
function useCalendar(props) {
  const adapter = useDate();
  const model = useProxiedModel(props, "modelValue", [], (v) => wrapInArray(v).map((i) => adapter.date(i)));
  const displayValue = computed(() => {
    if (props.displayValue) return adapter.date(props.displayValue);
    if (model.value.length > 0) return adapter.date(model.value[0]);
    if (props.min) return adapter.date(props.min);
    if (Array.isArray(props.allowedDates)) return adapter.date(props.allowedDates[0]);
    return adapter.date();
  });
  const year = useProxiedModel(props, "year", void 0, (v) => {
    const value = !isNullOrUndefined(v) ? Number(v) : adapter.getYear(displayValue.value);
    return adapter.startOfYear(adapter.setYear(adapter.date(), value));
  }, (v) => adapter.getYear(v));
  const month = useProxiedModel(props, "month", void 0, (v) => {
    const value = !isNullOrUndefined(v) ? Number(v) : adapter.getMonth(displayValue.value);
    const date2 = adapter.setYear(adapter.startOfMonth(adapter.date()), adapter.getYear(year.value));
    return adapter.setMonth(date2, value);
  }, (v) => adapter.getMonth(v));
  const weekdayLabels = computed(() => {
    const firstDayOfWeek = adapter.toJsDate(adapter.startOfWeek(adapter.date(), props.firstDayOfWeek)).getDay();
    return adapter.getWeekdays(props.firstDayOfWeek, props.weekdayFormat).filter((_, i) => props.weekdays.includes((i + firstDayOfWeek) % 7));
  });
  const weeksInMonth = computed(() => {
    const weeks = adapter.getWeekArray(month.value, props.firstDayOfWeek);
    const days = weeks.flat();
    const daysInMonth2 = 6 * 7;
    if (props.weeksInMonth === "static" && days.length < daysInMonth2) {
      const lastDay = days[days.length - 1];
      let week = [];
      for (let day = 1; day <= daysInMonth2 - days.length; day++) {
        week.push(adapter.addDays(lastDay, day));
        if (day % 7 === 0) {
          weeks.push(week);
          week = [];
        }
      }
    }
    return weeks;
  });
  function genDays(days, today) {
    return days.filter((date2) => {
      return props.weekdays.includes(adapter.toJsDate(date2).getDay());
    }).map((date2, index) => {
      const isoDate = adapter.toISO(date2);
      const isAdjacent = !adapter.isSameMonth(date2, month.value);
      const isStart = adapter.isSameDay(date2, adapter.startOfMonth(month.value));
      const isEnd = adapter.isSameDay(date2, adapter.endOfMonth(month.value));
      const isSame = adapter.isSameDay(date2, month.value);
      const weekdaysCount = props.weekdays.length;
      return {
        date: date2,
        formatted: adapter.format(date2, "keyboardDate"),
        isAdjacent,
        isDisabled: isDisabled(date2),
        isEnd,
        isHidden: isAdjacent && !props.showAdjacentMonths,
        isSame,
        isSelected: model.value.some((value) => adapter.isSameDay(date2, value)),
        isStart,
        isToday: adapter.isSameDay(date2, today),
        isWeekEnd: index % weekdaysCount === weekdaysCount - 1,
        isWeekStart: index % weekdaysCount === 0,
        isoDate,
        localized: adapter.format(date2, "dayOfMonth"),
        month: adapter.getMonth(date2),
        year: adapter.getYear(date2)
      };
    });
  }
  const daysInWeek = computed(() => {
    const lastDay = adapter.startOfWeek(displayValue.value, props.firstDayOfWeek);
    const week = [];
    for (let day = 0; day <= 6; day++) {
      week.push(adapter.addDays(lastDay, day));
    }
    const today = adapter.date();
    return genDays(week, today);
  });
  const daysInMonth = computed(() => {
    const days = weeksInMonth.value.flat();
    const today = adapter.date();
    return genDays(days, today);
  });
  const weekNumbers = computed(() => {
    return weeksInMonth.value.map((week) => {
      return week.length ? adapter.getWeek(week[0], props.firstDayOfWeek, props.firstDayOfYear) : null;
    });
  });
  const {
    minDate,
    maxDate
  } = useCalendarRange(props);
  function isDisabled(value) {
    if (props.disabled) return true;
    const date2 = adapter.date(value);
    if (minDate.value && adapter.isBefore(adapter.endOfDay(date2), minDate.value)) return true;
    if (maxDate.value && adapter.isAfter(date2, maxDate.value)) return true;
    if (Array.isArray(props.allowedDates) && props.allowedDates.length > 0) {
      return !props.allowedDates.some((d) => adapter.isSameDay(adapter.date(d), date2));
    }
    if (isFunction(props.allowedDates)) {
      return !props.allowedDates(date2);
    }
    return false;
  }
  return {
    displayValue,
    daysInMonth,
    daysInWeek,
    genDays,
    model,
    weeksInMonth,
    weekdayLabels,
    weekNumbers
  };
}
function useCalendarRange(props) {
  const adapter = useDate();
  const minDate = computed(() => {
    if (!props.min) return null;
    const date2 = adapter.date(props.min);
    return adapter.isValid(date2) ? date2 : null;
  });
  const maxDate = computed(() => {
    if (!props.max) return null;
    const date2 = adapter.date(props.max);
    return adapter.isValid(date2) ? date2 : null;
  });
  function clampDate(date2) {
    if (minDate.value && adapter.isBefore(date2, minDate.value)) {
      return minDate.value;
    }
    if (maxDate.value && adapter.isAfter(date2, maxDate.value)) {
      return maxDate.value;
    }
    return date2;
  }
  function isInAllowedRange(date2) {
    return (!minDate.value || adapter.isAfter(date2, minDate.value)) && (!maxDate.value || adapter.isBefore(date2, maxDate.value));
  }
  return {
    minDate,
    maxDate,
    clampDate,
    isInAllowedRange
  };
}
function useVirtualFocus(items, options) {
  const {
    control,
    orientation = "vertical",
    circular = false,
    columns: _columns,
    onHighlight
  } = options;
  const highlightedId = shallowRef();
  let previousEl = null;
  function applyHighlight(id) {
    const item = items().find((i) => i.id === id);
    if (!item?.el) return false;
    const el = toValue(item.el);
    if (!el) return false;
    const controlEl = toValue(control);
    if (controlEl) {
      const itemId = el.getAttribute("id");
      if (itemId) controlEl.setAttribute("aria-activedescendant", itemId);
    }
    if (previousEl) delete previousEl.dataset.highlighted;
    el.dataset.highlighted = "";
    previousEl = el;
    el.scrollIntoView?.({
      block: "nearest"
    });
    onHighlight?.(id);
    return true;
  }
  function mod(n, i) {
    return (i % n + n) % n;
  }
  function indexOf(id) {
    return items().findIndex((item) => item.id === id);
  }
  function go(id) {
    highlightedId.value = id;
    applyHighlight(id);
  }
  function step(stride) {
    const all = items();
    if (!all.length) return;
    const current = isNullOrUndefined(highlightedId.value) ? -1 : indexOf(highlightedId.value);
    const dir = stride > 0 ? 1 : -1;
    const abs = Math.abs(stride);
    const maxHops = Math.ceil(all.length / abs);
    let index = current + stride;
    let hops = 0;
    if (circular) {
      index = mod(all.length, index);
    } else if (index < 0 || index >= all.length) {
      return;
    }
    while (hops < maxHops) {
      const item = all[index];
      if (item && !toValue(item.disabled)) {
        go(item.id);
        return;
      }
      hops++;
      index = circular ? mod(all.length, index + stride) : index + dir * abs;
      if (!circular && (index < 0 || index >= all.length)) return;
    }
  }
  function first() {
    const item = items().find((i) => !toValue(i.disabled));
    if (item) go(item.id);
  }
  function last() {
    const item = [...items()].reverse().find((i) => !toValue(i.disabled));
    if (item) go(item.id);
  }
  function rowFirst() {
    const cols = toValue(_columns) ?? 0;
    if (!cols) return first();
    const all = items();
    const cur = isNullOrUndefined(highlightedId.value) ? 0 : Math.max(0, indexOf(highlightedId.value));
    const start = cur - cur % cols;
    for (let i = start; i < Math.min(start + cols, all.length); i++) {
      if (!toValue(all[i]?.disabled)) {
        go(all[i].id);
        return;
      }
    }
  }
  function rowLast() {
    const cols = toValue(_columns) ?? 0;
    if (!cols) return last();
    const all = items();
    const cur = isNullOrUndefined(highlightedId.value) ? 0 : Math.max(0, indexOf(highlightedId.value));
    const start = cur - cur % cols;
    const end = Math.min(start + cols, all.length);
    for (let i = end - 1; i >= start; i--) {
      if (!toValue(all[i]?.disabled)) {
        go(all[i].id);
        return;
      }
    }
  }
  function highlight(id) {
    const prev = highlightedId.value;
    highlightedId.value = id;
    if (!applyHighlight(id)) highlightedId.value = prev;
  }
  function focusHighlighted() {
    if (isNullOrUndefined(highlightedId.value)) return;
    const item = items().find((i) => i.id === highlightedId.value);
    if (!item?.el) return;
    toValue(item.el)?.focus();
  }
  function clear() {
    highlightedId.value = void 0;
    toValue(control)?.removeAttribute("aria-activedescendant");
    if (previousEl) {
      delete previousEl.dataset.highlighted;
      previousEl = null;
    }
  }
  function isRtl(e) {
    const el = e.currentTarget;
    return el ? getComputedStyle(el).direction === "rtl" : false;
  }
  function onKeydown(e) {
    const cols = toValue(_columns);
    const rtl = isRtl(e);
    if (cols) {
      switch (e.key) {
        case "ArrowRight":
          e.preventDefault();
          step(rtl ? -1 : 1);
          break;
        case "ArrowLeft":
          e.preventDefault();
          step(rtl ? 1 : -1);
          break;
        case "ArrowDown":
          e.preventDefault();
          step(cols);
          break;
        case "ArrowUp":
          e.preventDefault();
          step(-cols);
          break;
        case "Home":
          e.preventDefault();
          e.ctrlKey ? first() : rowFirst();
          break;
        case "End":
          e.preventDefault();
          e.ctrlKey ? last() : rowLast();
          break;
      }
      return;
    }
    const prevKeys = [];
    const nextKeys = [];
    if (orientation === "vertical" || orientation === "both") {
      prevKeys.push("ArrowUp");
      nextKeys.push("ArrowDown");
    }
    if (orientation === "horizontal" || orientation === "both") {
      prevKeys.push(rtl ? "ArrowRight" : "ArrowLeft");
      nextKeys.push(rtl ? "ArrowLeft" : "ArrowRight");
    }
    if (prevKeys.includes(e.key)) {
      e.preventDefault();
      step(-1);
    } else if (nextKeys.includes(e.key)) {
      e.preventDefault();
      step(1);
    } else if (e.key === "Home") {
      e.preventDefault();
      first();
    } else if (e.key === "End") {
      e.preventDefault();
      last();
    }
  }
  onScopeDispose(clear);
  return {
    highlightedId,
    highlight,
    focusHighlighted,
    clear,
    next: () => step(1),
    prev: () => step(-1),
    first,
    last,
    onKeydown
  };
}
function keyToDirection(key) {
  switch (key) {
    case "ArrowUp":
      return "up";
    case "ArrowDown":
      return "down";
    case "ArrowLeft":
      return "left";
    case "ArrowRight":
      return "right";
    default:
      return null;
  }
}
function useGridSelection({
  items,
  columns,
  initialValue,
  itemAttribute,
  onSelect,
  onNavigation,
  onHighlight,
  onEscape
}) {
  const hasFocusIn = shallowRef(false);
  const containerEl = shallowRef();
  const virtualFocus = useVirtualFocus(() => items().map((item) => ({
    id: item.value,
    disabled: item.isDisabled,
    el: () => containerEl.value?.querySelector(`[${itemAttribute}="${item.value}"]`)
  })), {
    control: containerEl,
    columns
  });
  if (onHighlight) {
    watch(virtualFocus.highlightedId, (id) => onHighlight(id));
  }
  function onFocusin(e) {
    const grid = containerEl.value;
    if (!grid || grid.contains(e.relatedTarget)) return;
    hasFocusIn.value = true;
    const targetEl = e.target;
    const targetId = targetEl?.getAttribute?.(itemAttribute);
    if (!isNullOrUndefined(targetId) && items().some((item) => String(item.value) === targetId)) {
      virtualFocus.highlight(targetId);
      return;
    }
    const initial = initialValue(virtualFocus.highlightedId.value);
    if (!isNullOrUndefined(initial)) virtualFocus.highlight(initial);
    virtualFocus.focusHighlighted();
  }
  function onFocusout(e) {
    if (containerEl.value?.contains(e.relatedTarget)) return;
    hasFocusIn.value = false;
    virtualFocus.clear();
  }
  function onActivate() {
    const id = virtualFocus.highlightedId.value;
    if (isNullOrUndefined(id)) return;
    const item = items().find((x) => x.value === id);
    if (item && !item.isDisabled) {
      onSelect(id);
    }
  }
  function selectItem(value) {
    virtualFocus.highlight(value);
    virtualFocus.focusHighlighted();
    onSelect(value);
  }
  function focusItem(value) {
    virtualFocus.highlight(value);
    if (isNullOrUndefined(virtualFocus.highlightedId.value)) virtualFocus.first();
    virtualFocus.focusHighlighted();
  }
  function onContainerKeydown(e) {
    if (e.key === "Escape") {
      e.preventDefault();
      onEscape?.();
      return;
    }
    if ((e.key === "Enter" || e.key === " ") && !isNullOrUndefined(virtualFocus.highlightedId.value)) {
      e.preventDefault();
      onActivate();
      return;
    }
    const direction = keyToDirection(e.key);
    if (direction && onNavigation?.(direction, e, virtualFocus.highlightedId.value)) return;
    virtualFocus.onKeydown(e);
    virtualFocus.focusHighlighted();
  }
  const containerProps = computed(() => ({
    ref: (el) => containerEl.value = el ?? void 0,
    tabindex: hasFocusIn.value ? -1 : 0,
    onKeydown: onContainerKeydown,
    onFocusin,
    onFocusout
  }));
  return {
    containerProps,
    containerEl,
    selectItem,
    focusItem,
    clear: virtualFocus.clear
  };
}
function useRangePicker({
  multiple,
  model,
  compare,
  normalizeEnd = (v) => v,
  previewValue: externalPreview
}) {
  const rangeStart = computed(() => model.value.length >= 1 ? model.value[0] : void 0);
  const rangeEnd = computed(() => model.value.length >= 2 ? model.value[model.value.length - 1] : void 0);
  const previewValue = externalPreview ?? shallowRef();
  const previewRange = computed(() => {
    if (multiple.value !== "range") return null;
    if (!rangeStart.value || rangeEnd.value) return null;
    if (!previewValue.value) return null;
    const start = rangeStart.value;
    const preview = previewValue.value;
    if (compare(start, preview) === 0) return null;
    return compare(start, preview) < 0 ? [start, preview] : [preview, start];
  });
  function isSelected(value) {
    if (multiple.value === "range" && model.value.length === 2) {
      return isWithinRange2(value);
    }
    for (const v of model.value) {
      if (compare(v, value) === 0) return true;
    }
    return false;
  }
  function isWithinRange2(value) {
    if (multiple.value !== "range" || model.value.length < 2) return false;
    const [start, end] = [model.value[0], model.value[model.value.length - 1]];
    return compare(value, start) >= 0 && compare(value, end) <= 0;
  }
  function isRangeStart(value) {
    if (multiple.value !== "range" || model.value.length < 2) return false;
    return compare(value, model.value[0]) === 0;
  }
  function isRangeEnd(value) {
    if (multiple.value !== "range" || model.value.length < 2) return false;
    return compare(value, model.value[model.value.length - 1]) === 0;
  }
  function isRangeMiddle(value) {
    return isWithinRange2(value) && !isRangeStart(value) && !isRangeEnd(value);
  }
  function select(value) {
    if (multiple.value === "range") {
      onRangeSelect(value);
    } else if (multiple.value) {
      onMultipleSelect(value);
    } else {
      model.value = [value];
    }
  }
  function onRangeSelect(value) {
    if (model.value.length === 0) {
      model.value = [value];
    } else if (model.value.length === 1) {
      const start = model.value[0];
      if (compare(value, start) === 0) {
        model.value = [];
      } else if (compare(value, start) < 0) {
        model.value = [value, normalizeEnd(start)];
      } else {
        model.value = [start, normalizeEnd(value)];
      }
    } else {
      model.value = [value];
    }
  }
  function setPreview(value) {
    previewValue.value = value;
  }
  function clearPreview() {
    previewValue.value = void 0;
  }
  function isPreviewStart(value) {
    const range2 = previewRange.value;
    if (!range2) return false;
    return compare(value, range2[0]) === 0;
  }
  function isPreviewEnd(value) {
    const range2 = previewRange.value;
    if (!range2) return false;
    return compare(value, range2[1]) === 0;
  }
  function isPreviewMiddle(value) {
    const range2 = previewRange.value;
    if (!range2) return false;
    return compare(value, range2[0]) > 0 && compare(value, range2[1]) < 0;
  }
  function isInPreviewRange(value) {
    return isPreviewStart(value) || isPreviewEnd(value) || isPreviewMiddle(value);
  }
  function onMultipleSelect(value) {
    const index = model.value.findIndex((v) => compare(v, value) === 0);
    if (index === -1) {
      model.value = [...model.value, value];
    } else {
      const copy = [...model.value];
      copy.splice(index, 1);
      model.value = copy;
    }
  }
  return {
    isSelected,
    isRangeStart,
    isRangeEnd,
    isRangeMiddle,
    select,
    setPreview,
    clearPreview,
    isPreviewStart,
    isPreviewEnd,
    isPreviewMiddle,
    isInPreviewRange
  };
}
const makeVDatePickerMonthProps = propsFactory({
  color: String,
  hideWeekdays: Boolean,
  multiple: [Boolean, Number, String],
  showWeek: Boolean,
  readonly: Boolean,
  transition: {
    type: String,
    default: "picker-transition"
  },
  reverseTransition: {
    type: String,
    default: "picker-reverse-transition"
  },
  events: {
    type: [Array, Function, Object],
    default: () => null
  },
  eventColor: {
    type: [Array, Function, Object, String],
    default: () => null
  },
  noAutoNavigation: Boolean,
  previewValue: null,
  ...omit(makeCalendarProps(), ["displayValue"])
}, "VDatePickerMonth");
const VDatePickerMonth = genericComponent()({
  name: "VDatePickerMonth",
  props: makeVDatePickerMonthProps(),
  emits: {
    "update:modelValue": (date2) => true,
    "update:month": (date2) => true,
    "update:year": (date2) => true,
    "update:previewValue": (_value) => true,
    "boundary-navigate": (_payload) => true
  },
  setup(props, {
    emit,
    slots
  }) {
    const uid = useId();
    const {
      t
    } = useLocale();
    const {
      daysInMonth,
      model,
      weekNumbers,
      weekdayLabels
    } = useCalendar(props);
    const adapter = useDate();
    const isReverse = shallowRef(false);
    const transition = toRef(() => {
      return !isReverse.value ? props.transition : props.reverseTransition;
    });
    function compareDays(a, b) {
      if (adapter.isSameDay(a, b)) return 0;
      return adapter.isBefore(a, b) ? -1 : 1;
    }
    const previewValue = useProxiedModel(props, "previewValue");
    const range2 = useRangePicker({
      multiple: computed(() => {
        if (props.multiple === "range") return "range";
        return !!props.multiple;
      }),
      model,
      compare: compareDays,
      normalizeEnd: (value) => adapter.endOfDay(value),
      previewValue
    });
    const selectionColor = toRef(() => props.color || "surface-variant");
    const {
      backgroundColorClasses: rangeColorClasses,
      backgroundColorStyles: rangeColorStyles
    } = useBackgroundColor(selectionColor);
    const atMax = computed(() => {
      const max = ["number", "string"].includes(typeof props.multiple) ? Number(props.multiple) : Infinity;
      return model.value.length >= max;
    });
    const dayRows = computed(() => {
      return chunkArray(daysInMonth.value, props.weekdays.length);
    });
    function isSelectedDay(item) {
      return range2.isSelected(item.date);
    }
    function isDayDisabled(item) {
      return item.isDisabled || item.isHidden || atMax.value && !isSelectedDay(item);
    }
    function getDateAriaLabel(item) {
      const fullDate = adapter.format(item.date, "fullDateWithWeekday");
      const localeKey = item.isToday ? "currentDate" : "selectDate";
      return t(`$vuetify.datePicker.ariaLabel.${localeKey}`, fullDate);
    }
    function onClick(value) {
      range2.select(adapter.startOfDay(value));
    }
    function initialFocusDate(current) {
      const isVisible = (d) => !d.isAdjacent && !d.isDisabled;
      if (current != null) {
        const cur = daysInMonth.value.find((d) => d.isoDate === current);
        if (cur && !cur.isAdjacent) return current;
      }
      const selected = daysInMonth.value.find((d) => isVisible(d) && model.value.some((m) => adapter.isSameDay(m, d.date)));
      return (selected ?? daysInMonth.value.find(isVisible))?.isoDate;
    }
    const {
      containerProps,
      containerEl,
      selectItem,
      focusItem,
      clear
    } = useGridSelection({
      items: () => daysInMonth.value.map((d) => ({
        value: d.isoDate,
        isDisabled: isDayDisabled(d)
      })),
      columns: () => props.weekdays.length,
      initialValue: initialFocusDate,
      itemAttribute: "data-v-date",
      onSelect: onDaySelect,
      onNavigation: onNavigationBoundary,
      onEscape
    });
    function onDaySelect(isoDate) {
      const item = daysInMonth.value.find((d) => d.isoDate === isoDate);
      if (!item || isDayDisabled(item)) return;
      onClick(item.date);
      if (item.isAdjacent) {
        emit("update:month", adapter.getMonth(item.date));
        emit("update:year", adapter.getYear(item.date));
        nextTick(() => focusItem(isoDate));
      }
    }
    function onNavigationBoundary(direction, e, curId) {
      if (curId == null) return false;
      const cols = props.weekdays.length;
      const rtl = getComputedStyle(e.currentTarget).direction === "rtl";
      let stride;
      let calendarDays;
      if (direction === "left") {
        stride = rtl ? 1 : -1;
        calendarDays = stride;
      } else if (direction === "right") {
        stride = rtl ? -1 : 1;
        calendarDays = stride;
      } else if (direction === "up") {
        stride = -cols;
        calendarDays = -7;
      } else {
        stride = cols;
        calendarDays = 7;
      }
      const all = daysInMonth.value;
      const curIndex = all.findIndex((d) => d.isoDate === curId);
      if (curIndex < 0) return false;
      const targetItem = all[curIndex + stride];
      if (targetItem && !targetItem.isHidden) return false;
      e.preventDefault();
      let targetIsoDate;
      if (targetItem) {
        targetIsoDate = targetItem.isoDate;
      } else {
        const step = calendarDays < 0 ? -1 : 1;
        let candidate = adapter.addDays(adapter.date(curId), calendarDays);
        while (!props.weekdays.includes(adapter.toJsDate(candidate).getDay())) {
          candidate = adapter.addDays(candidate, step);
        }
        targetIsoDate = adapter.toISO(candidate);
      }
      if (props.noAutoNavigation) {
        emit("boundary-navigate", {
          direction,
          targetIsoDate
        });
        return true;
      }
      const targetDate = adapter.date(targetIsoDate);
      emit("update:month", adapter.getMonth(targetDate));
      emit("update:year", adapter.getYear(targetDate));
      nextTick(() => focusItem(targetIsoDate));
      return true;
    }
    function onEscape() {
      const rawTarget = model.value[0] ?? adapter.date();
      const targetIso = adapter.toISO(adapter.date(rawTarget));
      const inCurrentMonth = daysInMonth.value.find((d) => d.isoDate === targetIso && !d.isAdjacent);
      if (inCurrentMonth) {
        focusItem(targetIso);
        return;
      }
      const targetDate = adapter.date(rawTarget);
      emit("update:month", adapter.getMonth(targetDate));
      emit("update:year", adapter.getYear(targetDate));
      nextTick(() => focusItem(targetIso));
    }
    function onDayClick(item) {
      if (item.isAdjacent) {
        onDaySelect(item.isoDate);
      } else {
        selectItem(item.isoDate);
      }
    }
    function focusGrid() {
      containerEl.value?.focus();
    }
    watch(daysInMonth, (val, oldVal) => {
      if (!oldVal || val[0].isoDate === oldVal[0].isoDate) return;
      isReverse.value = adapter.isBefore(val[0].date, oldVal[0].date);
      clear();
    });
    function getEventColors(date2) {
      const {
        events,
        eventColor
      } = props;
      let eventData;
      let eventColors = [];
      if (Array.isArray(events)) {
        eventData = events.includes(date2);
      } else if (events instanceof Function) {
        eventData = events(date2) || false;
      } else if (events) {
        eventData = events[date2] || false;
      } else {
        eventData = false;
      }
      if (!eventData) {
        return [];
      } else if (eventData !== true) {
        eventColors = wrapInArray(eventData);
      } else if (isString(eventColor)) {
        eventColors = [eventColor];
      } else if (isFunction(eventColor)) {
        eventColors = wrapInArray(eventColor(date2));
      } else if (Array.isArray(eventColor)) {
        eventColors = eventColor;
      } else if (isObject(eventColor)) {
        eventColors = wrapInArray(eventColor[date2]);
      }
      return !eventColors.length ? ["surface-variant"] : eventColors.filter(Boolean).map((color) => isString(color) ? color : "surface-variant");
    }
    function genEvents(date2) {
      const eventColors = getEventColors(date2);
      if (!eventColors.length) return null;
      return createElementVNode("div", {
        "class": "v-date-picker-month__events"
      }, [eventColors.map((color) => createVNode(VBadge, {
        "dot": true,
        "color": color
      }, null))]);
    }
    useRender(() => createElementVNode("div", {
      "class": "v-date-picker-month",
      "style": {
        "--v-date-picker-days-in-week": props.weekdays.length
      }
    }, [props.showWeek && createElementVNode("div", {
      "key": "weeks",
      "class": "v-date-picker-month__weeks"
    }, [!props.hideWeekdays && createElementVNode("div", {
      "key": "hide-week-days",
      "class": "v-date-picker-month__day"
    }, [createTextVNode(" ")]), weekNumbers.value.map((week) => createElementVNode("div", {
      "class": normalizeClass(["v-date-picker-month__day", "v-date-picker-month__day--adjacent"])
    }, [week]))]), createVNode(MaybeTransition, {
      "name": transition.value
    }, {
      default: () => [createElementVNode("div", mergeProps({
        "key": daysInMonth.value[0].date?.toString(),
        "class": "v-date-picker-month__days",
        "role": "grid",
        "onMouseleave": range2.clearPreview
      }, containerProps.value), [!props.hideWeekdays && createElementVNode("div", {
        "key": "weekday-labels",
        "class": "v-date-picker-month__days-row"
      }, [weekdayLabels.value.map((weekDay) => createElementVNode("div", {
        "class": normalizeClass(["v-date-picker-month__day", "v-date-picker-month__weekday"])
      }, [weekDay]))]), dayRows.value.map((row, rowIndex) => createElementVNode("div", {
        "class": "v-date-picker-month__days-row",
        "role": "row"
      }, [row.map((item, colIndex) => {
        const i = rowIndex * props.weekdays.length + colIndex;
        const isSelected = isSelectedDay(item);
        const disabled = isDayDisabled(item);
        const rangeStart = range2.isRangeStart(item.date);
        const rangeEnd = range2.isRangeEnd(item.date);
        const rangeMiddle = range2.isRangeMiddle(item.date);
        const previewStart = range2.isPreviewStart(item.date);
        const previewEnd = range2.isPreviewEnd(item.date);
        const previewMiddle = range2.isPreviewMiddle(item.date);
        const slotProps = {
          props: {
            class: "v-date-picker-month__day-btn",
            color: isSelected && !rangeMiddle || item.isToday ? props.color : void 0,
            disabled,
            readonly: props.readonly,
            icon: true,
            ripple: false,
            tabindex: -1,
            variant: isSelected && !rangeMiddle ? "flat" : item.isToday ? "outlined" : "text",
            "aria-label": getDateAriaLabel(item),
            "aria-current": item.isToday ? "date" : void 0,
            id: `${uid}-day-${item.isoDate}`,
            "data-v-date": !disabled ? item.isoDate : void 0,
            onMousedown: (e) => e.preventDefault(),
            // preserve virtual focus
            onClick: () => onDayClick(item),
            onMouseenter: () => range2.setPreview(item.date),
            onFocus: () => range2.setPreview(item.date),
            onBlur: range2.clearPreview
          },
          item,
          i
        };
        const hasRangeBg = rangeStart || rangeEnd || rangeMiddle;
        const hasPreviewBg = previewStart || previewEnd || previewMiddle;
        return createElementVNode("div", {
          "class": normalizeClass(["v-date-picker-month__day", {
            "v-date-picker-month__day--adjacent": item.isAdjacent,
            "v-date-picker-month__day--hide-adjacent": item.isHidden,
            "v-date-picker-month__day--selected": isSelected,
            "v-date-picker-month__day--week-end": item.isWeekEnd,
            "v-date-picker-month__day--week-start": item.isWeekStart,
            "v-date-picker-month__day--range-start": rangeStart,
            "v-date-picker-month__day--range-end": rangeEnd,
            "v-date-picker-month__day--range-middle": rangeMiddle,
            "v-date-picker-month__day--preview-start": previewStart,
            "v-date-picker-month__day--preview-end": previewEnd,
            "v-date-picker-month__day--preview-middle": previewMiddle
          }]),
          "role": "gridcell"
        }, [(hasRangeBg || hasPreviewBg) && createElementVNode("div", {
          "key": "range-bg",
          "class": normalizeClass(["v-date-picker-month__range-bg", hasRangeBg ? "v-date-picker-month__range-bg--range" : "v-date-picker-month__range-bg--preview", rangeColorClasses.value]),
          "style": normalizeStyle(rangeColorStyles.value)
        }, null), (props.showAdjacentMonths || !item.isAdjacent) && (slots.day?.(slotProps) ?? createVNode(VBtn, slotProps.props, {
          default: () => [item.localized, genEvents(item.isoDate)]
        }))]);
      })]))])]
    })]));
    return {
      focusGrid,
      focusItem
    };
  }
});
const makeVDatePickerMonthsProps = propsFactory({
  color: String,
  columns: {
    type: Number,
    default: 2
  },
  height: [String, Number],
  min: null,
  max: null,
  modelValue: Number,
  year: Number,
  allowedMonths: [Array, Function]
}, "VDatePickerMonths");
const VDatePickerMonths = genericComponent()({
  name: "VDatePickerMonths",
  props: makeVDatePickerMonthsProps(),
  emits: {
    "update:modelValue": (date2) => true,
    escape: () => true
  },
  setup(props, {
    emit,
    slots
  }) {
    const adapter = useDate();
    const model = useProxiedModel(props, "modelValue");
    const uid = useId();
    const months = computed(() => {
      let date2 = adapter.startOfYear(adapter.date());
      if (props.year) {
        date2 = adapter.setYear(date2, props.year);
      }
      return range(12).map((i) => {
        const text = adapter.format(date2, "monthShort");
        const label = adapter.format(date2, "month");
        const isDisabled = !!(!isMonthAllowed(i) || props.min && adapter.isAfter(adapter.startOfMonth(adapter.date(props.min)), date2) || props.max && adapter.isAfter(date2, adapter.startOfMonth(adapter.date(props.max))));
        date2 = adapter.getNextMonth(date2);
        return {
          isDisabled,
          text,
          label,
          value: i
        };
      });
    });
    watchEffect(() => {
      model.value = model.value ?? adapter.getMonth(adapter.date());
    });
    function isMonthAllowed(month) {
      if (Array.isArray(props.allowedMonths) && props.allowedMonths.length) {
        return props.allowedMonths.includes(month);
      }
      if (isFunction(props.allowedMonths)) {
        return props.allowedMonths(month);
      }
      return true;
    }
    function onMonthSelect(value) {
      if (model.value === value) emit("update:modelValue", value);
      else model.value = value;
    }
    const {
      containerProps,
      selectItem
    } = useGridSelection({
      items: () => months.value,
      columns: () => props.columns,
      initialValue: (current) => current ?? model.value ?? adapter.getMonth(adapter.date()),
      itemAttribute: "data-v-month",
      onSelect: onMonthSelect,
      onEscape: () => emit("escape")
    });
    useRender(() => createElementVNode("div", {
      "class": "v-date-picker-months",
      "style": {
        height: convertToUnit(props.height)
      }
    }, [createElementVNode("div", mergeProps({
      "class": "v-date-picker-months__content",
      "style": {
        "--v-date-picker-months-columns": props.columns
      }
    }, containerProps.value), [months.value.map((month, i) => {
      const btnProps = {
        id: `${uid}-month-${i}`,
        active: model.value === i,
        ariaLabel: month.label,
        color: model.value === i ? props.color : void 0,
        disabled: month.isDisabled,
        rounded: true,
        tabindex: -1,
        text: month.text,
        variant: model.value === month.value ? "flat" : "text",
        "data-v-month": month.value,
        onMousedown: (e) => e.preventDefault(),
        // preserve virtual focus
        onClick: () => selectItem(i)
      };
      return slots.month?.({
        month,
        i,
        props: btnProps
      }) ?? createVNode(VBtn, mergeProps({
        "key": "month"
      }, btnProps), null);
    })])]));
    return {};
  }
});
function mounted(el, binding) {
  return;
}
function unmounted(el, binding) {
  const observe = el._observe?.[binding.instance.$.uid];
  if (!observe) return;
  observe.observer.unobserve(el);
  delete el._observe[binding.instance.$.uid];
}
const Intersect = {
  mounted,
  unmounted,
  updated: (el, binding) => {
    if (el._observe?.[binding.instance.$.uid]) {
      unmounted(el, binding);
    }
  }
};
const makeVDatePickerYearsProps = propsFactory({
  color: String,
  columns: {
    type: Number,
    default: 3
  },
  height: [String, Number],
  min: null,
  max: null,
  modelValue: Number,
  allowedYears: [Array, Function]
}, "VDatePickerYears");
const VDatePickerYears = genericComponent()({
  name: "VDatePickerYears",
  props: makeVDatePickerYearsProps(),
  directives: {
    vIntersect: Intersect
  },
  emits: {
    "update:modelValue": (year) => true,
    escape: () => true
  },
  setup(props, {
    emit,
    slots
  }) {
    const adapter = useDate();
    const model = useProxiedModel(props, "modelValue");
    const containerRef = templateRef();
    const uid = useId();
    const years = computed(() => {
      const year = adapter.getYear(adapter.date());
      let min = year - 100;
      let max = year + 52;
      if (props.min) {
        min = adapter.getYear(adapter.date(props.min));
      }
      if (props.max) {
        max = adapter.getYear(adapter.date(props.max));
      }
      let date2 = adapter.startOfYear(adapter.date());
      date2 = adapter.setYear(date2, min);
      return range(max - min + 1, min).map((i) => {
        const text = adapter.format(date2, "year");
        date2 = adapter.setYear(date2, adapter.getYear(date2) + 1);
        return {
          text,
          value: i,
          isDisabled: !isYearAllowed(i)
        };
      });
    });
    watchEffect(() => {
      model.value = model.value ?? adapter.getYear(adapter.date());
    });
    function isYearAllowed(year) {
      if (Array.isArray(props.allowedYears) && props.allowedYears.length) {
        return props.allowedYears.includes(year);
      }
      if (isFunction(props.allowedYears)) {
        return props.allowedYears(year);
      }
      return true;
    }
    function onYearSelect(value) {
      if (model.value === value) emit("update:modelValue", value);
      else model.value = value;
    }
    const {
      containerProps,
      containerEl,
      selectItem
    } = useGridSelection({
      items: () => years.value,
      columns: () => props.columns,
      initialValue: (current) => current ?? model.value ?? adapter.getYear(adapter.date()),
      itemAttribute: "data-v-year",
      onSelect: onYearSelect,
      onEscape: () => emit("escape")
    });
    function scrollToSelected() {
      const container = containerRef.el;
      const target = containerEl.value?.querySelector(`[data-v-year="${model.value}"]`);
      if (!container || !target) return;
      const containerRect = container.getBoundingClientRect();
      const targetRect = target.getBoundingClientRect();
      container.scrollTop += targetRect.top - containerRect.top - container.clientHeight / 2 + targetRect.height / 2;
    }
    useRender(() => withDirectives(createElementVNode("div", {
      "class": "v-date-picker-years",
      "ref": containerRef,
      "tabindex": -1,
      "style": {
        height: convertToUnit(props.height)
      }
    }, [createElementVNode("div", mergeProps({
      "class": "v-date-picker-years__content",
      "style": {
        "--v-date-picker-years-columns": props.columns
      }
    }, containerProps.value), [years.value.map((year, i) => {
      const btnProps = {
        id: `${uid}-year-${year.value}`,
        active: model.value === year.value,
        color: model.value === year.value ? props.color : void 0,
        rounded: true,
        tabindex: -1,
        text: year.text,
        disabled: year.isDisabled,
        variant: model.value === year.value ? "flat" : "text",
        "data-v-year": year.value,
        onMousedown: (e) => e.preventDefault(),
        // preserve virtual focus
        onClick: () => selectItem(year.value)
      };
      return slots.year?.({
        year,
        i,
        props: btnProps
      }) ?? createVNode(VBtn, mergeProps({
        "key": "month"
      }, btnProps), null);
    })])]), [[Intersect, {
      handler: scrollToSelected
    }, null, {
      once: true
    }]]));
    return {};
  }
});
const makeTransitionProps = propsFactory({
  disabled: Boolean,
  group: Boolean,
  hideOnLeave: Boolean,
  leaveAbsolute: Boolean,
  mode: String,
  origin: String
}, "transition");
function createCssTransition(name, origin, mode) {
  return genericComponent()({
    name,
    props: makeTransitionProps({
      mode,
      origin
    }),
    setup(props, {
      slots
    }) {
      const functions = {
        onBeforeEnter(el) {
          if (props.origin) {
            el.style.transformOrigin = props.origin;
          }
        },
        onLeave(el) {
          if (props.leaveAbsolute) {
            const {
              offsetTop,
              offsetLeft,
              offsetWidth,
              offsetHeight
            } = el;
            el._transitionInitialStyles = {
              position: el.style.position,
              top: el.style.top,
              left: el.style.left,
              width: el.style.width,
              height: el.style.height
            };
            el.style.position = "absolute";
            el.style.top = `${offsetTop}px`;
            el.style.left = `${offsetLeft}px`;
            el.style.width = `${offsetWidth}px`;
            el.style.height = `${offsetHeight}px`;
          }
          if (props.hideOnLeave) {
            el.style.setProperty("display", "none", "important");
          }
        },
        onAfterLeave(el) {
          if (props.leaveAbsolute && el?._transitionInitialStyles) {
            const {
              position,
              top,
              left,
              width,
              height
            } = el._transitionInitialStyles;
            delete el._transitionInitialStyles;
            el.style.position = position || "";
            el.style.top = top || "";
            el.style.left = left || "";
            el.style.width = width || "";
            el.style.height = height || "";
          }
        }
      };
      return () => {
        const tag = props.group ? TransitionGroup : Transition;
        return h(tag, {
          name: props.disabled ? "" : name,
          css: !props.disabled,
          ...props.group ? void 0 : {
            mode: props.mode
          },
          ...props.disabled ? {} : functions
        }, slots.default);
      };
    }
  });
}
function createJavascriptTransition(name, functions, mode = "in-out") {
  return genericComponent()({
    name,
    props: {
      mode: {
        type: String,
        default: mode
      },
      disabled: {
        type: Boolean,
        default: PREFERS_REDUCED_MOTION()
      },
      group: Boolean,
      hideOnLeave: Boolean
    },
    setup(props, {
      slots
    }) {
      const tag = props.group ? TransitionGroup : Transition;
      return () => {
        return h(tag, {
          name: props.disabled ? "" : name,
          css: !props.disabled,
          // mode: props.mode, // TODO: vuejs/vue-next#3104
          ...props.disabled ? {} : {
            ...functions,
            onLeave: (el) => {
              if (props.hideOnLeave) {
                el.style.setProperty("display", "none", "important");
              } else {
                functions.onLeave?.(el);
              }
            }
          }
        }, slots.default);
      };
    }
  });
}
function ExpandTransitionGenerator(expandedParentClass = "", type = "y") {
  return {
    onBeforeEnter(el) {
      el._parent = el.parentNode;
      el._initialStyle = {
        transition: el.style.transition,
        overflow: el.style.overflow,
        width: el.style.width,
        height: el.style.height
      };
    },
    onEnter(el) {
      const initialStyle = el._initialStyle;
      if (!initialStyle) return;
      el.style.setProperty("transition", "none", "important");
      el.style.overflow = "hidden";
      const offsetWidth = `${el.offsetWidth}px`;
      const offsetHeight = `${el.offsetHeight}px`;
      if (["x", "both"].includes(type)) el.style.width = "0";
      if (["y", "both"].includes(type)) el.style.height = "0";
      void el.offsetHeight;
      el.style.transition = initialStyle.transition;
      if (expandedParentClass && el._parent) {
        el._parent.classList.add(expandedParentClass);
      }
      requestAnimationFrame(() => {
        if (["x", "both"].includes(type)) el.style.width = offsetWidth;
        if (["y", "both"].includes(type)) el.style.height = offsetHeight;
      });
    },
    onAfterEnter: resetStyles,
    onEnterCancelled: resetStyles,
    onLeave(el) {
      el._initialStyle = {
        transition: "",
        overflow: el.style.overflow,
        width: el.style.width,
        height: el.style.height
      };
      el.style.overflow = "hidden";
      if (["x", "both"].includes(type)) el.style.width = `${el.offsetWidth}px`;
      if (["y", "both"].includes(type)) el.style.height = `${el.offsetHeight}px`;
      void el.offsetHeight;
      requestAnimationFrame(() => {
        if (["x", "both"].includes(type)) el.style.width = "0";
        if (["y", "both"].includes(type)) el.style.height = "0";
      });
    },
    onAfterLeave,
    onLeaveCancelled: onAfterLeave
  };
  function onAfterLeave(el) {
    if (expandedParentClass && el._parent) {
      el._parent.classList.remove(expandedParentClass);
    }
    resetStyles(el);
  }
  function resetStyles(el) {
    if (!el._initialStyle) return;
    const {
      width: w,
      height: h2
    } = el._initialStyle;
    el.style.overflow = el._initialStyle.overflow;
    if (w != null && ["x", "both"].includes(type)) el.style.width = w;
    if (h2 != null && ["y", "both"].includes(type)) el.style.height = h2;
    delete el._initialStyle;
  }
}
createCssTransition("fab-transition", "center center", "out-in");
createCssTransition("dialog-bottom-transition");
createCssTransition("dialog-top-transition");
const VFadeTransition = createCssTransition("fade-transition");
createCssTransition("scale-transition");
createCssTransition("scroll-x-transition");
createCssTransition("scroll-x-reverse-transition");
createCssTransition("scroll-y-transition");
createCssTransition("scroll-y-reverse-transition");
createCssTransition("slide-x-transition");
createCssTransition("slide-x-reverse-transition");
createCssTransition("slide-y-transition");
createCssTransition("slide-y-reverse-transition");
createJavascriptTransition("expand-transition", ExpandTransitionGenerator());
createJavascriptTransition("expand-x-transition", ExpandTransitionGenerator("", "x"));
createJavascriptTransition("expand-both-transition", ExpandTransitionGenerator("", "both"));
const VPickerTitle = createSimpleFunctional("v-picker-title");
const makeVSheetProps = propsFactory({
  color: String,
  ...makeBorderProps(),
  ...makeComponentProps(),
  ...makeDimensionProps(),
  ...makeElevationProps(),
  ...makeLocationProps(),
  ...makePositionProps(),
  ...makeRoundedProps(),
  ...makeTagProps(),
  ...makeThemeProps()
}, "VSheet");
const VSheet = genericComponent()({
  name: "VSheet",
  props: makeVSheetProps(),
  setup(props, {
    slots
  }) {
    const {
      themeClasses
    } = provideTheme(props);
    const {
      backgroundColorClasses,
      backgroundColorStyles
    } = useBackgroundColor(() => props.color);
    const {
      borderClasses
    } = useBorder(props);
    const {
      dimensionStyles
    } = useDimension(props);
    const {
      elevationClasses
    } = useElevation(props);
    const {
      locationStyles
    } = useLocation(props);
    const {
      positionClasses
    } = usePosition(props);
    const {
      roundedClasses,
      roundedStyles
    } = useRounded(props);
    useRender(() => createVNode(props.tag, {
      "class": normalizeClass(["v-sheet", themeClasses.value, backgroundColorClasses.value, borderClasses.value, elevationClasses.value, positionClasses.value, roundedClasses.value, props.class]),
      "style": normalizeStyle([backgroundColorStyles.value, dimensionStyles.value, locationStyles.value, roundedStyles.value, props.style])
    }, slots));
    return {};
  }
});
const makeVPickerProps = propsFactory({
  bgColor: String,
  divided: Boolean,
  landscape: Boolean,
  title: String,
  hideHeader: Boolean,
  hideTitle: Boolean,
  ...makeVSheetProps()
}, "VPicker");
const VPicker = genericComponent()({
  name: "VPicker",
  props: makeVPickerProps(),
  setup(props, {
    slots
  }) {
    const {
      backgroundColorClasses,
      backgroundColorStyles
    } = useBackgroundColor(() => props.color);
    useRender(() => {
      const sheetProps = VSheet.filterProps(props);
      const hasTitle = !props.hideTitle && !!(props.title || slots.title);
      return createVNode(VSheet, mergeProps(sheetProps, {
        "color": props.bgColor,
        "class": ["v-picker", {
          "v-picker--divided": props.divided,
          "v-picker--landscape": props.landscape,
          "v-picker--with-actions": !!slots.actions
        }, props.class],
        "style": props.style
      }), {
        default: () => [!props.hideHeader && createElementVNode("div", {
          "key": "header",
          "class": normalizeClass(["v-picker__header-wrapper", backgroundColorClasses.value]),
          "style": normalizeStyle([backgroundColorStyles.value])
        }, [hasTitle && createVNode(VPickerTitle, {
          "key": "picker-title"
        }, {
          default: () => [slots.title?.() ?? props.title]
        }), slots.header && createElementVNode("div", {
          "class": "v-picker__header"
        }, [slots.header()])]), createElementVNode("div", {
          "class": "v-picker__body"
        }, [slots.default?.()]), slots.actions && createVNode(VDefaultsProvider, {
          "defaults": {
            VBtn: {
              slim: true,
              variant: "text"
            }
          }
        }, {
          default: () => [createElementVNode("div", {
            "class": "v-picker__actions"
          }, [slots.actions()])]
        })]
      });
    });
    return {};
  }
});
const makeVDatePickerProps = propsFactory({
  // TODO: implement in v3.5
  // calendarIcon: {
  //   type: String,
  //   default: '$calendar',
  // },
  // keyboardIcon: {
  //   type: String,
  //   default: '$edit',
  // },
  // inputMode: {
  //   type: String as PropType<'calendar' | 'keyboard'>,
  //   default: 'calendar',
  // },
  // inputText: {
  //   type: String,
  //   default: '$vuetify.datePicker.input.placeholder',
  // },
  // inputPlaceholder: {
  //   type: String,
  //   default: 'dd/mm/yyyy',
  // },
  header: {
    type: String,
    default: "$vuetify.datePicker.header"
  },
  headerColor: String,
  headerDateFormat: {
    type: String,
    default: "normalDateWithWeekday"
  },
  landscapeHeaderWidth: [Number, String],
  ...omit(makeVDatePickerControlsProps(), ["active", "monthText", "yearText"]),
  ...makeVDatePickerMonthProps({
    weeksInMonth: "static"
  }),
  ...omit(makeVDatePickerMonthsProps(), ["modelValue", "columns"]),
  ...omit(makeVDatePickerYearsProps(), ["modelValue", "columns"]),
  ...makeVPickerProps({
    title: "$vuetify.datePicker.title"
  }),
  modelValue: null
}, "VDatePicker");
const VDatePicker = genericComponent()({
  name: "VDatePicker",
  props: makeVDatePickerProps(),
  emits: {
    "update:modelValue": (date2) => true,
    "update:month": (date2) => true,
    "update:year": (date2) => true,
    "update:previewValue": (_value) => true,
    // 'update:inputMode': (date: any) => true,
    "update:viewMode": (date2) => true,
    "boundary-navigate": (_payload) => true
  },
  setup(props, {
    emit,
    slots
  }) {
    const adapter = useDate();
    const {
      t
    } = useLocale();
    const {
      rtlClasses
    } = useRtl();
    const model = useProxiedModel(props, "modelValue", void 0, (v) => wrapInArray(v).map((i) => adapter.date(i)), (v) => props.multiple ? v : v[0]);
    const viewMode = useProxiedModel(props, "viewMode");
    const previewValue = useProxiedModel(props, "previewValue");
    const {
      minDate,
      maxDate,
      clampDate
    } = useCalendarRange(props);
    const internal = computed(() => {
      const today = adapter.date();
      const value = model.value?.[0] ? adapter.date(model.value[0]) : clampDate(today);
      return value && adapter.isValid(value) ? value : today;
    });
    const headerColor = toRef(() => props.headerColor ?? props.color);
    const _month = useProxiedModel(props, "month");
    const month = computed({
      get: () => Number(_month.value ?? adapter.getMonth(adapter.startOfMonth(internal.value))),
      set: (v) => _month.value = v
    });
    const _year = useProxiedModel(props, "year");
    const year = computed({
      get: () => Number(_year.value ?? adapter.getYear(adapter.startOfYear(adapter.setMonth(internal.value, month.value)))),
      set: (v) => _year.value = v
    });
    const isReversing = shallowRef(false);
    const header = computed(() => {
      if (props.multiple === "range" && model.value.length === 2) {
        const [startDate, endDate] = model.value;
        const daysBetween = adapter.getDiff(endDate, startDate, "days") + 1;
        return t("$vuetify.datePicker.itemsSelected", daysBetween);
      }
      if (props.multiple && model.value.length > 1) {
        return t("$vuetify.datePicker.itemsSelected", model.value.length);
      }
      const formattedDate = model.value[0] && adapter.isValid(model.value[0]) ? adapter.format(adapter.date(model.value[0]), props.headerDateFormat) : t(props.header);
      return props.landscape && formattedDate.split(" ").length === 3 ? formattedDate.replace(" ", "\n") : formattedDate;
    });
    const monthStart = toRef(() => {
      let date2 = adapter.date();
      date2 = adapter.setDate(date2, 1);
      date2 = adapter.setMonth(date2, month.value);
      date2 = adapter.setYear(date2, year.value);
      return date2;
    });
    const monthYearText = toRef(() => adapter.format(monthStart.value, "monthAndYear"));
    const monthText = toRef(() => adapter.format(monthStart.value, "monthShort"));
    const yearText = toRef(() => adapter.format(monthStart.value, "year"));
    const headerTransition = toRef(() => `date-picker-header${isReversing.value ? "-reverse" : ""}-transition`);
    const disabled = computed(() => {
      if (props.disabled) return true;
      const targets = [];
      if (viewMode.value !== "month") {
        targets.push(...["prev-month", "next-month", "prev-year", "next-year"]);
      } else {
        let _date = adapter.date();
        _date = adapter.startOfMonth(_date);
        _date = adapter.setMonth(_date, month.value);
        _date = adapter.setYear(_date, year.value);
        if (minDate.value) {
          const prevMonthEnd = adapter.addDays(adapter.startOfMonth(_date), -1);
          const prevYearEnd = adapter.addDays(adapter.startOfYear(_date), -1);
          adapter.isAfter(minDate.value, prevMonthEnd) && targets.push("prev-month");
          adapter.isAfter(minDate.value, prevYearEnd) && targets.push("prev-year");
        }
        if (maxDate.value) {
          const nextMonthStart = adapter.addDays(adapter.endOfMonth(_date), 1);
          const nextYearStart = adapter.addDays(adapter.endOfYear(_date), 1);
          adapter.isAfter(nextMonthStart, maxDate.value) && targets.push("next-month");
          adapter.isAfter(nextYearStart, maxDate.value) && targets.push("next-year");
        }
      }
      return targets;
    });
    const allowedYears = computed(() => {
      return props.allowedYears || isYearAllowed;
    });
    const allowedMonths = computed(() => {
      return props.allowedMonths || isMonthAllowed;
    });
    function isAllowedInRange(start, end) {
      const allowedDates = props.allowedDates;
      if (!isFunction(allowedDates)) return true;
      const days = 1 + daysDiff(adapter, start, end);
      for (let i = 0; i < days; i++) {
        if (allowedDates(adapter.addDays(start, i))) return true;
      }
      return false;
    }
    function isYearAllowed(year2) {
      if (isFunction(props.allowedDates)) {
        const startOfYear2 = adapter.parseISO(`${year2}-01-01`);
        return isAllowedInRange(startOfYear2, adapter.endOfYear(startOfYear2));
      }
      if (Array.isArray(props.allowedDates) && props.allowedDates.length) {
        for (const date2 of props.allowedDates) {
          if (adapter.getYear(adapter.date(date2)) === year2) return true;
        }
        return false;
      }
      return true;
    }
    function isMonthAllowed(month2) {
      if (isFunction(props.allowedDates)) {
        const monthTwoDigits = String(month2 + 1).padStart(2, "0");
        const startOfMonth2 = adapter.parseISO(`${year.value}-${monthTwoDigits}-01`);
        return isAllowedInRange(startOfMonth2, adapter.endOfMonth(startOfMonth2));
      }
      if (Array.isArray(props.allowedDates) && props.allowedDates.length) {
        for (const date2 of props.allowedDates) {
          if (adapter.getYear(adapter.date(date2)) === year.value && adapter.getMonth(adapter.date(date2)) === month2) return true;
        }
        return false;
      }
      return true;
    }
    function onClickNextMonth() {
      if (month.value < 11) {
        month.value++;
      } else {
        year.value++;
        month.value = 0;
        onUpdateYear();
      }
      onUpdateMonth();
    }
    function onClickPrevMonth() {
      if (month.value > 0) {
        month.value--;
      } else {
        year.value--;
        month.value = 11;
        onUpdateYear();
      }
      onUpdateMonth();
    }
    function onClickNextYear() {
      year.value++;
      if (maxDate.value) {
        const monthTwoDigits = String(month.value + 1).padStart(2, "0");
        const monthStart2 = adapter.parseISO(`${year.value}-${monthTwoDigits}-01`);
        if (adapter.isAfter(monthStart2, maxDate.value)) {
          month.value = adapter.getMonth(maxDate.value);
        }
      }
      onUpdateYear();
    }
    function onClickPrevYear() {
      year.value--;
      if (minDate.value) {
        const monthTwoDigits = String(month.value + 1).padStart(2, "0");
        const monthStart2 = adapter.endOfMonth(adapter.parseISO(`${year.value}-${monthTwoDigits}-01`));
        if (adapter.isAfter(minDate.value, monthStart2)) {
          month.value = adapter.getMonth(minDate.value);
        }
      }
      onUpdateYear();
    }
    const monthGridRef = shallowRef();
    function focusDate(isoDate) {
      monthGridRef.value?.focusItem(isoDate);
    }
    function onClickDate() {
      viewMode.value = "month";
    }
    function onEscape() {
      viewMode.value = "month";
      nextTick(() => monthGridRef.value?.focusGrid());
    }
    function onClickMonth() {
      viewMode.value = viewMode.value === "months" ? "month" : "months";
    }
    function onClickYear() {
      viewMode.value = viewMode.value === "year" ? "month" : "year";
    }
    function onUpdateMonth() {
      if (viewMode.value === "months") onClickMonth();
    }
    function onUpdateYear() {
      if (viewMode.value === "year") onClickYear();
    }
    watch(model, (val, oldVal) => {
      if (props.noAutoNavigation) return;
      const arrBefore = wrapInArray(oldVal);
      const arrAfter = wrapInArray(val);
      if (!arrAfter.length) return;
      const before = adapter.date(arrBefore[arrBefore.length - 1]);
      const after = adapter.date(arrAfter[arrAfter.length - 1]);
      if (adapter.isSameDay(before, after)) return;
      const newMonth = adapter.getMonth(after);
      const newYear = adapter.getYear(after);
      if (newMonth !== month.value) {
        month.value = newMonth;
        onUpdateMonth();
      }
      if (newYear !== year.value) {
        year.value = newYear;
        onUpdateYear();
      }
      isReversing.value = adapter.isBefore(before, after);
    });
    useRender(() => {
      const pickerProps = VPicker.filterProps(props);
      const datePickerControlsProps = omit(VDatePickerControls.filterProps(props), ["viewMode"]);
      const datePickerHeaderProps = VDatePickerHeader.filterProps(props);
      const datePickerMonthProps = VDatePickerMonth.filterProps(props);
      const datePickerMonthsProps = omit(VDatePickerMonths.filterProps(props), ["modelValue"]);
      const datePickerYearsProps = omit(VDatePickerYears.filterProps(props), ["modelValue"]);
      const headerProps = {
        color: headerColor.value,
        header: header.value,
        transition: headerTransition.value
      };
      return createVNode(VPicker, mergeProps(pickerProps, {
        "color": headerColor.value,
        "class": ["v-date-picker", `v-date-picker--${viewMode.value}`, {
          "v-date-picker--show-week": props.showWeek
        }, rtlClasses.value, props.class],
        "style": [{
          "--v-date-picker-landscape-header-width": convertToUnit(props.landscapeHeaderWidth)
        }, props.style]
      }), {
        title: () => slots.title?.() ?? createElementVNode("div", {
          "class": "v-date-picker__title"
        }, [t(props.title)]),
        header: () => slots.header ? createVNode(VDefaultsProvider, {
          "defaults": {
            VDatePickerHeader: {
              ...headerProps
            }
          }
        }, {
          default: () => [slots.header?.(headerProps)]
        }) : createVNode(VDatePickerHeader, mergeProps({
          "key": "header"
        }, datePickerHeaderProps, headerProps, {
          "onClick": viewMode.value !== "month" ? onClickDate : void 0
        }), {
          prepend: slots.prepend,
          append: slots.append
        }),
        default: () => createElementVNode(Fragment, null, [createVNode(VDatePickerControls, mergeProps(datePickerControlsProps, {
          "disabled": disabled.value,
          "viewMode": viewMode.value,
          "text": monthYearText.value,
          "monthText": monthText.value,
          "yearText": yearText.value,
          "onClick:next": onClickNextMonth,
          "onClick:prev": onClickPrevMonth,
          "onClick:nextYear": onClickNextYear,
          "onClick:prevYear": onClickPrevYear,
          "onClick:month": onClickMonth,
          "onClick:year": onClickYear
        }), {
          default: slots.controls
        }), createVNode(VFadeTransition, {
          "hideOnLeave": true
        }, {
          default: () => [viewMode.value === "months" ? createVNode(VDatePickerMonths, mergeProps({
            "key": "date-picker-months"
          }, datePickerMonthsProps, {
            "modelValue": month.value,
            "onUpdate:modelValue": [($event) => month.value = $event, onUpdateMonth],
            "min": minDate.value,
            "max": maxDate.value,
            "year": year.value,
            "allowedMonths": allowedMonths.value,
            "onEscape": onEscape
          }), {
            month: slots.month
          }) : viewMode.value === "year" ? createVNode(VDatePickerYears, mergeProps({
            "key": "date-picker-years"
          }, datePickerYearsProps, {
            "modelValue": year.value,
            "onUpdate:modelValue": [($event) => year.value = $event, onUpdateYear],
            "min": minDate.value,
            "max": maxDate.value,
            "allowedYears": allowedYears.value,
            "onEscape": onEscape
          }), {
            year: slots.year
          }) : createVNode(VDatePickerMonth, mergeProps({
            "ref": monthGridRef,
            "key": "date-picker-month"
          }, datePickerMonthProps, {
            "modelValue": model.value,
            "onUpdate:modelValue": ($event) => model.value = $event,
            "month": month.value,
            "onUpdate:month": [($event) => month.value = $event, onUpdateMonth],
            "year": year.value,
            "onUpdate:year": [($event) => year.value = $event, onUpdateYear],
            "previewValue": previewValue.value,
            "onUpdate:previewValue": ($event) => previewValue.value = $event,
            "onBoundaryNavigate": (payload) => emit("boundary-navigate", payload),
            "min": minDate.value,
            "max": maxDate.value
          }), {
            day: slots.day
          })]
        })]),
        actions: slots.actions
      });
    });
    return {
      focusDate
    };
  }
});
const datePickerForwardedKeys = ["min", "max", "allowedDates", "allowedMonths", "allowedYears", "weekdays", "firstDayOfWeek", "firstDayOfYear", "weekdayFormat", "weeksInMonth", "hideWeekdays", "showWeek", "color", "readonly", "disabled", "prevIcon", "nextIcon", "transition", "reverseTransition", "controlHeight"];
const makeVDateRangePickerProps = propsFactory({
  independentMonths: Boolean,
  modelValue: null,
  ...makeVPickerProps({
    title: "$vuetify.datePicker.title"
  }),
  ...pick(makeVDatePickerProps(), datePickerForwardedKeys)
}, "VDateRangePicker");
const VDateRangePicker = genericComponent()({
  name: "VDateRangePicker",
  props: makeVDateRangePickerProps(),
  emits: {
    "update:modelValue": (_value) => true
  },
  setup(props, {
    slots
  }) {
    const adapter = useDate();
    const model = useProxiedModel(props, "modelValue", []);
    const today = adapter.date();
    const leftMonth = shallowRef(adapter.getMonth(today));
    const leftYear = shallowRef(adapter.getYear(today));
    const rightMonth = shallowRef(0);
    const rightYear = shallowRef(0);
    function shiftMonth(month, year, delta2) {
      const total = month + delta2 + year * 12;
      return {
        month: (total % 12 + 12) % 12,
        year: Math.floor(total / 12)
      };
    }
    function syncRightFromLeft() {
      const next = shiftMonth(leftMonth.value, leftYear.value, 1);
      rightMonth.value = next.month;
      rightYear.value = next.year;
    }
    function syncLeftFromRight() {
      const previous = shiftMonth(rightMonth.value, rightYear.value, -1);
      leftMonth.value = previous.month;
      leftYear.value = previous.year;
    }
    const leftIndex = computed(() => leftYear.value * 12 + leftMonth.value);
    const rightIndex = computed(() => rightYear.value * 12 + rightMonth.value);
    const leftNextDisabled = computed(() => leftIndex.value + 1 >= rightIndex.value);
    const rightPrevDisabled = computed(() => rightIndex.value - 1 <= leftIndex.value);
    const rootRef = shallowRef();
    const leftPickerRef = shallowRef();
    const rightPickerRef = shallowRef();
    const previewValue = shallowRef();
    function panelContaining(iso) {
      if (leftPickerRef.value?.$el?.querySelector(`[data-v-date="${iso}"]`)) return leftPickerRef.value;
      if (rightPickerRef.value?.$el?.querySelector(`[data-v-date="${iso}"]`)) return rightPickerRef.value;
      return null;
    }
    function isMonthInView(date2) {
      const month = adapter.getMonth(date2);
      const year = adapter.getYear(date2);
      return month === leftMonth.value && year === leftYear.value || month === rightMonth.value && year === rightYear.value;
    }
    function navigateToSelection() {
      const dates = model.value;
      if (!dates.length) return;
      const first = adapter.date(dates[0]);
      const last = adapter.date(dates[dates.length - 1]);
      leftMonth.value = adapter.getMonth(first);
      leftYear.value = adapter.getYear(first);
      if (!props.independentMonths) {
        syncRightFromLeft();
        return;
      }
      const firstIndex = leftIndex.value;
      const lastIndex = adapter.getYear(last) * 12 + adapter.getMonth(last);
      const targetRightIndex = Math.max(lastIndex, firstIndex + 1);
      rightYear.value = Math.floor(targetRightIndex / 12);
      rightMonth.value = targetRightIndex - rightYear.value * 12;
    }
    syncRightFromLeft();
    navigateToSelection();
    let syncing = false;
    watch([leftMonth, leftYear], () => {
      if (syncing || props.independentMonths) return;
      syncing = true;
      syncRightFromLeft();
      syncing = false;
    });
    watch([rightMonth, rightYear], () => {
      if (syncing || props.independentMonths) return;
      syncing = true;
      syncLeftFromRight();
      syncing = false;
    });
    watch(() => props.independentMonths, (value, oldValue) => {
      if (value || !oldValue) return;
      const start = model.value[0];
      if (start != null) {
        const startIndex = adapter.getYear(start) * 12 + adapter.getMonth(start);
        if (startIndex === rightIndex.value) {
          syncing = true;
          syncLeftFromRight();
          syncing = false;
          return;
        }
      }
      syncing = true;
      syncRightFromLeft();
      syncing = false;
    });
    watch(model, (val) => {
      const dates = val;
      if (!dates.length) return;
      const first = adapter.date(dates[0]);
      const last = adapter.date(dates[dates.length - 1]);
      if (isMonthInView(first) && isMonthInView(last)) return;
      navigateToSelection();
    });
    function focusCell(iso) {
      panelContaining(iso)?.focusDate(iso);
    }
    function onBoundaryNavigate({
      direction,
      targetIsoDate
    }) {
      if (panelContaining(targetIsoDate)) {
        focusCell(targetIsoDate);
        return;
      }
      const target = adapter.date(targetIsoDate);
      const targetMonth = adapter.getMonth(target);
      const targetYear = adapter.getYear(target);
      const targetIndex = targetYear * 12 + targetMonth;
      if (targetIndex < leftIndex.value) {
        const wouldCollide = props.independentMonths && rightIndex.value <= targetIndex;
        leftMonth.value = targetMonth;
        leftYear.value = targetYear;
        if (wouldCollide) {
          const next = shiftMonth(targetMonth, targetYear, 1);
          rightMonth.value = next.month;
          rightYear.value = next.year;
        }
      } else if (targetIndex > rightIndex.value) {
        const wouldCollide = props.independentMonths && leftIndex.value >= targetIndex;
        rightMonth.value = targetMonth;
        rightYear.value = targetYear;
        if (wouldCollide) {
          const previous = shiftMonth(targetMonth, targetYear, -1);
          leftMonth.value = previous.month;
          leftYear.value = previous.year;
        }
      } else {
        const goingBack = direction === "left" || direction === "up";
        if (goingBack) {
          rightMonth.value = targetMonth;
          rightYear.value = targetYear;
        } else {
          leftMonth.value = targetMonth;
          leftYear.value = targetYear;
        }
      }
      nextTick(() => focusCell(targetIsoDate));
    }
    useRender(() => {
      const pickerProps = VPicker.filterProps(props);
      const datePickerProps = pick(props, datePickerForwardedKeys);
      return createVNode(VPicker, mergeProps({
        "ref": rootRef
      }, pickerProps, {
        "class": ["v-date-range-picker", props.class],
        "style": props.style,
        "hideHeader": true
      }), {
        default: () => createElementVNode(Fragment, null, [createVNode(VDatePicker, mergeProps({
          "ref": leftPickerRef
        }, datePickerProps, {
          "modelValue": model.value,
          "onUpdate:modelValue": ($event) => model.value = $event,
          "month": leftMonth.value,
          "onUpdate:month": ($event) => leftMonth.value = $event,
          "year": leftYear.value,
          "onUpdate:year": ($event) => leftYear.value = $event,
          "class": "v-date-range-picker__panel",
          "multiple": "range",
          "noAutoNavigation": true,
          "showAdjacentMonths": false,
          "hideHeader": true,
          "previewValue": previewValue.value,
          "onUpdate:previewValue": (value) => previewValue.value = value,
          "onBoundaryNavigate": onBoundaryNavigate
        }), {
          controls: ({
            monthYearText,
            prevMonth,
            nextMonth,
            disabled
          }) => createElementVNode(Fragment, null, [createVNode(VBtn, {
            "density": "comfortable",
            "disabled": disabled.includes("prev-month"),
            "icon": props.prevIcon,
            "variant": "text",
            "onClick": prevMonth
          }, null), createVNode(VSpacer, null, null), createElementVNode("div", null, [monthYearText]), createVNode(VSpacer, null, null), createVNode(VBtn, {
            "class": normalizeClass({
              "v-date-range-picker__nav-hidden": !props.independentMonths
            }),
            "density": "comfortable",
            "disabled": !props.independentMonths || leftNextDisabled.value || disabled.includes("next-month"),
            "icon": props.nextIcon,
            "variant": "text",
            "onClick": nextMonth
          }, null)])
        }), createElementVNode("div", {
          "class": "v-date-range-picker__divider",
          "role": "separator"
        }, null), createVNode(VDatePicker, mergeProps({
          "ref": rightPickerRef
        }, datePickerProps, {
          "modelValue": model.value,
          "onUpdate:modelValue": ($event) => model.value = $event,
          "month": rightMonth.value,
          "onUpdate:month": ($event) => rightMonth.value = $event,
          "year": rightYear.value,
          "onUpdate:year": ($event) => rightYear.value = $event,
          "class": "v-date-range-picker__panel",
          "multiple": "range",
          "noAutoNavigation": true,
          "showAdjacentMonths": false,
          "hideHeader": true,
          "previewValue": previewValue.value,
          "onUpdate:previewValue": (value) => previewValue.value = value,
          "onBoundaryNavigate": onBoundaryNavigate
        }), {
          controls: ({
            monthYearText,
            prevMonth,
            nextMonth,
            disabled
          }) => createElementVNode(Fragment, null, [createVNode(VBtn, {
            "class": normalizeClass({
              "v-date-range-picker__nav-hidden": !props.independentMonths
            }),
            "density": "comfortable",
            "disabled": !props.independentMonths || rightPrevDisabled.value || disabled.includes("prev-month"),
            "icon": props.prevIcon,
            "variant": "text",
            "onClick": prevMonth
          }, null), createVNode(VSpacer, null, null), createElementVNode("div", null, [monthYearText]), createVNode(VSpacer, null, null), createVNode(VBtn, {
            "density": "comfortable",
            "disabled": disabled.includes("next-month"),
            "icon": props.nextIcon,
            "variant": "text",
            "onClick": nextMonth
          }, null)])
        })]),
        actions: slots.footer
      });
    });
    return {};
  }
});
const vuetify_hjFy4UiBVKu2U8_BW9ggkFzfvErKr3wFgTHpa6TF5Ds = /* @__PURE__ */ defineNuxtPlugin((nuxtApp) => {
  const vuetify = createVuetify({
    components: { VDateRangePicker },
    theme: {
      defaultTheme: "light",
      themes: {
        light: { colors: { primary: "#1769e8" } }
      }
    }
  });
  nuxtApp.vueApp.use(vuetify);
});
const plugins = [
  unhead_k2P3m_ZDyjlr2mMYnoDPwavjsDN8hBlk9cFai0bbopU,
  plugin$1,
  revive_payload_server_MVtmlZaQpj6ApFmshWfUWl5PehCebzaBf2NuRMiIbms,
  plugin,
  components_plugin_z4hgvsiddfKkfXTP6M8M4zG5Cb7sGnDhcryKVM45Di4,
  vuetify_hjFy4UiBVKu2U8_BW9ggkFzfvErKr3wFgTHpa6TF5Ds
];
const layouts = {
  auth: defineAsyncComponent(() => import("./_nuxt/auth-BAYWvNQB.js").then((m) => m.default || m)),
  default: defineAsyncComponent(() => import("./_nuxt/default-BHrF_4AT.js").then((m) => m.default || m))
};
const routeRulesMatcher = _routeRulesMatcher;
const LayoutLoader = defineComponent$1({
  name: "LayoutLoader",
  inheritAttrs: false,
  props: {
    name: String,
    layoutProps: Object
  },
  setup(props, context) {
    return () => h(layouts[props.name], props.layoutProps, context.slots);
  }
});
const nuxtLayoutProps = {
  name: {
    type: [String, Boolean, Object],
    default: null
  },
  fallback: {
    type: [String, Object],
    default: null
  }
};
const __nuxt_component_0 = defineComponent$1({
  name: "NuxtLayout",
  inheritAttrs: false,
  props: nuxtLayoutProps,
  setup(props, context) {
    const nuxtApp = useNuxtApp();
    const injectedRoute = inject$1(PageRouteSymbol);
    const shouldUseEagerRoute = !injectedRoute || injectedRoute === useRoute$1();
    const route = shouldUseEagerRoute ? useRoute$2() : injectedRoute;
    const layout = computed(() => {
      let layout2 = unref(props.name) ?? route?.meta.layout ?? routeRulesMatcher(route?.path).appLayout ?? "default";
      if (layout2 && !(layout2 in layouts)) {
        if (props.fallback) {
          layout2 = unref(props.fallback);
        }
      }
      return layout2;
    });
    const layoutRef = shallowRef();
    context.expose({ layoutRef });
    const done = nuxtApp.deferHydration();
    let lastLayout;
    return () => {
      const hasLayout = !!layout.value && layout.value in layouts;
      const hasTransition = hasLayout && !!(route?.meta.layoutTransition ?? appLayoutTransition);
      const transitionProps = hasTransition && _mergeTransitionProps([
        route?.meta.layoutTransition,
        appLayoutTransition,
        {
          onBeforeLeave() {
            nuxtApp["~transitionPromise"] = new Promise((resolve) => {
              nuxtApp["~transitionFinish"] = resolve;
            });
          },
          onAfterLeave() {
            nuxtApp["~transitionFinish"]?.();
            delete nuxtApp["~transitionFinish"];
            delete nuxtApp["~transitionPromise"];
          }
        }
      ]);
      const previouslyRenderedLayout = lastLayout;
      lastLayout = layout.value;
      return _wrapInTransition(transitionProps, {
        default: () => h(
          Suspense,
          {
            suspensible: true,
            onResolve: async () => {
              await nextTick(done);
            }
          },
          {
            default: () => h(
              LayoutProvider,
              {
                layoutProps: mergeProps(context.attrs, route.meta.layoutProps ?? {}, { ref: layoutRef }),
                key: layout.value || void 0,
                name: layout.value,
                shouldProvide: !props.name,
                isRenderingNewLayout: (name) => {
                  return name !== previouslyRenderedLayout && name === layout.value;
                },
                hasTransition
              },
              context.slots
            )
          }
        )
      }).default();
    };
  }
});
const LayoutProvider = defineComponent$1({
  name: "NuxtLayoutProvider",
  inheritAttrs: false,
  props: {
    name: {
      type: [String, Boolean]
    },
    layoutProps: {
      type: Object
    },
    hasTransition: {
      type: Boolean
    },
    shouldProvide: {
      type: Boolean
    },
    isRenderingNewLayout: {
      type: Function,
      required: true
    }
  },
  setup(props, context) {
    const name = props.name;
    if (props.shouldProvide) {
      provide(LayoutMetaSymbol, {
        // When name=false, always return true so NuxtPage doesn't skip rendering
        isCurrent: (route) => name === false || name === (route.meta.layout ?? routeRulesMatcher(route.path).appLayout ?? "default")
      });
    }
    const injectedRoute = inject$1(PageRouteSymbol);
    const isNotWithinNuxtPage = injectedRoute && injectedRoute === useRoute$1();
    const enclosingLayout = inject$1(LayoutMetaSymbol, null);
    if (isNotWithinNuxtPage) {
      const vueRouterRoute = useRoute$2();
      const reactiveChildRoute = {};
      for (const _key in vueRouterRoute) {
        const key = _key;
        Object.defineProperty(reactiveChildRoute, key, {
          enumerable: true,
          get: () => {
            const useEagerRoute = props.isRenderingNewLayout(props.name) && (!enclosingLayout || enclosingLayout.isCurrent(vueRouterRoute));
            return useEagerRoute ? vueRouterRoute[key] : injectedRoute[key];
          }
        });
      }
      provide(PageRouteSymbol, shallowReactive(reactiveChildRoute));
    }
    return () => {
      if (!name || typeof name === "string" && !(name in layouts)) {
        return context.slots.default?.();
      }
      return h(
        LayoutLoader,
        { key: name, layoutProps: props.layoutProps, name },
        context.slots
      );
    };
  }
});
const defineRouteProvider = (name = "RouteProvider") => defineComponent$1({
  name,
  props: {
    route: {
      type: Object,
      required: true
    },
    vnode: Object,
    vnodeRef: Object,
    renderKey: String,
    trackRootNodes: Boolean
  },
  setup(props) {
    const previousKey = props.renderKey;
    const previousRoute = props.route;
    const route = {};
    for (const key in props.route) {
      Object.defineProperty(route, key, {
        get: () => previousKey === props.renderKey ? props.route[key] : previousRoute[key],
        enumerable: true
      });
    }
    provide(PageRouteSymbol, shallowReactive(route));
    return () => {
      if (!props.vnode) {
        return props.vnode;
      }
      return h(props.vnode, { ref: props.vnodeRef });
    };
  }
});
const RouteProvider = defineRouteProvider();
const __nuxt_component_1 = defineComponent$1({
  name: "NuxtPage",
  inheritAttrs: false,
  props: {
    name: {
      type: String
    },
    transition: {
      type: [Boolean, Object],
      default: void 0
    },
    keepalive: {
      type: [Boolean, Object],
      default: void 0
    },
    route: {
      type: Object
    },
    pageKey: {
      type: [Function, String],
      default: null
    }
  },
  setup(props, { attrs, slots, expose }) {
    const nuxtApp = useNuxtApp();
    const pageRef = ref();
    inject$1(PageRouteSymbol, null);
    expose({ pageRef });
    inject$1(LayoutMetaSymbol, null);
    nuxtApp.deferHydration();
    return () => {
      return h(RouterView, { name: props.name, route: props.route, ...attrs }, {
        default: markStableSlot((routeProps) => {
          return h(Suspense, { suspensible: true }, {
            default() {
              return h(RouteProvider, {
                vnode: slots.default ? normalizeSlot(slots.default, routeProps) : routeProps.Component,
                route: routeProps.route,
                vnodeRef: pageRef
              });
            }
          });
        })
      });
    };
  }
});
function markStableSlot(fn) {
  const wrapped = ((routeProps) => {
    const result = fn(routeProps);
    if (Array.isArray(result)) {
      return result;
    }
    if (result == null || !isVNode(result)) {
      return [createCommentVNode()];
    }
    return [result];
  });
  wrapped._n = true;
  return wrapped;
}
function normalizeSlot(slot, data) {
  const slotContent = slot(data);
  return slotContent.length === 1 ? h(slotContent[0]) : h(Fragment, void 0, slotContent);
}
const _export_sfc = (sfc, props) => {
  const target = sfc.__vccOpts || sfc;
  for (const [key, val] of props) {
    target[key] = val;
  }
  return target;
};
const _sfc_main$2 = {};
function _sfc_ssrRender(_ctx, _push, _parent, _attrs) {
  const _component_NuxtLayout = __nuxt_component_0;
  const _component_NuxtPage = __nuxt_component_1;
  _push(ssrRenderComponent(_component_NuxtLayout, _attrs, {
    default: withCtx((_, _push2, _parent2, _scopeId) => {
      if (_push2) {
        _push2(ssrRenderComponent(_component_NuxtPage, null, null, _parent2, _scopeId));
      } else {
        return [
          createVNode(_component_NuxtPage)
        ];
      }
    }),
    _: 1
  }, _parent));
}
const _sfc_setup$2 = _sfc_main$2.setup;
_sfc_main$2.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("app.vue");
  return _sfc_setup$2 ? _sfc_setup$2(props, ctx) : void 0;
};
const AppComponent = /* @__PURE__ */ _export_sfc(_sfc_main$2, [["ssrRender", _sfc_ssrRender]]);
const _sfc_main$1 = {
  __name: "nuxt-error-page",
  __ssrInlineRender: true,
  props: {
    error: Object
  },
  setup(__props) {
    const props = __props;
    const _error = props.error;
    const status = Number(_error.statusCode || 500);
    const is404 = status === 404;
    const statusText = _error.statusMessage ?? (is404 ? "Page Not Found" : "Internal Server Error");
    const description = _error.message || _error.toString();
    const stack = void 0;
    const _Error404 = defineAsyncComponent(() => import("./_nuxt/error-404-QoVANqZd.js"));
    const _Error = defineAsyncComponent(() => import("./_nuxt/error-500-B4oHMI4b.js"));
    const ErrorTemplate = is404 ? _Error404 : _Error;
    return (_ctx, _push, _parent, _attrs) => {
      _push(ssrRenderComponent(unref(ErrorTemplate), mergeProps({ status: unref(status), statusText: unref(statusText), statusCode: unref(status), statusMessage: unref(statusText), description: unref(description), stack: unref(stack) }, _attrs), null, _parent));
    };
  }
};
const _sfc_setup$1 = _sfc_main$1.setup;
_sfc_main$1.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("node_modules/nuxt/dist/app/components/nuxt-error-page.vue");
  return _sfc_setup$1 ? _sfc_setup$1(props, ctx) : void 0;
};
const _sfc_main = {
  __name: "nuxt-root",
  __ssrInlineRender: true,
  setup(__props) {
    const IslandRenderer = () => null;
    const nuxtApp = useNuxtApp();
    nuxtApp.deferHydration();
    nuxtApp.ssrContext.url;
    const SingleRenderer = false;
    provide(PageRouteSymbol, useRoute$1());
    nuxtApp.hooks.callHookWith((hooks) => hooks.map((hook) => hook()), "vue:setup", []);
    const error = /* @__PURE__ */ useError();
    const abortRender = error.value && !nuxtApp.ssrContext.error;
    function invokeAppErrorHandler(err, target, info) {
      const errorHandler = nuxtApp.vueApp.config.errorHandler;
      if (errorHandler && !errorHandler.__nuxt_default) {
        try {
          errorHandler(err, target, info);
        } catch (handlerError) {
          console.error("[nuxt] Error in `app.config.errorHandler`", handlerError);
        }
      }
    }
    onErrorCaptured((err, target, info) => {
      nuxtApp.hooks.callHook("vue:error", err, target, info).catch((hookError) => console.error("[nuxt] Error in `vue:error` hook", hookError));
      {
        const p = nuxtApp.runWithContext(() => showError(err));
        onServerPrefetch(() => p);
        invokeAppErrorHandler(err, target, info);
        return false;
      }
    });
    const islandContext = nuxtApp.ssrContext.islandContext;
    return (_ctx, _push, _parent, _attrs) => {
      ssrRenderSuspense(_push, {
        default: () => {
          if (unref(abortRender)) {
            _push(`<div></div>`);
          } else if (unref(error)) {
            _push(ssrRenderComponent(unref(_sfc_main$1), { error: unref(error) }, null, _parent));
          } else if (unref(islandContext)) {
            _push(ssrRenderComponent(unref(IslandRenderer), { context: unref(islandContext) }, null, _parent));
          } else if (unref(SingleRenderer)) {
            ssrRenderVNode(_push, createVNode(resolveDynamicComponent(unref(SingleRenderer)), null, null), _parent);
          } else {
            _push(ssrRenderComponent(unref(AppComponent), null, null, _parent));
          }
        },
        _: 1
      });
    };
  }
};
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("node_modules/nuxt/dist/app/components/nuxt-root.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
let entry;
{
  entry = async function createNuxtAppServer(ssrContext) {
    const vueApp = createApp(_sfc_main);
    const nuxt = createNuxtApp({ vueApp, ssrContext });
    try {
      await applyPlugins(nuxt, plugins);
      await nuxt.hooks.callHook("app:created", vueApp);
    } catch (error) {
      await nuxt.hooks.callHook("app:error", error);
      nuxt.payload.error ||= createError(error);
    }
    if (ssrContext && (ssrContext["~renderResponse"] || ssrContext._renderResponse)) {
      throw new Error("skipping render");
    }
    return vueApp;
  };
}
const entry_default = ((ssrContext) => entry(ssrContext));
export {
  VDateRangePicker as V,
  _export_sfc as _,
  useNuxtApp as a,
  useRuntimeConfig as b,
  nuxtLinkDefaults as c,
  usePermissions as d,
  entry_default as default,
  encodeRoutePath as e,
  useAuth as f,
  __nuxt_component_0$1 as g,
  useApi as h,
  useRoute$1 as i,
  usePermissionsStore as j,
  navigateTo as n,
  resolveRouteObject as r,
  tryUseNuxtApp as t,
  useRouter as u
};
//# sourceMappingURL=server.mjs.map
