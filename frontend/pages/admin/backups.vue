<template>
  <section class="backups-page">
    <header class="page-heading">
      <div>
        <p class="eyebrow">SYSTEM</p>
        <h1>Backups</h1>
        <p class="heading-copy">Backups of the database and attachment files, kept on the server for {{ settings.keep_days }} days and sent to Telegram.</p>
      </div>
      <button type="button" class="primary-button" :disabled="running || loading" @click="backupNow">
        <DatabaseBackup :size="16" /> {{ running ? 'Backing up…' : 'Backup now' }}
      </button>
    </header>

    <AppAlert v-if="loadError" title="Couldn’t load backups" :message="loadError"><template #action><button type="button" @click="load">Try again</button></template></AppAlert>

    <div class="settings-grid">
      <!-- Schedule -->
      <form class="card" novalidate @submit.prevent="saveSchedule">
        <div class="card-heading">
          <span class="card-icon"><CalendarClock :size="18" /></span>
          <div><h2>Daily backup</h2><p>Runs every day at the time you choose ({{ settings.timezone }}).</p></div>
        </div>
        <label class="switch-row">
          <span><strong>Back up every day</strong><small>{{ schedule.enabled ? nextRunText : 'Off: only “Backup now” makes backups.' }}</small></span>
          <input v-model="schedule.enabled" type="checkbox" role="switch" class="switch" />
        </label>
        <div class="field-row">
          <label class="field">
            <span>Time</span>
            <input v-model="schedule.time" type="time" step="60" :disabled="!schedule.enabled" :aria-invalid="scheduleCheck.invalid('time')" />
          </label>
          <label class="field">
            <span>Keep backups for</span>
            <div class="input-suffix">
              <input v-model.number="schedule.keep_days" type="number" min="1" max="90" inputmode="numeric" :aria-invalid="scheduleCheck.invalid('keep_days')" />
              <em>days</em>
            </div>
          </label>
        </div>
        <AppAlert v-if="scheduleError || scheduleCheck.messages.length" compact :message="scheduleError" :items="scheduleCheck.messages" />
        <div class="card-actions">
          <button type="submit" class="primary-button" :disabled="savingSchedule">{{ savingSchedule ? 'Saving…' : 'Save schedule' }}</button>
        </div>
      </form>

      <!-- Telegram -->
      <form class="card" novalidate @submit.prevent="saveTelegram">
        <div class="card-heading">
          <span class="card-icon is-telegram"><Send :size="18" /></span>
          <div><h2>Telegram</h2><p>Each backup file is sent to this chat by your bot.</p></div>
        </div>
        <label class="switch-row">
          <span><strong>Send backups to Telegram</strong><small>{{ settings.telegram_ready ? 'Bot and chat are set.' : 'Set the bot token and chat ID below.' }}</small></span>
          <input v-model="telegram.enabled" type="checkbox" role="switch" class="switch" />
        </label>
        <label class="field">
          <span>Chat ID <small>e.g. -1001234567890 for a group</small></span>
          <input v-model.trim="telegram.chat_id" type="text" inputmode="numeric" autocomplete="off" :aria-invalid="telegramCheck.invalid('telegram_chat_id')" />
        </label>
        <label class="field">
          <span>Bot token <small>{{ settings.telegram_token_set ? (settings.telegram_from_env ? 'set on the server (environment)' : 'saved · leave empty to keep it') : 'from @BotFather' }}</small></span>
          <input v-model.trim="telegram.bot_token" type="password" autocomplete="new-password" :placeholder="settings.telegram_token_set ? '••••••••••' : '123456:ABC-DEF…'" />
        </label>
        <AppAlert v-if="telegramError || telegramCheck.messages.length" compact :message="telegramError" :items="telegramCheck.messages" />
        <div class="card-actions">
          <button v-if="settings.telegram_token_set && !settings.telegram_from_env" type="button" class="ghost-button" :disabled="savingTelegram" @click="clearToken">Remove token</button>
          <button type="button" class="ghost-button" :disabled="testing || !settings.telegram_ready" @click="testTelegram">{{ testing ? 'Sending…' : 'Send test message' }}</button>
          <button type="submit" class="primary-button" :disabled="savingTelegram">{{ savingTelegram ? 'Saving…' : 'Save Telegram' }}</button>
        </div>
      </form>
    </div>

    <!-- Backups on the server -->
    <section class="card list-card">
      <div class="card-heading">
        <span class="card-icon"><Archive :size="18" /></span>
        <div><h2>Backups on the server</h2><p>Older than {{ settings.keep_days }} days are deleted automatically.</p></div>
        <label class="upload-button" :class="{ 'is-busy': uploading }">
          <Upload :size="15" /> {{ uploading ? 'Uploading…' : 'Upload backup' }}
          <input type="file" accept=".gz,.sql,.tar.gz,application/gzip,application/sql" class="sr-only" :disabled="uploading" @change="onUpload" />
        </label>
      </div>
      <AppAlert v-if="uploadError" class="list-alert" compact title="Couldn’t upload the backup" :message="uploadError" dismissible @dismiss="uploadError = ''" />
      <div v-if="loading" class="list-state">Loading backups…</div>
      <div v-else-if="!items.length" class="list-state">No backups yet. Press “Backup now”, or wait for the daily backup.</div>
      <ul v-else class="backup-list">
        <li v-for="item in items" :key="item.name" class="backup-row">
          <span class="backup-icon" :class="`is-${item.trigger}`"><FileArchive :size="18" /></span>
          <div class="backup-info">
            <strong>{{ formatDateTime(item.created_at) }}</strong>
            <span class="backup-meta">
              <em class="tag" :class="`trigger-${item.trigger}`">{{ triggerLabel(item.trigger) }}</em>
              <span>{{ formatBytes(item.size) }}</span>
              <span class="contents" :title="item.missing_attachments ? `${item.missing_attachments} attachment file(s) were missing from storage and could not be included` : ''">{{ contentsLabel(item) }}</span>
              <span v-if="item.created_by && item.trigger !== 'scheduled'">by {{ item.created_by }}</span>
              <span class="telegram-status" :class="`tg-${item.telegram}`" :title="item.telegram_error || ''">
                <Send :size="11" /> {{ telegramLabel(item) }}
              </span>
            </span>
          </div>
          <div class="backup-actions">
            <button type="button" class="icon-action" title="Download" :aria-label="`Download ${item.name}`" @click="download(item)"><Download :size="15" /></button>
            <button v-if="settings.telegram_ready" type="button" class="icon-action" title="Send to Telegram" :aria-label="`Send ${item.name} to Telegram`" :disabled="resending === item.name" @click="resend(item)"><Send :size="15" /></button>
            <button type="button" class="icon-action is-restore" title="Restore this backup" :aria-label="`Restore ${item.name}`" @click="openRestore(item)"><RotateCcw :size="15" /></button>
            <button type="button" class="icon-action is-delete" title="Delete" :aria-label="`Delete ${item.name}`" @click="deleteTarget = item"><Trash2 :size="15" /></button>
          </div>
        </li>
      </ul>
    </section>

    <!-- Restore -->
    <AppModal :model-value="!!restoreTarget" form title="Restore this backup?" eyebrow="RESTORE DATABASE" :busy="restoring" width="500px" @update:model-value="(open) => { if (!open) restoreTarget = null }" @submit="restore">
      <template #icon><RotateCcw :size="20" /></template>
      <div class="restore-copy">
        <p>Everything in the database is replaced with the backup from <strong>{{ restoreTarget ? formatDateTime(restoreTarget.created_at) : '' }}</strong>. Changes made after it are lost.</p>
        <p v-if="restoreTarget?.full">Its {{ restoreTarget.attachments === 1 ? '1 attachment file is' : `${restoreTarget.attachments} attachment files are` }} put back too.</p>
        <p v-else>This backup holds the database only; attachment files on the server stay as they are.</p>
        <p>A safety copy of the current database and files is made first, so you can go back. Everyone may have to sign in again.</p>
        <label class="field">
          <span>Type <b>RESTORE</b> to confirm</span>
          <input v-model.trim="restoreConfirm" type="text" autocomplete="off" autocapitalize="characters" :aria-invalid="restoreError ? 'true' : undefined" />
        </label>
        <AppAlert v-if="restoreError" compact :message="restoreError" />
      </div>
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="restoring" @click="restoreTarget = null">Cancel</button>
        <button type="submit" class="bg-expense text-white disabled:opacity-60" :disabled="restoring || restoreConfirm !== 'RESTORE'">{{ restoring ? 'Restoring…' : 'Restore database' }}</button>
      </template>
    </AppModal>

    <!-- Delete -->
    <AppModal :model-value="!!deleteTarget" title="Delete this backup?" eyebrow="DELETE BACKUP" :busy="deleting" width="440px" @update:model-value="(open) => { if (!open) deleteTarget = null }">
      <p class="restore-copy">The backup from <strong>{{ deleteTarget ? formatDateTime(deleteTarget.created_at) : '' }}</strong> is removed from the server. A copy already sent to Telegram stays there.</p>
      <template #footer>
        <button type="button" class="border border-border bg-white text-muted" :disabled="deleting" @click="deleteTarget = null">Cancel</button>
        <button type="button" class="bg-expense text-white disabled:opacity-60" :disabled="deleting" @click="removeBackup">{{ deleting ? 'Deleting…' : 'Delete backup' }}</button>
      </template>
    </AppModal>
  </section>
