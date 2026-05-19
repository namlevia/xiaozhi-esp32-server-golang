<template>
  <div class="dashboard-page">
    <section class="stats-grid">
      <article class="metric-card">
        <div class="metric-top">
          <span class="metric-icon users">
            <el-icon><User /></el-icon>
          </span>
          <span class="metric-trend">{{ authStore.isAdmin ? t('dashboard.globalUsers') : t('dashboard.linkedAccount') }}</span>
        </div>
        <strong>{{ authStore.isAdmin ? stats.totalUsers : 1 }}</strong>
        <p>{{ authStore.isAdmin ? t('dashboard.totalUsers') : t('dashboard.currentAccount') }}</p>
      </article>

      <article class="metric-card">
        <div class="metric-top">
          <span class="metric-icon devices">
            <el-icon><Monitor /></el-icon>
          </span>
          <span class="metric-trend">{{ t('dashboard.online', { count: stats.onlineDevices }) }}</span>
        </div>
        <strong>{{ stats.totalDevices }}</strong>
        <p>{{ authStore.isAdmin ? t('dashboard.totalDevices') : t('dashboard.myDevices') }}</p>
      </article>

      <article class="metric-card">
        <div class="metric-top">
          <span class="metric-icon agents">
            <el-icon><Cpu /></el-icon>
          </span>
          <span class="metric-trend">{{ t('dashboard.active') }}</span>
        </div>
        <strong>{{ stats.totalAgents }}</strong>
        <p>{{ authStore.isAdmin ? t('dashboard.totalAgents') : t('dashboard.myAgents') }}</p>
      </article>

      <article class="metric-card">
        <div class="metric-top">
          <span class="metric-icon status">
            <el-icon><Connection /></el-icon>
          </span>
          <span class="metric-trend">{{ t('dashboard.realtimeMonitoring') }}</span>
        </div>
        <strong>{{ stats.onlineDevices }}</strong>
        <p>{{ t('dashboard.onlineDevices') }}</p>
      </article>
    </section>

    <section class="dashboard-grid" :class="{ compact: !authStore.isAdmin }">
      <div class="dashboard-main">
        <el-card v-if="authStore.isAdmin" class="dashboard-card service-card">
          <template #header>
            <div class="card-header">
              <div>
                <p class="card-eyebrow">ĐỊA CHỈ DỊCH VỤ</p>
                <h3>{{ t('dashboard.serviceAddress') }}</h3>
              </div>
              <el-button type="warning" size="small" :loading="otaTestLoading" @click="runOtaTest">
                {{ t('dashboard.otaTest') }}
              </el-button>
            </div>
          </template>

          <div v-loading="addressLoading" class="address-card-content">
            <template v-if="!addressLoading && (serviceAddress.otaUrl || serviceAddress.wsUrl)">
              <div class="address-list">
                <div class="address-row">
                  <span class="address-label">OTA</span>
                  <span class="address-value" :title="serviceAddress.otaUrl">{{ serviceAddress.otaUrl || '—' }}</span>
                  <el-button v-if="serviceAddress.otaUrl" link type="primary" :icon="CopyDocument" @click="copyAddress(serviceAddress.otaUrl)" />
                </div>
                <div class="address-row">
                  <span class="address-label">WS</span>
                  <span class="address-value" :title="serviceAddress.wsUrl">{{ serviceAddress.wsUrl || '—' }}</span>
                  <el-button v-if="serviceAddress.wsUrl" link type="primary" :icon="CopyDocument" @click="copyAddress(serviceAddress.wsUrl)" />
                </div>
                <div v-if="serviceAddress.mqttEndpoint" class="address-row">
                  <span class="address-label">MQTT</span>
                  <span class="address-value" :title="serviceAddress.mqttEndpoint">{{ serviceAddress.mqttEndpoint }}</span>
                  <el-button link type="primary" :icon="CopyDocument" @click="copyAddress(serviceAddress.mqttEndpoint)" />
                </div>
                <div v-if="serviceAddress.udpAddress" class="address-row">
                  <span class="address-label">UDP</span>
                  <span class="address-value" :title="serviceAddress.udpAddress">{{ serviceAddress.udpAddress }}</span>
                  <el-button link type="primary" :icon="CopyDocument" @click="copyAddress(serviceAddress.udpAddress)" />
                </div>
              </div>

              <div v-if="otaTestResult !== null" class="ota-test-block">
                <span class="apple-chip is-primary">{{ t('dashboard.otaResponse') }}</span>
                <pre class="ota-test-pre">{{ otaTestResult }}</pre>
              </div>
            </template>

            <div v-else-if="!addressLoading" class="empty-inline">{{ t('dashboard.noOtaConfig') }}</div>
          </div>
        </el-card>

        <el-card v-if="authStore.isAdmin" class="dashboard-card health-card">
          <template #header>
            <div class="card-header">
              <div>
                <p class="card-eyebrow">HEALTH CHECK</p>
                <h3>Trạng thái stack local</h3>
              </div>
              <el-button size="small" :loading="healthLoading" @click="loadHealthCheck">Làm mới</el-button>
            </div>
          </template>

          <div v-loading="healthLoading" class="health-content">
            <div v-if="healthCheckedAt" class="health-summary">
              <el-tag :type="healthStatusType(healthStatus)" effect="light">{{ healthStatusLabel(healthStatus) }}</el-tag>
              <span>Kiểm tra lúc {{ formatDateTime(healthCheckedAt) }}</span>
            </div>
            <div v-if="healthItems.length" class="health-list">
              <div v-for="item in healthItems" :key="item.name" class="health-row">
                <div class="health-row-main">
                  <strong>{{ item.name }}</strong>
                  <small>{{ item.message || '—' }}</small>
                  <small v-if="item.url" class="health-url">{{ item.url }}</small>
                </div>
                <div class="health-row-side">
                  <el-tag size="small" :type="healthStatusType(item.status)" effect="plain">{{ healthStatusLabel(item.status) }}</el-tag>
                  <span v-if="item.latency_ms != null">{{ item.latency_ms }}ms</span>
                </div>
              </div>
            </div>
            <div v-else class="empty-inline">Chưa có dữ liệu health check.</div>
          </div>
        </el-card>

        <el-card v-if="authStore.isAdmin" class="dashboard-card">
          <template #header>
            <div class="card-header">
              <div>
                <p class="card-eyebrow">CẤU HÌNH</p>
                <h3>{{ t('dashboard.configuration') }}</h3>
              </div>
            </div>
          </template>

          <div class="config-actions">
            <button class="action-card action-primary" type="button" @click="$router.push('/admin/config-wizard')">
              <span class="action-icon"><el-icon><Guide /></el-icon></span>
              <span class="action-copy">
                <strong>{{ t('nav.configWizard') }}</strong>
                <small>{{ t('dashboard.configWizardHint') }}</small>
              </span>
            </button>

            <button class="action-card" type="button" @click="exportConfig">
              <span class="action-icon"><el-icon><Download /></el-icon></span>
              <span class="action-copy">
                <strong>{{ t('dashboard.exportConfig') }}</strong>
                <small>{{ t('dashboard.exportConfigHint') }}</small>
              </span>
            </button>

            <button class="action-card" type="button" @click="importConfig">
              <span class="action-icon"><el-icon><Upload /></el-icon></span>
              <span class="action-copy">
                <strong>{{ t('dashboard.importConfig') }}</strong>
                <small>{{ t('dashboard.importConfigHint') }}</small>
              </span>
            </button>
          </div>

          <input
            ref="fileInput"
            type="file"
            accept=".yaml,.yml,.json"
            style="display: none"
            @change="handleFileChange"
          />
        </el-card>
      </div>

      <div class="dashboard-side">
        <el-card class="dashboard-card info-card">
          <template #header>
            <div class="card-header">
              <div>
                <p class="card-eyebrow">HỆ THỐNG</p>
                <h3>{{ t('dashboard.system') }}</h3>
              </div>
            </div>
          </template>

          <div class="info-list">
            <div class="info-row">
              <span>{{ t('dashboard.version') }}</span>
              <strong>v1.0.0</strong>
            </div>
            <div class="info-row">
              <span>{{ t('dashboard.startedAt') }}</span>
              <strong>{{ programStartedAt }}</strong>
            </div>
            <div class="info-row">
              <span>{{ t('dashboard.currentUser') }}</span>
              <strong>{{ authStore.user?.username || '—' }}</strong>
            </div>
            <div class="info-row">
              <span>{{ t('dashboard.userRole') }}</span>
              <el-tag :type="authStore.isAdmin ? 'danger' : 'primary'" effect="light">
                {{ authStore.isAdmin ? t('layout.admin') : t('layout.user') }}
              </el-tag>
            </div>
          </div>
        </el-card>

        <el-card class="dashboard-card quick-card">
          <template #header>
            <div class="card-header">
              <div>
                <p class="card-eyebrow">LỐI TẮT</p>
                <h3>{{ t('dashboard.shortcuts') }}</h3>
              </div>
            </div>
          </template>

          <div class="quick-actions">
            <template v-if="authStore.isAdmin">
              <button class="quick-action" type="button" @click="$router.push('/admin/users')">
                <span class="quick-action-icon"><el-icon><User /></el-icon></span>
                <span>
                  <strong>{{ t('menu.users') }}</strong>
                  <small>{{ t('dashboard.usersHint') }}</small>
                </span>
              </button>
              <button class="quick-action" type="button" @click="$router.push('/admin/llm-config')">
                <span class="quick-action-icon"><el-icon><Setting /></el-icon></span>
                <span>
                  <strong>LLM {{ t('menu.config') }}</strong>
                  <small>{{ t('dashboard.llmHint') }}</small>
                </span>
              </button>
              <button class="quick-action" type="button" @click="$router.push('/admin/vad-config')">
                <span class="quick-action-icon"><el-icon><Cpu /></el-icon></span>
                <span>
                  <strong>VAD {{ t('menu.config') }}</strong>
                  <small>{{ t('dashboard.vadHint') }}</small>
                </span>
              </button>
            </template>

            <template v-else>
              <button class="quick-action" type="button" @click="$router.push('/agents')">
                <span class="quick-action-icon"><el-icon><Monitor /></el-icon></span>
                <span>
                  <strong>{{ t('menu.agents') }}</strong>
                  <small>{{ t('dashboard.agentsHint') }}</small>
                </span>
              </button>
              <div class="empty-inline">{{ t('dashboard.userQuickHint') }}</div>
            </template>
          </div>
        </el-card>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/stores/auth'
