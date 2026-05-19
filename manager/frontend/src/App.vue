<template>
  <div id="app">
    <router-view />
  </div>
</template>

<script>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/utils/api'

export default {
  name: 'App',
  setup() {
    const router = useRouter()

    const checkSystemStatus = async () => {
      try {
        // Kiểm tra xem hệ thống có cần khởi tạo hay không
        const response = await api.get('/setup/status')
        
        if (response.data.needs_setup) {
          // Nếu cần khởi tạo và hiện chưa ở trang hướng dẫn thì chuyển hướng sang đó
          if (router.currentRoute.value.path !== '/setup') {
            router.push('/setup')
          }
        }
      } catch (error) {
        console.error('Kiểm tra trạng thái hệ thống thất bại:', error)
      }
    }

    onMounted(() => {
      checkSystemStatus()
    })
  }
}
</script>

<style>
#app {
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  height: 100dvh;
}

html,
body {
  height: 100%;
}

body {
  margin: 0;
}

* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}


/* Tối ưu giao diện di động */
@media (max-width: 767px) {
  /* Tối ưu cỡ chữ trên di động */
  body {
    font-size: 14px;
    -webkit-text-size-adjust: 100%;
    -webkit-tap-highlight-color: transparent;
  }

  /* Tối ưu cuộn trên di động */
  * {
    -webkit-overflow-scrolling: touch;
  }

  /* Tối ưu thao tác chạm trên di động */
  a, button, input, textarea {
    touch-action: manipulation;
  }

  /* Ẩn phần tử chỉ dành cho desktop */
  .desktop-only {
    display: none !important;
  }
}

/* Giao diện desktop */
@media (min-width: 768px) {
  /* Ẩn phần tử chỉ dành cho di động */
  .mobile-only {
    display: none !important;
  }
}

/* Hiệu ứng toàn cục */
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.22s ease, transform 0.22s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translateY(4px);
}

/* Thích ứng vùng an toàn trên di động */
@supports (padding: max(0px)) {
  .mobile-safe-top {
    padding-top: max(20px, env(safe-area-inset-top));
  }
  
  .mobile-safe-bottom {
    padding-bottom: max(20px, env(safe-area-inset-bottom));
  }
}
</style>
