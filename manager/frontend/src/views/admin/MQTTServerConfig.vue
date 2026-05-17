<template>
  <div class="mqtt-server-config">
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      class="config-form"
      v-loading="loading"
    >
      <div class="config-layout">
        <el-card class="config-card config-card-main" shadow="never">
          <template #header>
            <div class="card-head">
              <div>
                <p class="card-kicker">MQTT Server</p>
                <h3>Lắng nghe và truy cập</h3>
                <p class="card-description">Cấu hình địa chỉ lắng nghe và trạng thái bật/tắt của MQTT Server tích hợp để thiết bị và chương trình chính truy cập.</p>
              </div>
              <el-tag :type="serverReady ? 'success' : 'warning'" effect="plain" round>
                {{ serverReady ? 'Đã đủ tham số dịch vụ' : 'Cần bổ sung' }}
              </el-tag>
            </div>
          </template>

          <div class="field-grid">
            <el-form-item label="Trạng thái bật/tắt" prop="enable">
              <div class="switch-field">
                <div>
                  <div class="switch-title">Bật MQTT Server tích hợp</div>
                  <div class="field-help">Khi tắt, hệ thống sẽ không lắng nghe kết nối MQTT từ thiết bị.</div>
                </div>
                <el-switch v-model="form.enable" />
              </div>
            </el-form-item>

            <el-form-item label="Host lắng nghe" prop="listen_host">
              <el-input v-model="form.listen_host" placeholder="Ví dụ: 0.0.0.0" />
            </el-form-item>

            <el-form-item label="Cổng lắng nghe" prop="listen_port">
              <el-input-number v-model="form.listen_port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
            </el-form-item>
          </div>
        </el-card>

        <div class="side-stack">
          <el-card class="config-card" shadow="never">
            <template #header>
              <div class="card-head">
                <div>
                  <p class="card-kicker">Authentication</p>
                  <h3>Xác thực và chữ ký</h3>
                  <p class="card-description">Nếu bật xác thực, hãy nhập tài khoản quản trị dùng khi chương trình chính kết nối MQTT Server và giữ khóa ký đồng bộ với OTA.</p>
                </div>
                <el-tag :type="form.enable_auth ? 'warning' : 'info'" effect="plain" round>
                  {{ form.enable_auth ? 'Đã bật xác thực' : 'Truy cập ẩn danh' }}
                </el-tag>
              </div>
            </template>

            <div class="field-stack">
              <el-form-item label="Bật xác thực" prop="enable_auth">
                <div class="switch-field">
                  <div>
                    <div class="switch-title">Kiểm tra tên người dùng và mật khẩu MQTT</div>
                    <div class="field-help">Sau khi bật, hệ thống sẽ kiểm tra tên người dùng và mật khẩu khi client kết nối.</div>
                  </div>
                  <el-switch v-model="form.enable_auth" />
                </div>
              </el-form-item>

              <el-form-item label="Người dùng quản trị" prop="username">
                <el-input v-model="form.username" placeholder="Có thể để trống nếu chưa bật xác thực" />
              </el-form-item>

              <el-form-item label="Mật khẩu quản trị" prop="password">
                <el-input v-model="form.password" type="password" placeholder="Có thể để trống nếu chưa bật xác thực" show-password />
              </el-form-item>

              <el-form-item label="Khóa ký" prop="signature_key">
                <el-input v-model="form.signature_key" placeholder="Nhập khóa ký đồng bộ với OTA" />
                <div class="field-help">Khóa này cần khớp với khóa ký trong trang cấu hình OTA.</div>
              </el-form-item>
            </div>
          </el-card>

          <el-card class="config-card" shadow="never">
            <template #header>
              <div class="card-head">
                <div>
                  <p class="card-kicker">MQTTS</p>
                  <h3>Cấu hình TLS</h3>
                  <p class="card-description">Chỉ bật TLS và điền đường dẫn chứng chỉ khi thiết bị cần kết nối qua MQTTS.</p>
                </div>
                <el-tag :type="form.tls.enable ? 'success' : 'info'" effect="plain" round>
                  {{ form.tls.enable ? 'Đã bật TLS' : 'Chưa bật TLS' }}
                </el-tag>
              </div>
            </template>

            <div class="field-stack">
              <el-form-item label="Bật TLS" prop="tls.enable">
                <div class="switch-field">
                  <div>
                    <div class="switch-title">Cho phép thiết bị kết nối qua MQTTS</div>
                    <div class="field-help">Sau khi bật, hãy điền đồng thời cổng TLS, file chứng chỉ và file khóa.</div>
                  </div>
                  <el-switch v-model="form.tls.enable" />
                </div>
              </el-form-item>

              <el-form-item label="Cổng TLS" prop="tls.port">
                <el-input-number v-model="form.tls.port" :min="1" :max="65535" controls-position="right" style="width: 100%" />
              </el-form-item>

              <el-form-item label="File chứng chỉ" prop="tls.pem">
                <el-input v-model="form.tls.pem" placeholder="Ví dụ: certs/server.pem" />
              </el-form-item>

              <el-form-item label="File khóa" prop="tls.key">
                <el-input v-model="form.tls.key" placeholder="Ví dụ: certs/server.key" />
              </el-form-item>
            </div>
          </el-card>
        </div>
      </div>

      <div class="footer-bar">
        <p class="footer-note">
          Sau khi lưu, cấu hình MQTT Server mặc định sẽ được cập nhật; nếu đồng thời bật gửi OTA qua MQTT, hãy đảm bảo khóa ký khớp nhau.
        </p>
        <div class="footer-actions">
          <el-button plain :loading="loading" @click="loadConfig">Đặt lại theo cấu hình hiện tại</el-button>
          <el-button type="primary" :loading="saving" @click="handleSave">Lưu cấu hình</el-button>
        </div>
      </div>
    </el-form>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../utils/api'

