import FaqSection from '@/components/landing/FaqSection'
import FeatureGrid from '@/components/landing/FeatureGrid'
import LandingCta from '@/components/landing/LandingCta'
import LandingFooter from '@/components/landing/LandingFooter'
import LandingHeader from '@/components/landing/LandingHeader'
import LandingHero from '@/components/landing/LandingHero'
import ObservabilitySection from '@/components/landing/ObservabilitySection'
import RequestFlow from '@/components/landing/RequestFlow'

export default function LandingView() {
  return (
    <>
      <LandingHeader />
      <main className="min-h-screen bg-background text-foreground">
        <LandingHero />
        <FeatureGrid />
        <RequestFlow />
        <ObservabilitySection />
        <FaqSection />
        <LandingCta />
      </main>
      <LandingFooter />
    </>
  )
}
