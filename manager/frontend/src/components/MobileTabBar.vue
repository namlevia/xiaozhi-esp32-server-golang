<template>
  <van-tabbar
    v-model="activeTab"
    @change="handleTabChange"
    fixed
    placeholder
    safe-area-inset-bottom
    class="mobile-tabbar"
  >
    <van-tabbar-item
      v-for="tab in tabs"
      :key="tab.name"
      :icon="tab.icon"
      :name="tab.name"
    >
      {{ tab.label }}
    </van-tabbar-item>
  </van-tabbar>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const route = useRoute()
const { t } = useI18n()
const authStore = useAuthStore()

const activeTab = ref('')

// Xác định tab bar theo vai trò người dùng
const tabs = computed(() => {
  if (authStore.isAdmin) {
    // Tab bar cho quản trị viên
    return [
      { name: 'dashboard', label: t('menu.home'), icon: 'home-o', path: '/dashboard' },
      { name: 'config', label: t('menu.config'), icon: 'setting-o', path: '/admin/vad-config' },
      { name: 'manage', label: t('menu.manage'), icon: 'apps-o', path: '/admin/users' },
      { name: 'more', label: t('menu.more'), icon: 'ellipsis', path: '/more' }
    ]
  } else {
    // Tab bar cho người dùng thường
    return [
      { name: 'agents', label: t('menu.agents'), icon: 'apps-o', path: '/agents' },
      { name: 'speakers', label: t('menu.speakers'), icon: 'user-o', path: '/user/speakers' },
      { name: 'more', label: t('menu.more'), icon: 'ellipsis', path: '/more' }
    ]
  }
})

// Đặt tab đang active theo route hiện tại
const updateActiveTab = () => {
  const currentPath = route.path
  const currentTab = tabs.value.find(tab => {
    if (tab.path === currentPath) {
      return true
    }
    // Hỗ trợ khớp theo tiền tố đường dẫn
    if (currentPath.startsWith(tab.path)) {
      return true
    }
    return false
  })
  
  if (currentTab) {
    activeTab.value = currentTab.name
  }
}

// Xử lý chuyển tab
const handleTabChange = (name) => {
  const tab = tabs.value.find(item => item.name === name)
  if (tab && tab.path !== route.path) {
    router.push(tab.path)
  }
}

// Theo dõi thay đổi route
watch(
  () => route.path,
  () => {
    updateActiveTab()
  },
  { immediate: true }
)

onMounted(() => {
  updateActiveTab()
})
</script>

<style scoped>
.mobile-tabbar {
  background: transparent;
}

:deep(.van-tabbar) {
  z-index: 1200;
  left: 12px;
  right: 12px;
  bottom: 12px;
  width: auto;
  border-radius: 22px;
  border: 1px solid rgba(255, 255, 255, 0.84);
  box-shadow: var(--apple-shadow-lg);
  overflow: hidden;
}

:deep(.van-tabbar-item--active) {
  color: var(--apple-primary);
}

:deep(.van-tabbar-item) {
  min-height: 58px;
}
</style>
