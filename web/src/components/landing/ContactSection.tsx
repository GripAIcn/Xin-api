import { Link } from 'react-router-dom'

import Reveal from '@/components/landing/Reveal'
import { Section, SectionHeading } from '@/components/landing/Section'
import {
  CONTACT_EMAIL,
  contactChannels,
} from '@/components/landing/content'

const channelLinkClassName =
  'mt-5 inline-flex min-h-11 items-center text-sm font-medium text-primary transition-opacity hover:opacity-80 md:min-h-0'

export default function ContactSection() {
  return (
    <Section className="border-t">
      <Reveal>
        <SectionHeading
          eyebrow="沟通渠道"
          title="选一个入口，把问题交给我们"
          description="技术支持与功能建议走 GitHub Issues，公开可追溯；账号、用量与渠道配置问题，登录控制台处理更快。"
        />
      </Reveal>

      <div className="mt-12 grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {contactChannels.map((channel) => {
          const Icon = channel.icon
          const { action } = channel
          return (
            <Reveal key={channel.title}>
              <article className="flex h-full flex-col rounded-3xl border bg-card p-6 transition-[border-color,transform] duration-200 hover:-translate-y-0.5 hover:border-primary/40">
                <span className="flex size-11 items-center justify-center rounded-xl border border-primary/15 bg-primary/10 text-primary">
                  <Icon className="size-5" aria-hidden="true" />
                </span>
                <h3 className="mt-6 text-lg font-semibold tracking-tight">
                  {channel.title}
                </h3>
                <p className="mt-2 text-sm leading-6 text-muted-foreground">
                  {channel.desc}
                </p>
                <div className="mt-auto">
                  {action.external ? (
                    <a
                      href={action.href}
                      target="_blank"
                      rel="noreferrer"
                      className={channelLinkClassName}
                    >
                      {action.label}
                    </a>
                  ) : action.href.startsWith('mailto:') ? (
                    // mailto 不是站内路由，必须用原生 <a>，否则会被 Router 当作相对路径
                    <a href={action.href} className={channelLinkClassName}>
                      {action.label}
                    </a>
                  ) : (
                    <Link to={action.href} className={channelLinkClassName}>
                      {action.label}
                    </Link>
                  )}
                </div>
              </article>
            </Reveal>
          )
        })}
      </div>

      <Reveal>
        <div className="mt-14 rounded-3xl border bg-marketing-panel p-8 md:p-10">
          <h2 className="text-xl font-semibold tracking-tight">商务合作</h2>
          <p className="mt-3 max-w-2xl text-sm leading-6 text-muted-foreground">
            私有化部署、定制协议接入、渠道合作等需求，请发送邮件至{' '}
            <a
              href={`mailto:${CONTACT_EMAIL}`}
              className="font-medium text-primary transition-opacity hover:opacity-80"
            >
              {CONTACT_EMAIL}
            </a>
            ，说明使用场景与联系方式，我们会尽快回复。
          </p>
          <a
            href={`mailto:${CONTACT_EMAIL}`}
            className={channelLinkClassName}
          >
            发送邮件
          </a>
        </div>
      </Reveal>
    </Section>
  )
}
