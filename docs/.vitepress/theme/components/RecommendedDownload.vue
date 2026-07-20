<template>
  <section class="recommended-download" aria-labelledby="recommended-download-title">
    <div class="download-copy">
      <p class="eyebrow">{{ text.eyebrow }}</p>
      <h2 id="recommended-download-title">{{ title }}</h2>
      <p class="description">{{ description }}</p>
      <div class="meta" v-if="versionLabel || platformLabel">
        <span v-if="platformLabel">{{ platformLabel }}</span>
        <span v-if="versionLabel">{{ versionLabel }}</span>
      </div>
    </div>
    <div class="download-actions">
      <a class="primary-action" :href="downloadUrl" rel="noopener noreferrer">
        {{ primaryText }}
      </a>
      <a
        v-if="mirrorUrl"
        class="secondary-action"
        :href="mirrorUrl"
        rel="noopener noreferrer"
        :title="text.mirrorTooltip"
      >
        {{ text.mirror }}
      </a>
      <a class="secondary-action" :href="releasesUrl" rel="noopener noreferrer">
        {{ text.allDownloads }}
      </a>
      <a class="secondary-action" :href="privacyUrl">
        {{ text.privacy }}
      </a>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { withBase } from 'vitepress'

type Locale = 'zh' | 'en'
type Os = 'darwin' | 'windows' | 'linux'
type Arch = 'arm64' | 'amd64'

interface ReleaseAsset {
  name: string
  browser_download_url: string
}

interface LatestRelease {
  tag_name?: string
  assets?: ReleaseAsset[]
}

interface NavigatorUAData {
  platform?: string
  getHighEntropyValues?: (hints: string[]) => Promise<{ architecture?: string }>
}

interface PlatformInfo {
  os?: Os
  arch?: Arch
}

const props = defineProps<{
  locale?: Locale
}>()

const releasesUrl = 'https://github.com/BenedictKing/ccx/releases/latest'
const apiUrl = 'https://api.github.com/repos/BenedictKing/ccx/releases/latest'

const platform = ref<PlatformInfo>({})
const assetUrl = ref('')
const latestTag = ref('')

const locale = computed<Locale>(() => props.locale ?? 'zh')

const copy = {
  zh: {
    eyebrow: '桌面客户端下载',
    title: '推荐下载 CCX Desktop',
    fallbackTitle: '下载 CCX Desktop 最新版',
    description: '自动识别你的系统，优先推荐适合当前设备的安装包。',
    fallbackDescription: '查看最新版本，并选择适合你系统的安装包。',
    allDownloads: '查看全部下载',
    privacy: '隐私政策',
    download: '下载',
    latest: '最新版',
    version: '版本',
    platform: '当前平台',
    mirror: '加速下载（中国）',
    mirrorTooltip: '通过 gh-proxy.org 镜像下载，适合 GitHub 直连较慢的用户',
    names: {
      darwin: 'macOS',
      windows: 'Windows',
      linux: 'Linux',
    },
  },
  en: {
    eyebrow: 'Desktop app download',
    title: 'Recommended CCX Desktop download',
    fallbackTitle: 'Download the latest CCX Desktop',
    description: 'Detects your system and recommends the installer for this device.',
    fallbackDescription: 'Open the latest release and choose the installer for your system.',
    allDownloads: 'All downloads',
    privacy: 'Privacy Policy',
    download: 'Download',
    latest: 'Latest release',
    version: 'Version',
    platform: 'Platform',
    mirror: 'Mirror download (China)',
    mirrorTooltip: 'Download via the gh-proxy.org mirror for users with limited GitHub access',
    names: {
      darwin: 'macOS',
      windows: 'Windows',
      linux: 'Linux',
    },
  },
} as const

const text = computed(() => copy[locale.value])
const matchedPlatformName = computed(() => {
  if (!platform.value.os) {
    return ''
  }

  return text.value.names[platform.value.os]
})
const platformLabel = computed(() => {
  if (!matchedPlatformName.value) {
    return ''
  }

  const arch = platform.value.arch ? ` ${platform.value.arch}` : ''
  return `${text.value.platform}: ${matchedPlatformName.value}${arch}`
})
const versionLabel = computed(() => latestTag.value ? `${text.value.version}: ${latestTag.value}` : '')
const downloadUrl = computed(() => assetUrl.value || releasesUrl)
const mirrorUrl = computed(() => assetUrl.value ? `https://v6.gh-proxy.org/${assetUrl.value}` : '')
const privacyUrl = computed(() => withBase(locale.value === 'en' ? '/en/guide/privacy' : '/guide/privacy'))
const primaryText = computed(() => {
  if (!matchedPlatformName.value) {
    return text.value.latest
  }

  return `${text.value.download} ${matchedPlatformName.value}`
})
const title = computed(() => matchedPlatformName.value ? text.value.title : text.value.fallbackTitle)
const description = computed(() => matchedPlatformName.value ? text.value.description : text.value.fallbackDescription)

