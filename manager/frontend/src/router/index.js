import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { isMobile } from '../utils/device'

// Tải component động theo loại thiết bị
const getLoginComponent = () => {
  return isMobile()
    ? import('../views/mobile/MobileLogin.vue')
    : import('../views/Login.vue')
}

const routes = [
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('../views/Setup.vue')
  },
  {
    path: '/test',
    name: 'Test',
    component: () => import('../views/Test.vue')
  },
  {
    path: '/test-route',
    name: 'TestRoute',
    component: () => import('../views/TestRoute.vue')
  },
  {
    path: '/simple-login',
    name: 'SimpleLogin',
    component: () => import('../views/SimpleLogin.vue')
  },
  {
    path: '/login',
    name: 'Login',
    component: getLoginComponent
  },

  {
    path: '/openapi-docs',
    name: 'OpenAPIDocs',
    component: () => import('../views/OpenAPIDocs.vue'),
    meta: { titleKey: 'routes.openapiDocs' }
  },
  {
    path: '/',
    name: 'Layout',
    component: () => import('../components/Layout.vue'),
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: '/dashboard',
        name: 'Dashboard',
        component: () => import('../views/Dashboard.vue'),
        meta: { titleKey: 'menu.dashboard', requiresAdmin: true }
      },
      // Route dành cho quản trị viên
      {
        path: '/admin',
        name: 'Admin',
        meta: { requiresAuth: true, requiresAdmin: true },
        children: [
          {
            path: 'config-wizard',
            name: 'ConfigWizard',
            component: () => import('../views/admin/ConfigWizard.vue'),
            meta: { titleKey: 'routes.configWizard' }
          },
          {
            path: 'vad-config',
            name: 'VADConfig',
            component: () => import('../views/admin/VADConfig.vue'),
            meta: { titleKey: 'routes.vadConfig' }
          },
          {
            path: 'asr-config',
            name: 'ASRConfig',
            component: () => import('../views/admin/ASRConfig.vue'),
            meta: { titleKey: 'routes.asrConfig' }
          },
          {
            path: 'llm-config',
            name: 'LLMConfig',
            component: () => import('../views/admin/LLMConfig.vue'),
            meta: { titleKey: 'routes.llmConfig' }
          },
          {
            path: 'tts-config',
            name: 'TTSConfig',
            component: () => import('../views/admin/TTSConfig.vue'),
            meta: { titleKey: 'routes.ttsConfig' }
          },
          {
            path: 'speaker-config',
            name: 'SpeakerConfig',
            component: () => import('../views/admin/SpeakerConfig.vue'),
            meta: { titleKey: 'routes.speakerConfig' }
          },
          {
            path: 'ota-config',
            name: 'OTAConfig',
            component: () => import('../views/admin/OTAConfig.vue'),
            meta: { titleKey: 'routes.otaConfig' }
          },
          {
            path: 'mqtt-config',
            name: 'MQTTConfig',
            component: () => import('../views/admin/MQTTConfig.vue'),
            meta: { titleKey: 'routes.mqttConfig' }
          },
          {
            path: 'udp-config',
            name: 'UDPConfig',
            component: () => import('../views/admin/UDPConfig.vue'),
            meta: { titleKey: 'routes.udpConfig' }
          },
          {
            path: 'mqtt-server-config',
            name: 'MQTTServerConfig',
            component: () => import('../views/admin/MQTTServerConfig.vue'),
            meta: { titleKey: 'routes.mqttServerConfig' }
          },
          {
            path: 'mcp-config',
            name: 'MCPConfig',
            component: () => import('../views/admin/MCPConfig.vue'),
            meta: { titleKey: 'routes.mcpConfig' }
          },
          {
            path: 'mcp-market',
            name: 'MCPMarket',
            component: () => import('../views/admin/MCPMarket.vue'),
            meta: { titleKey: 'routes.mcpMarket' }
          },
          {
            path: 'memory-config',
            name: 'MemoryConfig',
            component: () => import('../views/admin/MemoryConfig.vue'),
            meta: { titleKey: 'routes.memoryConfig' }
          },
          {
            path: 'knowledge-search-config',
            name: 'KnowledgeSearchConfig',
            component: () => import('../views/admin/KnowledgeSearchConfig.vue'),
            meta: { titleKey: 'routes.knowledgeSearchConfig' }
          },
          {
            path: 'chat-settings',
            name: 'ChatSettings',
            component: () => import('../views/admin/ChatSettings.vue'),
            meta: { titleKey: 'routes.chatSettings' }
          },
          {
            path: 'vision-config',
            name: 'VisionConfig',
            component: () => import('../views/admin/VisionConfig.vue'),
            meta: { titleKey: 'routes.visionConfig' }
          },
          {
            path: 'pool-stats',
            name: 'PoolStats',
            component: () => import('../views/admin/PoolStats.vue'),
            meta: { titleKey: 'routes.poolStats' }
          },
          {
            path: 'global-roles',
            name: 'GlobalRoles',
            component: () => import('../views/admin/GlobalRoles.vue'),
            meta: { titleKey: 'routes.globalRoles' }
          },
          {
            path: 'users',
            name: 'Users',
            component: () => import('../views/admin/Users.vue'),
            meta: { titleKey: 'routes.users' }
          },
          {
            path: 'devices',
            name: 'AdminDevices',
            component: () => import('../views/admin/Devices.vue'),
            meta: { titleKey: 'routes.devices' }
          },
          {
            path: 'agents',
            name: 'AdminAgents',
            component: () => import('../views/admin/Agents.vue'),
            meta: { titleKey: 'routes.agents' }
          }
        ]
      },
      // Route dành cho người dùng
      {
        path: '/console',
        redirect: '/agents',
        meta: { titleKey: 'routes.agentWorkspace' }
      },
      {
        path: '/agents',
        name: 'Agents',
        component: () => import('../views/user/Agents.vue'),
        meta: { titleKey: 'routes.myAgents' }
      },
      {
        path: '/user/agents',
        name: 'UserAgents',
        component: () => import('../views/user/Agents.vue'),
        meta: { titleKey: 'routes.myAgents' }
      },
      {
        path: '/agents/:id/edit',
        name: 'AgentEdit',
        component: () => import('../views/user/AgentEdit.vue'),
        meta: { titleKey: 'routes.editAgent' }
      },
      {
        path: '/user/agents/:id/edit',
        name: 'UserAgentEdit',
        component: () => import('../views/user/AgentEdit.vue'),
        meta: { titleKey: 'routes.editAgent' }
      },
      {
        path: '/user/agents/:id/devices',
        name: 'AgentDevices',
        component: () => import('../views/user/AgentDevices.vue'),
        meta: { titleKey: 'routes.agentDevices' }
      },
      {
        path: '/user/devices',
        name: 'UserDevices',
        component: () => import('../views/user/AgentDevices.vue'),
        meta: { titleKey: 'routes.deviceList' }
      },
      {
        path: '/speakers',
        name: 'Speakers',
        component: () => import('../views/user/Speakers.vue'),
        meta: { titleKey: 'routes.speakers' }
      },
      {
        path: '/user/speakers',
        name: 'UserSpeakers',
        component: () => import('../views/user/Speakers.vue'),
        meta: { titleKey: 'routes.speakers' }
      },
      {
        path: '/voice-clones',
        name: 'VoiceClones',
        component: () => import('../views/user/VoiceClones.vue'),
        meta: { titleKey: 'routes.voiceClones' }
      },
      {
        path: '/more',
        name: 'MobileMore',
        component: () => import('../views/mobile/MobileMore.vue'),
        meta: { titleKey: 'routes.more' }
      },
      {
        path: '/user/agents/:id/history',
        name: 'AgentHistory',
        component: () => import('../views/user/AgentHistory.vue'),
        meta: { titleKey: 'routes.chatHistory' }
      },

      {
        path: '/user/api-tokens',
        name: 'UserAPITokens',
        component: () => import('../views/user/APITokens.vue'),
        meta: { titleKey: 'routes.apiTokens' }
      },
      {
        path: '/user/knowledge-bases',
        name: 'UserKnowledgeBases',
        component: () => import('../views/user/KnowledgeBases.vue'),
        meta: { titleKey: 'routes.knowledgeBases' }
      },
      {
        path: 'user/roles',
        name: 'UserRoles',
        component: () => import('../views/user/Roles.vue'),
        meta: { titleKey: 'routes.myRoles' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  
  // Nếu truy cập trang hướng dẫn thì cho qua trực tiếp
  if (to.path === '/setup') {
    next()
    return
  }
  
  // Nếu vào trang đăng nhập khi đã đăng nhập thì chuyển theo vai trò; admin chưa hoàn tất wizard lần đầu sẽ sang config wizard
  if (to.path === '/login' && authStore.isAuthenticated) {
    if (authStore.user?.role === 'admin') {
      if (!localStorage.getItem('admin_first_login_done')) {
        next('/admin/config-wizard')
      } else {
        next('/dashboard')
      }
    } else {
      next('/agents')
    }
    return
  }
  
  // Nếu route này yêu cầu xác thực
  if (to.meta.requiresAuth) {
    if (!authStore.isAuthenticated) {
      // Không có token thì chuyển về trang đăng nhập
      next('/login')
      return
    }
    
    // Có token nhưng chưa có thông tin người dùng thì thử xác thực lại token
    if (!authStore.user && !authStore.isValidating) {
      try {
        await authStore.getProfile()
      } catch (error) {
        // Nếu là lỗi 401 (token không hợp lệ) thì chuyển về trang đăng nhập
        if (error.response?.status === 401) {
          next('/login')
          return
        }
        // Nếu là lỗi mạng (backend kết nối thất bại) thì vẫn cho đi tiếp, nhưng sẽ hiển thị lỗi
        if (error.code === 'ERR_NETWORK' || error.message?.includes('Failed to fetch') || error.message?.includes('ERR_CONNECTION_REFUSED')) {
          // Khi lỗi mạng xảy ra, nếu máy cục bộ đã có thông tin người dùng thì vẫn cho truy cập
          if (!authStore.user) {
            next('/login')
            return
          }
          // Không gọi next() ở đây để luồng tiếp tục chạy tới next() cuối cùng
        } else {
          // Với lỗi khác thì vẫn cho đi tiếp, có thể backend chỉ đang tạm thời không khả dụng
          // Không gọi next() ở đây để luồng tiếp tục chạy tới next() cuối cùng
        }
      }
    }
    
    // Nếu đang trong quá trình xác thực thì chờ hoàn tất, tối đa 2 giây
    if (authStore.isValidating) {
      let waitCount = 0
      while (authStore.isValidating && waitCount < 20) {
        await new Promise(resolve => setTimeout(resolve, 100))
        waitCount++
      }
    }
  }
  
  // Nếu vào đường dẫn gốc thì chuyển theo vai trò; admin chưa hoàn tất wizard lần đầu sẽ sang config wizard
  if (to.path === '/' && authStore.isAuthenticated) {
    if (authStore.user?.role === 'admin') {
      if (!localStorage.getItem('admin_first_login_done')) {
        next('/admin/config-wizard')
      } else {
        next('/dashboard')
      }
    } else {
      next('/agents')
    }
    return
  }
  
  // Nếu người dùng thường vào trang quản trị thì chuyển về workspace tác tử
  if (to.meta.requiresAdmin && authStore.user?.role !== 'admin') {
    next('/agents')
    return
  }
  
  next()
})

export default router
