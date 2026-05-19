<template>
  <div class="config-page">
    <div class="page-actions">
      <el-button
        type="warning"
        plain
        :loading="testingAll"
        @click="testAllConfigs"
        :disabled="!getEnabledConfigs().length"
      >
        Kiểm tra tất cả
      </el-button>
      <el-button type="primary" @click="showDialog = true">
        <el-icon><Plus /></el-icon>
        Thêm cấu hình
      </el-button>
    </div>

    <el-table :data="configs" style="width: 100%" v-loading="loading">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" label="Tên cấu hình" />
      <el-table-column prop="config_id" label="ID cấu hình" width="150" />
      <el-table-column prop="provider" label="Nhà cung cấp">
        <template #default="scope">
          {{ scope.row.provider }}
        </template>
      </el-table-column>
      <el-table-column prop="enabled" label="Trạng thái bật" width="80" align="center">
        <template #default="scope">
          <el-switch 
            v-model="scope.row.enabled" 
            @change="toggleEnable(scope.row)"
          />
        </template>
      </el-table-column>
      <el-table-column prop="is_default" label="Cấu hình mặc định" width="80" align="center">
        <template #default="scope">
          <el-switch 
            v-model="scope.row.is_default" 
            @change="toggleDefault(scope.row)"
            :disabled="scope.row.is_default && getEnabledConfigs().length === 1"
          />
        </template>
      </el-table-column>
      <el-table-column label="Kết quả kiểm tra" width="120" align="center">
        <template #default="scope">
          <template v-if="testResults[scope.row.config_id]">
            <el-tooltip v-if="testResults[scope.row.config_id].ok" :content="formatTestResultTip(testResults[scope.row.config_id])" placement="top">
              <span class="test-result test-ok">{{ formatTestResultLabel(testResults[scope.row.config_id]) }}</span>
            </el-tooltip>
            <el-tooltip v-else :content="testResults[scope.row.config_id].message" placement="top" :show-after="200">
              <span class="test-result test-err">Lỗi</span>
            </el-tooltip>
          </template>
          <span v-else class="test-result test-none">-</span>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="Thời gian tạo" width="180">
        <template #default="scope">
          {{ formatDate(scope.row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="Thao tác" width="260">
        <template #default="scope">
          <el-button size="small" @click="editConfig(scope.row)">Sửa</el-button>
          <el-button
            size="small"
            type="warning"
            :loading="testingId === scope.row.config_id"
            @click="testConfig(scope.row, 'asr')"
          >
            Kiểm tra
          </el-button>
          <el-button
            size="small"
            type="danger"
            @click="deleteConfig(scope.row.id)"
          >
            Xóa
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Hộp thoại thêm/chỉnh sửa cấu hình -->
    <el-dialog
      v-model="showDialog"
      :title="editingConfig ? 'Chỉnh sửa cấu hình ASR' : 'Thêm cấu hình ASR'"
      width="720px"
      @close="handleDialogClose"
    >
      <ASRConfigForm ref="formRef" :model="form" :rules="rules" />
      
      <template #footer>
        <el-button @click="handleDialogClose">Hủy</el-button>
        <el-button type="warning" plain @click="testCurrentConfig" :loading="testingCurrent">
          Kiểm tra
        </el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">
          Lưu
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { testSingleConfig, testWithData, parseJsonData } from '../../utils/configTest'
import ASRConfigForm from './forms/ASRConfigForm.vue'
import { resolveASRProvider } from './forms/configProviderUtils'

const configs = ref([])
const testingId = ref(null)
const testingAll = ref(false)
const testingCurrent = ref(false)
const testResults = ref({})
const loading = ref(false)
const saving = ref(false)
const showDialog = ref(false)
const editingConfig = ref(null)
const formRef = ref()

const validateAliyunPcm = (rule, value, callback) => {
  if (value !== 'pcm') {
    callback(new Error('Định dạng phải là pcm'))
    return
  }
  callback()
}

const validateAliyun16000 = (rule, value, callback) => {
  if (Number(value) !== 16000) {
    callback(new Error('Sample rate phải là 16000'))
    return
  }
  callback()
}

const form = reactive({
  name: '',
  config_id: '',
  provider: '',
  is_default: false,
  enabled: true,
  wyoming_vietnamese_asr: {
    base_url: 'http://127.0.0.1:8082',
    sample_rate: 16000,
    timeout_ms: 30000
  },
  funasr: {
    host: 'localhost',
    port: 10095,
    mode: 'offline',
    sample_rate: 16000,
    chunk_size: [5, 10, 5],
    chunk_interval: 10,
    max_connections: 100,
    timeout: 30,
    auto_end: false
  },
  aliyun_funasr: {
    api_key: '',
    ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/inference/',
    model: 'fun-asr-realtime',
    format: 'pcm',
    sample_rate: 16000,
    vocabulary_id: '',
    disfluency_removal_enabled: false,
    timeout: 30
  },
  doubao: {
    appid: '',
    access_token: '',
    ws_url: 'wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async',
    model_name: 'bigmodel',
    end_window_size: 800,
    enable_punc: true,
    enable_itn: true,
    enable_ddc: false,
    chunk_duration: 200,
    timeout: 30
  },
  aliyun_qwen3: {
    api_key: '',
    ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/realtime',
    model: 'qwen3-asr-flash-realtime',
    format: 'pcm',
    sample_rate: 16000,
    language: 'zh',
    auto_end: false,
    vad_threshold: 0.0,
    vad_silence_ms: 400,
    timeout: 30
  },
  xunfei: {
    appid: '',
    api_key: '',
    api_secret: '',
    host: 'iat-api.xfyun.cn',
    path: '/v2/iat',
    domain: 'iat',
    language: 'zh_cn',
    accent: 'mandarin',
    sample_rate: 16000,
    timeout: 30
  }
})

// Dùng rule động theo provider hiện tại để tránh các trường doubao/funasr đang ẩn vẫn bị bắt buộc, làm thao tác lưu không gửi request
const rules = computed(() => {
  const base = {
    name: [{ required: true, message: 'Vui lòng nhập tên cấu hình', trigger: 'blur' }],
    config_id: [{ required: true, message: 'Vui lòng nhập ID cấu hình', trigger: 'blur' }],
    provider: [{ required: true, message: 'Vui lòng chọn nhà cung cấp', trigger: 'change' }]
  }
  if (form.provider === 'wyoming_vietnamese_asr') {
    return {
      ...base,
      'wyoming_vietnamese_asr.base_url': [{ required: true, message: 'Vui lòng nhập URL dịch vụ Go', trigger: 'blur' }],
      'wyoming_vietnamese_asr.sample_rate': [{ required: true, message: 'Vui lòng chọn sample rate', trigger: 'change' }],
      'wyoming_vietnamese_asr.timeout_ms': [{ required: true, message: 'Vui lòng nhập timeout', trigger: 'blur' }]
    }
  }
  if (form.provider === 'funasr') {
    return {
      ...base,
      'funasr.host': [{ required: true, message: 'Vui lòng nhập địa chỉ host', trigger: 'blur' }],
      'funasr.port': [{ required: true, message: 'Vui lòng nhập cổng', trigger: 'blur' }],
      'funasr.mode': [{ required: true, message: 'Vui lòng chọn chế độ', trigger: 'change' }],
      'funasr.sample_rate': [{ required: true, message: 'Vui lòng chọn sample rate', trigger: 'change' }],
      'funasr.chunk_size': [{ required: true, message: 'Vui lòng nhập kích thước chunk', trigger: 'blur' }],
      'funasr.chunk_interval': [{ required: true, message: 'Vui lòng nhập khoảng cách chunk', trigger: 'blur' }],
      'funasr.max_connections': [{ required: true, message: 'Vui lòng nhập số kết nối tối đa', trigger: 'blur' }],
      'funasr.timeout': [{ required: true, message: 'Vui lòng nhập thời gian timeout', trigger: 'blur' }]
    }
  }
  if (form.provider === 'aliyun_funasr') {
    return {
      ...base,
      'aliyun_funasr.ws_url': [{ required: true, message: 'Vui lòng nhập WS URL', trigger: 'blur' }],
      'aliyun_funasr.model': [{ required: true, message: 'Vui lòng nhập tên model', trigger: 'blur' }],
      'aliyun_funasr.format': [
        { required: true, message: 'Vui lòng chọn định dạng âm thanh', trigger: 'change' },
        { validator: validateAliyunPcm, trigger: 'change' }
      ],
      'aliyun_funasr.sample_rate': [
        { required: true, message: 'Vui lòng chọn sample rate', trigger: 'change' },
        { validator: validateAliyun16000, trigger: 'change' }
      ],
      'aliyun_funasr.timeout': [{ required: true, message: 'Vui lòng nhập thời gian timeout', trigger: 'blur' }]
    }
  }
  if (form.provider === 'doubao') {
    return {
      ...base,
      'doubao.appid': [{ required: true, message: 'Vui lòng nhập App ID', trigger: 'blur' }],
      'doubao.access_token': [{ required: true, message: 'Vui lòng nhập access token', trigger: 'blur' }],
      'doubao.ws_url': [{ required: true, message: 'Vui lòng nhập WebSocket URL', trigger: 'blur' }],
      'doubao.resource_id': [{ required: true, message: 'Vui lòng chọn quy cách tài nguyên', trigger: 'change' }],
      'doubao.end_window_size': [{ required: true, message: 'Vui lòng nhập kích thước cửa sổ kết thúc', trigger: 'blur' }],
      'doubao.timeout': [{ required: true, message: 'Vui lòng nhập thời gian timeout', trigger: 'blur' }]
    }
  }
  if (form.provider === 'aliyun_qwen3') {
    return {
      ...base,
      'aliyun_qwen3.ws_url': [{ required: true, message: 'Vui lòng nhập WS URL', trigger: 'blur' }],
      'aliyun_qwen3.model': [{ required: true, message: 'Vui lòng nhập tên model', trigger: 'blur' }],
      'aliyun_qwen3.format': [{ required: true, message: 'Vui lòng chọn định dạng âm thanh', trigger: 'change' }],
      'aliyun_qwen3.sample_rate': [{ required: true, message: 'Vui lòng chọn sample rate', trigger: 'change' }],
      'aliyun_qwen3.language': [{ required: true, message: 'Vui lòng nhập ngôn ngữ', trigger: 'blur' }],
      'aliyun_qwen3.timeout': [{ required: true, message: 'Vui lòng nhập thời gian timeout', trigger: 'blur' }]
    }
  }
  if (form.provider === 'xunfei') {
    return {
      ...base,
      'xunfei.appid': [{ required: true, message: 'Vui lòng nhập App ID', trigger: 'blur' }],
      'xunfei.api_key': [{ required: true, message: 'Vui lòng nhập API Key', trigger: 'blur' }],
      'xunfei.api_secret': [{ required: true, message: 'Vui lòng nhập API Secret', trigger: 'blur' }],
      'xunfei.host': [{ required: true, message: 'Vui lòng nhập Host', trigger: 'blur' }],
      'xunfei.path': [{ required: true, message: 'Vui lòng nhập Path', trigger: 'blur' }],
      'xunfei.sample_rate': [{ required: true, message: 'Vui lòng nhập sample rate', trigger: 'change' }],
      'xunfei.timeout': [{ required: true, message: 'Vui lòng nhập thời gian timeout', trigger: 'blur' }]
    }
  }
  return base
})

const loadConfigs = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/asr-configs')
    configs.value = (response.data.data || []).map(normalizeASRConfigRow)
  } catch (error) {
    ElMessage.error('Tải cấu hình thất bại')
  } finally {
    loading.value = false
  }
}

function normalizeASRConfigRow(row) {
  const data = parseJsonData(row?.json_data)
  return {
    ...row,
    provider: resolveASRProvider(row?.provider, row?.config_id, data)
  }
}

const editConfig = (config) => {
  config = normalizeASRConfigRow(config)
  editingConfig.value = config
  form.name = config.name
  form.config_id = config.config_id
  form.provider = config.provider
  form.is_default = config.is_default
  form.enabled = config.enabled
  
  // Parse JSON cấu hình và điền vào các trường tương ứng
  try {
    const configObj = JSON.parse(config.json_data || '{}')
    
    // Tương thích cả định dạng cũ lẫn mới: kiểm tra dữ liệu đang bọc theo provider hay là nội dung trực tiếp
    if (configObj.wyoming_vietnamese_asr) {
      form.wyoming_vietnamese_asr = { ...form.wyoming_vietnamese_asr, ...configObj.wyoming_vietnamese_asr }
    } else if (config.provider === 'wyoming_vietnamese_asr' && (configObj.base_url || configObj.api_url || configObj.url)) {
      form.wyoming_vietnamese_asr = { ...form.wyoming_vietnamese_asr, ...configObj }
    } else if (configObj.funasr) {
      // Định dạng cũ: có lớp provider bao ngoài
      const funasrConfig = { ...form.funasr, ...configObj.funasr }
      // Tương thích chunk_size: nếu là số đơn hoặc sai định dạng thì đổi về giá trị mặc định [5, 10, 5]
      if (typeof funasrConfig.chunk_size === 'number') {
        funasrConfig.chunk_size = [5, 10, 5]
      } else if (!Array.isArray(funasrConfig.chunk_size) || funasrConfig.chunk_size.length !== 3) {
        funasrConfig.chunk_size = [5, 10, 5]
      }
      form.funasr = funasrConfig
    } else if (configObj.aliyun_funasr) {
      // Định dạng cũ: có lớp provider bao ngoài
      form.aliyun_funasr = { ...form.aliyun_funasr, ...configObj.aliyun_funasr }
    } else if (configObj.doubao) {
      // Định dạng cũ: có lớp provider bao ngoài
      form.doubao = { ...form.doubao, ...configObj.doubao }
    } else if (config.provider === 'funasr' && configObj.host) {
      // Định dạng mới: chứa trực tiếp nội dung cấu hình
      const funasrConfig = { ...form.funasr, ...configObj }
      // Tương thích chunk_size: nếu là số đơn hoặc sai định dạng thì đổi về giá trị mặc định [5, 10, 5]
      if (typeof funasrConfig.chunk_size === 'number') {
        funasrConfig.chunk_size = [5, 10, 5]
      } else if (!Array.isArray(funasrConfig.chunk_size) || funasrConfig.chunk_size.length !== 3) {
        funasrConfig.chunk_size = [5, 10, 5]
      }
      form.funasr = funasrConfig
    } else if (config.provider === 'aliyun_funasr' && (configObj.ws_url || configObj.model || configObj.api_key)) {
      // Định dạng mới: chứa trực tiếp nội dung cấu hình
      form.aliyun_funasr = { ...form.aliyun_funasr, ...configObj }
    } else if (config.provider === 'doubao' && (configObj.appid || configObj.access_token)) {
      // Định dạng mới: chứa trực tiếp nội dung cấu hình
      form.doubao = { ...form.doubao, ...configObj }
    } else if (configObj.aliyun_qwen3) {
      // Định dạng cũ: có lớp provider bao ngoài
      form.aliyun_qwen3 = { ...form.aliyun_qwen3, ...configObj.aliyun_qwen3 }
    } else if (config.provider === 'aliyun_qwen3' && (configObj.ws_url || configObj.model || configObj.api_key)) {
      // Định dạng mới: chứa trực tiếp nội dung cấu hình
      form.aliyun_qwen3 = { ...form.aliyun_qwen3, ...configObj }
    } else if (configObj.xunfei) {
      form.xunfei = { ...form.xunfei, ...configObj.xunfei }
    } else if (config.provider === 'xunfei' && (configObj.appid || configObj.api_key || configObj.api_secret)) {
      form.xunfei = { ...form.xunfei, ...configObj }
    }
  } catch (error) {
    console.error('Parse JSON cấu hình thất bại:', error)
  }
  
  showDialog.value = true
}

const handleSave = async () => {
  if (!formRef.value) {
    ElMessage.warning('Form chưa sẵn sàng, vui lòng thử lại sau')
    return
  }
  await formRef.value.validate(async (valid) => {
    if (valid) {
      saving.value = true
      try {
        // Nếu đang thêm cấu hình mới và hiện chưa có cấu hình nào thì tự đặt làm mặc định
        const isFirstConfig = !editingConfig.value && configs.value.length === 0
        
        const configData = {
          name: form.name,
          config_id: form.config_id,
          provider: form.provider,
          is_default: isFirstConfig || form.is_default, // Khi thêm bản ghi đầu tiên thì tự đặt làm mặc định
          enabled: form.enabled !== undefined ? form.enabled : true, // Đảm bảo luôn có trường enabled
          json_data: formRef.value.getJsonData()
        }
        
        if (editingConfig.value) {
          await api.put(`/admin/asr-configs/${editingConfig.value.id}`, configData)
          ElMessage.success('Cập nhật cấu hình thành công')
        } else {
          await api.post('/admin/asr-configs', configData)
          ElMessage.success('Tạo cấu hình thành công')
        }
        
        showDialog.value = false
        loadConfigs()
      } catch (error) {
        const msg = error.response?.data?.error || error.response?.data?.message || error.message
        ElMessage.error('Lưu thất bại: ' + msg)
      } finally {
        saving.value = false
      }
    }
  })
}

const toggleEnable = async (config) => {
  try {
    await api.post(`/admin/configs/${config.id}/toggle`)
    ElMessage.success(`${config.enabled ? 'Bật' : 'Tắt'}thành công`)
  } catch (error) {
    // Khôi phục trạng thái switch
    config.enabled = !config.enabled
    ElMessage.error('Thao tác thất bại')
  }
}

const toggleDefault = async (config) => {
  try {
    if (!config.enabled) {
      ElMessage.warning('Vui lòng bật cấu hình trước khi đặt làm mặc định')
      config.is_default = false
      return
    }
    
    const configData = {
      name: config.name,
      config_id: config.config_id,
      provider: config.provider,
      is_default: config.is_default,
      enabled: config.enabled,
      json_data: config.json_data
    }
    
    await api.put(`/admin/asr-configs/${config.id}`, configData)
    ElMessage.success(config.is_default ? 'Đặt làm mặc định thành công' : 'HủyMặc địnhthành công')
    
    // Làm mới danh sách để cập nhật trạng thái mặc định của các cấu hình khác
    loadConfigs()
  } catch (error) {
    // Khôi phục trạng thái switch
    config.is_default = !config.is_default
    ElMessage.error('Thao tác thất bại')
  }
}

const getEnabledConfigs = () => {
  return configs.value.filter(config => config.enabled)
}

function formatTestResultLabel(r) {
  if (!r?.ok) return 'Lỗi'
  return r.first_packet_ms != null ? `Đạt ${r.first_packet_ms}ms` : 'Đạt'
}
function formatTestResultTip(r) {
  if (!r?.ok) return ''
  return r.first_packet_ms != null ? `Đạt, thời gian ${r.first_packet_ms}ms` : 'Đạt'
}
function formatTestMessage(result) {
  const base = result.message || ''
  return result.first_packet_ms != null ? `${base} ${result.first_packet_ms}ms` : base
}

const testConfig = async (row, type) => {
  testingId.value = row.config_id
  try {
    const result = await testSingleConfig(type, row.config_id)
    testResults.value = { ...testResults.value, [row.config_id]: result }
    if (result.ok) {
      ElMessage.success(`${row.name || row.config_id}：${formatTestMessage(result)}`)
    } else {
      ElMessage.warning(`${row.name || row.config_id}：${result.message}`)
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.error || 'Kiểm traYêu cầu thất bại')
  } finally {
    testingId.value = null
  }
}

const testAllConfigs = async () => {
  const list = getEnabledConfigs()
  if (!list.length) {
    ElMessage.warning('Không có cấu hình nào đang bật')
    return
  }
  testingAll.value = true
  testResults.value = {}
  let okCount = 0
  try {
    for (const row of list) {
      try {
        const result = await testSingleConfig('asr', row.config_id)
        testResults.value = { ...testResults.value, [row.config_id]: result }
        if (result.ok) okCount++
      } catch (_) {
        testResults.value = { ...testResults.value, [row.config_id]: { ok: false, message: 'Yêu cầu thất bại' } }
      }
    }
    ElMessage.success(`Đã hoàn tất kiểm tra tất cả: ${okCount}/${list.length} Đạt`)
  } catch (err) {
    ElMessage.error(err.response?.data?.error || 'Kiểm traYêu cầu thất bại')
  } finally {
    testingAll.value = false
  }
}

const testCurrentConfig = async () => {
  if (!formRef.value) return
  try {
    await formRef.value.validate()
  } catch (_) {
    return
  }
  const configId = form.config_id?.trim()
  if (!configId) {
    ElMessage.warning('Vui lòng nhập ID cấu hình')
    return
  }
  const payload = {
    ...parseJsonData(formRef.value.getJsonData()),
    name: form.name,
    config_id: configId,
    provider: form.provider,
    is_default: form.is_default
  }
  testingCurrent.value = true
  try {
    const result = await testWithData('asr', { [configId]: payload })
    if (result.ok) {
      ElMessage.success(formatTestMessage(result) || 'Kiểm traĐạt')
    } else {
      ElMessage.warning(result.message || 'Kiểm tra chưa đạt')
    }
  } catch (err) {
    ElMessage.error(err.response?.data?.error || 'Kiểm traYêu cầu thất bại')
  } finally {
    testingCurrent.value = false
  }
}

const deleteConfig = async (id) => {
  try {
    await ElMessageBox.confirm('Bạn có chắc muốn xóa cấu hình này không?', 'Gợi ý', {
      confirmButtonText: 'Xác nhận',
      cancelButtonText: 'Hủy',
      type: 'warning'
    })
    
    await api.delete(`/admin/asr-configs/${id}`)
    ElMessage.success('Xóathành công')
    loadConfigs()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('Xóa thất bại')
    }
  }
}