import api from '@/utils/api'
import { ElMessage } from 'element-plus'
import {
  User,
  Monitor,
  Connection,
  Setting,
  Download,
  Upload,
  Cpu,
  Guide,
  CopyDocument
} from '@element-plus/icons-vue'

const { t } = useI18n()
const authStore = useAuthStore()

const addressLoading = ref(false)
const serviceAddress = ref({
  otaUrl: '',
  wsUrl: '',
  mqttEndpoint: '',
  udpAddress: ''
})

async function loadServiceAddress() {
  addressLoading.value = true
  serviceAddress.value = { otaUrl: '', wsUrl: '', mqttEndpoint: '', udpAddress: '' }
  try {
    const [otaRes, udpRes] = await Promise.all([
      api.get('/admin/ota-configs'),
      api.get('/admin/udp-configs')
    ])
    const otaList = otaRes.data?.data || []
    const config = otaList.find(c => c.is_default) || otaList[0]
    if (config?.json_data) {
      const data = JSON.parse(config.json_data || '{}')

      let envData = data.external || {}
      const hasExternalWs = envData.websocket?.url
      const hasExternalOta = envData.ota_url
      if (!hasExternalWs && !hasExternalOta) {
        envData = data.test || {}
      }

      let otaUrl = envData.ota_url || ''
      if (!otaUrl) {
        const wsUrl = envData.websocket?.url || ''
        if (wsUrl) {
          const matched = wsUrl.match(/^(wss?):\/\/([^:/]+)(?::(\d+))?/)
          if (matched) {
            const protocol = matched[1] === 'wss' ? 'https' : 'http'
            const port = matched[3] || (matched[1] === 'wss' ? '443' : '80')
            otaUrl = `${protocol}://${matched[2]}:${port}/xiaozhi/ota/`
          }
        }
      }
      serviceAddress.value.otaUrl = otaUrl
      serviceAddress.value.wsUrl = envData.websocket?.url || ''

      const mqttEnabled = envData.mqtt?.enable
      const endpoint = envData.mqtt?.endpoint || ''
      if (mqttEnabled && endpoint) {
        serviceAddress.value.mqttEndpoint = endpoint
      }
    }

    const udpList = udpRes.data?.data || []
    const udpConfig = udpList.find(c => c.is_default) || udpList[0]
    if (udpConfig?.json_data) {
      const udpData = JSON.parse(udpConfig.json_data || '{}')
      const host = udpData.external_host || ''
      const port = udpData.external_port
      if (host && port != null) {
        serviceAddress.value.udpAddress = `${host}:${port}`
      }
    }
  } catch (err) {
    console.error(`${t('dashboard.loadAddressFailed')}:`, err)
  } finally {
    addressLoading.value = false
  }
}

