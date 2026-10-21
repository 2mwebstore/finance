<template>
  <div class="rich-text" :class="{ 'is-focused': focused, 'is-disabled': disabled, 'is-over': textLength > maxLength }">
    <div class="rt-toolbar" role="toolbar" :aria-label="`${ariaLabel || 'Text'} formatting`">
      <button
        v-for="tool in tools"
        :key="tool.command"
        type="button"
        class="rt-tool"
        :class="{ 'is-on': active[tool.command] }"
        :aria-pressed="!!active[tool.command]"
        :title="tool.label"
        :aria-label="tool.label"
        :disabled="disabled"
        @mousedown.prevent
        @click="run(tool.command)"
      >
        <component :is="tool.icon" :size="16" :stroke-width="2.2" />
      </button>
      <span class="rt-separator" aria-hidden="true"></span>
      <button type="button" class="rt-tool" title="Clear formatting" aria-label="Clear formatting" :disabled="disabled" @mousedown.prevent @click="run('removeFormat')">
        <RemoveFormatting :size="16" />
      </button>
      <span class="rt-count" aria-live="polite">{{ textLength }}/{{ maxLength }}</span>
    </div>
    <div
      ref="editor"
      class="rt-editor rich-text-content"
      :contenteditable="!disabled"
      role="textbox"
      aria-multiline="true"
      :aria-label="ariaLabel"
      :data-placeholder="placeholder"
      @input="onInput"
      @focus="onFocus"
      @blur="onBlur"
      @paste="onPaste"
      @keyup="refreshActive"
      @mouseup="refreshActive"
    ></div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import { Bold, Italic, List, ListOrdered, RemoveFormatting, Underline } from 'lucide-vue-next'
import { isRichTextEmpty, richTextToPlain, sanitizeRichText } from '~/utils/richText'

const props = withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  ariaLabel?: string
  maxLength?: number
  disabled?: boolean
}>(), {
  placeholder: '',
  ariaLabel: '',
  maxLength: 1000,
  disabled: false,
})
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const tools: { command: string; label: string; icon: Component }[] = [
  { command: 'bold', label: 'Bold (Ctrl+B)', icon: Bold },
  { command: 'italic', label: 'Italic (Ctrl+I)', icon: Italic },
  { command: 'underline', label: 'Underline (Ctrl+U)', icon: Underline },
  { command: 'insertUnorderedList', label: 'Bulleted list', icon: List },
  { command: 'insertOrderedList', label: 'Numbered list', icon: ListOrdered },
]

const editor = ref<HTMLElement | null>(null)
const focused = ref(false)
const active = reactive<Record<string, boolean>>({})
const textLength = computed(() => richTextToPlain(props.modelValue).length)
// The last value we emitted, so our own updates don't rewrite the editor (and move the caret).
let lastEmitted: string | null = null

function render(value: string) {
  if (editor.value) editor.value.innerHTML = sanitizeRichText(value)
}

function emitValue() {
  if (!editor.value) return
  let html = editor.value.innerHTML
  if (isRichTextEmpty(html) && !/<li/i.test(html)) {
    // Browsers leave a stray <br> behind; clear it so the placeholder shows again.
    editor.value.innerHTML = ''
    html = ''
  }
  const value = isRichTextEmpty(html) ? '' : sanitizeRichText(html)
  lastEmitted = value
  emit('update:modelValue', value)
}

function onInput() {
  emitValue()
  refreshActive()
}

function run(command: string) {
  if (props.disabled) return
  editor.value?.focus()
  // Use <b>/<i>/<u> tags rather than inline styles (styles would be stripped on save).
  document.execCommand('styleWithCSS', false, 'false')
  document.execCommand(command, false)
  emitValue()
  refreshActive()
}

function refreshActive() {
  for (const tool of tools) {
    try {
      active[tool.command] = document.queryCommandState(tool.command)
    } catch {
      active[tool.command] = false
    }
  }
}

function onFocus() {
  focused.value = true
  refreshActive()
}

function onBlur() {
  focused.value = false
  for (const tool of tools) active[tool.command] = false
}

// Paste as plain text so outside formatting, images and scripts never come along.
function onPaste(event: ClipboardEvent) {
  event.preventDefault()
  const text = event.clipboardData?.getData('text/plain') || ''
  document.execCommand('insertText', false, text)
}

watch(() => props.modelValue, (value) => {
  if (value === lastEmitted) return
  lastEmitted = value
  render(value)
})

onMounted(() => {
  lastEmitted = props.modelValue
  render(props.modelValue)
})
</script>

<style scoped>
.rich-text { overflow: hidden; border: 1px solid #dfe7f1; border-radius: 10px; background: #fff; transition: border-color .15s ease, box-shadow .15s ease; }
.rich-text.is-focused { border-color: #85b4f7; box-shadow: 0 0 0 3px rgb(37 99 235 / 10%); }
.rich-text.is-over { border-color: #f0a3ae; }
.rich-text.is-disabled { background: #f6f8fb; }
.rt-toolbar { display: flex; align-items: center; gap: 2px; border-bottom: 1px solid #edf1f7; padding: 5px 6px; background: #f9fbfd; }
.rt-tool { display: grid; width: 32px; height: 30px; place-items: center; border: 0; border-radius: 7px; color: #50668a; background: transparent; cursor: pointer; transition: background .12s ease, color .12s ease; }
.rt-tool:hover:not(:disabled) { color: #1b3156; background: #eef3fa; }
.rt-tool.is-on { color: #1769e8; background: #e5efff; }
.rt-tool:disabled { opacity: .5; cursor: not-allowed; }
.rt-separator { width: 1px; height: 18px; margin: 0 4px; background: #dfe7f1; }
.rt-count { margin-left: auto; padding-right: 4px; color: #93a2b8; font-size: 11px; font-variant-numeric: tabular-nums; }
.rich-text.is-over .rt-count { color: #c93449; font-weight: 700; }
/* Normal weight on purpose: inheriting a bold label weight makes the browser treat text as already bold. */
.rt-editor { min-height: 92px; max-height: 240px; overflow-y: auto; padding: 10px 12px; color: #263f65; font-size: 14px; font-weight: 400; line-height: 1.55; outline: none; word-break: break-word; }
.rt-editor:empty::before { content: attr(data-placeholder); color: #9aa8bc; pointer-events: none; }
@media (max-width: 540px) {
  .rt-tool { width: 36px; height: 34px; }
  .rt-editor { min-height: 100px; font-size: 15px; }
}
</style>
