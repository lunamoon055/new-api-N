import { afterEach, describe, expect, it } from 'bun:test'
import type { ReactElement } from 'react'
import { renderToStaticMarkup } from 'react-dom/server'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { createInstance } from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { useAuthStore } from '../src/stores/auth-store'
import { ModelCard } from '../src/features/pricing/components/model-card'
import { ModelCardCapabilities } from '../src/features/pricing/components/model-card-capabilities'
import { ModelCardPricing } from '../src/features/pricing/components/model-card-pricing'
import type {
  IntegrationParameters,
  PricingModel,
} from '../src/features/pricing/types'
import en from '../src/i18n/locales/en.json'
import zh from '../src/i18n/locales/zh.json'
import v6kProfiles from '../../../common/v6k_video_models.json'

const i18n = createInstance()
await i18n.init({
  lng: 'zh',
  resources: { en, zh },
  interpolation: { escapeValue: false },
})

function model(overrides: Partial<PricingModel> = {}): PricingModel {
  return {
    id: 1,
    model_name: 'public-video',
    quota_type: 1,
    model_ratio: 0,
    completion_ratio: 0,
    enable_groups: ['default'],
    price_configured: true,
    model_price: 0.1,
    video_billing_mode: 'dynamic',
    ...overrides,
  }
}

function video(parameters?: Partial<IntegrationParameters>): PricingModel {
  return model({
    mapped_endpoints: [
      {
        type: 'openai-video',
        kind: 'video',
        path: '/v1/videos',
        method: 'POST',
        documentation_fields: parameters
          ? { max_images: 0, max_audios: 0, max_videos: 0, ...parameters }
          : undefined,
      },
    ],
  })
}

function render(element: ReactElement): string {
  return renderToStaticMarkup(
    <I18nextProvider i18n={i18n}>
      <QueryClientProvider client={new QueryClient()}>
        {element}
      </QueryClientProvider>
    </I18nextProvider>
  )
}

function text(html: string): string {
  return html.replace(/<[^>]*>/g, '')
}

afterEach(() => useAuthStore.getState().auth.setUser(null))

