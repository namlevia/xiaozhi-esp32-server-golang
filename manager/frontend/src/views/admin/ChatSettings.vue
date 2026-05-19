<template>
  <div class="config-page">
    <div class="page-actions">
      <el-button @click="loadSettings" :loading="loading">Làm mới</el-button>
      <el-button type="primary" @click="saveSettings" :loading="saving">Lưu cài đặt</el-button>
    </div>

    <el-card v-loading="loading">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="180px" style="max-width: 720px;">
        <el-divider content-position="left">Xác thực</el-divider>
        <el-form-item label="Bật xác thực kích hoạt thiết bị" prop="auth.enable">
          <el-switch v-model="form.auth.enable" />
        </el-form-item>
        <el-form-item label="Xác minh số khi đăng nhập" prop="auth.login_captcha_enabled">
          <el-switch
            v-model="form.auth.login_captcha_enabled"
            active-text="Bật"
            inactive-text="Tắt"
          />
          <div class="form-help">
            Khi bật, trang đăng nhập cần hoàn thành phép toán số; khi tắt chỉ kiểm tra tên người dùng và mật khẩu. Mặc định bật.
          </div>
        </el-form-item>

        <el-divider content-position="left">Tham số trò chuyện</el-divider>
        <el-form-item label="Thời gian rỗi tối đa của phiên (ms)" prop="chat.max_idle_duration">
          <el-input-number v-model="form.chat.max_idle_duration" :min="0" :step="1000" style="width: 100%;" />
          <div class="form-help">
            Đơn vị ms. Đặt 0 nghĩa là không giới hạn thời gian rỗi của phiên và không tự ngắt do rỗi. Khuyến nghị: 30000~120000.
          </div>
        </el-form-item>
        <el-form-item label="Ngưỡng im lặng kết thúc câu (ms)" prop="chat.chat_max_silence_duration">
          <el-input-number v-model="form.chat.chat_max_silence_duration" :min="0" :step="10" style="width: 100%;" />
          <div class="form-help">
            Dùng để xác định kết thúc câu: khi chuyển từ có âm thanh sang im lặng đủ ngưỡng, hệ thống xem câu đã kết thúc và xử lý tiếp. Mặc định 400ms. Ngưỡng nhỏ phản hồi nhanh hơn nhưng dễ cắt ngang; ngưỡng lớn ổn định hơn nhưng chậm hơn, khuyến nghị 300~600ms.
          </div>
        </el-form-item>
        <el-form-item label="Chế độ ngắt thời gian thực" prop="chat.realtime_mode">
          <el-select v-model="form.chat.realtime_mode" style="width: 100%;">
            <el-option :value="1" label="1 - chế độ ngắt bằng VAD" />
            <el-option :value="2" label="2 - chế độ ngắt bằng ASR" />
            <el-option :value="3" label="3 - ngắt khi ASR nhận diện người nói" />
            <el-option :value="4" label="4 - ngắt khi ASR có kết quả" />
          </el-select>
        </el-form-item>
        <el-form-item label="Mô tả prompt hệ thống toàn cục" prop="chat.global_system_prompt">
          <el-input
            v-model="form.chat.global_system_prompt"
            type="textarea"
            :rows="6"
            maxlength="8000"
            show-word-limit
            placeholder="Nội dung này sẽ được ghép vào đầu system prompt; nên nhập ràng buộc cấp nền tảng và thiết lập danh tính."
          />
          <div class="form-help">
            Thứ tự hiệu lực: mô tả prompt hệ thống toàn cục → prompt vai trò/thiết bị → thông tin thời gian chạy như thời gian/bộ nhớ.
          </div>
        </el-form-item>
      </el-form>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import api from '../../utils/api'

const loading = ref(false)
const saving = ref(false)
const formRef = ref()

const form = reactive({
  auth: {
    enable: false,
    login_captcha_enabled: true
  },
  chat: {
    max_idle_duration: 30000,
    chat_max_silence_duration: 400,
    realtime_mode: 4,
    global_system_prompt: ''
  }
})

const rules = {
  'chat.max_idle_duration': [
    { required: true, message: 'Vui lòng nhập thời gian rỗi tối đa của phiên', trigger: 'blur' }
  ],
  'chat.chat_max_silence_duration': [
    { required: true, message: 'Vui lòng nhập ngưỡng im lặng kết thúc câu', trigger: 'blur' }
  ],
  'chat.realtime_mode': [
    { required: true, message: 'Vui lòng chọn chế độ ngắt thời gian thực', trigger: 'change' }
  ],
  'chat.global_system_prompt': [
    { max: 8000, message: 'Mô tả prompt hệ thống toàn cục không được vượt quá 8000 ký tự', trigger: 'blur' }
  ]
}

const loadSettings = async () => {
  loading.value = true
  try {
    const res = await api.get('/admin/chat-settings')
    const data = res.data?.data || {}
    form.auth.enable = !!data.auth?.enable
    form.auth.login_captcha_enabled = data.auth?.login_captcha_enabled !== false
    form.chat.max_idle_duration = Number(data.chat?.max_idle_duration ?? 30000)
    form.chat.chat_max_silence_duration = Number(data.chat?.chat_max_silence_duration ?? 400)
    form.chat.realtime_mode = Number(data.chat?.realtime_mode ?? 4)
    form.chat.global_system_prompt = String(data.chat?.global_system_prompt ?? '')
  } catch (error) {
    ElMessage.error('Tải cài đặt trò chuyện thất bại')
    console.error(error)
  } finally {
    loading.value = false
  }
}

const saveSettings = async () => {
  if (!formRef.value) return
  const valid = await formRef.value.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    await api.put('/admin/chat-settings', {
      auth: {
        enable: !!form.auth.enable,
        login_captcha_enabled: form.auth.login_captcha_enabled !== false
      },
      chat: {
        max_idle_duration: Number(form.chat.max_idle_duration),
        chat_max_silence_duration: Number(form.chat.chat_max_silence_duration),
        realtime_mode: Number(form.chat.realtime_mode),
        global_system_prompt: String(form.chat.global_system_prompt || '')
      }
    })
    ElMessage.success('Lưu cài đặt trò chuyện thành công')
  } catch (error) {
    ElMessage.error('Lưu cài đặt trò chuyện thất bại')
    console.error(error)
  } finally {
    saving.value = false
  }
}

onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.config-page {
  padding: 20px;
}

.page-actions {
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.form-help {
  margin-top: 6px;
  color: #909399;
  font-size: 12px;
  line-height: 1.5;
}
</style>
