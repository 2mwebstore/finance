import { defineComponent, ref, computed, watch, mergeProps, unref, useSSRContext } from 'vue';
import { ssrRenderAttrs, ssrRenderAttr, ssrRenderComponent, ssrInterpolate, ssrRenderList } from 'vue/server-renderer';
import { onClickOutside } from '@vueuse/core';
import { CalendarDays, ChevronDown } from 'lucide-vue-next';
import { _ as _export_sfc, V as VDateRangePicker } from './server.mjs';

function dateKey(date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
}
function currentMonthRange() {
  const now = /* @__PURE__ */ new Date();
  return {
    from: dateKey(new Date(now.getFullYear(), now.getMonth(), 1)),
    to: dateKey(new Date(now.getFullYear(), now.getMonth() + 1, 0))
  };
}
function formatDateKey(value) {
  if (!value) return "";
  const parsed = /* @__PURE__ */ new Date(`${value}T00:00:00`);
  return Number.isNaN(parsed.valueOf()) ? value : new Intl.DateTimeFormat("en-US", { month: "short", day: "numeric", year: "numeric" }).format(parsed);
}
const _sfc_main = /* @__PURE__ */ defineComponent({
  __name: "DateRangePicker",
  __ssrInlineRender: true,
  props: {
    modelValue: {}
  },
  emits: ["update:modelValue"],
  setup(__props, { emit: __emit }) {
    const props = __props;
    const emit = __emit;
    const root = ref(null);
    const open = ref(false);
    const draftDates = ref([]);
    const selectedKeys = computed(() => draftDates.value.map(dateKey2).filter(Boolean));
    const invalidRange = computed(() => selectedKeys.value.length > 1 && selectedKeys.value.at(-1) < selectedKeys.value[0]);
    const rangeLabel = computed(() => props.modelValue.from && props.modelValue.to ? `${formatDateKey(props.modelValue.from)} \u2013 ${formatDateKey(props.modelValue.to)}` : "Select date range");
    const presets = [
      { label: "This month", getRange: () => currentMonthRange() },
      { label: "Last month", getRange: () => monthRange(-1) },
      { label: "Last 30 days", getRange: () => recentRange(30) },
      { label: "Today", getRange: () => recentRange(1) }
    ];
    function dateKey2(value) {
      if (typeof value === "string" && /^\d{4}-\d{2}-\d{2}/.test(value)) return value.slice(0, 10);
      const date = value instanceof Date ? value : new Date(value);
      if (Number.isNaN(date.valueOf())) return "";
      return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
    }
    function pickerDates(range) {
      if (!range.from || !range.to) return [];
      return [range.from, range.to].map((value) => /* @__PURE__ */ new Date(`${value}T12:00:00`));
    }
    function monthRange(offset) {
      const now = /* @__PURE__ */ new Date();
      const first = new Date(now.getFullYear(), now.getMonth() + offset, 1);
      const last = new Date(first.getFullYear(), first.getMonth() + 1, 0);
      return { from: dateKey2(first), to: dateKey2(last) };
    }
    function recentRange(days) {
      const end = /* @__PURE__ */ new Date();
      const start = new Date(end.getFullYear(), end.getMonth(), end.getDate() - days + 1);
      return { from: dateKey2(start), to: dateKey2(end) };
    }
    function onPickerUpdate(value) {
      draftDates.value = value;
      if (value.length >= 2) applyDraft();
    }
    function applyDraft() {
      if (invalidRange.value || selectedKeys.value.length < 2) return;
      emit("update:modelValue", { from: selectedKeys.value[0], to: selectedKeys.value.at(-1) });
      open.value = false;
    }
    watch(() => props.modelValue, (range) => {
      draftDates.value = pickerDates(range);
    }, { deep: true });
    onClickOutside(root, () => {
      open.value = false;
    });
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<div${ssrRenderAttrs(mergeProps({
        ref_key: "root",
        ref: root,
        class: ["date-range-picker", { open: unref(open) }]
      }, _attrs))} data-v-f94fbffc><button type="button" class="range-trigger"${ssrRenderAttr("aria-expanded", unref(open))} aria-haspopup="dialog" data-v-f94fbffc>`);
      _push(ssrRenderComponent(unref(CalendarDays), { size: 15 }, null, _parent));
      _push(`<span data-v-f94fbffc>${ssrInterpolate(unref(rangeLabel))}</span>`);
      _push(ssrRenderComponent(unref(ChevronDown), {
        size: 14,
        class: "range-chevron"
      }, null, _parent));
      _push(`</button>`);
      if (unref(open)) {
        _push(`<div class="range-popover" role="dialog" aria-label="Choose date range" data-v-f94fbffc><div class="range-presets" data-v-f94fbffc><!--[-->`);
        ssrRenderList(presets, (preset) => {
          _push(`<button type="button" data-v-f94fbffc>${ssrInterpolate(preset.label)}</button>`);
        });
        _push(`<!--]--></div><div class="range-picker-scroll" data-v-f94fbffc>`);
        _push(ssrRenderComponent(unref(VDateRangePicker), {
          "model-value": unref(draftDates),
          color: "primary",
          width: 690,
          "hide-header": "",
          "onUpdate:modelValue": onPickerUpdate
        }, null, _parent));
        _push(`</div>`);
        if (unref(invalidRange)) {
          _push(`<p class="range-error" data-v-f94fbffc>End date must be on or after the start date.</p>`);
        } else {
          _push(`<!---->`);
        }
        _push(`<div class="range-actions" data-v-f94fbffc><button type="button" class="range-clear" data-v-f94fbffc>Clear</button><span class="range-hint" data-v-f94fbffc>Choose a start and end date</span><button type="button" class="range-cancel" data-v-f94fbffc>Close</button></div></div>`);
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
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("components/DateRangePicker.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
const __nuxt_component_0 = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-f94fbffc"]]);

export { __nuxt_component_0 as _, currentMonthRange as c };
//# sourceMappingURL=DateRangePicker-BmLhEXsC.mjs.map
