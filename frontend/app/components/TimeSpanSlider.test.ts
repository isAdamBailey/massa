import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import TimeSpanSlider from './TimeSpanSlider.vue'

const stops = [
  { value: '7d', tick: '7d', label: '7 days' },
  { value: '30d', tick: '30d', label: '30 days' },
  { value: '90d', tick: '90d', label: '90 days' },
  { value: 'all', tick: 'All', label: 'All time' }
]

async function mount(modelValue = '30d') {
  return mountSuspended(TimeSpanSlider, {
    props: { modelValue, stops, accent: 'verdigris' as const }
  })
}

describe('TimeSpanSlider', () => {
  it('exposes the current stop as a native range input with a spoken value', async () => {
    const wrapper = await mount('90d')
    const input = wrapper.get('input[type="range"]')
    expect(input.attributes('min')).toBe('0')
    expect(input.attributes('max')).toBe('3')
    expect((input.element as HTMLInputElement).value).toBe('2')
    expect(input.attributes('aria-valuetext')).toBe('90 days')
  })

  it('updates the model live on input without committing', async () => {
    const wrapper = await mount()
    const input = wrapper.get('input[type="range"]')
    ;(input.element as HTMLInputElement).value = '0'
    await input.trigger('input')
    expect(wrapper.emitted('update:modelValue')!.at(-1)).toEqual(['7d'])
    expect(wrapper.emitted('commit')).toBeUndefined()
  })

  it('commits on change', async () => {
    const wrapper = await mount()
    const input = wrapper.get('input[type="range"]')
    ;(input.element as HTMLInputElement).value = '3'
    await input.trigger('change')
    expect(wrapper.emitted('commit')).toEqual([['all']])
  })

  it('jumps to a stop and commits when its tick label is tapped, without adding a tab stop', async () => {
    const wrapper = await mount()
    const ticks = wrapper.findAll('button')
    expect(ticks.map(t => t.attributes('tabindex'))).toEqual(['-1', '-1', '-1', '-1'])
    await ticks[0]!.trigger('click')
    expect(wrapper.emitted('update:modelValue')!.at(-1)).toEqual(['7d'])
    expect(wrapper.emitted('commit')).toEqual([['7d']])
  })
})