function copyAddress(text) {
  if (!text) return
  navigator.clipboard.writeText(text).then(() => {
    ElMessage.success(t('dashboard.copied'))
  }).catch(() => {
    ElMessage.error(t('dashboard.copyFailed'))
  })
}

const otaTestLoading = ref(false)
const otaTestResult = ref(null)
const healthLoading = ref(false)
const healthStatus = ref('unknown')
const healthCheckedAt = ref('')
const healthItems = ref([])

function healthStatusLabel(status) {
  switch (status) {
    case 'healthy': return 'Ổn định'
    case 'degraded': return 'Cần chú ý'
    case 'unreachable': return 'Không kết nối'
    case 'disabled': return 'Đã tắt'
    default: return 'Không rõ'
  }
}

function healthStatusType(status) {
  switch (status) {
    case 'healthy': return 'success'
    case 'degraded': return 'warning'
    case 'unreachable': return 'danger'
    case 'disabled': return 'info'
    default: return 'info'
  }
}

function formatDateTime(value) {
  return value ? new Date(value).toLocaleString('vi-VN') : '—'
}

async function loadHealthCheck() {
  healthLoading.value = true
  try {
    const res = await api.get('/admin/health-check', { timeout: 10000 })
    const data = res.data?.data || res.data || {}
    healthStatus.value = data.status || 'unknown'
    healthCheckedAt.value = data.checked_at || ''
    healthItems.value = data.items || []
  } catch (error) {
    healthStatus.value = 'unreachable'
    healthCheckedAt.value = new Date().toISOString()
    healthItems.value = [{ name: 'Health check', status: 'unreachable', message: error.response?.data?.error || error.message || 'Yêu cầu health check thất bại' }]
  } finally {
    healthLoading.value = false
  }
}

