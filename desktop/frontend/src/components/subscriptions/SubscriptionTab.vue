<script setup lang="ts">
import { computed, ref } from 'vue'
import { CheckCircle2, Copy, Github, Loader2, ShieldCheck } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useChannelPresets } from '@/composables/useChannelPresets'
import { useCopilotOAuth } from '@/composables/useCopilotOAuth'
import { useLanguage } from '@/composables/useLanguage'

const { t } = useLanguage()
const { creating, error, result, createChannel } = useChannelPresets()

const copilotApiKeys = ref<string[]>([])
const copilotProxyUrl = ref('')
const copilotCreateError = ref('')

const {
  copilotOAuthLoading,
  copilotPolling,
  copilotOAuthError,
  copilotOAuthSuccess,
  copilotUserCode,
  copilotUserCodeCopied,
  clearCopilotPollTimer,
  copyCopilotUserCode,
  startCopilotOAuth,
  openCopilotAuthorization,
} = useCopilotOAuth(copilotApiKeys, t, () => copilotProxyUrl.value)

const latestCopilotToken = computed(() => copilotApiKeys.value[copilotApiKeys.value.length - 1] || '')

async function startCopilotAuthorization() {
  copilotApiKeys.value = []
  copilotCreateError.value = ''
  await startCopilotOAuth()
}

function cancelCopilotAuthorization() {
  clearCopilotPollTimer()
  copilotPolling.value = false
  copilotOAuthLoading.value = false
}

async function addCopilotChannel() {
  const token = latestCopilotToken.value
  if (!token) return
  copilotCreateError.value = ''
  try {
    await createChannel({
      provider: 'github-copilot',
      target: 'responses',
      baseUrl: 'https://api.githubcopilot.com',
      apiKey: token,
      name: 'desktop-github-copilot',
      proxyUrl: copilotProxyUrl.value.trim(),
    }, { reloadPresets: false })
  } catch (err) {
    copilotCreateError.value = err instanceof Error ? err.message : String(err)
  }
}
</script>

<template>
  <div class="flex h-full min-h-0 flex-col gap-5">
    <div class="bg-glass dark:bg-glass-dark border border-border rounded-2xl p-5 shrink-0">
      <div class="flex items-start justify-between gap-4">
        <div>
          <div class="flex items-center gap-2 text-primary mb-2">
            <ShieldCheck class="w-4 h-4" />
            <span class="text-xs font-bold uppercase tracking-[0.2em]">{{ t('subscription.headerEyebrow') }}</span>
          </div>
          <h3 class="text-xl font-bold text-foreground">{{ t('subscription.title') }}</h3>
          <p class="text-sm text-muted-foreground mt-1 max-w-2xl">
            {{ t('subscription.description') }}
          </p>
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-4 xl:grid-cols-[minmax(0,520px)_1fr]">
      <section class="bg-glass dark:bg-glass-dark border border-border rounded-2xl p-5 space-y-5">
        <div class="flex items-start gap-3">
          <div class="mt-0.5 flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-secondary ring-1 ring-border">
            <Github class="h-5 w-5 text-foreground" />
          </div>
          <div class="min-w-0 flex-1">
            <div class="flex flex-wrap items-center gap-2">
              <h4 class="text-base font-semibold text-foreground">GitHub Copilot</h4>
              <span
                v-if="copilotOAuthSuccess || latestCopilotToken"
                class="rounded border border-emerald-500/20 bg-emerald-500/10 px-1.5 py-0.5 text-[10px] text-emerald-700 dark:text-emerald-400"
              >
                {{ t('subscription.authorized') }}
              </span>
            </div>
            <p class="mt-1 text-sm text-muted-foreground">{{ t('copilotOAuth.description') }}</p>
          </div>
        </div>

        <div class="space-y-1.5">
          <Label class="text-xs text-muted-foreground">{{ t('channelEditor.transport.proxyUrl.label') }}</Label>
          <Input
            v-model="copilotProxyUrl"
            class="font-mono text-xs"
            :placeholder="t('channelEditor.transport.proxyUrl.placeholder')"
          />
          <p class="text-xs text-muted-foreground">{{ t('channelEditor.transport.proxyUrl.hint') }}</p>
        </div>

        <div v-if="copilotUserCode" class="flex flex-wrap items-center gap-2 text-sm">
          <span class="text-muted-foreground">{{ t('copilotOAuth.userCode') }}</span>
          <code class="rounded bg-muted px-2 py-0.5 font-mono text-xs">{{ copilotUserCode }}</code>
          <button
            type="button"
            class="inline-flex h-6 w-6 items-center justify-center rounded border border-border text-muted-foreground transition-colors hover:text-foreground"
            :title="copilotUserCodeCopied ? t('common.copied') : t('common.copy')"
            :aria-label="copilotUserCodeCopied ? t('common.copied') : t('common.copy')"
            @click="copyCopilotUserCode"
          >
            <CheckCircle2 v-if="copilotUserCodeCopied" class="h-3.5 w-3.5 text-emerald-700 dark:text-emerald-400" />
            <Copy v-else class="h-3.5 w-3.5" />
          </button>
          <button type="button" class="text-xs text-primary underline" @click="openCopilotAuthorization">
            {{ t('copilotOAuth.openAuthorize') }}
          </button>
        </div>

        <p v-if="copilotOAuthSuccess" class="text-xs text-emerald-600">{{ t('copilotOAuth.success') }}</p>
        <p v-if="copilotOAuthError" class="text-xs text-destructive">{{ copilotOAuthError }}</p>
        <p v-if="copilotCreateError || error" class="text-xs text-destructive">{{ copilotCreateError || error }}</p>
        <p v-if="result?.provider === 'github-copilot'" class="text-xs text-emerald-600">{{ result.message }}</p>

        <div class="flex flex-wrap items-center gap-2">
          <Button :disabled="copilotOAuthLoading || copilotPolling" @click="startCopilotAuthorization">
            <Loader2 v-if="copilotOAuthLoading || copilotPolling" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
            {{ t('copilotOAuth.button') }}
          </Button>
          <button
            v-if="copilotPolling || copilotOAuthLoading"
            type="button"
            class="text-xs text-muted-foreground underline"
            @click="cancelCopilotAuthorization"
          >
            {{ t('copilotOAuth.cancel') }}
          </button>
        </div>

        <div class="border-t border-border/70 pt-4">
          <Button :disabled="creating || !latestCopilotToken" @click="addCopilotChannel">
            <Loader2 v-if="creating" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
            {{ t('subscription.addCopilotChannel') }}
          </Button>
        </div>
      </section>
    </div>
  </div>
</template>
