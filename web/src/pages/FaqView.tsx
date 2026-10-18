import FaqSection from '@/components/landing/FaqSection'
import LandingCta from '@/components/landing/LandingCta'
import LandingFooter from '@/components/landing/LandingFooter'
import LandingHeader from '@/components/landing/LandingHeader'
import PageHero from '@/components/landing/PageHero'

/** 常见问题页：独立可索引页面，长尾流量入口 */
export default function FaqView() {
  return (
    <>
      <LandingHeader />
      <main className="min-h-screen bg-background text-foreground">
        <PageHero
          eyebrow="常见问题"
          title="关于 Xin API 的常见问题"
          description="产品定位、支持的模型与协议、渠道高可用保障，以及用量统计与计费方式，四个高频问题的完整回答都在这里。"
        />
        <FaqSection />
        <LandingCta />
      </main>
      <LandingFooter />
    </>
  )
}
