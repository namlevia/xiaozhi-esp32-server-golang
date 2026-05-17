<template>
  <div class="config-page">
    <div class="page-actions">
      <el-input
        v-model="searchKeyword"
        placeholder="Tìm người dùng..."
        style="width: 200px"
        prefix-icon="Search"
        clearable
      />
      <el-button type="primary" @click="openAddDialog">
        <el-icon><Plus /></el-icon>
        Thêm người dùng
      </el-button>
    </div>

    <el-table :data="filteredUserList" v-loading="tableLoading" style="width: 100%">
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="username" label="Tên đăng nhập" width="150" />
      <el-table-column prop="email" label="Email" width="200" />
      <el-table-column prop="role" label="Vai trò" width="120">
        <template #default="{ row }">
          <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'">
            {{ row.role === 'admin' ? 'Quản trị viên' : 'Người dùng thường' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="created_at" label="Thời gian tạo" width="180">
        <template #default="{ row }">
          {{ formatDateTime(row.created_at) }}
        </template>
      </el-table-column>
      <el-table-column label="Thao tác" width="360">
        <template #default="{ row }">
          <el-button size="small" @click="openEditDialog(row)">Chỉnh sửa</el-button>
          <el-button size="small" type="success" @click="openQuotaDialog(row)" :disabled="row.role === 'admin'">Hạn mức clone</el-button>
          <el-button size="small" type="warning" @click="openResetPasswordDialog(row)">
            Đặt lại mật khẩu
          </el-button>
          <el-button
            size="small"
            type="danger"
            @click="handleDeleteUser(row)"
            :disabled="row.role === 'admin'"
          >
            Xóa
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      v-model="userDialogVisible"
      :title="isEditMode ? 'Chỉnh sửa người dùng' : 'Thêm người dùng'"
      width="500px"
      @close="resetUserForm"
    >
      <el-form
        ref="userFormRef"
        :model="userForm"
        :rules="userFormRules"
        label-width="80px"
      >
        <el-form-item label="Tên đăng nhập" prop="username">
          <el-input
            v-model="userForm.username"
            :disabled="isEditMode"
            placeholder="Nhập tên đăng nhập"
          />
        </el-form-item>

        <el-form-item label="Email" prop="email">
          <el-input v-model="userForm.email" placeholder="Nhập email" />
        </el-form-item>

        <el-form-item v-if="!isEditMode" label="Mật khẩu" prop="password">
          <el-input
            v-model="userForm.password"
            type="password"
            placeholder="Nhập mật khẩu (ít nhất 6 ký tự)"
            show-password
          />
        </el-form-item>

        <el-form-item label="Vai trò" prop="role">
          <el-select v-model="userForm.role" placeholder="Chọn vai trò" style="width: 100%">
            <el-option label="Người dùng thường" value="user" />
            <el-option label="Quản trị viên" value="admin" />
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="userDialogVisible = false">Hủy</el-button>
        <el-button type="primary" @click="handleUserSubmit" :loading="userSubmitLoading">
          {{ isEditMode ? 'Lưu' : 'Thêm' }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="resetPasswordDialogVisible"
      title="Đặt lại mật khẩu"
      width="400px"
      @close="resetPasswordForm"
    >
      <el-form
        ref="passwordFormRef"
        :model="passwordForm"
        :rules="passwordFormRules"
        label-width="80px"
      >
        <el-form-item label="Người dùng">
          <el-input v-model="currentUser.username" disabled />
        </el-form-item>

        <el-form-item label="Mật khẩu mới" prop="newPassword">
          <el-input
            v-model="passwordForm.newPassword"
            type="password"
            placeholder="Nhập mật khẩu mới (ít nhất 6 ký tự)"
            show-password
          />
        </el-form-item>

        <el-form-item label="Xác nhận mật khẩu" prop="confirmPassword">
          <el-input
            v-model="passwordForm.confirmPassword"
            type="password"
            placeholder="Nhập lại mật khẩu mới"
            show-password
          />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="resetPasswordDialogVisible = false">Hủy</el-button>
        <el-button type="primary" @click="handleResetPassword" :loading="resetPasswordLoading">
          Xác nhận đặt lại
        </el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="quotaDialogVisible"
      :title="`Hạn mức nhân bản giọng - ${quotaUser.username || ''}`"
      width="900px"
      @close="resetQuotaDialog"
    >
      <div class="quota-hint">Phân bổ số lần clone theo từng cấu hình TTS: -1 là không giới hạn, 0 là cấm tạo, số nguyên dương là số lần clone tối đa.</div>
      <el-table :data="quotaRows" v-loading="quotaLoading" style="margin-top: 12px">
        <el-table-column prop="tts_config_name" label="Tên cấu hình TTS" min-width="180" />
        <el-table-column prop="tts_config_id" label="TTS Config ID" min-width="180" />
        <el-table-column prop="provider" label="Provider" width="120" />
        <el-table-column label="Đã dùng" width="100">
          <template #default="{ row }">{{ row.used_count }}</template>
        </el-table-column>
        <el-table-column label="Còn lại" width="100">
          <template #default="{ row }">{{ row.remaining_count < 0 ? 'Không giới hạn' : row.remaining_count }}</template>
        </el-table-column>
        <el-table-column label="Số lần tối đa" width="180">
          <template #default="{ row }">
            <el-input-number v-model="row.max_count" :min="-1" :step="1" :precision="0" controls-position="right" style="width: 140px" />
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="quotaDialogVisible = false">Hủy</el-button>
        <el-button type="primary" :loading="quotaSaving" @click="saveQuotaSettings">Lưu hạn mức</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus } from '@element-plus/icons-vue'
import api from '../../utils/api'

// Trạng thái dữ liệu
const userList = ref([])
const tableLoading = ref(false)
const userDialogVisible = ref(false)
const resetPasswordDialogVisible = ref(false)
const userSubmitLoading = ref(false)
const resetPasswordLoading = ref(false)
const quotaDialogVisible = ref(false)
const quotaLoading = ref(false)
const quotaSaving = ref(false)
const quotaRows = ref([])
const quotaUser = ref({})
const quotaOriginalMaxMap = ref({})
const isEditMode = ref(false)
const currentUser = ref({})
const searchKeyword = ref('')

// Thuộc tính tính toán
const filteredUserList = computed(() => {
  if (!searchKeyword.value) {
    return userList.value
  }
  return userList.value.filter(user =>
    user.username.toLowerCase().includes(searchKeyword.value.toLowerCase()) ||
    user.email.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

// Tham chiếu form
const userFormRef = ref()
const passwordFormRef = ref()

// Dữ liệu form người dùng
const userForm = reactive({
  username: '',
  email: '',
  password: '',
  role: ''
})

// Dữ liệu form mật khẩu
const passwordForm = reactive({
  newPassword: '',
  confirmPassword: ''
})

// Quy tắc kiểm tra form người dùng
const userFormRules = {
  username: [
    { required: true, message: 'Vui lòng nhập tên đăng nhập', trigger: 'blur' }
  ],
  email: [
    { required: true, message: 'Vui lòng nhập email', trigger: 'blur' },
    { type: 'email', message: 'Vui lòng nhập đúng định dạng email', trigger: 'blur' }
  ],
  password: [
    { required: true, message: 'Vui lòng nhập mật khẩu', trigger: 'blur' },
    { min: 6, message: 'Mật khẩu phải có ít nhất 6 ký tự', trigger: 'blur' }
  ],
  role: [
    { required: true, message: 'Vui lòng chọn vai trò', trigger: 'change' }
  ]
}

const passwordFormRules = {
  newPassword: [
    { required: true, message: 'Vui lòng nhập mật khẩu mới', trigger: 'blur' },
    { min: 6, message: 'Mật khẩu phải có ít nhất 6 ký tự', trigger: 'blur' }
  ],
  confirmPassword: [
    { required: true, message: 'Vui lòng xác nhận mật khẩu', trigger: 'blur' },
    {
      validator: (rule, value, callback) => {
        if (value !== passwordForm.newPassword) {
          callback(new Error('Hai lần nhập mật khẩu không khớp'))
        } else {
          callback()
        }
      },
      trigger: 'blur'
    }
  ]
}

// Tải danh sách người dùng
const loadUserList = async () => {
  tableLoading.value = true
  try {
    const response = await api.get('/admin/users')
    userList.value = response.data.data || []
  } catch (error) {
    ElMessage.error('Tải danh sách người dùng thất bại')
  } finally {
    tableLoading.value = false
  }
}

// Mở hộp thoại thêm người dùng
const openAddDialog = () => {
  isEditMode.value = false
  userDialogVisible.value = true
}

// Mở hộp thoại sửa người dùng
const openEditDialog = (user) => {
  isEditMode.value = true
  currentUser.value = user
  userForm.username = user.username
  userForm.email = user.email
  userForm.role = user.role
  userDialogVisible.value = true
}

// Đặt lại form người dùng
const resetUserForm = () => {
  userForm.username = ''
  userForm.email = ''
  userForm.password = ''
  userForm.role = ''
  currentUser.value = {}
  if (userFormRef.value) {
    userFormRef.value.resetFields()
  }
}

// Xử lý gửi form người dùng
const handleUserSubmit = async () => {
  if (!userFormRef.value) return
  
  try {
    await userFormRef.value.validate()
    userSubmitLoading.value = true
    
    if (isEditMode.value) {
      // 编辑用户
      await api.put(`/admin/users/${currentUser.value.id}`, {
        email: userForm.email,
        role: userForm.role
      })
      ElMessage.success('Cập nhật người dùng thành công')
    } else {
      // 添加用户
      await api.post('/admin/users', {
        username: userForm.username,
        email: userForm.email,
        password: userForm.password,
        role: userForm.role
      })
      ElMessage.success('Thêm người dùng thành công')
    }
    
    userDialogVisible.value = false
    loadUserList()
  } catch (error) {
    ElMessage.error(isEditMode.value ? 'Cập nhật người dùng thất bại' : 'Thêm người dùng thất bại')
  } finally {
    userSubmitLoading.value = false
  }
}

// Xóa người dùng
const handleDeleteUser = async (user) => {
  try {
    await ElMessageBox.confirm(
      `Bạn có chắc muốn xóa người dùng "${user.username}" không?`,
      'Xác nhận xóa',
      {
        confirmButtonText: 'Xác nhận',
        cancelButtonText: 'Hủy',
        type: 'warning'
      }
    )
    
    await api.delete(`/admin/users/${user.id}`)
    ElMessage.success('Xóa người dùng thành công')
    loadUserList()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('Xóa người dùng thất bại')
    }
  }
}

// Mở hộp thoại đặt lại mật khẩu
const openResetPasswordDialog = (user) => {
  currentUser.value = user
  resetPasswordDialogVisible.value = true
}

// Mở phần thiết lập hạn mức clone
const openQuotaDialog = async (user) => {
  quotaUser.value = user
  quotaDialogVisible.value = true
  await loadQuotaSettings(user.id)
}

const loadQuotaSettings = async (userID) => {
  quotaLoading.value = true
  try {
    const response = await api.get(`/admin/users/${userID}/voice-clone-quotas`)
    const quotas = response.data?.data?.quotas || []
    quotaRows.value = quotas.map((item) => ({
      ...item,
      max_count: Number.isFinite(Number(item.max_count)) ? Number(item.max_count) : -1,
      used_count: Number(item.used_count || 0),
      remaining_count: Number.isFinite(Number(item.remaining_count)) ? Number(item.remaining_count) : -1
    }))
    quotaOriginalMaxMap.value = quotaRows.value.reduce((acc, row) => {
      acc[row.tts_config_id] = Number(row.max_count)
      return acc
    }, {})
  } catch (error) {
    ElMessage.error('Tải hạn mức clone thất bại')
    quotaRows.value = []
    quotaOriginalMaxMap.value = {}
  } finally {
    quotaLoading.value = false
  }
}

const saveQuotaSettings = async () => {
  if (!quotaUser.value?.id) return
  const normalizedItems = quotaRows.value.map((row) => ({
    tts_config_id: row.tts_config_id,
    max_count: Number(row.max_count)
  }))
  for (const item of normalizedItems) {
    if (!item.tts_config_id) {
      ElMessage.error('Tồn tại tts_config_id không hợp lệ')
      return
    }
    if (!Number.isInteger(item.max_count) || item.max_count < -1) {
      ElMessage.error('max_count chỉ được là số nguyên lớn hơn hoặc bằng -1')
      return
    }
  }

  const items = normalizedItems.filter((item) => quotaOriginalMaxMap.value[item.tts_config_id] !== item.max_count)
  if (items.length === 0) {
    ElMessage.info('Hạn mức chưa thay đổi')
    return
  }

  quotaSaving.value = true
  try {
    await api.put(`/admin/users/${quotaUser.value.id}/voice-clone-quotas`, { items })
    ElMessage.success('Lưu hạn mức nhân bản giọng thành công')
    await loadQuotaSettings(quotaUser.value.id)
  } catch (error) {
    ElMessage.error('Lưu hạn mức clone thất bại')
  } finally {
    quotaSaving.value = false
  }
}

const resetQuotaDialog = () => {
  quotaRows.value = []
  quotaUser.value = {}
  quotaOriginalMaxMap.value = {}
}

// Đặt lại form mật khẩu
const resetPasswordForm = () => {
  passwordForm.newPassword = ''
  passwordForm.confirmPassword = ''
  if (passwordFormRef.value) {
    passwordFormRef.value.resetFields()
  }
}

// Xử lý đặt lại mật khẩu
const handleResetPassword = async () => {
  if (!passwordFormRef.value) return
  
  try {
    await passwordFormRef.value.validate()
    
    await ElMessageBox.confirm(
      `Bạn có chắc muốn đặt lại mật khẩu cho người dùng "${currentUser.value.username}" không?`,
      'Xác nhận đặt lại mật khẩu',
      {
        confirmButtonText: 'Xác nhận',
        cancelButtonText: 'Hủy',
        type: 'warning'
      }
    )
    
    resetPasswordLoading.value = true
    
    await api.post(`/admin/users/${currentUser.value.id}/reset-password`, {
      new_password: passwordForm.newPassword
    })
    
    ElMessage.success('Đặt lại mật khẩu thành công')
    resetPasswordDialogVisible.value = false
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error('Đặt lại mật khẩu thất bại')
    }
  } finally {
    resetPasswordLoading.value = false
  }
}

// Định dạng ngày giờ
const formatDateTime = (dateString) => {
  if (!dateString) return '--'
  return new Date(dateString).toLocaleString('zh-CN')
}

// Tải dữ liệu khi component được mount
onMounted(() => {
  loadUserList()
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
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}

.quota-hint {
  color: #666;
  font-size: 13px;
}
</style>