function formatOtaResponseDisplay(str) {
  if (str == null || str === '') return ''
  const content = String(str).trim()
  if (!content) return ''
  try {
    return JSON.stringify(JSON.parse(content), null, 2)
  } catch {
    return content
  }
}

async function runOtaTest() {
  otaTestLoading.value = true
  otaTestResult.value = null
  try {
    const res = await api.post('/admin/configs/test', { types: ['ota'] }, { timeout: 30000 })
    const data = res.data?.data ?? res.data
    const ota = data?.ota
    if (ota && typeof ota === 'object') {
      const entry = Object.entries(ota).find(([key]) => !key.startsWith('_'))
      if (entry) {
        const [, value] = entry
        let displayText = ''

        if (value.websocket) {
          const ws = value.websocket
          displayText += `WebSocket: ${ws.ok ? '✓' : '✗'} ${ws.message}`
          displayText += ws.first_packet_ms != null ? ` (${ws.first_packet_ms}ms)\n` : '\n'
        }

        if (value.mqtt_udp) {
          const mqtt = value.mqtt_udp
          displayText += `MQTT UDP: ${mqtt.ok ? '✓' : '✗'} ${mqtt.message}`
          displayText += mqtt.first_packet_ms != null ? ` (${mqtt.first_packet_ms}ms)\n` : '\n'
        }

        if (value.ota_response !== undefined && value.ota_response !== '') {
          displayText += `\n--- ${t('dashboard.otaResponse')} ---\n${formatOtaResponseDisplay(value.ota_response)}`
        }

        otaTestResult.value = displayText.trim() || t('dashboard.otaDetailsMissing')
        ElMessage[value.ok ? 'success' : 'warning'](value.message || (value.ok ? t('dashboard.otaPassed') : t('dashboard.otaNotPassed')))
      } else {
        otaTestResult.value = t('dashboard.otaResultMissing')
      }
    } else {
      otaTestResult.value = typeof data === 'string' ? data : JSON.stringify(data || {}, null, 2)
    }
  } catch (error) {
    const errorMsg = (error.response?.data && typeof error.response.data === 'object')
      ? JSON.stringify(error.response.data, null, 2)
      : (error.response?.data?.message || error.message || t('dashboard.requestFailed'))
    otaTestResult.value = errorMsg
    ElMessage.error(t('dashboard.otaRequestFailed'))
  } finally {
    otaTestLoading.value = false
  }
}