onMounted(async () => {
  platform.value = await detectPlatform()
  await loadLatestAsset()
})

async function detectPlatform(): Promise<PlatformInfo> {
  const nav = window.navigator
  const uaData = (nav as Navigator & { userAgentData?: NavigatorUAData }).userAgentData
  const platformText = `${uaData?.platform ?? ''} ${nav.platform ?? ''} ${nav.userAgent ?? ''}`.toLowerCase()
  const os = detectOs(platformText)
  const arch = await detectArch(uaData, platformText, os)

  return { os, arch }
}

function detectOs(value: string): Os | undefined {
  if (/mac|darwin/.test(value)) {
    return 'darwin'
  }

  if (/win/.test(value)) {
    return 'windows'
  }

  if (/linux|x11/.test(value)) {
    return 'linux'
  }

  return undefined
}

async function detectArch(uaData: NavigatorUAData | undefined, platformText: string, os: Os | undefined): Promise<Arch | undefined> {
  const highEntropyArch = await readHighEntropyArch(uaData)
  const normalized = `${highEntropyArch ?? ''} ${platformText}`.toLowerCase()

  if (/arm|aarch64/.test(normalized)) {
    return 'arm64'
  }

  if (/x86_64|x64|amd64|win64|wow64/.test(normalized)) {
    return 'amd64'
  }

  if (os === 'darwin') {
    return 'arm64'
  }

  if (os === 'windows' || os === 'linux') {
    return 'amd64'
  }

  return undefined
}

async function readHighEntropyArch(uaData: NavigatorUAData | undefined): Promise<string> {
  try {
    const values = await uaData?.getHighEntropyValues?.(['architecture'])
    return values?.architecture ?? ''
  } catch {
    return ''
  }
}

async function loadLatestAsset() {
  if (!platform.value.os) {
    return
  }

  try {
    const response = await fetch(apiUrl, { headers: { Accept: 'application/vnd.github+json' } })
    if (!response.ok) {
      return
    }

    const release = await response.json() as LatestRelease
    latestTag.value = release.tag_name ?? ''
    const asset = release.assets?.find((item) => matchesAsset(item.name, platform.value))
    assetUrl.value = asset?.browser_download_url ?? ''
  } catch {
    assetUrl.value = ''
  }
}

function matchesAsset(name: string, info: PlatformInfo): boolean {
  if (!info.os) {
    return false
  }

  const archPattern = info.arch ?? '(?:arm64|amd64)'
  const escapedArch = typeof archPattern === 'string' ? archPattern : '(?:arm64|amd64)'
  const patterns: Record<Os, RegExp> = {
    darwin: new RegExp(`^CCX-Desktop-.*-darwin-${escapedArch}\\.dmg$`),
    windows: new RegExp(`^CCX-Desktop-.*-windows-${escapedArch}-setup\\.exe$`),
    linux: new RegExp(`^CCX-Desktop-.*-linux-${escapedArch}\\.AppImage$`),
  }

  return patterns[info.os].test(name)
}
</script>

<style scoped>
.recommended-download {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 32px;
  max-width: 960px;
  margin: 0 auto 56px;
  padding: 28px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 24px;
  background: var(--vp-c-bg-soft);
}

.download-copy {
  min-width: 0;
}

.eyebrow {
  margin: 0 0 8px;
  color: var(--vp-c-brand-1);
  font-size: 14px;
  font-weight: 700;
}

h2 {
  margin: 0;
  color: var(--vp-c-text-1);
  font-size: 28px;
  line-height: 1.25;
}

.description {
  max-width: 560px;
  margin: 10px 0 0;
  color: var(--vp-c-text-2);
  line-height: 1.7;
}

.meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
}

.meta span {
  padding: 4px 10px;
  border: 1px solid var(--vp-c-divider);
  border-radius: 999px;
  color: var(--vp-c-text-2);
  font-size: 12px;
}

.download-actions {
  display: flex;
  flex-shrink: 0;
  gap: 12px;
}

.primary-action,
.secondary-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 44px;
  padding: 0 18px;
  border-radius: 999px;
  font-weight: 700;
  text-decoration: none;
  transition: border-color 0.2s, background-color 0.2s, color 0.2s;
}

.primary-action {
  background: var(--vp-c-brand-1);
  color: var(--vp-c-white);
}

.primary-action:hover {
  background: var(--vp-c-brand-2);
}

.secondary-action {
  border: 1px solid var(--vp-c-divider);
  color: var(--vp-c-text-1);
}

.secondary-action:hover {
  border-color: var(--vp-c-brand-1);
  color: var(--vp-c-brand-1);
}

@media (max-width: 768px) {
  .recommended-download {
    align-items: stretch;
    flex-direction: column;
    margin: 0 24px 40px;
    padding: 24px;
  }

  h2 {
    font-size: 24px;
  }

  .download-actions {
    flex-direction: column;
  }
}
</style>