</template>

<script setup lang="ts">
import { Archive, CalendarClock, DatabaseBackup, Download, FileArchive, RotateCcw, Send, Trash2, Upload } from 'lucide-vue-next'
import { useToast } from '~/composables/useToast'
import { useFormErrors } from '~/composables/useFormErrors'
import { type Check, rule } from '~/utils/validation'

interface BackupItem {
  name: string
  size: number
  created_at: string
  trigger: 'scheduled' | 'manual' | 'pre-restore' | 'uploaded' | string
  telegram: 'sent' | 'failed' | 'off' | 'skipped' | string
  telegram_error?: string
  created_by?: string
  /** true: database and attachment files; false: database only (older backups, uploaded .sql). */
  full: boolean
  attachments: number
  missing_attachments?: number
}
interface BackupSettings {
  enabled: boolean
  time: string
  keep_days: number
  telegram: boolean
  telegram_chat_id: string
  telegram_token_set: boolean
  telegram_from_env: boolean
  telegram_ready: boolean
  timezone: string
  next_run?: string
}

const { api, apiBase } = useApi()
const { getToken, tryRefresh } = useAuth()
const toast = useToast()

const items = ref<BackupItem[]>([])
const settings = ref<BackupSettings>({
  enabled: true, time: '12:00', keep_days: 7, telegram: true, telegram_chat_id: '',
  telegram_token_set: false, telegram_from_env: false, telegram_ready: false, timezone: 'Asia/Phnom_Penh',
})
const loading = ref(true)
const loadError = ref('')
const running = ref(false)

