import { defineComponent, ref, computed, watch, nextTick, mergeProps, unref, useSSRContext } from "vue";
import { ssrRenderAttrs, ssrRenderAttr, ssrIncludeBooleanAttr, ssrRenderClass, ssrInterpolate, ssrRenderComponent, ssrRenderList } from "vue/server-renderer";
import { onClickOutside } from "@vueuse/core";
import { ChevronDown, Search, Check } from "lucide-vue-next";
import { _ as _export_sfc } from "../server.mjs";
const _sfc_main = /* @__PURE__ */ defineComponent({
  __name: "SearchSelect",
  __ssrInlineRender: true,
  props: {
    modelValue: {},
    options: {},
    multiple: { type: Boolean, default: false },
    placeholder: { default: "Select an option" },
    searchPlaceholder: { default: "Search…" },
    emptyText: { default: "No matching options" },
    ariaLabel: { default: "" },
    disabled: { type: Boolean, default: false }
  },
  emits: ["update:modelValue"],
  setup(__props, { emit: __emit }) {
    const props = __props;
    const root = ref(null);
    const searchInput = ref(null);
    const open = ref(false);
    const search = ref("");
    const selectedValues = computed(() => Array.isArray(props.modelValue) ? props.modelValue : props.modelValue ? [props.modelValue] : []);
    const selectedOption = computed(() => props.options.find((option) => option.value === props.modelValue));
    const hasSelection = computed(() => props.multiple ? selectedValues.value.length > 0 : !!selectedOption.value);
    const selectedLabel = computed(() => {
      if (props.multiple) {
        const labels = props.options.filter((option) => selectedValues.value.includes(option.value)).map((option) => option.label);
        return labels.length ? labels.join(", ") : props.placeholder;
      }
      return selectedOption.value?.label || props.placeholder;
    });
    const filteredOptions = computed(() => {
      const query = search.value.trim().toLocaleLowerCase();
      if (!query) return props.options;
      return props.options.filter((option) => option.label.toLocaleLowerCase().includes(query));
    });
    function isSelected(option) {
      if (!props.multiple) return props.modelValue === option.value;
      return selectedValues.value.includes(option.value);
    }
    watch(open, async (isOpen) => {
      if (isOpen) {
        await nextTick();
        searchInput.value?.focus();
      }
    });
    onClickOutside(root, () => {
      open.value = false;
    });
    return (_ctx, _push, _parent, _attrs) => {
      _push(`<div${ssrRenderAttrs(mergeProps({
        ref_key: "root",
        ref: root,
        class: ["search-select", { "is-open": unref(open) }]
      }, _attrs))} data-v-a11aafa2><button type="button" class="search-select-trigger" role="combobox"${ssrRenderAttr("aria-expanded", unref(open))}${ssrRenderAttr("aria-label", __props.ariaLabel || __props.placeholder)}${ssrIncludeBooleanAttr(__props.disabled) ? " disabled" : ""} data-v-a11aafa2><span class="${ssrRenderClass(unref(hasSelection) ? "selected-label" : "placeholder-label")}" data-v-a11aafa2>${ssrInterpolate(unref(selectedLabel))}</span>`);
      _push(ssrRenderComponent(unref(ChevronDown), { size: 15 }, null, _parent));
      _push(`</button>`);
      if (unref(open)) {
        _push(`<div class="search-select-menu" data-v-a11aafa2><div class="search-input-wrap" data-v-a11aafa2>`);
        _push(ssrRenderComponent(unref(Search), { size: 14 }, null, _parent));
        _push(`<input${ssrRenderAttr("value", unref(search))} type="search"${ssrRenderAttr("placeholder", __props.searchPlaceholder)} data-v-a11aafa2></div><div class="search-select-options" role="listbox"${ssrRenderAttr("aria-multiselectable", __props.multiple)} data-v-a11aafa2><!--[-->`);
        ssrRenderList(unref(filteredOptions), (option) => {
          _push(`<button type="button" role="option"${ssrRenderAttr("aria-selected", isSelected(option))}${ssrIncludeBooleanAttr(option.disabled) ? " disabled" : ""} class="${ssrRenderClass([{ selected: isSelected(option) }, "search-select-option"])}" data-v-a11aafa2><span data-v-a11aafa2>${ssrInterpolate(option.label)}</span>`);
          if (isSelected(option)) {
            _push(ssrRenderComponent(unref(Check), { size: 14 }, null, _parent));
          } else {
            _push(`<!---->`);
          }
          _push(`</button>`);
        });
        _push(`<!--]-->`);
        if (!unref(filteredOptions).length) {
          _push(`<p class="search-select-empty" data-v-a11aafa2>${ssrInterpolate(__props.emptyText)}</p>`);
        } else {
          _push(`<!---->`);
        }
        _push(`</div></div>`);
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
  (ssrContext.modules || (ssrContext.modules = /* @__PURE__ */ new Set())).add("components/SearchSelect.vue");
  return _sfc_setup ? _sfc_setup(props, ctx) : void 0;
};
const __nuxt_component_1 = /* @__PURE__ */ _export_sfc(_sfc_main, [["__scopeId", "data-v-a11aafa2"]]);
export {
  __nuxt_component_1 as _
};
//# sourceMappingURL=SearchSelect-BFyTiPOW.js.map
