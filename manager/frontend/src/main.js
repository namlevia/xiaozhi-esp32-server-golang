import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
// Nạp Vant 4 theo nhu cầu để giảm kích thước bundle
import { 
  NavBar, 
  Tabbar, 
  TabbarItem, 
  Form, 
  Field, 
  CellGroup, 
  Button, 
  Tabs, 
  Tab, 
  Cell,
  Popup,
  Icon
} from 'vant'
import 'vant/lib/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import App from './App.vue'
import router from './router'
import { i18n } from './locales'
import './styles/apple-light.css'

const app = createApp(App)

// Đăng ký toàn bộ icon của Element Plus
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

// Đăng ký các component Vant theo nhu cầu
app.use(NavBar)
app.use(Tabbar)
app.use(TabbarItem)
app.use(Form)
app.use(Field)
app.use(CellGroup)
app.use(Button)
app.use(Tabs)
app.use(Tab)
app.use(Cell)
app.use(Popup)
app.use(Icon)

app.use(createPinia())
app.use(router)
app.use(i18n)
app.use(ElementPlus)  // Dùng cho giao diện desktop

app.mount('#app')
