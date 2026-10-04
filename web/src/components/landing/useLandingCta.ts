import { useEffect, useState } from 'react'

import { useAuthStore } from '@/stores/auth'

export interface LandingCta {
  to: string
  label: string
}

const GUEST_CTA: LandingCta = { to: '/login', label: '登录控制台' }
const AUTH_CTA: LandingCta = { to: '/dashboard', label: '进入控制台' }

// SSR / 水合安全：首帧恒为访客 CTA，挂载后在 effect 中与 auth store 同步
export function useLandingCta(): LandingCta {
  const [cta, setCta] = useState<LandingCta>(GUEST_CTA)

  useEffect(() => {
    const sync = (authenticated: boolean) => {
      setCta(authenticated ? AUTH_CTA : GUEST_CTA)
    }
    sync(useAuthStore.getState().isAuthenticated)
    return useAuthStore.subscribe((state) => sync(state.isAuthenticated))
  }, [])

  return cta
}