// Forms are copies, so typing doesn't change the page until it is saved.
const schedule = reactive({ enabled: true, time: '12:00', keep_days: 7 as number | string })
const telegram = reactive({ enabled: true, chat_id: '', bot_token: '' })

function applySettings(next: BackupSettings) {
  settings.value = next
  Object.assign(schedule, { enabled: next.enabled, time: next.time, keep_days: next.keep_days })
  Object.assign(telegram, { enabled: next.telegram, chat_id: next.telegram_chat_id, bot_token: '' })
}

async function load() {
  loading.value = true
  loadError.value = ''
  try {
    const res = await api<{ items: BackupItem[]; settings: BackupSettings }>('/backups')
    items.value = res.items || []
    applySettings(res.settings)
  } catch (e: any) {
    loadError.value = e?.data?.error || 'Please try again.'
  } finally {
    loading.value = false
  }
}

const nextRunText = computed(() => {
  if (!settings.value.next_run) return `Next backup at ${settings.value.time}.`
  const next = new Date(settings.value.next_run)
  return next.getTime() <= Date.now() + 60_000 ? 'Next backup: within a minute.' : `Next backup: ${formatDateTime(settings.value.next_run)}.`
})

// --- schedule (same rules as the API: HH:MM, 1–90 days) ---
const daysRule: Check = value => (Number.isInteger(Number(value)) && Number(value) >= 1 && Number(value) <= 90) ? '' : 'keep days must be 1 to 90'
const timeRule: Check = value => /^([01]\d|2[0-3]):[0-5]\d$/.test(String(value || '')) ? '' : 'time must be HH:MM'
const scheduleCheck = reactive(useFormErrors(
  () => ({ ...schedule }),
  () => ({ time: [rule.required('time'), timeRule], keep_days: [rule.required('keep days'), daysRule] }),
))
const scheduleError = ref('')
const savingSchedule = ref(false)

