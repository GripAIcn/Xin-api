import { ChevronDown } from 'lucide-react'

import { Section, SectionHeading } from '@/components/landing/Section'
import Reveal from '@/components/landing/Reveal'
import { faqItems } from '@/components/landing/content'

const ISSUES_URL = 'https://github.com/GripAIcn/Xin-api/issues'

export default function FaqSection() {
  return (
    <Section id="faq" className="border-t">
      <div className="grid gap-12 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)] lg:gap-20">
        <Reveal>
          <SectionHeading
            eyebrow="常见问题"
            title="关于 Xin API，你可能想知道"
            description="还有别的疑问？欢迎在 GitHub Issues 留言，我们会尽快回复。"
          />
          <a
            href={ISSUES_URL}
            target="_blank"
            rel="noreferrer"
            className="mt-6 inline-flex min-h-11 items-center text-sm font-medium text-primary transition-opacity hover:opacity-80 md:min-h-0"
          >
            前往 GitHub Issues
          </a>
        </Reveal>

        <Reveal>
          <div className="border-t">
            {faqItems.map((item) => (
              <details key={item.q} className="group border-b">
                <summary className="flex min-h-14 cursor-pointer list-none items-center justify-between gap-6 py-5 font-medium marker:hidden [&::-webkit-details-marker]:hidden">
                  <span>{item.q}</span>
                  <ChevronDown
                    aria-hidden="true"
                    className="size-4 shrink-0 text-muted-foreground transition-transform duration-200 group-open:rotate-180"
                  />
                </summary>
                <p className="pb-5 text-sm leading-7 text-muted-foreground">
                  {item.a}
                </p>
              </details>
            ))}
          </div>
        </Reveal>
      </div>
    </Section>
  )
}
