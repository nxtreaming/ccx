import { ref } from 'vue'
import { GetEnvFile, SaveEnvFile } from '@bindings/github.com/BenedictKing/ccx/desktop/desktopservice'

type EnvFileState = {
  path: string
  content: string
  exists: boolean
}

const envFile = ref<EnvFileState>({ path: '', content: '', exists: false })
const envContent = ref('')
const envLoading = ref(false)
const envSaving = ref(false)
const envMessage = ref('')
const envError = ref('')

const loadEnvFile = async () => {
  envLoading.value = true
  envError.value = ''
  try {
    const data = await GetEnvFile() as EnvFileState
    envFile.value = data
    envContent.value = data.content || ''
  } catch (error) {
    envError.value = error instanceof Error ? error.message : String(error)
  } finally {
    envLoading.value = false
  }
}

const saveEnvFile = async (content?: string) => {
  envSaving.value = true
  envMessage.value = ''
  envError.value = ''
  try {
    const nextContent = content ?? envContent.value
    await SaveEnvFile(nextContent)
    envContent.value = nextContent
    await loadEnvFile()
    envMessage.value = '.env 已保存，重启服务后生效'
  } catch (error) {
    envError.value = error instanceof Error ? error.message : String(error)
  } finally {
    envSaving.value = false
  }
}

export function useEnvFile() {
  return {
    envFile,
    envContent,
    envLoading,
    envSaving,
    envMessage,
    envError,
    loadEnvFile,
    saveEnvFile,
  }
}
