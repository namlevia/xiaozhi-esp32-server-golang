<template>
  <div class="admin-agents">
    <div class="toolbar">
      <el-button type="primary" @click="openAddDialog">
        <el-icon><Plus /></el-icon>
        {{ t('agent.add') }}
      </el-button>
      <el-button @click="loadAgents">
        <el-icon><Refresh /></el-icon>
        {{ t('common.refresh') }}
      </el-button>
    </div>

    <el-table :data="agents" v-loading="loading" stripe>
      <el-table-column prop="id" label="ID" width="80" />
      <el-table-column prop="name" :label="t('agent.name')" min-width="140" />
      <el-table-column :label="t('agent.nickname')" min-width="130">
        <template #default="{ row }">{{ row.nickname || row.name }}</template>
      </el-table-column>
      <el-table-column :label="t('agent.ownerUser')" width="150">
        <template #default="{ row }">{{ row.username || `User ${row.user_id}` }}</template>
      </el-table-column>
      <el-table-column :label="t('agent.roleIntro')" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">{{ row.custom_prompt || t('agent.notSet') }}</template>
      </el-table-column>
      <el-table-column :label="t('agent.llm')" width="150">
        <template #default="{ row }">{{ row.llm_config?.name || t('agent.notSet') }}</template>
      </el-table-column>
      <el-table-column :label="`${t('agent.ttsConfig')} / ${t('agent.ttsVoice')}`" width="190" show-overflow-tooltip>
        <template #default="{ row }">{{ getVoiceText(row) }}</template>
      </el-table-column>
      <el-table-column :label="t('routes.knowledgeBases')" width="90">
        <template #default="{ row }">{{ row.knowledge_base_ids?.length || 0 }}</template>
      </el-table-column>
      <el-table-column :label="t('device.device')" width="90">
        <template #default="{ row }">{{ row.device_count || 0 }}</template>
      </el-table-column>
      <el-table-column :label="t('agent.asrSpeed')" width="110">
        <template #default="{ row }">
          <el-tag :type="getASRSpeedType(row.asr_speed)">{{ getASRSpeedText(row.asr_speed) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('agent.memoryMode')" width="100">
        <template #default="{ row }">
          <el-tag :type="getMemoryModeType(row.memory_mode)">{{ getMemoryModeText(row.memory_mode) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('agent.speakerChatMode')" width="150">
        <template #default="{ row }">
          <el-tag :type="getSpeakerChatModeType(row.speaker_chat_mode)">{{ getSpeakerChatModeText(row.speaker_chat_mode) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column :label="t('device.actions')" width="330" fixed="right">
        <template #default="{ row }">
          <el-button size="small" @click="editAgent(row)">{{ t('common.edit') }}</el-button>
          <el-button size="small" type="primary" @click="openDiagnostics(row, 'mcp')">MCP</el-button>
          <el-button size="small" type="success" @click="openDiagnostics(row, 'openclaw')">OpenClaw</el-button>
          <el-button size="small" type="danger" @click="deleteAgent(row)">{{ t('common.delete') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog
      v-model="showAddDialog"
      :title="editingAgent ? t('agent.edit') : t('agent.add')"
      width="760px"
      :close-on-click-modal="false"
    >
      <AgentForm
        ref="agentFormRef"
        v-model="agentForm"
        is-admin
        :mode="editingAgent ? 'edit' : 'create'"
      />
      <AgentRuntimeDiagnostics
        v-if="editingAgent"
        class="dialog-diagnostics"
        :agent-id="editingAgent.id"
        scope="admin"
        preload-status
      />
      <template #footer>
        <el-button @click="showAddDialog = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveAgent" :loading="saving">
          {{ editingAgent ? t('agent.update') : t('common.add') }}
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="showDiagnosticsDialog" :title="diagnosticsTitle" width="760px">
      <AgentRuntimeDiagnostics
        v-if="diagnosticAgent"
        :key="`${diagnosticAgent.id}-${diagnosticPanel}`"
        :agent-id="diagnosticAgent.id"
        scope="admin"
        :default-panels="[diagnosticPanel]"
      />
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh } from '@element-plus/icons-vue'
import api from '../../utils/api'
import AgentForm from '../../components/common/AgentForm.vue'
import AgentRuntimeDiagnostics from '../../components/common/AgentRuntimeDiagnostics.vue'
import { agentToForm, createDefaultAgentForm } from '../../composables/useAgentFormOptions'

const { t } = useI18n()
const agents = ref([])
const loading = ref(false)
const showAddDialog = ref(false)
const editingAgent = ref(null)
const saving = ref(false)
const agentFormRef = ref(null)
const agentForm = ref(createDefaultAgentForm({ isAdmin: true }))
const showDiagnosticsDialog = ref(false)
const diagnosticAgent = ref(null)
const diagnosticPanel = ref('mcp')
const diagnosticsTitle = computed(() => {
  const name = diagnosticAgent.value?.name || `Agent ${diagnosticAgent.value?.id || ''}`
  return diagnosticPanel.value === 'openclaw' ? `${name} - OpenClaw` : `${name} - MCP`
})

const loadAgents = async () => {
  loading.value = true
  try {
    const response = await api.get('/admin/agents')
    agents.value = response.data.data || []
  } catch (error) {
    ElMessage.error(t('agent.loadAgentsFailed'))
  } finally {
    loading.value = false
  }
}

const openAddDialog = () => {
  editingAgent.value = null
  agentForm.value = createDefaultAgentForm({ isAdmin: true })
  showAddDialog.value = true
}

const editAgent = (agent) => {
  editingAgent.value = agent
  agentForm.value = agentToForm(agent, { isAdmin: true })
  showAddDialog.value = true
}

const saveAgent = async () => {
  if (!agentFormRef.value) return
  const valid = await agentFormRef.value.validate().catch(() => false)
  if (!valid) return

  saving.value = true
  try {
    const payload = agentFormRef.value.buildPayload()
    if (editingAgent.value) {
      await api.put(`/admin/agents/${editingAgent.value.id}`, payload)
      ElMessage.success(t('agent.updateSuccess'))
    } else {
      await api.post('/admin/agents', payload)
      ElMessage.success(t('agent.addSuccess'))
    }
    showAddDialog.value = false
    await loadAgents()
  } catch (error) {
    ElMessage.error(error.response?.data?.error || (editingAgent.value ? t('agent.updateFailed') : t('agent.addFailed')))
  } finally {
    saving.value = false
  }
}

const deleteAgent = async (agent) => {
  try {
    await ElMessageBox.confirm(t('agent.confirmDelete', { name: agent.name }), t('device.confirmDeleteTitle'), {
      confirmButtonText: t('common.confirm'),
      cancelButtonText: t('common.cancel'),
      type: 'warning'
    })
    await api.delete(`/admin/agents/${agent.id}`)
    ElMessage.success(t('agent.deleteSuccess'))
    await loadAgents()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error.response?.data?.error || t('agent.deleteFailed'))
    }
  }
}

const getVoiceText = (agent) => {
  const tts = agent.tts_config?.name || agent.tts_config?.provider || t('agent.notSet')
  return agent.voice ? `${tts} · ${agent.voice}` : tts
}

const getASRSpeedText = (speed) => ({ normal: t('agent.normal'), patient: t('agent.patient'), fast: t('agent.fast') }[speed] || t('agent.normal'))
const getASRSpeedType = (speed) => ({ patient: 'warning', fast: 'success' }[speed] || '')
const getMemoryModeText = (mode) => ({ none: t('agent.noMemory'), short: t('agent.shortMemory'), long: t('agent.longMemory') }[mode] || t('agent.shortMemory'))
const getMemoryModeType = (mode) => ({ none: 'info', long: 'success' }[mode] || '')
const getSpeakerChatModeText = (mode) => ({ off: t('agent.off'), identified_only: t('agent.identifiedOnlyShort') }[mode] || t('agent.off'))
const getSpeakerChatModeType = (mode) => ({ off: 'info', identified_only: 'warning' }[mode] || 'info')

const openDiagnostics = (agent, panel = 'mcp') => {
  diagnosticAgent.value = agent
  diagnosticPanel.value = panel
  showDiagnosticsDialog.value = true
}

onMounted(() => {
  loadAgents()
})
</script>

<style scoped>
.admin-agents {
  padding: 20px;
}

.toolbar {
  margin-bottom: 20px;
  display: flex;
  gap: 12px;
  justify-content: flex-end;
  flex-wrap: wrap;
}

.dialog-diagnostics {
  margin-top: 16px;
}
</style>