async function saveSchedule() {
  scheduleError.value = ''
  if (!scheduleCheck.check()) return
  savingSchedule.value = true
  try {
    const next = await api<BackupSettings>('/backups/settings', {
      method: 'PUT',
      body: { enabled: schedule.enabled, time: schedule.time, keep_days: Number(schedule.keep_days) },
    })
    applySettings(next)
    toast.success(next.enabled ? `Every day at ${next.time}. Backups are kept ${next.keep_days} days.` : 'Daily backups are off.', { title: 'Schedule saved' })
    await refreshList()
  } catch (e: any) {
    if (!scheduleCheck.fromApi(e)) scheduleError.value = e?.data?.error || 'Could not save the schedule.'
  } finally {
    savingSchedule.value = false
  }
}

// --- Telegram ---
const telegramCheck = reactive(useFormErrors(
  () => ({ telegram_chat_id: telegram.chat_id }),
  () => ({ telegram_chat_id: [(value: unknown) => !value || /^-?\d+$|^@\w{5,}$/.test(String(value)) ? '' : 'chat ID must be a number (or @channel name)'] }),
))
const telegramError = ref('')
const savingTelegram = ref(false)
const testing = ref(false)

async function saveTelegram() {
  telegramError.value = ''
  if (!telegramCheck.check()) return
  savingTelegram.value = true
  try {
    const next = await api<BackupSettings>('/backups/settings', {
      method: 'PUT',
      body: { telegram: telegram.enabled, telegram_chat_id: telegram.chat_id, ...(telegram.bot_token ? { telegram_bot_token: telegram.bot_token } : {}) },
    })
    applySettings(next)
    toast.success(next.telegram_ready ? 'Backups will be sent to the chat.' : 'Saved. Add the bot token and chat ID to start sending.', { title: 'Telegram saved' })
  } catch (e: any) {
    if (!telegramCheck.fromApi(e)) telegramError.value = e?.data?.error || 'Could not save the Telegram settings.'
  } finally {
    savingTelegram.value = false
  }
}

async function clearToken() {
  savingTelegram.value = true
  telegramError.value = ''
  try {
    applySettings(await api<BackupSettings>('/backups/settings', { method: 'PUT', body: { clear_telegram_token: true } }))
    toast.info('Backups won’t be sent until a token is saved again.', { title: 'Bot token removed' })
  } catch (e: any) {
    telegramError.value = e?.data?.error || 'Could not remove the token.'
  } finally {
    savingTelegram.value = false
  }
}

async function testTelegram() {
  testing.value = true
  telegramError.value = ''
  try {
    await api('/backups/telegram-test', { method: 'POST' })
    toast.success('Check the Telegram chat for the message.', { title: 'Test message sent' })
  } catch (e: any) {
    telegramError.value = e?.data?.error || 'Could not send the test message.'
  } finally {
    testing.value = false
  }
}

// --- backups ---
async function refreshList() {
  try {
    const res = await api<{ items: BackupItem[]; settings: BackupSettings }>('/backups')
    items.value = res.items || []
    settings.value = res.settings
  } catch {
    // The list stays as it was; the next action reloads it.
  }
}

async function backupNow() {
  running.value = true
  try {
    const item = await api<BackupItem>('/backups', { method: 'POST' })
    const where = item.telegram === 'sent' ? 'kept on the server and sent to Telegram' : item.telegram === 'failed' ? `kept on the server, but sending to Telegram failed (${item.telegram_error})` : item.telegram === 'skipped' ? 'kept on the server (too large for Telegram)' : 'kept on the server'
    toast[item.telegram === 'failed' || item.telegram === 'skipped' ? 'warning' : 'success'](`${contentsLabel(item)}, ${formatBytes(item.size)}, ${where}.`, { title: 'Backup made' })
    await refreshList()
  } catch (e: any) {
    toast.error(e?.data?.error || 'Please try again.', { title: 'Backup failed' })
  } finally {
    running.value = false
  }
}

