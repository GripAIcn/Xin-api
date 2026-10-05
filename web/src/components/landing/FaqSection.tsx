import { ChevronDown } from 'lucide-react'

import { faqItems } from '@/components/landing/content'

export default function FaqSection() {
  return (
    <section id="faq" className="scroll-mt-20 py-20">
      <div className="mx-auto w-full max-w-6xl px-4 sm:px-6 lg:px-8">
        <div className="mx-auto max-w-2xl text-center">
          <span className="rounded-full bg-primary/10 px-3 py-1 text-xs font-medium text-primary">
            常见问题
          </span>
          <h2 className="mt-4 text-3xl font-bold tracking-tight">常见问题</h2>
        </div>
        <div className="mx-auto mt-10 max-w-3xl space-y-3">
          {faqItems.map((item) => (
            <details
              key={item.q}
              className="group rounded-2xl border bg-card"
            >
              <summary className="flex min-h-11 cursor-pointer list-none items-center justify-between gap-4 px-5 py-4 font-medium marker:hidden [&::-webkit-details-marker]:hidden">
                <span>{item.q}</span>
                <ChevronDown
                  aria-hidden="true"
                  className="size-4 shrink-0 text-muted-foreground transition-transform duration-200 group-open:rotate-180"
                />
              </summary>
              <div className="px-5 pb-4 text-sm leading-relaxed text-muted-foreground">
                {item.a}
              </div>
            </details>
          ))}
        </div>
      </div>
    </section>
  )
}
