import ContactSection from '@/components/landing/ContactSection'
import LandingFooter from '@/components/landing/LandingFooter'
import LandingHeader from '@/components/landing/LandingHeader'
import PageHero from '@/components/landing/PageHero'

/** 联系我们页：独立可索引页面，长尾流量入口 */
export default function ContactView() {
  return (
    <>
      <LandingHeader />
      <main className="min-h-screen bg-background text-foreground">
        <PageHero
          eyebrow="联系我们"
          title="联系我们"
          description="技术支持、功能建议、开源共建与商务合作，都能在这里找到对应的沟通入口。"
        />
        <ContactSection />
      </main>
      <LandingFooter />
    </>
  )
}