const stats = ref({
  totalUsers: 0,
  totalDevices: 0,
  totalAgents: 0,
  onlineDevices: 0
})

const programStartedAt = ref('—')
const fileInput = ref(null)

onMounted(async () => {
  await loadStats()
  if (authStore.isAdmin) {
    loadServiceAddress()
    loadHealthCheck()
  }
})

const loadStats = async () => {
  try {
    const response = await api.get('/dashboard/stats')
    stats.value = {
      totalUsers: response.data.totalUsers || 0,
      totalDevices: response.data.totalDevices || 0,
      totalAgents: response.data.totalAgents || 0,
      onlineDevices: response.data.onlineDevices || 0
    }
    programStartedAt.value = response.data?.programStartedAt
      ? new Date(response.data.programStartedAt).toLocaleString('vi-VN')
      : '—'
  } catch (error) {
    console.error(`${t('dashboard.loadStatsFailed')}:`, error)
    stats.value = {
      totalUsers: 0,
      totalDevices: 0,
      totalAgents: 0,
      onlineDevices: 0
    }
    programStartedAt.value = '—'
  }
}

const exportConfig = async () => {
  try {
    const response = await fetch('/api/admin/configs/export', {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${authStore.token}`
      }
    })

    if (response.ok) {
      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = 'config.yaml'
      document.body.appendChild(link)
      link.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(link)
      ElMessage.success(t('dashboard.exportSuccess'))
    } else {
      ElMessage.error(t('dashboard.exportFailed'))
    }
  } catch (error) {
    console.error(`${t('dashboard.exportFailed')}:`, error)
    ElMessage.error(t('dashboard.exportFailed'))
  }
}

const importConfig = () => {
  fileInput.value.click()
}

const handleFileChange = async (event) => {
  const file = event.target.files[0]
  if (!file) return

  const validExtensions = ['.yaml', '.yml', '.json']
  const fileExtension = file.name.toLowerCase().substring(file.name.lastIndexOf('.'))

  if (!validExtensions.includes(fileExtension)) {
    ElMessage.error(t('dashboard.invalidConfigFile'))
    return
  }

  const formData = new FormData()
  formData.append('file', file)

  try {
    const response = await fetch('/api/admin/configs/import', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${authStore.token}`
      },
      body: formData
    })

    if (response.ok) {
      ElMessage.success(t('dashboard.importSuccess'))
    } else {
      const error = await response.json()
      ElMessage.error(error.error || t('dashboard.importFailed'))
    }
  } catch (error) {
    console.error(`${t('dashboard.importFailed')}:`, error)
    ElMessage.error(t('dashboard.importFailed'))
  }

  event.target.value = ''
}
</script>

<style scoped>
.dashboard-page {
  display: grid;
  gap: 20px;
}
.card-eyebrow {
  margin: 0;
  color: var(--apple-text-tertiary);
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.metric-card {
  padding: 22px;
  border-radius: 24px;
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.88);
  box-shadow: var(--apple-shadow-md);
}

.metric-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
  margin-bottom: 18px;
}

.metric-icon {
  width: 42px;
  height: 42px;
  border-radius: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
}

.metric-icon.users {
  color: var(--apple-primary);
  background: var(--apple-primary-soft);
}

.metric-icon.devices {
  color: #176a31;
  background: var(--apple-success-soft);
}

.metric-icon.agents {
  color: #875f00;
  background: var(--apple-warning-soft);
}

.metric-icon.status {
  color: #8a1f19;
  background: var(--apple-danger-soft);
}

.metric-trend {
  color: var(--apple-text-secondary);
  font-size: 12px;
  font-weight: 600;
}