const resending = ref('')
async function resend(item: BackupItem) {
  resending.value = item.name
  try {
    await api(`/backups/${item.name}/telegram`, { method: 'POST' })
    toast.success('Check the Telegram chat.', { title: 'Sent to Telegram' })
  } catch (e: any) {
    toast.error(e?.data?.error || 'Please try again.', { title: 'Couldn’t send to Telegram' })
  } finally {
    resending.value = ''
    await refreshList()
  }
}

// Files are behind the API's auth, so download with the access token and save from a blob URL.
async function download(item: BackupItem) {
  const url = `${apiBase}/backups/${item.name}/download`
  const get = () => {
    const token = getToken()
    return fetch(url, { headers: token ? { Authorization: `Bearer ${token}` } : {}, credentials: 'include' })
  }
  try {
    let res = await get()
    if (res.status === 401 && await tryRefresh()) res = await get()
    if (!res.ok) throw new Error(res.status === 404 ? 'This backup is no longer on the server.' : 'Could not download the backup.')
    const blobUrl = URL.createObjectURL(await res.blob())
    const link = document.createElement('a')
    link.href = blobUrl
    link.download = item.name
    link.click()
    setTimeout(() => URL.revokeObjectURL(blobUrl), 10_000)
  } catch (e: any) {
    toast.error(e?.message || 'Please try again.', { title: 'Download failed' })
  }
}

const deleteTarget = ref<BackupItem | null>(null)
const deleting = ref(false)
async function removeBackup() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await api(`/backups/${deleteTarget.value.name}`, { method: 'DELETE' })
    toast.success('It was removed from the server.', { title: 'Backup deleted' })
    deleteTarget.value = null
    await refreshList()
  } catch (e: any) {
    toast.error(e?.data?.error || 'Please try again.', { title: 'Couldn’t delete the backup' })
  } finally {
    deleting.value = false
  }
}

// --- upload & restore ---
const uploading = ref(false)
const uploadError = ref('')
async function onUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  uploadError.value = ''
  if (!/\.(sql|gz)$/i.test(file.name)) {
    uploadError.value = 'Choose a backup file (.tar.gz, .sql.gz or .sql).'
    return
  }
  if (file.size > 100 * 1024 * 1024) {
    uploadError.value = 'The file is larger than 100 MB.'
    return
  }
  uploading.value = true
  try {
    const body = new FormData()
    body.append('file', file)
    const item = await api<BackupItem>('/backups/upload', { method: 'POST', body })
    toast.success('It is in the list. Use “Restore” on it to load it into the database.', { title: 'Backup uploaded' })
    await refreshList()
    openRestore(items.value.find(other => other.name === item.name) || item)
  } catch (e: any) {
    uploadError.value = e?.data?.error || 'Could not upload the file.'
  } finally {
    uploading.value = false
  }
}

const restoreTarget = ref<BackupItem | null>(null)
const restoreConfirm = ref('')
const restoreError = ref('')
const restoring = ref(false)
function openRestore(item: BackupItem) {
  restoreTarget.value = item
  restoreConfirm.value = ''
  restoreError.value = ''
}

async function restore() {
  if (!restoreTarget.value) return
  if (restoreConfirm.value !== 'RESTORE') {
    restoreError.value = 'Type RESTORE to confirm.'
    return
  }
  restoring.value = true
  restoreError.value = ''
  try {
    const res = await api<{ statements: number; attachments: number; safety_copy: BackupItem }>(`/backups/${restoreTarget.value.name}/restore`, {
      method: 'POST',
      body: { confirm: 'RESTORE' },
    })
    const files = res.attachments ? ` and ${res.attachments === 1 ? '1 attachment file' : `${res.attachments} attachment files`}` : ''
    toast.success(`Database${files} restored. A safety copy of the previous data was kept (${formatDateTime(res.safety_copy.created_at)}).`, { title: 'Backup restored', duration: 9000 })
    restoreTarget.value = null
    await load()
  } catch (e: any) {
    restoreError.value = e?.data?.error || 'Could not restore the backup.'
  } finally {
    restoring.value = false
  }
}

// --- formatting ---
function formatBytes(bytes: number) {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function formatDateTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.valueOf()) ? value : new Intl.DateTimeFormat('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: 'numeric', minute: '2-digit' }).format(date)
}

