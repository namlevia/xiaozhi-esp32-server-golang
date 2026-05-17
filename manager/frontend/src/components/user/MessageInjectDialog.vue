<template>
  <el-dialog
    v-model="visible"
    :title="t('device.voicePushTitle')"
    width="620px"
    class="inject-message-dialog"
    :close-on-click-modal="false"
    @closed="resetForm"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
    >
      <el-form-item :label="t('device.selectDevice')" prop="device_id">
        <el-select
          v-model="form.device_id"
          :placeholder="t('device.selectDevicePlaceholder')"
          style="width: 100%"
          filterable
          :disabled="deviceSelectDisabled"
          popper-class="inject-device-select-popper"
        >
          <el-option
            v-for="device in devices"
            :key="device.id || device.device_code"
            :label="getDeviceOptionLabel(device)"
            :value="device.device_name || ''"
          >
            <div class="device-option">
              <div class="device-option-header">
                <span class="device-name">{{ getDeviceNickName(device) }}</span>
                <el-tag :type="isDeviceOnline(device.last_active_at) ? 'success' : 'danger'" size="small">
                  {{ isDeviceOnline(device.last_active_at) ? t('device.online') : t('device.offline') }}
                </el-tag>
              </div>
              <div class="device-code">{{ t('device.deviceId') }}: {{ getDeviceIdText(device) }}</div>
              <div v-if="device.device_code" class="device-code">{{ t('device.activationCode') }}: {{ device.device_code }}</div>
              <div class="device-agent">{{ t('device.linkedAgent') }}: {{ device.agent_name || t('device.unbound') }}</div>
            </div>
          </el-option>
        </el-select>
      </el-form-item>

      <el-form-item :label="t('device.pushContent')" prop="message">
        <el-input
          v-model="form.message"
          type="textarea"
          :rows="4"
          :placeholder="t('device.pushContentPlaceholder')"
          maxlength="500"
          show-word-limit
        />
      </el-form-item>

      <el-form-item :label="t('device.directPlayback')" prop="skip_llm">
        <div class="switch-field">
          <div class="switch-copy">
            <div class="switch-title">{{ directPlayback ? t('device.enabled') : t('device.disabled') }}</div>
            <div class="switch-desc">
              {{ directPlayback ? t('device.directPlaybackOn') : t('device.directPlaybackOff') }}
            </div>
          </div>
          <el-switch
            v-model="directPlayback"
            inline-prompt
            :active-text="t('device.enabled')"
            :inactive-text="t('device.disabled')"
          />
        </div>
      </el-form-item>

      <el-form-item :label="t('device.returnToIdle')" prop="auto_listen">
        <div class="switch-field">
          <div class="switch-copy">
            <div class="switch-title">{{ returnToIdleAfterPlayback ? t('device.enabled') : t('device.disabled') }}</div>
            <div class="switch-desc">
              {{ returnToIdleAfterPlayback ? t('device.returnToIdleOn') : t('device.returnToIdleOff') }}
            </div>
          </div>
          <el-switch
            v-model="returnToIdleAfterPlayback"
            inline-prompt
            :active-text="t('device.enabled')"
            :inactive-text="t('device.disabled')"
          />
        </div>
      </el-form-item>
    </el-form>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="handleClose">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="submitting" @click="handleSubmit">
          {{ submitting ? t('device.pushing') : t('device.voicePush') }}
        </el-button>
      </div>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import api from '../../utils/api'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  devices: {
    type: Array,
    default: () => []
  },
  defaultDeviceId: {
    type: String,
    default: ''
  },
  lockDevice: {
    type: Boolean,
    default: false
  }
})

const emit = defineEmits(['update:modelValue', 'success'])
const { t } = useI18n()

const formRef = ref()
const submitting = ref(false)
const visible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

const directPlayback = computed({
  get: () => form.skip_llm,
  set: (value) => {
    form.skip_llm = value
  }
})

const returnToIdleAfterPlayback = computed({
  get: () => !form.auto_listen,
  set: (value) => {
    form.auto_listen = !value
  }
})

const deviceSelectDisabled = computed(() => props.lockDevice && !!props.defaultDeviceId)

const form = reactive({
  device_id: '',
  message: '',
  skip_llm: false,
  auto_listen: true
})

const rules = {
  device_id: [
    { required: true, message: t('device.selectDevice'), trigger: 'change' }
  ],
  message: [
    { required: true, message: t('device.pushContentRequired'), trigger: 'blur' },
    { min: 1, max: 500, message: t('device.pushContentLength'), trigger: 'blur' }
  ]
}

const isDeviceOnline = (lastActiveAt) => {
  if (!lastActiveAt) return false
  const lastActive = new Date(lastActiveAt)
  return (Date.now() - lastActive.getTime()) < 5 * 60 * 1000
}

const getDeviceNickName = (device) => {
  const nickName = String(device?.nick_name || '').trim()
  if (nickName) return nickName
  return String(device?.device_name || '').trim() || t('device.unnamed')
}

const getDeviceIdText = (device) => String(device?.device_name || '').trim() || '-'

const getDeviceOptionLabel = (device) => {
  const nickName = getDeviceNickName(device)
  const deviceId = getDeviceIdText(device)
  return `${nickName} (${deviceId})`
}

const resetForm = () => {
  form.device_id = props.defaultDeviceId || ''
  form.message = ''
  form.skip_llm = false
  form.auto_listen = true
  formRef.value?.clearValidate?.()
}

watch(
  () => [props.modelValue, props.defaultDeviceId],
  ([visible]) => {
    if (!visible) return
    resetForm()
  }
)

const closeDialog = () => {
  visible.value = false
}

const handleSubmit = async () => {
  if (!formRef.value) return

  try {
    await formRef.value.validate()
  } catch {
    return
  }

  submitting.value = true
  try {
    const response = await api.post('/user/devices/inject-message', {
      device_id: form.device_id,
      message: form.message,
      skip_llm: form.skip_llm,
      auto_listen: form.auto_listen
    })
    if (response.data?.success) {
      ElMessage.success(t('device.voicePushSuccess'))
      emit('success', response.data?.data || null)
      closeDialog()
    }
  } catch (error) {
    ElMessage.error(error.response?.data?.error || t('device.voicePushFailed'))
  } finally {
    submitting.value = false
  }
}

const handleClose = () => {
  resetForm()
  closeDialog()
}
</script>

<style scoped>
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}

.device-option {
  padding: 8px 0;
}

.device-option-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
  gap: 12px;
}

.device-name {
  font-weight: 600;
  color: var(--apple-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.device-code,
.device-agent {
  font-size: 12px;
  color: rgba(107, 114, 128, 0.72);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.switch-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  width: 100%;
  padding: 14px 16px;
  border-radius: 18px;
  background: rgba(248, 250, 252, 0.9);
  border: 1px solid rgba(229, 229, 234, 0.72);
}

.switch-copy {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.switch-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--apple-text);
}

.switch-desc {
  font-size: 12px;
  line-height: 1.5;
  color: var(--apple-text-secondary);
}

:deep(.inject-device-select-popper .el-select-dropdown__item) {
  height: auto;
  line-height: 1.4;
  padding-top: 8px;
  padding-bottom: 8px;
  white-space: normal;
}

@media (max-width: 768px) {
  .dialog-footer {
    flex-wrap: wrap;
  }

  .dialog-footer .el-button {
    flex: 1;
    min-width: 120px;
  }

  .switch-field {
    align-items: flex-start;
  }
}
</style>
