import FaqSection from '@/components/landing/FaqSection'
import FeatureGrid from '@/components/landing/FeatureGrid'
import LandingCta from '@/components/landing/LandingCta'
import LandingFooter from '@/components/landing/LandingFooter'
import LandingHeader from '@/components/landing/LandingHeader'
import LandingHero from '@/components/landing/LandingHero'
import ObservabilitySection from '@/components/landing/ObservabilitySection'
import RequestFlow from '@/components/landing/RequestFlow'
import Roadmap from '@/components/landing/Roadmap'

export default function LandingView() {
  return (
    <>
      <LandingHeader />
      <main className="min-h-screen bg-background text-foreground">
        <LandingHero />
        <FeatureGrid />
        <RequestFlow />
        <ObservabilitySection />
        <Roadmap />
        <FaqSection />
        <LandingCta />
      </main>
      <LandingFooter />
    </>
  )
}