const loading = ref(false)
const saving = ref(false)
const configId = ref(null)
const formRef = ref(null)

const createDefaultFormState = () => ({
  enable: true,
  listen_host: '0.0.0.0',
  listen_port: 1883,
  username: '',
  password: '',
  signature_key: 'xiaozhi_ota_signature_key',
  enable_auth: false,
  tls: {
    enable: false,
    port: 8883,
    pem: '',
    key: ''
  }
})

const form = reactive(createDefaultFormState())

const validateUsername = (_, value, callback) => {
  if (form.enable_auth && !String(value || '').trim()) {
    callback(new Error('Khi bật xác thực, tên người dùng quản trị không được để trống'))
    return
  }
  callback()
}

const validatePassword = (_, value, callback) => {
  if (form.enable_auth && !String(value || '').trim()) {
    callback(new Error('Khi bật xác thực, mật khẩu quản trị không được để trống'))
    return
  }
  callback()
}

const validateTlsPort = (_, value, callback) => {
  if (form.tls.enable && (!value || value < 1 || value > 65535)) {
    callback(new Error('Khi bật TLS, số cổng phải nằm trong khoảng 1-65535'))
    return
  }
  callback()
}

const validateTlsPem = (_, value, callback) => {
  if (form.tls.enable && !String(value || '').trim()) {
    callback(new Error('Khi bật TLS, đường dẫn file chứng chỉ không được để trống'))
    return
  }
  callback()
}

const validateTlsKey = (_, value, callback) => {
  if (form.tls.enable && !String(value || '').trim()) {
    callback(new Error('Khi bật TLS, đường dẫn file khóa không được để trống'))
    return
  }
  callback()
}

const rules = {
  listen_host: [{ required: true, message: 'Vui lòng nhập địa chỉ host lắng nghe', trigger: 'blur' }],
  listen_port: [
    { required: true, message: 'Vui lòng nhập cổng lắng nghe', trigger: 'blur' },
    { type: 'number', min: 1, max: 65535, message: 'Số cổng phải nằm trong khoảng 1-65535', trigger: 'blur' }
  ],
  username: [{ validator: validateUsername, trigger: 'blur' }],
  password: [{ validator: validatePassword, trigger: 'blur' }],
  signature_key: [{ required: true, message: 'Vui lòng nhập khóa ký', trigger: 'blur' }],
  'tls.port': [{ validator: validateTlsPort, trigger: 'blur' }],
  'tls.pem': [{ validator: validateTlsPem, trigger: 'blur' }],
  'tls.key': [{ validator: validateTlsKey, trigger: 'blur' }]
}

const serverReady = computed(() => {
  return Boolean(String(form.listen_host || '').trim() && Number(form.listen_port))
})

const resetForm = () => {
  Object.assign(form, createDefaultFormState())
}

