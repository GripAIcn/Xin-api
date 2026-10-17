import { useEffect, useState } from 'react'
import { RouterProvider, type RouterProviderProps } from 'react-router-dom'

import { Toaster } from '@/components/ui/sonner'

interface AppProps {
  router: RouterProviderProps['router']
}

export default function App({ router }: AppProps) {
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
