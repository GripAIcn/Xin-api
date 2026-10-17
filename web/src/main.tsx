import { StrictMode } from 'react'
import { createRoot, hydrateRoot } from 'react-dom/client'

import App from './App'
import { ThemeProvider } from './components/theme-provider'
import { router } from './router'
import './assets/main.css'

const container = document.getElementById('root')!
const app = (
  <StrictMode>
    <ThemeProvider>
      <App router={router} />
    </ThemeProvider>
  </StrictMode>
)

// 预渲染后的 #root 已含服务端注入的静态 HTML，走水合；否则客户端挂载
if (container.hasChildNodes()) {
  hydrateRoot(container, app)
} else {
  createRoot(container).render(app)
}
