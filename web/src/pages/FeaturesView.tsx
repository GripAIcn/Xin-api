import FeatureGrid from '@/components/landing/FeatureGrid'
import LandingCta from '@/components/landing/LandingCta'
import LandingFooter from '@/components/landing/LandingFooter'
import LandingHeader from '@/components/landing/LandingHeader'
import PageHero from '@/components/landing/PageHero'
import RequestFlow from '@/components/landing/RequestFlow'

/** 功能介绍页：独立可索引页面，长尾流量入口 */
export default function FeaturesView() {
  return (
    <>
      <LandingHeader />
      <main className="min-h-screen bg-background text-foreground">
        <PageHero
          eyebrow="功能介绍"
          title="一个网关，管住多供应商大模型接入"
          description="统一协议接入、多维指标智能选路、分布式熔断与全链路可观测，四件事收在同一个控制面里，接入新供应商不再需要改客户端代码。"
        />
        <FeatureGrid />
        <RequestFlow />
        <LandingCta />
      </main>
      <LandingFooter />
    </>
  )
}
