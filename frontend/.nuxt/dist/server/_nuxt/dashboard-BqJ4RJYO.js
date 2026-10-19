import { _ as __nuxt_component_0 } from "./nuxt-link-Cf-wdoG-.js";
import { c as currentMonthRange, _ as __nuxt_component_0$1 } from "./DateRangePicker-BmLhEXsC.js";
import { _ as __nuxt_component_1 } from "./SearchSelect-BFyTiPOW.js";
import { defineComponent, ref, computed, mergeProps, unref, createVNode, resolveDynamicComponent, withCtx, createTextVNode, openBlock, createBlock, toDisplayString, isRef, useSSRContext } from "vue";
import { ssrRenderAttrs, ssrRenderComponent, ssrInterpolate, ssrRenderList, ssrRenderClass, ssrRenderVNode, ssrRenderAttr, ssrRenderStyle, ssrIncludeBooleanAttr } from "vue/server-renderer";
import { TrendingUp, TrendingDown, CircleDollarSign, ReceiptText, Building2, ShieldCheck, UsersRound, FileChartColumnIncreasing, CalendarDays, ArrowUpRight, Check, ListFilter } from "lucide-vue-next";
import { f as formatMoney } from "./money-BqQfZu3a.js";
import { h as useApi, f as useAuth, d as usePermissions, _ as _export_sfc } from "../server.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/ufo/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/defu/dist/defu.mjs";
import "@vueuse/core";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/ofetch/dist/node.mjs";
import "#internal/nuxt/paths";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/hookable/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/unctx/dist/index.mjs";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/h3/dist/index.mjs";
import "vue-router";
import "/Users/sila/Downloads/income-expense-source (1)/frontend/node_modules/klona/dist/index.mjs";
import "@vuetify/v0/utilities";
const _sfc_main = /* @__PURE__ */ defineComponent({
  __name: "dashboard",
  __ssrInlineRender: true,
  setup(__props) {
    useApi();
    useAuth();
    const { canViewBranches, canViewTransactions, canViewUsers, canViewRoles, canViewReports } = usePermissions();
    const data = ref(null);
    const branches = ref([]);
    const period = ref("this_month");
    const branchId = ref("");
    const currency = ref("USD");
    const dateRange = ref(currentMonthRange());
    const branchOptions = computed(() => [
      { value: "", label: "All accessible branches" },
      ...branches.value.map((branch) => ({ value: branch.id, label: `${branch.code} — ${branch.name}` }))
    ]);
    const loading = ref(true);
    const err = ref("");
    const dateValidationError = ref("");
    const periodOptions = [
      { value: "this_month", label: "Monthly" },
      { value: "this_week", label: "Weekly" },
      { value: "today", label: "Today" }
    ];
    const chartColors = ["#1769e8", "#58a0ff", "#86bcff", "#22a77a", "#f3b544", "#8b7cf6", "#a8b5c7"];
    const periodLabel = computed(() => ({ today: "Today", this_week: "This week", this_month: "This month", custom: "Custom range" })[period.value] || "Selected period");
    const currencyMark = computed(() => currency.value === "USD" ? "$" : "៛");
    const selectedIncomeDisplay = computed(() => formatMoney(currency.value === "USD" ? data.value?.income_usd : data.value?.income_khr, currency.value));
    const selectedExpenseDisplay = computed(() => formatMoney(currency.value === "USD" ? data.value?.expense_usd : data.value?.expense_khr, currency.value));
    const selectedNetDisplay = computed(() => formatMoney(currency.value === "USD" ? data.value?.net_usd : data.value?.net_khr, currency.value));
    computed(() => numberValue(currency.value === "USD" ? data.value?.net_usd : data.value?.net_khr));
    const metrics = computed(() => [
      {
        label: "Total Income",
        value: selectedIncomeDisplay.value,
        note: `${currency.value} · selected period`,
        icon: TrendingUp,
        colorClass: "metric-blue"
      },
      {
        label: "Total Expense",
        value: selectedExpenseDisplay.value,
        note: `${currency.value} · selected period`,
        icon: TrendingDown,
        colorClass: "metric-red"
      },
      {
        label: "Net Profit",
        value: selectedNetDisplay.value,
        note: `${currency.value} · after expenses`,
        icon: CircleDollarSign,
        colorClass: "metric-teal"
      },
      {
        label: "Total Transactions",
        value: new Intl.NumberFormat("en-US").format(data.value?.transaction_count || 0),
        note: "Approved in this period",
        icon: ReceiptText,
        colorClass: "metric-violet"
      }
    ]);
    const dailyItems = computed(() => {
      if (!data.value) return [];
      const byDate = new Map((data.value.daily || []).map((item) => [item.date, item]));
      const start = parseDate(data.value.date_from);
      const end = parseDate(data.value.date_to);
      const days = [];
      if (!start || !end) return [];
      for (const day = new Date(start); day <= end; day.setUTCDate(day.getUTCDate() + 1)) {
        const key = day.toISOString().slice(0, 10);
        const item = byDate.get(key);
        days.push({
          date: key,
          income: numberValue(currency.value === "USD" ? item?.income_usd : item?.income_khr),
          expense: numberValue(currency.value === "USD" ? item?.expense_usd : item?.expense_khr)
        });
      }
      if (days.length <= 7) return days;
      const count = 7;
      return Array.from({ length: count }, (_, index) => {
        const from = Math.floor(index * days.length / count);
        const to = Math.floor((index + 1) * days.length / count);
        const group = days.slice(from, to);
        return {
          date: group[0]?.date || days[from]?.date || "",
          income: group.reduce((sum, day) => sum + day.income, 0),
          expense: group.reduce((sum, day) => sum + day.expense, 0)
        };
      });
    });
    const chartMaximum = computed(() => Math.max(1, ...dailyItems.value.flatMap((item) => [item.income, item.expense])));
    const chartBars = computed(() => {
      const count = dailyItems.value.length;
      if (!count) return [];
      const slot = 648 / count;
      const width = Math.min(28, Math.max(8, slot * 0.25));
      const chartHeight = 170;
      return dailyItems.value.map((item, index) => {
        const center = 56 + slot * index + slot / 2;
        const incomeHeight = item.income / chartMaximum.value * chartHeight;
        const expenseHeight = item.expense / chartMaximum.value * chartHeight;
        const date = parseDate(item.date);
        return {
          label: date ? new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", timeZone: "UTC" }).format(date) : "",
          width,
          incomeHeight,
          expenseHeight,
          incomeX: center - width - 2,
          incomeY: 204 - incomeHeight,
          expenseX: center + 2,
          expenseY: 204 - expenseHeight,
          labelX: center
        };
      });
    });
    const chartTicks = computed(() => [0, 0.25, 0.5, 0.75, 1].map((ratio) => {
      const value = chartMaximum.value * ratio;
      return { value, y: 204 - ratio * 170, label: formatAxisAmount(value) };
    }));
    const categorySegments = computed(() => {
      const rows = (currency.value === "USD" ? data.value?.expense_by_category : data.value?.expense_by_category)?.filter((item) => item.currency === currency.value).map((item) => ({ name: item.category_name || "Uncategorized", amount: numberValue(item.total) })).filter((item) => item.amount > 0).sort((a, b) => b.amount - a.amount) || [];
      const visible = rows.slice(0, 6);
      if (rows.length > 6) visible.push({ name: "Other", amount: rows.slice(6).reduce((sum, item) => sum + item.amount, 0) });
      const total = visible.reduce((sum, item) => sum + item.amount, 0);
      return visible.map((item, index) => ({
        ...item,
        color: chartColors[index % chartColors.length],
        percent: total ? Math.round(item.amount / total * 100) : 0
      }));
    });
    const donutStyle = computed(() => {
      if (!categorySegments.value.length) return { background: "#e8eef7" };
      let cursor = 0;
      const slices = categorySegments.value.map((item) => {
        const start = cursor;
        cursor += item.percent;
        return `${item.color} ${start}% ${cursor}%`;
      });
      return { background: `conic-gradient(${slices.join(", ")})` };
    });
    const quickLinks = computed(() => {
      const links = [];
      if (canViewBranches.value) links.push({ label: "Branches", description: "Manage business locations", to: "/admin/branches", icon: Building2 });
      if (canViewRoles.value) links.push({ label: "Roles & Permissions", description: "Review team access", to: "/admin/roles", icon: ShieldCheck });
      if (canViewUsers.value) links.push({ label: "Users", description: "Manage your team", to: "/admin/users", icon: UsersRound });
      if (canViewReports.value) links.push({ label: "Reports", description: "Explore detailed reports", to: "/reports", icon: FileChartColumnIncreasing });
      return links;
    });
    function parseDate(value) {
      if (!value) return null;
      const parsed = /* @__PURE__ */ new Date(`${value.slice(0, 10)}T00:00:00Z`);
      return Number.isNaN(parsed.valueOf()) ? null : parsed;
    }
    function numberValue(value) {
      const parsed = Number(value || 0);
      return Number.isFinite(parsed) ? parsed : 0;
    }
    function formatNumber(value) {
      return formatMoney(value, currency.value);
    }
    function formatAxisAmount(value) {
      return `${currencyMark.value}${new Intl.NumberFormat("en-US", { notation: "compact", maximumFractionDigits: 1 }).format(value)}`;
    }
    function formatDate(value) {
      const date = parseDate(value);
      return date ? new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", year: "numeric", timeZone: "UTC" }).format(date) : value;
    }
    function formatTransactionDate(value) {
      const date = new Date(value);
      if (Number.isNaN(date.valueOf())) return value;
      return new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" }).format(date);
    }
    function formatStatus(value) {
      return value ? value.charAt(0).toUpperCase() + value.slice(1) : "Approved";
    }
    function branchValue(branch, metric) {
      const suffix = currency.value.toLowerCase();
      const field = `${metric}_${suffix}`;
      return numberValue(branch[field]);
    }
    function formatBranchAmount(branch, metric) {
      return formatNumber(branchValue(branch, metric));
    }
    function branchNet(branch) {
      return branchValue(branch, "net");
    }
    return (_ctx, _push, _parent, _attrs) => {
      const _component_NuxtLink = __nuxt_component_0;
      const _component_DateRangePicker = __nuxt_component_0$1;
      const _component_SearchSelect = __nuxt_component_1;
      _push(`<div${ssrRenderAttrs(mergeProps({ class: "dashboard-page" }, _attrs))} data-v-c2230777><div class="dashboard-heading" data-v-c2230777><div data-v-c2230777><p class="eyebrow" data-v-c2230777>BUSINESS OVERVIEW</p><h1 data-v-c2230777>Dashboard</h1><p class="heading-copy" data-v-c2230777>A clear view of your income and expenses.</p></div>`);
      if (unref(data)) {
        _push(`<div class="date-summary" data-v-c2230777>`);
        _push(ssrRenderComponent(unref(CalendarDays), { size: 17 }, null, _parent));
        _push(`<span data-v-c2230777>${ssrInterpolate(formatDate(unref(data).date_from))} <span class="date-arrow" data-v-c2230777>→</span> ${ssrInterpolate(formatDate(unref(data).date_to))}</span></div>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</div>`);
      if (unref(loading)) {
        _push(`<div class="dashboard-message" data-v-c2230777><div class="loading-mark" data-v-c2230777><span data-v-c2230777></span><span data-v-c2230777></span><span data-v-c2230777></span></div><span data-v-c2230777>Loading your dashboard…</span></div>`);
      } else if (unref(err)) {
        _push(`<div class="dashboard-message error-message" data-v-c2230777><div data-v-c2230777><p class="font-semibold" data-v-c2230777>We couldn’t load your dashboard.</p><p class="mt-1 text-sm" data-v-c2230777>${ssrInterpolate(unref(err))}</p></div><button class="retry-button" data-v-c2230777>Try again</button></div>`);
      } else if (unref(data)) {
        _push(`<div class="dashboard-layout" data-v-c2230777><main class="dashboard-main" data-v-c2230777><section class="metric-grid" aria-label="Financial summary" data-v-c2230777><!--[-->`);
        ssrRenderList(unref(metrics), (metric) => {
          _push(`<article class="metric-card" data-v-c2230777><div class="metric-topline" data-v-c2230777><div class="${ssrRenderClass([metric.colorClass, "metric-icon"])}" data-v-c2230777>`);
          ssrRenderVNode(_push, createVNode(resolveDynamicComponent(metric.icon), {
            size: 21,
            "stroke-width": 2.1
          }, null), _parent);
          _push(`</div><span class="metric-period" data-v-c2230777>${ssrInterpolate(unref(periodLabel))}</span></div><p class="metric-label" data-v-c2230777>${ssrInterpolate(metric.label)}</p><p class="metric-value" data-v-c2230777>${ssrInterpolate(metric.value)}</p><p class="metric-note" data-v-c2230777>${ssrInterpolate(metric.note)}</p></article>`);
        });
        _push(`<!--]--></section><section class="chart-grid" data-v-c2230777><article class="panel trend-panel" data-v-c2230777><div class="panel-heading" data-v-c2230777><div data-v-c2230777><h2 data-v-c2230777>Income vs Expense</h2><p data-v-c2230777>Approved transactions in the selected period</p></div><div class="chart-legend" data-v-c2230777><span data-v-c2230777><i class="legend-dot income-dot" data-v-c2230777></i>Income</span><span data-v-c2230777><i class="legend-dot expense-dot" data-v-c2230777></i>Expense</span></div></div><div class="trend-chart-wrap" data-v-c2230777><svg class="trend-chart" viewBox="0 0 720 252" role="img" aria-label="Income and expense chart" data-v-c2230777><!--[-->`);
        ssrRenderList(unref(chartTicks), (tick) => {
          _push(`<g data-v-c2230777><line x1="56"${ssrRenderAttr("y1", tick.y)} x2="704"${ssrRenderAttr("y2", tick.y)} class="chart-grid-line" data-v-c2230777></line><text x="46"${ssrRenderAttr("y", tick.y + 4)} text-anchor="end" class="chart-axis-label" data-v-c2230777>${ssrInterpolate(tick.label)}</text></g>`);
        });
        _push(`<!--]--><!--[-->`);
        ssrRenderList(unref(chartBars), (bar) => {
          _push(`<g data-v-c2230777><rect${ssrRenderAttr("x", bar.incomeX)}${ssrRenderAttr("y", bar.incomeY)}${ssrRenderAttr("width", bar.width)}${ssrRenderAttr("height", bar.incomeHeight)} rx="4" class="income-bar" data-v-c2230777></rect><rect${ssrRenderAttr("x", bar.expenseX)}${ssrRenderAttr("y", bar.expenseY)}${ssrRenderAttr("width", bar.width)}${ssrRenderAttr("height", bar.expenseHeight)} rx="4" class="expense-bar" data-v-c2230777></rect><text${ssrRenderAttr("x", bar.labelX)} y="238" text-anchor="middle" class="chart-axis-label" data-v-c2230777>${ssrInterpolate(bar.label)}</text></g>`);
        });
        _push(`<!--]--></svg></div>`);
        if (!unref(chartBars).some((bar) => bar.incomeHeight || bar.expenseHeight)) {
          _push(`<div class="chart-empty" data-v-c2230777> No approved activity in this period yet. </div>`);
        } else {
          _push(`<!---->`);
        }
        _push(`</article><article class="panel category-panel" data-v-c2230777><div class="panel-heading" data-v-c2230777><div data-v-c2230777><h2 data-v-c2230777>Expense by Category</h2><p data-v-c2230777>Breakdown in ${ssrInterpolate(unref(currency))}</p></div><span class="period-pill" data-v-c2230777>${ssrInterpolate(unref(periodLabel))}</span></div><div class="category-content" data-v-c2230777><div class="donut-wrap" data-v-c2230777><div class="donut-chart" style="${ssrRenderStyle(unref(donutStyle))}" role="img" aria-label="Expenses by category" data-v-c2230777><div class="donut-hole" data-v-c2230777><span data-v-c2230777>Total expense</span><strong data-v-c2230777>${ssrInterpolate(unref(selectedExpenseDisplay))}</strong><small data-v-c2230777>${ssrInterpolate(unref(currency))}</small></div></div></div>`);
        if (unref(categorySegments).length) {
          _push(`<div class="category-legend" data-v-c2230777><!--[-->`);
          ssrRenderList(unref(categorySegments), (category) => {
            _push(`<div class="category-row" data-v-c2230777><span class="category-name" data-v-c2230777><i class="legend-dot" style="${ssrRenderStyle({ backgroundColor: category.color })}" data-v-c2230777></i>${ssrInterpolate(category.name)}</span><span class="category-total" data-v-c2230777>${ssrInterpolate(formatNumber(category.amount))}</span><span class="category-percent" data-v-c2230777>${ssrInterpolate(category.percent)}%</span></div>`);
          });
          _push(`<!--]--></div>`);
        } else {
          _push(`<div class="category-empty" data-v-c2230777>No expense categories for this period.</div>`);
        }
        _push(`</div></article></section><section class="table-grid" data-v-c2230777><article class="panel table-panel transactions-panel" data-v-c2230777><div class="panel-heading" data-v-c2230777><div data-v-c2230777><h2 data-v-c2230777>Recent Transactions</h2><p data-v-c2230777>Latest approved activity</p></div>`);
        if (unref(canViewTransactions)) {
          _push(ssrRenderComponent(_component_NuxtLink, {
            to: "/transactions",
            class: "view-link"
          }, {
            default: withCtx((_, _push2, _parent2, _scopeId) => {
              if (_push2) {
                _push2(`View all `);
                _push2(ssrRenderComponent(unref(ArrowUpRight), { size: 15 }, null, _parent2, _scopeId));
              } else {
                return [
                  createTextVNode("View all "),
                  createVNode(unref(ArrowUpRight), { size: 15 })
                ];
              }
            }),
            _: 1
          }, _parent));
        } else {
          _push(`<!---->`);
        }
        _push(`</div>`);
        if (unref(data).recent_transactions?.length) {
          _push(`<div class="table-scroll" data-v-c2230777><table class="data-table transactions-table" data-v-c2230777><thead data-v-c2230777><tr data-v-c2230777><th data-v-c2230777>Date</th><th data-v-c2230777>Type</th><th data-v-c2230777>Description</th><th data-v-c2230777>Branch</th><th class="amount-cell" data-v-c2230777>Amount</th><th data-v-c2230777>Status</th></tr></thead><tbody data-v-c2230777><!--[-->`);
          ssrRenderList(unref(data).recent_transactions.slice(0, 6), (transaction) => {
            _push(`<tr data-v-c2230777><td class="date-cell" data-v-c2230777>${ssrInterpolate(formatTransactionDate(transaction.transaction_date))}</td><td data-v-c2230777><span class="${ssrRenderClass([transaction.type === "income" ? "type-income" : "type-expense", "type-pill"])}" data-v-c2230777>${ssrInterpolate(transaction.type)}</span></td><td class="description-cell" data-v-c2230777>${ssrInterpolate(transaction.description || (transaction.type === "expense" ? transaction.category?.name : "") || transaction.transaction_number)}</td><td data-v-c2230777>${ssrInterpolate(transaction.branch?.name || transaction.branch?.code || "—")}</td><td class="amount-cell" data-v-c2230777>${ssrInterpolate(unref(formatMoney)(transaction.amount, transaction.currency))}</td><td data-v-c2230777><span class="status-pill" data-v-c2230777>`);
            _push(ssrRenderComponent(unref(Check), { size: 11 }, null, _parent));
            _push(` ${ssrInterpolate(formatStatus(transaction.status))}</span></td></tr>`);
          });
          _push(`<!--]--></tbody></table></div>`);
        } else {
          _push(`<div class="table-empty" data-v-c2230777>No approved transactions in this period.</div>`);
        }
        _push(`</article><article class="panel table-panel branch-panel" data-v-c2230777><div class="panel-heading" data-v-c2230777><div data-v-c2230777><h2 data-v-c2230777>Branch Summary</h2><p data-v-c2230777>Income, expense and net by branch</p></div>`);
        if (unref(canViewBranches)) {
          _push(ssrRenderComponent(_component_NuxtLink, {
            to: "/admin/branches",
            class: "view-link"
          }, {
            default: withCtx((_, _push2, _parent2, _scopeId) => {
              if (_push2) {
                _push2(`View all `);
                _push2(ssrRenderComponent(unref(ArrowUpRight), { size: 15 }, null, _parent2, _scopeId));
              } else {
                return [
                  createTextVNode("View all "),
                  createVNode(unref(ArrowUpRight), { size: 15 })
                ];
              }
            }),
            _: 1
          }, _parent));
        } else {
          _push(`<!---->`);
        }
        _push(`</div>`);
        if (unref(data).by_branch?.length) {
          _push(`<div class="table-scroll" data-v-c2230777><table class="data-table branch-table" data-v-c2230777><thead data-v-c2230777><tr data-v-c2230777><th data-v-c2230777>Branch</th><th class="amount-cell" data-v-c2230777>Income</th><th class="amount-cell" data-v-c2230777>Expense</th><th class="amount-cell" data-v-c2230777>Net</th></tr></thead><tbody data-v-c2230777><!--[-->`);
          ssrRenderList(unref(data).by_branch, (branch) => {
            _push(`<tr data-v-c2230777><td data-v-c2230777><span class="branch-name" data-v-c2230777>${ssrInterpolate(branch.branch_name)}</span><small data-v-c2230777>${ssrInterpolate(branch.branch_code)}</small></td><td class="amount-cell" data-v-c2230777>${ssrInterpolate(formatBranchAmount(branch, "income"))}</td><td class="amount-cell" data-v-c2230777>${ssrInterpolate(formatBranchAmount(branch, "expense"))}</td><td class="${ssrRenderClass([branchNet(branch) < 0 ? "negative-net" : "", "amount-cell net-cell"])}" data-v-c2230777>${ssrInterpolate(formatBranchAmount(branch, "net"))}</td></tr>`);
          });
          _push(`<!--]--></tbody></table></div>`);
        } else {
          _push(`<div class="table-empty" data-v-c2230777>No branch activity in this period.</div>`);
        }
        _push(`</article></section>`);
        if (unref(quickLinks).length) {
          _push(`<section class="quick-link-grid" aria-label="Quick links" data-v-c2230777><!--[-->`);
          ssrRenderList(unref(quickLinks), (link) => {
            _push(ssrRenderComponent(_component_NuxtLink, {
              key: link.label,
              to: link.to,
              class: "quick-link-card"
            }, {
              default: withCtx((_, _push2, _parent2, _scopeId) => {
                if (_push2) {
                  _push2(`<span class="quick-link-icon" data-v-c2230777${_scopeId}>`);
                  ssrRenderVNode(_push2, createVNode(resolveDynamicComponent(link.icon), { size: 20 }, null), _parent2, _scopeId);
                  _push2(`</span><span class="quick-link-copy" data-v-c2230777${_scopeId}><strong data-v-c2230777${_scopeId}>${ssrInterpolate(link.label)}</strong><small data-v-c2230777${_scopeId}>${ssrInterpolate(link.description)}</small></span>`);
                  _push2(ssrRenderComponent(unref(ArrowUpRight), {
                    class: "quick-link-arrow",
                    size: 16
                  }, null, _parent2, _scopeId));
                } else {
                  return [
                    createVNode("span", { class: "quick-link-icon" }, [
                      (openBlock(), createBlock(resolveDynamicComponent(link.icon), { size: 20 }))
                    ]),
                    createVNode("span", { class: "quick-link-copy" }, [
                      createVNode("strong", null, toDisplayString(link.label), 1),
                      createVNode("small", null, toDisplayString(link.description), 1)
                    ]),
                    createVNode(unref(ArrowUpRight), {
                      class: "quick-link-arrow",
                      size: 16
                    })
                  ];
                }
              }),
              _: 2
            }, _parent));
          });
          _push(`<!--]--></section>`);
        } else {
          _push(`<!---->`);
        }
        _push(`</main><aside class="filter-panel panel" data-v-c2230777><div class="filter-heading" data-v-c2230777><span class="filter-icon" data-v-c2230777>`);
        _push(ssrRenderComponent(unref(ListFilter), { size: 18 }, null, _parent));
        _push(`</span><div data-v-c2230777><h2 data-v-c2230777>Filter</h2><p data-v-c2230777>Update your overview</p></div></div><div class="filter-group" data-v-c2230777><label class="filter-label" data-v-c2230777>Date Range</label><div class="period-options" data-v-c2230777><!--[-->`);
        ssrRenderList(periodOptions, (option) => {
          _push(`<button class="${ssrRenderClass([unref(period) === option.value ? "selected" : "", "period-option"])}" data-v-c2230777>${ssrInterpolate(option.label)}</button>`);
        });
        _push(`<!--]--></div>`);
        _push(ssrRenderComponent(_component_DateRangePicker, {
          modelValue: unref(dateRange),
          "onUpdate:modelValue": [($event) => isRef(dateRange) ? dateRange.value = $event : null, ($event) => period.value = "custom"]
        }, null, _parent));
        _push(`</div><div class="filter-group" data-v-c2230777><label class="filter-label" data-v-c2230777>Branch</label>`);
        _push(ssrRenderComponent(_component_SearchSelect, {
          modelValue: unref(branchId),
          "onUpdate:modelValue": ($event) => isRef(branchId) ? branchId.value = $event : null,
          options: unref(branchOptions),
          placeholder: "All accessible branches",
          "aria-label": "Branch filter"
        }, null, _parent));
        _push(`</div><div class="filter-group" data-v-c2230777><label class="filter-label" data-v-c2230777>Currency</label><div class="currency-options" data-v-c2230777><button class="${ssrRenderClass(unref(currency) === "USD" ? "selected" : "")}" data-v-c2230777>USD <small data-v-c2230777>$</small></button><button class="${ssrRenderClass(unref(currency) === "KHR" ? "selected" : "")}" data-v-c2230777>KHR <small data-v-c2230777>៛</small></button></div></div>`);
        if (unref(dateValidationError)) {
          _push(`<div class="filter-error" data-v-c2230777>${ssrInterpolate(unref(dateValidationError))}</div>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<button class="apply-button"${ssrIncludeBooleanAttr(unref(loading)) ? " disabled" : ""} data-v-c2230777>`);
        _push(ssrRenderComponent(unref(ListFilter), { size: 15 }, null, _parent));
        _push(` Apply Filter</button><button class="reset-button" data-v-c2230777>Reset</button><p class="filter-footnote" data-v-c2230777>Totals include approved transactions only.</p></aside></div>`);
      } else {
        _push(`<!---->`);
      }
      _push(`</div>`);
    };
  }
});
const _sfc_setup = _sfc_main.setup;
_sfc_main.setup = (props, ctx) => {
  const ssrContext = useSSRContext();
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("pages/dashboard.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
const dashboard = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-c2230777"]]);
export {
  dashboard as default
};
//# sourceMappingURL=dashboard-BqJ4RJYO.js.map