const loadConfig = async () => {
  try {
    loading.value = true
    const response = await api.get('/admin/mqtt-server-configs')
    const configs = response.data?.data || []

    if (configs.length > 0) {
      const config = configs[0]
      configId.value = config.id

      let configData = {}
      try {
        configData = JSON.parse(config.json_data || '{}')
      } catch (error) {
        ElMessage.warning('Định dạng cấu hình MQTT Server bất thường, đã khôi phục giá trị mặc định')
        configData = {}
      }

      form.enable = configData.enable !== undefined ? configData.enable : true
      form.listen_host = String(configData.listen_host || '0.0.0.0')
      form.listen_port = Number(configData.listen_port) > 0 ? Number(configData.listen_port) : 1883
      form.username = String(configData.username || '')
      form.password = String(configData.password || '')
      form.signature_key = String(configData.signature_key || 'xiaozhi_ota_signature_key')
      form.enable_auth = configData.enable_auth !== undefined ? configData.enable_auth : false
      form.tls.enable = configData.tls?.enable !== undefined ? configData.tls.enable : false
      form.tls.port = Number(configData.tls?.port) > 0 ? Number(configData.tls.port) : 8883
      form.tls.pem = String(configData.tls?.pem || '')
      form.tls.key = String(configData.tls?.key || '')
    } else {
      configId.value = null
      resetForm()
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.message || 'Tải cấu hình MQTT Server thất bại')
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  saving.value = true
  try {
    const payload = {
      name: 'Cấu hình MQTT Server',
      config_id: 'mqtt_server_mqtt_server_config',
      provider: 'mqtt_server',
      json_data: JSON.stringify({
        enable: !!form.enable,
        listen_host: String(form.listen_host || '').trim(),
        listen_port: Number(form.listen_port),
        username: String(form.username || '').trim(),
        password: String(form.password || ''),
        signature_key: String(form.signature_key || '').trim(),
        enable_auth: !!form.enable_auth,
        tls: {
          enable: !!form.tls.enable,
          port: Number(form.tls.port),
          pem: String(form.tls.pem || '').trim(),
          key: String(form.tls.key || '').trim()
        }
      }),
      enabled: true,
      is_default: true
    }

    if (configId.value) {
      await api.put(`/admin/mqtt-server-configs/${configId.value}`, payload)
      ElMessage.success('Đã cập nhật cấu hình MQTT Server')
    } else {
      const response = await api.post('/admin/mqtt-server-configs', payload)
      configId.value = response.data?.data?.id || configId.value
      ElMessage.success('Đã lưu cấu hình MQTT Server')
    }

    await loadConfig()
  } catch (error) {
    ElMessage.error(error.response?.data?.message || 'Lưu cấu hình MQTT Server thất bại')
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.mqtt-server-config {
  padding: 0 24px 32px;
}

.config-form {
  display: grid;
  gap: 24px;
}

.config-layout {
  display: grid;
  grid-template-columns: minmax(0, 1.25fr) minmax(340px, 0.95fr);
  gap: 24px;
}

.side-stack {
  display: grid;
  gap: 24px;
}

.config-card {
  border: 1px solid rgba(255, 255, 255, 0.88);
  background: rgba(255, 255, 255, 0.88);
  box-shadow: var(--apple-shadow-md);
}

.card-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.card-kicker {
  display: block;
  margin: 0;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--apple-text-tertiary);
}

.card-head h3 {
  margin: 8px 0 0;
  font-size: 22px;
  line-height: 1.15;
  letter-spacing: -0.03em;
  color: var(--apple-text);
}

.card-description,
.field-help,
.footer-note {
  margin: 8px 0 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--apple-text-secondary);
}

.field-grid {
  display: grid;
  gap: 20px 18px;
}

.field-stack {
  display: grid;
  gap: 20px;
}

.switch-field {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px 18px;
  align-items: center;
}

.switch-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--apple-text);
}

.footer-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  padding: 0 4px;
}

.footer-note {
  max-width: 640px;
  margin: 0;
}

.footer-actions {
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 12px;
}

:deep(.el-card__header) {
  padding: 24px 24px 0;
  border-bottom: none;
  background: transparent;
}

:deep(.el-card__body) {
  padding: 24px;
}

:deep(.el-form-item) {
  margin-bottom: 0;
}

:deep(.el-form-item__label) {
  font-size: 14px;
  font-weight: 600;
  color: var(--apple-text);
}

@media (max-width: 1180px) {
  .config-layout {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .mqtt-server-config {
    padding: 0 16px 24px;
  }

  :deep(.el-card__body) {
    padding: 20px;
  }

  :deep(.el-card__header) {
    padding: 20px 20px 0;
  }

  .footer-bar {
    flex-direction: column;
    align-items: stretch;
  }

  .footer-actions {
    justify-content: stretch;
  }

  .footer-actions :deep(.el-button) {
    flex: 1;
  }
}
</style>
