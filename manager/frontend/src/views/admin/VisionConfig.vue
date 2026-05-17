<template>
  <div class="config-page">
    <!-- Cấu hình cơ bản部分 -->
    <el-card class="base-config-card" style="margin-bottom: 20px;">
      <template #header>
        <div class="card-header">
          <span>Cấu hình cơ bản</span>
        </div>
      </template>
      
      <el-form
        ref="baseFormRef"
        :model="baseForm"
        :rules="baseRules"
        label-width="120px"
        style="max-width: 600px;"
      >
        <el-form-item label="Bật xác thực" prop="enable_auth">
          <el-switch v-model="baseForm.enable_auth" />
          <div class="form-tip">Có bật xác thực cho API nhận diện hình ảnh hay không</div>
        </el-form-item>
        
        <el-form-item label="Vision URL" prop="vision_url">
          <el-input 
            v-model="baseForm.vision_url" 
            placeholder="Vui lòng nhập địa chỉ Vision API"
            style="width: 100%;"
          />
          <div class="form-tip">Địa chỉ HTTP trả về cho client để nhận diện hình ảnh</div>
        </el-form-item>
        
        <el-form-item>
          <el-button type="primary" @click="saveBaseConfig" :loading="baseSaving">
            LưuCấu hình cơ bản
          </el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 配置列表部分 -->
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Danh sách cấu hình model</span>
          <el-button type="primary" @click="showDialog = true">
            <el-icon><Plus /></el-icon>
            Thêm cấu hình
          </el-button>
        </div>
      </template>

      <el-table :data="configs" style="width: 100%" v-loading="loading">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="Tên cấu hình" />
        <el-table-column prop="provider" label="Nhà cung cấp" />
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
        <el-table-column prop="created_at" label="Thời gian tạo" width="180">
          <template #default="scope">
            {{ formatDate(scope.row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="Thao tác" width="180">
          <template #default="scope">
            <el-button size="small" @click="editConfig(scope.row)">Sửa</el-button>
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
    </el-card>

    <!-- Hộp thoại thêm/chỉnh sửa cấu hình -->
    <el-dialog
      v-model="showDialog"
      :title="editingConfig ? 'Chỉnh sửa cấu hình Vision' : 'Thêm cấu hình Vision'"
      width="700px"
      @close="handleDialogClose"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="120px"
      >
        <el-form-item label="Nhà cung cấp" prop="provider">
          <el-select v-model="form.provider" placeholder="Vui lòng chọn nhà cung cấp" style="width: 100%">
            <el-option label="Aliyun Vision" value="aliyun_vision" />
            <el-option label="Doubao Vision" value="doubao_vision" />
          </el-select>
        </el-form-item>
        
        <el-form-item label="Tên cấu hình" prop="name">
          <el-input v-model="form.name" placeholder="Vui lòng nhập tên cấu hình" />
        </el-form-item>
        
        <el-form-item label="Loại" prop="type">
          <el-input v-model="form.type" placeholder="Vui lòng nhập loại" />
        </el-form-item>
        
        <el-form-item label="Tên model" prop="model_name">
          <el-input v-model="form.model_name" placeholder="Vui lòng nhập tên model" />
        </el-form-item>
        
        <el-form-item label="API key" prop="api_key">
          <el-input v-model="form.api_key" type="password" placeholder="Vui lòng nhập API key" show-password />
        </el-form-item>
        
        <el-form-item label="Base URL" prop="base_url">
          <el-input v-model="form.base_url" placeholder="Vui lòng nhập Base URL" />
        </el-form-item>
        
        <el-form-item label="Số token tối đa" prop="max_tokens">
          <el-input-number v-model="form.max_tokens" :min="1" :max="100000" placeholder="Vui lòng nhập số token tối đa" style="width: 100%" />
        </el-form-item>
        
        <el-form-item label="Temperature" prop="temperature">
          <el-input-number v-model="form.temperature" :min="0" :max="2" :step="0.1" placeholder="Vui lòng nhập temperature" style="width: 100%" />
        </el-form-item>
        
        <el-form-item label="Top P" prop="top_p">
          <el-input-number v-model="form.top_p" :min="0" :max="1" :step="0.1" placeholder="Vui lòng nhập Top P" style="width: 100%" />
        </el-form-item>
        
        <el-form-item label="Thời gian timeout (giây)" prop="timeout">
          <el-input-number v-model="form.timeout" :min="1" :max="300" placeholder="Vui lòng nhập thời gian timeout" style="width: 100%" />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <el-button @click="handleDialogClose">Hủy</el-button>
        <el-button type="primary" @click="handleSave" :loading="saving">
          Lưu
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import api from '../../utils/api'
import { resolveVisionProvider } from './forms/configProviderUtils'

const configs = ref([])
const loading = ref(false)
const saving = ref(false)
const baseSaving = ref(false)
const showDialog = ref(false)
const editingConfig = ref(null)
const formRef = ref()
const baseFormRef = ref()

// Cấu hình cơ bản表单
const baseForm = reactive({
  enable_auth: false,
  vision_url: ''
})

// Cấu hình cơ bản验证规则
const baseRules = {
  vision_url: [
    { required: true, message: 'Vui lòng nhập Vision URL', trigger: 'blur' },
    { type: 'url', message: 'Vui lòng nhập URL hợp lệ', trigger: 'blur' }
  ]
}

const form = reactive({
  name: '',
  provider: 'aliyun_vision',
  is_default: false,
  enabled: true,
  type: 'openai',
  model_name: 'qwen-vl-max',
  api_key: '',
  base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
  max_tokens: 1000,
  temperature: 0.1,
  top_p: 0.1,
  timeout: 30
})

const generateConfig = () => {
  return JSON.stringify({
    provider: form.provider,
    type: form.type,
    model_name: form.model_name,
    api_key: form.api_key,
    base_url: form.base_url,
    max_tokens: form.max_tokens,
    temperature: form.temperature,
    top_p: form.top_p,
    timeout: form.timeout
  })
}

const rules = {
  name: [{ required: true, message: 'Vui lòng nhập tên cấu hình', trigger: 'blur' }],
  provider: [{ required: true, message: 'Vui lòng chọn nhà cung cấp', trigger: 'change' }],
  type: [{ required: true, message: 'Vui lòng nhập loại', trigger: 'blur' }],
  model_name: [{ required: true, message: 'Vui lòng nhập tên model', trigger: 'blur' }],
  api_key: [{ required: true, message: 'Vui lòng nhập API key', trigger: 'blur' }],
  base_url: [
    { required: true, message: 'Vui lòng nhập Base URL', trigger: 'blur' },
    { type: 'url', message: 'Vui lòng nhập URL hợp lệ', trigger: 'blur' }
  ],
  max_tokens: [{ required: true, message: 'Vui lòng nhập số token tối đa', trigger: 'blur' }],
  timeout: [{ required: true, message: 'Vui lòng nhập thời gian timeout', trigger: 'blur' }]
}

const parseJsonData = (jsonData) => {
  try {
    return JSON.parse(jsonData || '{}')
  } catch (error) {
    return {}
  }
}

const normalizeVisionConfigRow = (config) => {
  const data = parseJsonData(config.json_data)
  return {
    ...config,
    provider: resolveVisionProvider(config.provider, config.config_id, data)
  }
}

// 加载Cấu hình cơ bản
const loadBaseConfig = async () => {
  try {
    const response = await api.get('/admin/vision-base-config')
    const data = response.data.data || {}
    baseForm.enable_auth = data.enable_auth || false
    baseForm.vision_url = data.vision_url || ''
  } catch (error) {
    console.error('Tải cấu hình cơ bản thất bại:', error)
  }
}

// LưuCấu hình cơ bản
const saveBaseConfig = async () => {
  if (!baseFormRef.value) return
  
  await baseFormRef.value.validate(async (valid) => {
    if (valid) {
      baseSaving.value = true
      try {
        await api.put('/admin/vision-base-config', {
          enable_auth: baseForm.enable_auth,
          vision_url: baseForm.vision_url
        })
        ElMessage.success('Cấu hình cơ bảnLưuthành công')
      } catch (error) {
        ElMessage.error('Lưu thất bại, vui lòng kiểm tra kết nối mạng và nội dung nhập')
      } finally {
        baseSaving.value = false
      }
    }
  })
}

const loadConfigs = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/vision-configs')
    // 过滤掉vision_base配置，确保不在列表Trung bình显示
    const allConfigs = response.data.data || []
    configs.value = allConfigs.filter(config => config.config_id !== 'vision_base').map(normalizeVisionConfigRow)
  } catch (error) {
    ElMessage.error('Tải cấu hình thất bại')
  } finally {
    loading.value = false
  }
}

const editConfig = (config) => {
  const normalizedConfig = normalizeVisionConfigRow(config)
  editingConfig.value = normalizedConfig
  form.name = normalizedConfig.name
  form.provider = normalizedConfig.provider
  form.is_default = normalizedConfig.is_default
  form.enabled = normalizedConfig.enabled
  
  try {
    const configData = parseJsonData(normalizedConfig.json_data)
    form.type = configData.type || ''
    form.model_name = configData.model_name || ''
    form.api_key = configData.api_key || ''
    form.base_url = configData.base_url || ''
    form.max_tokens = configData.max_tokens || 4096
    form.temperature = configData.temperature !== undefined ? configData.temperature : 0.7
    form.top_p = configData.top_p !== undefined ? configData.top_p : 0.9
    form.timeout = configData.timeout || 30
  } catch (error) {
    console.error('Parse cấu hình thất bại:', error)
    ElMessage.warning('Định dạng cấu hình lỗi, đã đặt lại về mặc định')
  }
  
  showDialog.value = true
}

const handleSave = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (valid) {
      saving.value = true
      try {
        const isFirstConfig = !editingConfig.value && configs.value.length === 0
        
        const configData = {
          name: form.name,
          provider: form.provider,
          is_default: isFirstConfig || form.is_default,
          enabled: form.enabled !== undefined ? form.enabled : true,
          json_data: generateConfig()
        }
        
        if (editingConfig.value) {
          await api.put(`/admin/vision-configs/${editingConfig.value.id}`, configData)
          ElMessage.success('Cập nhật thành công')
        } else {
          await api.post('/admin/vision-configs', configData)
          ElMessage.success('Thêm thành công')
        }
        
        showDialog.value = false
        loadConfigs()
      } catch (error) {
        ElMessage.error('Lưu thất bại, vui lòng kiểm tra kết nối mạng và nội dung nhập')
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
      provider: config.provider,
      is_default: config.is_default,
      enabled: config.enabled,
      json_data: config.json_data
    }
    
    await api.put(`/admin/vision-configs/${config.id}`, configData)
    ElMessage.success(config.is_default ? 'Đặt làm mặc định thành công' : 'HủyMặc địnhthành công')
    loadConfigs()
  } catch (error) {
    config.is_default = !config.is_default
    ElMessage.error('Thao tác thất bại')
  }
}

const getEnabledConfigs = () => {
  return configs.value.filter(config => config.enabled)
}

const deleteConfig = async (id) => {
  try {
    await ElMessageBox.confirm('Bạn có chắc muốn xóa cấu hình này không?', 'Gợi ý', {
      confirmButtonText: 'Xác nhận',
      cancelButtonText: 'Hủy',
      type: 'warning'
    })
    
    await api.delete(`/admin/vision-configs/${id}`)
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
  Object.assign(form, {
    name: '',
    provider: 'aliyun_vision',
    is_default: false,
    enabled: true,
    type: 'openai',
    model_name: 'qwen-vl-max',
    api_key: '',
    base_url: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    max_tokens: 1000,
    temperature: 0.1,
    top_p: 0.1,
    timeout: 30
  })
  formRef.value?.clearValidate()
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
  loadBaseConfig()
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

.base-config-card {
  background: #f8f9fa;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-weight: 600;
  color: #333;
}

.form-tip {
  font-size: 12px;
  color: #666;
  margin-top: 4px;
  line-height: 1.4;
}
</style>
