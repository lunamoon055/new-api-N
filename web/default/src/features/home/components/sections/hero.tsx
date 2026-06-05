/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()

  return (
    <section className='ghlink-grainient-hero relative z-10 flex min-h-svh flex-col items-center justify-center overflow-hidden px-6 pt-28 pb-16 md:pt-32 md:pb-20'>
      <div className='relative z-10 flex max-w-5xl flex-col items-center text-center'>
        <h1
          className='landing-animate-fade-up text-[clamp(3rem,9vw,7.5rem)] leading-[0.94] font-semibold tracking-normal text-slate-950'
          style={{ animationDelay: '0ms' }}
        >
          <span className='block'>欢迎使用GHLINK API</span>
          <span className='mt-4 block text-slate-900/85'>激发你的想象</span>
        </h1>
        <div
          className='landing-animate-fade-up mt-10 flex flex-wrap items-center justify-center gap-3 opacity-0'
          style={{ animationDelay: '120ms' }}
        >
          {props.isAuthenticated ? (
            <Button
              className='group h-11 rounded-full px-6 shadow-[0_14px_45px_-22px_rgba(15,23,42,0.8)]'
              render={<Link to='/dashboard' />}
            >
              {t('Go to Dashboard')}
              <ArrowRight className='ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
            </Button>
          ) : (
            <>
              <Button
                className='group h-11 rounded-full px-6 shadow-[0_14px_45px_-22px_rgba(15,23,42,0.8)]'
                render={<Link to='/sign-up' />}
              >
                {t('Get Started')}
                <ArrowRight className='ml-1 size-3.5 transition-transform duration-200 group-hover:translate-x-0.5' />
              </Button>
              <Button
                variant='outline'
                className='h-11 rounded-full border-white/70 bg-white/55 px-6 text-slate-900 shadow-[0_14px_45px_-24px_rgba(15,23,42,0.75)] backdrop-blur-xl hover:bg-white/80'
                render={<Link to='/pricing' />}
              >
                {t('View Pricing')}
              </Button>
            </>
          )}
        </div>
      </div>
    </section>
  )
}