function triggerLabel(trigger: string) {
  return ({ scheduled: 'Daily', manual: 'Backup now', 'pre-restore': 'Safety copy', uploaded: 'Uploaded' } as Record<string, string>)[trigger] || trigger
}

function contentsLabel(item: BackupItem) {
  if (!item.full) return 'Database only'
  return item.attachments === 1 ? 'Database + 1 file' : `Database + ${item.attachments} files`
}

function telegramLabel(item: BackupItem) {
  return ({ sent: 'Sent', failed: 'Not sent', skipped: 'Too large', off: 'Not sent (off)' } as Record<string, string>)[item.telegram] || item.telegram
}

onMounted(load)
</script>

<style scoped>
.backups-page { color: #183054; }
.page-heading { display: flex; align-items: flex-end; justify-content: space-between; gap: 18px; margin-bottom: 18px; }
.eyebrow { margin: 0 0 6px; color: #6180ae; font-size: 11px; font-weight: 800; letter-spacing: .13em; }
.page-heading h1 { margin: 0; color: #102348; font-size: 26px; line-height: 1.15; font-weight: 750; letter-spacing: -.035em; }
.heading-copy { margin: 7px 0 0; color: #7083a2; font-size: 13px; }
.primary-button, .ghost-button { display: inline-flex; flex: 0 0 auto; min-height: 40px; align-items: center; justify-content: center; gap: 7px; border-radius: 10px; padding: 0 15px; font-size: 13px; font-weight: 700; white-space: nowrap; cursor: pointer; transition: filter .15s ease, background .15s ease; }
.primary-button { border: 0; color: #fff; background: #1769e8; box-shadow: 0 5px 12px rgb(23 105 232 / 20%); }
.primary-button:hover:not(:disabled) { filter: brightness(.95); }
.ghost-button { border: 1px solid #dfe7f1; color: #3d5a82; background: #fff; }
.ghost-button:hover:not(:disabled) { background: #f5f8fc; }
.primary-button:disabled, .ghost-button:disabled { opacity: .55; cursor: not-allowed; box-shadow: none; }
.settings-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 14px; margin-bottom: 14px; }
.card { display: flex; min-width: 0; flex-direction: column; gap: 14px; border: 1px solid #e2eaf4; border-radius: 14px; padding: 16px; background: #fff; box-shadow: 0 4px 16px rgb(24 60 110 / 4%); }
.card-heading { display: flex; align-items: flex-start; gap: 11px; }
.card-heading h2 { margin: 0; color: #172b4e; font-size: 15px; font-weight: 750; }
.card-heading p { margin: 3px 0 0; color: #7b8ca5; font-size: 12px; }
.card-heading > div { flex: 1; min-width: 0; }
.card-icon { display: grid; flex: 0 0 auto; width: 36px; height: 36px; place-items: center; border-radius: 10px; color: #1769e8; background: #eaf2ff; }
.card-icon.is-telegram { color: #229ed9; background: #e7f5fc; }
.switch-row { display: flex; align-items: center; justify-content: space-between; gap: 14px; border: 1px solid #e5ebf4; border-radius: 12px; padding: 11px 13px; background: #f8fafd; cursor: pointer; }
.switch-row > span { display: flex; min-width: 0; flex-direction: column; gap: 3px; }
.switch-row strong { color: #2c4468; font-size: 13px; font-weight: 700; }
.switch-row small { color: #7b8ca5; font-size: 12px; }
.switch { position: relative; flex: 0 0 auto; width: 42px; height: 24px; margin: 0; border-radius: 999px; background: #cfd9e6; cursor: pointer; appearance: none; transition: background .15s ease; }
.switch::before { content: ''; position: absolute; top: 3px; left: 3px; width: 18px; height: 18px; border-radius: 50%; background: #fff; box-shadow: 0 1px 3px rgb(16 40 80 / 25%); transition: transform .15s ease; }
.switch:checked { background: #1769e8; }
.switch:checked::before { transform: translateX(18px); }
.field-row { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.field { display: flex; min-width: 0; flex-direction: column; gap: 6px; color: #324b70; font-size: 13px; font-weight: 650; }
.field small { margin-left: 4px; color: #94a2b7; font-size: 11px; font-weight: 500; }
.field input { width: 100%; min-width: 0; min-height: 42px; border: 1px solid #dfe7f1; border-radius: 10px; outline: none; padding: 9px 12px; color: #263f65; background: #fff; font: inherit; font-size: 14px; font-weight: 450; }
.field input:focus { border-color: #85b4f7; box-shadow: 0 0 0 3px rgb(37 99 235 / 10%); }
.field input:disabled { color: #a6b3c5; background: #f6f8fb; }
.input-suffix { position: relative; }
.input-suffix input { padding-right: 52px; }
.input-suffix em { position: absolute; top: 50%; right: 12px; color: #8b9bb1; font-size: 13px; font-style: normal; transform: translateY(-50%); }
.card-actions { display: flex; flex-wrap: wrap; justify-content: flex-end; gap: 8px; margin-top: auto; }
.list-card { gap: 10px; }
.upload-button { display: inline-flex; flex: 0 0 auto; align-items: center; gap: 6px; border: 1px dashed #b9cbe4; border-radius: 10px; padding: 8px 12px; color: #1769e8; background: #fff; font-size: 13px; font-weight: 700; white-space: nowrap; cursor: pointer; }
.upload-button:hover { background: #f2f7ff; }
.upload-button.is-busy { opacity: .6; cursor: wait; }
.list-alert { margin-top: 2px; }
.list-state { padding: 28px 12px; color: #8193ab; font-size: 13px; text-align: center; }
.backup-list { margin: 0; padding: 0; list-style: none; }
.backup-row { display: flex; align-items: center; gap: 12px; border-top: 1px solid #edf1f7; padding: 12px 2px; }
.backup-icon { display: grid; flex: 0 0 auto; width: 38px; height: 38px; place-items: center; border-radius: 11px; color: #1769e8; background: #eef5ff; }
.backup-icon.is-pre-restore { color: #a46c0d; background: #fff4d7; }
.backup-icon.is-uploaded { color: #6d59b7; background: #f0edff; }
.backup-info { display: flex; min-width: 0; flex: 1; flex-direction: column; gap: 4px; }
.backup-info strong { color: #223b60; font-size: 13px; font-weight: 700; }
.backup-meta { display: flex; flex-wrap: wrap; align-items: center; gap: 6px 10px; color: #8294b0; font-size: 12px; }
.tag { border-radius: 999px; padding: 2px 8px; color: #3565a7; background: #eaf3ff; font-size: 10px; font-style: normal; font-weight: 800; text-transform: uppercase; letter-spacing: .03em; }
.trigger-pre-restore { color: #a46c0d; background: #fff4d7; }
.trigger-uploaded { color: #6d59b7; background: #f0edff; }
.trigger-manual { color: #14835f; background: #e4f7ee; }
.telegram-status { display: inline-flex; align-items: center; gap: 4px; }
.contents { color: #5f7392; }
.tg-sent { color: #14835f; }
.tg-failed { color: #c93449; }
.tg-skipped { color: #a46c0d; }
.backup-actions { display: flex; flex: 0 0 auto; gap: 5px; }
.icon-action { display: grid; width: 34px; height: 34px; place-items: center; border: 0; border-radius: 10px; color: #1769e8; background: #eaf2ff; cursor: pointer; }
.icon-action:hover:not(:disabled) { filter: brightness(.96); }
.icon-action:disabled { opacity: .5; cursor: wait; }
.icon-action.is-restore { color: #a46c0d; background: #fff4d7; }
.icon-action.is-delete { color: #c93449; background: #fdecef; }
.restore-copy { display: flex; flex-direction: column; gap: 10px; color: #4c5f7d; font-size: 13px; line-height: 1.5; }
.restore-copy p { margin: 0; }
.restore-copy strong { color: #1d365c; }
@media (max-width: 760px) {
  .page-heading { flex-direction: column; align-items: stretch; }
  .settings-grid { grid-template-columns: 1fr; }
  .field-row { grid-template-columns: 1fr 1fr; }
  .card-heading { flex-wrap: wrap; }
  .upload-button { width: 100%; justify-content: center; }
  .backup-row { flex-wrap: wrap; }
  .backup-actions { width: 100%; justify-content: flex-end; padding-left: 50px; }
}
</style>
