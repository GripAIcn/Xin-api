import { useEffect, useState } from 'react'
import { RouterProvider } from 'react-router-dom'

import { Toaster } from '@/components/ui/sonner'
import { router } from '@/router'

export default function App() {
  // Toaster（sonner/react-aria TopLayer）仅渲染客户端 portal，
  // 挂载后再挂载，保证与预渲染 HTML 的水合首帧完全一致
  const [mounted, setMounted] = useState(false)
  useEffect(() => {
    const markMounted = () => setMounted(true)
    markMounted()
  }, [])

  return (
    <>
      <RouterProvider router={router} />
      {mounted ? <Toaster richColors position="top-right" /> : null}
    </>
  )
}