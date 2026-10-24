<template>
  <AppModal
    :model-value="modelValue"
    form
    :title="branch ? `Telegram · ${branch.code}` : 'Telegram'"
    eyebrow="BRANCH TELEGRAM"
    subtitle="This branch's transaction updates go to its own Telegram chat."
    :busy="saving || testing"
    @update:model-value="(open) => { if (!open) close() }"
    @submit="save"
  >
    <template #icon><Send :size="20" /></template>
    <div v-if="loading" class="py-6 text-center text-sm text-muted">Loading…</div>
    <div v-else class="tg-form">
      <label class="switch-row">
        <span>
          <strong>Send updates to Telegram</strong>
          <small>{{ settings.ready ? 'On: updates go to the chat below.' : 'Set the chat (and a bot) below, then turn this on.' }}</small>
        </span>
        <input v-model="form.enabled" type="checkbox" role="switch" class="switch" />
      </label>

      <label class="field">
        <span>Chat ID <small>the branch's Telegram group</small></span>
        <input v-model.trim="form.chat_id" type="text" autocomplete="off" placeholder="-1001234567890" :aria-invalid="formCheck.invalid('chat_id')" />
        <em>Add the bot to the group, then use the group's chat ID (or a public @channel name).</em>
      </label>

      <label class="field">
        <span>Bot token <small>{{ tokenHint }}</small></span>
        <input v-model.trim="form.bot_token" type="password" autocomplete="new-password" :placeholder="settings.own_bot && !form.clear_bot_token ? '••••••••••' : '123456789:AA…'" :aria-invalid="formCheck.invalid('bot_token')" />
        <button v-if="settings.own_bot && !form.clear_bot_token" type="button" class="link-button" @click="form.clear_bot_token = true; form.bot_token = ''">
          {{ settings.default_bot_ready ? 'Use the default bot instead' : 'Remove this bot token' }}
        </button>
        <em v-else-if="form.clear_bot_token">The saved token is removed when you save.</em>
      </label>

      <fieldset class="events" :aria-invalid="formCheck.invalid('events')">
        <legend>Send these updates</legend>
        <label v-for="event in eventOptions" :key="event.value">
          <input v-model="form.events" type="checkbox" :value="event.value" />
          <span><strong>{{ event.label }}</strong><small>{{ event.hint }}</small></span>
        </label>
      </fieldset>

      <AppAlert v-if="formError || formCheck.messages.length" compact :title="formError ? 'Couldn’t save' : 'Check the highlighted fields'" :message="formError" :items="formCheck.messages" dismissible @dismiss="formError = ''; formCheck.reset()" />
    </div>
    <template #footer>
      <button type="button" class="border border-border bg-white text-muted" :disabled="saving || testing" @click="close">Cancel</button>
      <button type="button" class="border border-border bg-white text-primary disabled:opacity-60" :disabled="loading || saving || testing || !canTest" :title="canTest ? 'Send a test message with the saved settings' : 'Save a chat ID and a bot first'" @click="sendTest">{{ testing ? 'Sending…' : 'Send test' }}</button>
      <button type="submit" class="bg-primary text-white disabled:opacity-60" :disabled="loading || saving || testing">{{ saving ? 'Saving…' : 'Save' }}</button>
    </template>
  </AppModal>
</template>

<script setup lang="ts">
import { Send } from 'lucide-vue-next'
import type { Branch, BranchTelegramSettings } from '~/types/api'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'

const props = defineProps<{ modelValue: boolean; branch: Branch | null }>()
const emit = defineEmits<{ 'update:modelValue': [open: boolean]; saved: [settings: BranchTelegramSettings] }>()

const { api } = useApi()
const toast = useToast()

const eventOptions = [
  { value: 'created', label: 'New income or expense', hint: 'Including expenses waiting for approval' },
  { value: 'approved', label: 'Approved', hint: 'With who approved it and the note' },
  { value: 'rejected', label: 'Rejected', hint: 'With who rejected it and the reason' },
  { value: 'voided', label: 'Voided', hint: 'With the reason' },
  { value: 'reversed', label: 'Reversed', hint: 'The reversal entry and the reason' },
]

const emptySettings: BranchTelegramSettings = { enabled: false, chat_id: '', events: [], own_bot: false, default_bot_ready: false, ready: false }
const settings = ref<BranchTelegramSettings>({ ...emptySettings })
const form = reactive({ enabled: false, chat_id: '', bot_token: '', clear_bot_token: false, events: [] as string[] })
const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const formError = ref('')

// Same rules as the API (services.BranchTelegramService.Save).
const CHAT_RULE = /^(-?\d{5,20}|@[A-Za-z][A-Za-z0-9_]{4,31})$/
const TOKEN_RULE = /^\d{5,12}:[A-Za-z0-9_-]{30,64}$/
const hasBot = computed(() => !!form.bot_token || (settings.value.own_bot && !form.clear_bot_token) || settings.value.default_bot_ready)
const formCheck = reactive(useFormErrors(
  () => ({ ...form }),
  () => ({
    chat_id: [(value: unknown) => {
      const chat = String(value || '')
      if (!chat) return form.enabled ? 'enter the chat ID of the branch’s Telegram group' : ''
      return CHAT_RULE.test(chat) ? '' : 'enter the chat ID (like -1001234567890) or a public @channel name'
    }],
    bot_token: [(value: unknown) => {
      const token = String(value || '')
      if (token) return TOKEN_RULE.test(token) ? '' : 'enter the bot token from @BotFather (like 123456789:AA…)'
      return form.enabled && !hasBot.value ? 'add a bot token; no default bot is set up on the Backups page' : ''
    }],
    events: [(value: unknown) => form.enabled && !(value as string[]).length ? 'choose at least one event to send' : ''],
  }),
))