const resetForm = () => {
  editingConfig.value = null
  form.name = ''
  form.config_id = ''
  form.provider = ''
  form.is_default = false
  form.enabled = true
  form.wyoming_vietnamese_asr = {
    base_url: 'http://127.0.0.1:8082',
    sample_rate: 16000,
    timeout_ms: 30000
  }
  form.funasr = {
    host: 'localhost',
    port: 10095,
    mode: 'offline',
    sample_rate: 16000,
    chunk_size: [5, 10, 5],
    chunk_interval: 10,
    max_connections: 100,
    timeout: 30,
    auto_end: false
  }
  form.aliyun_funasr = {
    api_key: '',
    ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/inference/',
    model: 'fun-asr-realtime',
    format: 'pcm',
    sample_rate: 16000,
    vocabulary_id: '',
    disfluency_removal_enabled: false,
    timeout: 30
  }
  form.doubao = {
    appid: '',
    access_token: '',
    ws_url: 'wss://openspeech.bytedance.com/api/v3/sauc/bigmodel_async',
    resource_id: 'volc.bigasr.sauc.duration',
    model_name: 'bigmodel',
    end_window_size: 800,
    enable_punc: true,
    enable_itn: true,
    enable_ddc: false,
    chunk_duration: 200,
    timeout: 30
  }
  form.aliyun_qwen3 = {
    api_key: '',
    ws_url: 'wss://dashscope.aliyuncs.com/api-ws/v1/realtime',
    model: 'qwen3-asr-flash-realtime',
    format: 'pcm',
    sample_rate: 16000,
    language: 'zh',
    auto_end: false,
    vad_threshold: 0.0,
    vad_silence_ms: 400,
    timeout: 30
  }
  form.xunfei = {
    appid: '',
    api_key: '',
    api_secret: '',
    host: 'iat-api.xfyun.cn',
    path: '/v2/iat',
    domain: 'iat',
    language: 'zh_cn',
    accent: 'mandarin',
    sample_rate: 16000,
    timeout: 30
  }
}

const handleDialogClose = () => {
  showDialog.value = false
  resetForm()
  if (formRef.value) {
    formRef.value.resetFields()
  }
}

const formatDate = (dateString) => {
  return new Date(dateString).toLocaleString('zh-CN')
}

onMounted(() => {
  loadConfigs()
})
</script>

<style scoped>
.config-page {
  padding: 20px;
  background: rgba(255, 255, 255, 0.88);
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.page-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.test-result { font-size: 12px; }
.test-result.test-ok { color: var(--el-color-success); }
.test-result.test-err { color: var(--el-color-danger); cursor: help; }
.test-result.test-none { color: var(--el-text-color-placeholder); }
</style>
