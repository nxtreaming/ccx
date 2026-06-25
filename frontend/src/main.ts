import { createApp, unref } from 'vue'
import { createPinia } from 'pinia'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'
import vuetify from './plugins/vuetify'
import router from './router'
import App from './App.vue'
import './assets/style.css'
import { useAuthStore } from './stores/auth'
import { usePreferencesStore } from './stores/preferences'
import { applyDocumentLanguage, getRuntimeLocale } from './i18n'
import i18n from './i18n/vue-i18n'

const app = createApp(App)

const pinia = createPinia()
pinia.use(piniaPluginPersistedstate)

// TS 6.x Plugin 类型兼容（Vue 3.5 app.use 签名变更）
app.use(pinia as any) // eslint-disable-line @typescript-eslint/no-explicit-any
app.use(vuetify as any) // eslint-disable-line @typescript-eslint/no-explicit-any
app.use(router as any) // eslint-disable-line @typescript-eslint/no-explicit-any
app.use(i18n)

// 初始化 AuthStore（从 localStorage 恢复状态）
const authStore = useAuthStore()
authStore.initializeAuth()

const preferencesStore = usePreferencesStore()
preferencesStore.initializeUILanguage(getRuntimeLocale())
applyDocumentLanguage(unref(preferencesStore.uiLanguage))

// 同步 vue-i18n locale 为 preferences store 中的持久化值
i18n.global.locale.value = unref(preferencesStore.uiLanguage)

app.mount('#app')