const tokenHint = computed(() => {
  if (settings.value.own_bot && !form.clear_bot_token) return 'saved for this branch · leave empty to keep it'
  if (settings.value.default_bot_ready) return 'optional · empty uses the default bot from Backups'
  return 'from @BotFather'
})
// A test uses the saved settings.
const canTest = computed(() => !!settings.value.chat_id && (settings.value.own_bot || settings.value.default_bot_ready))

function apply(next: BranchTelegramSettings) {
  settings.value = { ...emptySettings, ...next }
  Object.assign(form, { enabled: next.enabled, chat_id: next.chat_id, bot_token: '', clear_bot_token: false, events: [...(next.events || [])] })
}

watch(() => [props.modelValue, props.branch?.id], async ([open]) => {
  if (!open || !props.branch) return
  formError.value = ''
  formCheck.reset()
  loading.value = true
  try {
    const result = await api<{ settings: BranchTelegramSettings }>(`/branches/${props.branch.id}/telegram`)
    apply(result.settings)
  } catch (e: any) {
    apply({ ...emptySettings })
    formError.value = e?.data?.error || 'Could not load the Telegram settings.'
  } finally {
    loading.value = false
  }
}, { immediate: true })

function close() {
  if (!saving.value && !testing.value) emit('update:modelValue', false)
}

async function save() {
  if (!props.branch) return
  formError.value = ''
  if (!formCheck.check()) return
  saving.value = true
  try {
    const result = await api<{ settings: BranchTelegramSettings }>(`/branches/${props.branch.id}/telegram`, {
      method: 'PUT',
      body: {
        enabled: form.enabled,
        chat_id: form.chat_id,
        events: form.events,
        ...(form.bot_token ? { bot_token: form.bot_token } : {}),
        ...(form.clear_bot_token && !form.bot_token ? { clear_bot_token: true } : {}),
      },
    })
    apply(result.settings)
    emit('saved', result.settings)
    toast.success(result.settings.ready ? `${props.branch.code} updates go to Telegram.` : `${props.branch.code} won’t send updates to Telegram.`, { title: 'Telegram saved' })
  } catch (e: any) {
    if (!formCheck.fromApi(e)) formError.value = e?.data?.error || 'Could not save the Telegram settings.'
  } finally {
    saving.value = false
  }
}

async function sendTest() {
  if (!props.branch) return
  testing.value = true
  formError.value = ''
  try {
    await api(`/branches/${props.branch.id}/telegram/test`, { method: 'POST' })
    toast.success('Check the branch’s Telegram chat.', { title: 'Test message sent' })
  } catch (e: any) {
    formError.value = e?.data?.error || 'Could not send the test message.'
  } finally {
    testing.value = false
  }
}
</script>

<style scoped>
.tg-form { display: flex; flex-direction: column; gap: 14px; }
.switch-row { display: flex; align-items: center; justify-content: space-between; gap: 14px; border: 1px solid #e5ebf4; border-radius: 12px; padding: 11px 13px; background: #f8fafd; cursor: pointer; }
.switch-row > span { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.switch-row strong { color: #2c4468; font-size: 13px; font-weight: 700; }
.switch-row small { color: #7b8ca5; font-size: 12px; }
.switch { position: relative; flex: 0 0 auto; width: 42px; height: 24px; margin: 0; border-radius: 999px; background: #cfd9e6; cursor: pointer; appearance: none; transition: background .15s ease; }
.switch::before { content: ''; position: absolute; top: 3px; left: 3px; width: 18px; height: 18px; border-radius: 50%; background: #fff; box-shadow: 0 1px 3px rgb(16 40 80 / 25%); transition: transform .15s ease; }
.switch:checked { background: #1769e8; }
.switch:checked::before { transform: translateX(18px); }
.field { display: flex; min-width: 0; flex-direction: column; gap: 6px; color: #324b70; font-size: 13px; font-weight: 650; }
.field small { margin-left: 4px; color: #94a2b7; font-size: 11px; font-weight: 500; }
.field em { color: #8b9bb1; font-size: 11.5px; font-style: normal; font-weight: 450; }
.field input { width: 100%; min-width: 0; min-height: 42px; border: 1px solid #dfe7f1; border-radius: 10px; outline: none; padding: 9px 12px; color: #263f65; background: #fff; font: inherit; font-size: 14px; font-weight: 450; }
.field input:focus { border-color: #85b4f7; box-shadow: 0 0 0 3px rgb(37 99 235 / 10%); }
.field input[aria-invalid='true'] { border-color: #f1a7a7; }
.link-button { align-self: flex-start; border: 0; padding: 0; color: #1769e8; background: none; font-size: 12px; font-weight: 650; cursor: pointer; }
.events { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 8px; margin: 0; border: 0; padding: 0; }
.events legend { margin-bottom: 8px; color: #324b70; font-size: 13px; font-weight: 650; }
.events label { display: flex; align-items: flex-start; gap: 9px; border: 1px solid #e5ebf4; border-radius: 10px; padding: 9px 11px; background: #fff; cursor: pointer; }
.events input { margin-top: 2px; accent-color: #1769e8; }
.events span { display: flex; min-width: 0; flex-direction: column; gap: 2px; }
.events strong { color: #2c4468; font-size: 12.5px; font-weight: 650; }
.events small { color: #8b9bb1; font-size: 11px; }
.events[aria-invalid='true'] label { border-color: #f6c8c8; }
@media (max-width: 520px) { .events { grid-template-columns: 1fr; } }
</style>
