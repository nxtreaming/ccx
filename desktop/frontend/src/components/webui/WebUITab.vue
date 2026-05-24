<script setup lang="ts">
import { computed, ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Globe } from 'lucide-vue-next'
import type { DesktopStatus } from '@/types'
import { GetProxyAccessKey, OpenWebUIInBrowser } from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'

const props = defineProps<{
  status: DesktopStatus
  loading: boolean
}>()

const iframeRef = ref<HTMLIFrameElement | null>(null)

const iframeSrc = computed(() => {
  if (!props.status.url) return ''
  const url = new URL(props.status.url.replace('http://127.0.0.1:', 'http://localhost:'))
  url.searchParams.set('ccx_desktop', '1')
  return url.toString()
})

const postProxyAccessKey = async () => {
  if (!iframeRef.value?.contentWindow || !iframeSrc.value) return
  try {
    const accessKey = await GetProxyAccessKey()
    const targetOrigin = new URL(iframeSrc.value).origin
    iframeRef.value.contentWindow.postMessage(
      { type: 'ccx-desktop-auth', accessKey },
      targetOrigin,
    )
  } catch {
    // Web UI 仍可手动输入 access key
  }
}

const refreshIframe = () => {
  if (!iframeRef.value) return
  iframeRef.value.src = iframeRef.value.src
}

defineExpose({ refreshIframe })

const openInBrowser = async () => {
  try {
    await OpenWebUIInBrowser()
  } catch {
    // handled by parent
  }
}
</script>

<template>
  <div class="h-full">
    <div v-if="status.running && iframeSrc" class="h-full rounded-lg overflow-hidden border-0">
      <iframe
        ref="iframeRef"
        :src="iframeSrc"
        class="w-full h-full border-0 block"
        style="background: white"
        title="CCX Web UI"
        @load="postProxyAccessKey"
      />
    </div>
    <Card v-else>
      <CardContent class="flex flex-col items-start gap-4 py-8">
        <p class="text-sm text-muted-foreground">CCX 服务尚未启动，无法显示 Web UI。</p>
        <Button size="sm" :disabled="loading" @click="openInBrowser">
          <Globe class="w-4 h-4 mr-1.5" />
          浏览器打开
        </Button>
      </CardContent>
    </Card>
  </div>
</template>
