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
import { Grainient } from '../grainient'

export function Hero() {
  return (
    <section className='relative z-10 flex min-h-svh flex-col items-center justify-center overflow-hidden bg-[#5227ff] px-6 py-24'>
      <div className='absolute inset-0 z-0'>
        <Grainient
          color1='#f1cdef'
          color2='#5227FF'
          color3='#ffffff'
          timeSpeed={0.25}
          colorBalance={0}
          warpStrength={1}
          warpFrequency={5}
          warpSpeed={2}
          warpAmplitude={50}
          blendAngle={0}
          blendSoftness={0.05}
          rotationAmount={500}
          noiseScale={2}
          grainAmount={0.1}
          grainScale={2}
          grainAnimated={false}
          contrast={1.5}
          gamma={1}
          saturation={1}
          centerX={0}
          centerY={0}
          zoom={0.9}
        />
      </div>
      <div className='absolute inset-0 z-10 bg-slate-950/20' />

      <div className='relative z-20 max-w-7xl text-center'>
        <h1
          className='landing-animate-fade-up text-[clamp(2.25rem,8vw,7rem)] leading-[1.08] font-semibold tracking-normal text-white drop-shadow-[0_18px_40px_rgba(15,23,42,0.35)]'
          style={{ animationDelay: '0ms' }}
        >
          <span className='inline-block'>欢迎使用</span>
          <wbr />
          <span className='inline-block'>GHLINK API</span>
        </h1>
      </div>
    </section>
  )
}