.metric-card strong {
  display: block;
  font-size: 34px;
  line-height: 1;
  letter-spacing: -0.05em;
}

.metric-card p {
  margin: 10px 0 0;
  color: var(--apple-text-secondary);
  font-size: 14px;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.3fr) minmax(320px, 0.9fr);
  gap: 18px;
}

.dashboard-grid.compact {
  grid-template-columns: 1fr 360px;
}

.dashboard-main,
.dashboard-side {
  display: grid;
  gap: 18px;
}

.dashboard-card :deep(.el-card__header) {
  padding-bottom: 18px;
}

.card-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.card-header h3 {
  margin: 4px 0 0;
  font-size: 18px;
}

.address-card-content {
  display: grid;
  gap: 16px;
}

.address-list {
  display: grid;
  gap: 12px;
}

.address-row {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 14px 16px;
  border-radius: 18px;
  background: rgba(248, 250, 252, 0.86);
  border: 1px solid rgba(229, 229, 234, 0.76);
}

.address-label {
  color: var(--apple-text-tertiary);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
}

.address-value {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--apple-text);
  font-weight: 500;
}

.ota-test-block {
  display: grid;
  gap: 10px;
}

.ota-test-pre {
  margin: 0;
  padding: 16px;
  border-radius: 18px;
  background: #f7f9fc;
  border: 1px solid rgba(229, 229, 234, 0.72);
  color: #445064;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  max-height: 180px;
  overflow: auto;
}

.config-actions,
.quick-actions {
  display: grid;
  gap: 12px;
}

.action-card,
.quick-action {
  width: 100%;
  padding: 16px;
  border: 1px solid rgba(229, 229, 234, 0.76);
  border-radius: 20px;
  background: rgba(255, 255, 255, 0.9);
  display: flex;
  align-items: center;
  gap: 14px;
  text-align: left;
  cursor: pointer;
  color: inherit;
}

.action-card:hover,
.quick-action:hover {
  transform: translateY(-1px);
  box-shadow: var(--apple-shadow-sm);
  border-color: rgba(0, 122, 255, 0.18);
}

.action-primary {
  background: linear-gradient(180deg, rgba(0, 122, 255, 0.12) 0%, rgba(0, 122, 255, 0.06) 100%);
}

.action-icon,
.quick-action-icon {
  width: 42px;
  height: 42px;
  border-radius: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 122, 255, 0.1);
  color: var(--apple-primary);
  font-size: 18px;
  flex: none;
}

.action-copy,
.quick-action span:last-child {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.action-copy strong,
.quick-action strong {
  font-size: 15px;
}

.action-copy small,
.quick-action small {
  color: var(--apple-text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.info-list {
  display: grid;
  gap: 12px;
}

.info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 0;
  border-bottom: 1px solid rgba(229, 229, 234, 0.72);
}

.info-row:last-child {
  border-bottom: 0;
  padding-bottom: 0;
}

.info-row span {
  color: var(--apple-text-secondary);
  font-size: 14px;
}

.info-row strong {
  color: var(--apple-text);
  font-size: 14px;
}

.empty-inline {
  color: var(--apple-text-secondary);
  font-size: 13px;
  line-height: 1.7;
}

.health-content,
.health-list,
.health-row-main {
  display: grid;
  gap: 12px;
}

.health-summary,
.health-row,
.health-row-side {
  display: flex;
  align-items: center;
  gap: 10px;
}

.health-summary {
  color: var(--apple-text-secondary);
  font-size: 13px;
}

.health-row {
  justify-content: space-between;
  padding: 14px 0;
  border-bottom: 1px solid rgba(229, 229, 234, 0.72);
}

.health-row:last-child {
  border-bottom: 0;
}

.health-row-main {
  gap: 4px;
  min-width: 0;
}

.health-row-main small,
.health-row-side span {
  color: var(--apple-text-secondary);
  font-size: 12px;
}

.health-url {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.health-row-side {
  flex: none;
  justify-content: flex-end;
}

@media (max-width: 1280px) {
  .dashboard-grid,
  .dashboard-grid.compact {
    grid-template-columns: 1fr;
  }

  .stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 768px) {
  .stats-grid {
    grid-template-columns: 1fr;
  }

  .address-row {
    grid-template-columns: 1fr;
    align-items: flex-start;
  }
}
</style>