describe('model card public display', () => {
  it('shows the configured SD duration ranges without upstream-count notices', () => {
    for (const [name, expected] of [
      ['SD2.0', '5–15 秒'],
      ['SD2.5', '5–30 秒'],
    ] as const) {
      const output = text(
        render(<ModelCardCapabilities model={video(v6kProfiles[name])} />)
      )
      expect(output).toContain(expected)
      expect(output).not.toContain('上游视频数量未公布')
    }
  })
  it('shows every tier and keeps generation and reference-video prices separate, including zero', () => {
    const output = text(
      render(
        <ModelCardPricing
          model={model({
            video_billing_mode: 'tiered_seconds',
            video_resolution_prices: { '480p': 0.1, '720p': 0.2, '1080p': 0.3 },
            video_reference_prices: { '480p': 0.4, '720p': 0, '1080p': 0.6 },
          })}
        />
      )
    )
    expect(output).toContain('480p$0.1/秒$0.4/秒')
    expect(output).toContain('720p$0.2/秒$0/秒')
    expect(output).toContain('1080p$0.3/秒$0.6/秒')
    expect(output).toContain('参考视频按输入时长另计')
  })

  it('keeps missing prices unconfigured and sorts tiers that exist only in reference pricing', () => {
    const output = text(
      render(
        <ModelCardPricing
          model={model({
            video_billing_mode: 'tiered_request',
            video_resolution_prices: { '720p': 2, '1080p': 3 },
            video_reference_prices: { '480p': 0.1, '720p': 0.2 },
          })}
        />
      )
    )
    expect(output).toContain('480p未配置$0.1/秒')
    expect(output).toContain('720p$2/次$0.2/秒')
    expect(output).toContain('1080p$3/次未配置')
    expect(output.indexOf('480p')).toBeLessThan(output.indexOf('720p'))
    expect(
      text(
        render(
          <ModelCardPricing
            model={model({ price_configured: false, model_price: 0 })}
          />
        )
      )
    ).toBe('价格待配置')
  })

  it('uses the existing group and recharge price conversion', () => {
    const output = text(
      render(
        <ModelCardPricing
          model={model({
            video_billing_mode: 'tiered_seconds',
            group_ratio: { default: 2 },
            video_resolution_prices: { '720p': 0.1 },
            video_reference_prices: { '720p': 0.2 },
          })}
          showRechargePrice
          priceRate={0.5}
          usdExchangeRate={1}
        />
      )
    )
    expect(output).toContain('720p$0.1/秒$0.2/秒')
  })

  it('preserves token/cache pricing and fixed request pricing', () => {
    const output = text(
      render(
        <ModelCardPricing
          model={model({
            quota_type: 0,
            model_ratio: 1,
            completion_ratio: 4,
            cache_ratio: 0,
            video_billing_mode: undefined,
          })}
        />
      )
    )
    expect(output).toContain('输入 $2/1M')
    expect(output).toContain('输出 $8/1M')
    expect(output).toContain('缓存 $0/1M')
    expect(
      text(
        render(
          <ModelCardPricing
            model={model({ video_billing_mode: 'fixed', model_price: 2 })}
          />
        )
      )
    ).toBe('$2/次')
  })

  it('preserves discrete durations and hides aspect ratios from cards', () => {
    const output = text(
      render(
        <ModelCardCapabilities
          model={video({
            durations: [15, 5, 10],
            ratios: ['16:9', '9:16', '1:1', '4:3', '3:4', '2:3', '3:2', '21:9'],
            max_images: 9,
            max_audios: 3,
            max_videos: 3,
          })}
        />
      )
    )
    expect(output).toContain('5 / 10 / 15 秒')
    expect(output).toContain('图片 9 张')
    for (const ratio of [
      '16:9',
      '9:16',
      '1:1',
      '4:3',
      '3:4',
      '2:3',
      '3:2',
      '21:9',
    ])
      expect(output).not.toContain(ratio)
    expect(
      text(
        render(
          <ModelCardCapabilities model={video({ durations: [5, 6, 7] })} />
        )
      )
    ).toContain('5–7 秒')
  })

  it('separates unknown capabilities, unsupported audio and the site reference-video limit', () => {
    const output = text(
      render(
        <ModelCardCapabilities
          model={video({
            durations: [5, 10],
            unknown_fields: ['durations', 'max_images', 'max_videos'],
            max_images: 30,
            max_audios: 0,
            max_videos: 10,
            reference_video_billed_per_second: true,
          })}
        />
      )
    )
    expect(output).toContain('时长未公布')
    expect(output).toContain('图片 未公布')
    expect(output).toContain('音频 不支持')
    expect(output).toContain('本站最多 10 段')
    expect(output).not.toContain('上游视频数量未公布')
    expect(output).not.toContain('图片 30 张')
    expect(text(render(<ModelCardCapabilities model={video()} />))).toBe(
      '模型能力未确认。'
    )
  })

  it('uses public documentation fields instead of provider-internal capability fields', () => {
    const m = video({ max_images: 9 })
    m.mapped_endpoints![0].parameters = {
      max_images: 100,
      max_audios: 100,
      max_videos: 100,
    }
    const output = text(render(<ModelCardCapabilities model={m} />))
    expect(output).toContain('图片 9 张')
    expect(output).not.toContain('100')
  })

  it('keeps full names and note/status controls, without token labels on video cards', () => {
    const m = {
      ...video({ max_images: 9 }),
      model_name: 'V6K/TTseadanco2.5-1080',
      supported_endpoint_types: ['openai-video'],
    }
    let output = render(<ModelCard model={m} onClick={() => {}} />)
    expect(output).toContain('V6K/TTseadanco2.5-1080')
    expect(output).toContain('复制模型名称：V6K/TTseadanco2.5-1080')
    expect(output).toContain('按秒计费')
    expect(output).not.toContain('1M')
    expect(output).not.toContain('暂无描述')
    expect(output).not.toContain('对外展示')
    useAuthStore
      .getState()
      .auth.setUser({ id: 1, username: 'display-test-admin', role: 100 })
    // Zustand uses the initial server snapshot during SSR; simulate this request's admin snapshot.
    const initialSnapshot = useAuthStore.getInitialState()
    const initialAuth = initialSnapshot.auth
    try {
      initialSnapshot.auth = {
        ...initialAuth,
        user: useAuthStore.getState().auth.user,
      }
      output = render(<ModelCard model={m} onClick={() => {}} />)
    } finally {
      initialSnapshot.auth = initialAuth
    }
    expect(output).toContain('对外展示')
    expect(output).toContain('添加备注')
    expect(output).toContain('role="switch"')
  })
})
