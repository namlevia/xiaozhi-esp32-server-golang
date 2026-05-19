<template>
  <div class="config-page">
    <el-card v-loading="loading" class="config-card">
      <el-alert
        title="Gợi ý"
        type="info"
        :closable="false"
        show-icon
        style="margin-bottom: 20px;"
      >
        <template #default>
          Nếu triển khai bằng docker-compose, hệ thống sẽ đọc địa chỉ API từ biến môi trường nên không cần cấu hình.
        </template>
      </el-alert>
      
      <el-form
        ref="formRef"
        :model="form"
        :rules="rules"
        label-width="120px"
      >
        <el-form-item label="Địa chỉ dịch vụ" prop="base_url">
          <el-input 
            v-model="form.base_url" 
            placeholder="Vui lòng nhập địa chỉ dịch vụ HTTP, ví dụ: http://192.168.208.214:8080"
            style="width: 100%"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            Vui lòng nhập địa chỉ HTTP, hệ thống sẽ tự động chuyển thành địa chỉ WebSocket
          </div>
        </el-form-item>
        
        <el-form-item label="Ngưỡng nhận diện" prop="threshold">
          <el-input-number 
            v-model="form.threshold" 
            :min="0" 
            :max="1" 
            :step="0.1" 
            :precision="2"
            placeholder="0.4"
            style="width: 100%"
          />
          <div class="form-tip">
            <el-icon><InfoFilled /></el-icon>
            Ngưỡng nhận diện người nói, phạm vi 0.0-1.0, mặc định 0.4. Giá trị càng lớn càng nghiêm ngặt
          </div>
        </el-form-item>
        
        <el-form-item label="Trạng thái bật">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      
      <div class="form-actions">
        <el-button type="primary" @click="handleSave" :loading="saving">
          Lưu cấu hình
        </el-button>
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { InfoFilled } from '@element-plus/icons-vue'
import api from '../../utils/api'

const loading = ref(false)
const saving = ref(false)
const formRef = ref()
const currentConfig = ref(null)

const form = reactive({
  base_url: 'http://192.168.208.214:8080',
  threshold: 0.4,
  enabled: true
})

const rules = {
  base_url: [
    { required: true, message: 'Vui lòng nhập địa chỉ dịch vụ', trigger: 'blur' },
    { 
      pattern: /^https?:\/\/.+/, 
      message: 'Vui lòng nhập địa chỉ HTTP hợp lệ, ví dụ: http://192.168.208.214:8080', 
      trigger: 'blur' 
    }
  ],
  threshold: [
    { required: true, message: 'Vui lòng nhập ngưỡng nhận diện', trigger: 'blur' },
    { 
      type: 'number', 
      min: 0, 
      max: 1, 
      message: 'Ngưỡng phải nằm trong khoảng 0.0 đến 1.0', 
      trigger: 'blur' 
    }
  ]
}

const loadConfig = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/speaker-configs')
    const configs = response.data.data || []
    
    if (configs.length > 0) {
      // Nếu đã có cấu hình thì dùng bản ghi đầu tiên, về lý thuyết chỉ nên có một bản
      currentConfig.value = configs[0]
      const configObj = JSON.parse(configs[0].json_data || '{}')
      
      // Phân tích dữ liệu cấu hình
      if (configObj.service && configObj.service.base_url) {
        form.base_url = configObj.service.base_url
      } else if (configObj.base_url) {
        // Tương thích với định dạng cũ
        form.base_url = configObj.base_url
      }
      // Đọc cấu hình ngưỡng
      if (configObj.service && configObj.service.threshold !== undefined) {
        form.threshold = configObj.service.threshold
      } else if (configObj.threshold !== undefined) {
        // Tương thích với định dạng cũ
        form.threshold = configObj.threshold
      } else {
        // Giá trị mặc định
        form.threshold = 0.4
      }
      // Công tắc tương ứng với json_data.enable trong nghiệp vụ, không dùng cột enabled trả về từ API
      form.enabled = configObj.enable !== undefined ? configObj.enable : true
    }
  } catch (error) {
    ElMessage.error('Tải cấu hình thất bại')
  } finally {
    loading.value = false
  }
}

const handleSave = async () => {
  if (!formRef.value) return
  
  await formRef.value.validate(async (valid) => {
    if (valid) {
      saving.value = true
      try {
        // Tạo dữ liệu cấu hình: trạng thái bật/tắt được ghi vào json_data.enable và dùng trường này làm chuẩn đầu ra
        const configData = {
          service: {
            base_url: form.base_url,
            threshold: form.threshold
          },
          enable: form.enabled
        }
        
        const saveData = {
          name: 'Cấu hình nhận diện người nói',
          config_id: 'asr_server',
          provider: 'asr_server',
          is_default: true,
          enabled: form.enabled,
          json_data: JSON.stringify(configData)
        }
        
        if (currentConfig.value) {
          // Cập nhật cấu hình hiện có
          await api.put(`/admin/speaker-configs/${currentConfig.value.id}`, saveData)
          ElMessage.success('Cập nhật cấu hình thành công')
        } else {
          // Tạo cấu hình mới
          await api.post('/admin/speaker-configs', saveData)
          ElMessage.success('Tạo cấu hình thành công')
        }
        
        // Tải lại cấu hình
        await loadConfig()
      } catch (error) {
        ElMessage.error('Lưu thất bại: ' + (error.response?.data?.message || error.message))
      } finally {
        saving.value = false
      }
    }
  })
}

onMounted(() => {
  loadConfig()
})
</script>

<style scoped>
.config-page {
  padding: 20px;
  background: rgba(255, 255, 255, 0.88);
  border-radius: 8px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.config-card {
  max-width: 800px;
}

.form-tip {
  margin-top: 8px;
  font-size: 12px;
  color: var(--apple-text-secondary);
  display: flex;
  align-items: center;
  gap: 4px;
}

.form-tip .el-icon {
  font-size: 14px;
  color: var(--apple-primary);
}

.form-actions {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid #ebeef5;
}
</style>
