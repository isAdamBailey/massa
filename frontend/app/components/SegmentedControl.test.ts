import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import SegmentedControl from './SegmentedControl.vue'

const options = [
  { value: 'weight', label: 'Weight' },
  { value: 'bmi', label: 'BMI' },
  { value: 'energy', label: 'Energy' },
  { value: 'overwhelm', label: 'Overwhelm' }
]

async function mount(props: Record<string, unknown> = {}) {
  return mountSuspended(SegmentedControl, {
    props: { modelValue: 'energy', options, groupLabel: 'Metric', stretch: true, ...props }
  })
}

describe('SegmentedControl', () => {
  it('lays primary options out as equal columns, one per option', async () => {
    const wrapper = await mount()
    const group = wrapper.get('[role="group"]')
    expect(group.attributes('style')).toContain('grid-template-columns: repeat(4, minmax(0, 1fr))')
  })

  it('slides the indicator to the active option in the group accent', async () => {
    const wrapper = await mount({ accent: 'copper' })
    const indicator = wrapper.get('span[aria-hidden="true"]')
    expect(indicator.attributes('style')).toContain('translateX(calc(2 * (100% + 4px)))')
    expect(indicator.classes()).toContain('bg-copper')
    expect(indicator.classes()).toContain('motion-reduce:transition-none')

    await wrapper.setProps({ modelValue: 'weight', accent: 'verdigris' })
    expect(indicator.attributes('style')).toContain('translateX(calc(0 * (100% + 4px)))')
    expect(indicator.classes()).toContain('bg-verdigris')
  })

  it('marks the active option pressed and updates the model on click', async () => {
    const wrapper = await mount()
    const buttons = wrapper.findAll('button')
    expect(buttons.map(b => b.attributes('aria-pressed'))).toEqual(['false', 'false', 'true', 'false'])

    await buttons[3]!.trigger('click')
    expect(wrapper.emitted('update:modelValue')!.at(-1)).toEqual(['overwhelm'])
  })

  it('renders quiet options as trackless chips with no indicator', async () => {
    const wrapper = await mount({ emphasis: 'quiet' })
    expect(wrapper.find('span[aria-hidden="true"]').exists()).toBe(false)
    expect(wrapper.get('[role="group"]').classes()).not.toContain('bg-graphite')
  })
})
