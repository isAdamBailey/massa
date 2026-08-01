import { describe, expect, it } from 'vitest'
import { useStatusSentence } from './useStatusSentence'

function elevated(topTags: { name: string, count: number }[]) {
  return { average: 6, count: 3, elevated: true, topTags }
}

describe('useStatusSentence', () => {
  it('returns null when there is nothing to report', () => {
    const { buildStatusSentence } = useStatusSentence()
    expect(buildStatusSentence(false, null, null)).toBeNull()
  })

  it('reports insufficient history when weight data exists but no trend yet', () => {
    const { buildStatusSentence } = useStatusSentence()
    expect(buildStatusSentence(true, null, null)).toEqual([
      { text: 'Not enough history yet to call a trend this week.' }
    ])
  })

  it('affirms every trend direction when overwhelm is not elevated', () => {
    const { buildStatusSentence } = useStatusSentence()
    expect(buildStatusSentence(true, 'better', null)).toEqual([
      { text: 'Doing great', highlight: true },
      { text: ' this week — nice momentum!' }
    ])
    expect(buildStatusSentence(true, 'worse', null)).toEqual([
      { text: 'Pick up the pace a little', highlight: true },
      { text: ' this week, no worries, it happens.' }
    ])
    expect(buildStatusSentence(true, 'steady', null)).toEqual([
      { text: 'Holding steady', highlight: true },
      { text: ' this week, a good place to be.' }
    ])
  })

  it('reports overwhelm alone with tags when there is no trend yet', () => {
    const { buildStatusSentence } = useStatusSentence()
    expect(buildStatusSentence(true, null, elevated([{ name: 'work', count: 3 }]))).toEqual([
      { text: 'This week’s been a lot, mostly ' },
      { text: 'work', tag: true },
      { text: '.' }
    ])
  })

  it('reports overwhelm alone without tags when none were logged', () => {
    const { buildStatusSentence } = useStatusSentence()
    expect(buildStatusSentence(true, null, elevated([]))).toEqual([
      { text: 'This week’s been a lot.' }
    ])
  })

  it('always contrasts with "but" when overwhelm is elevated, regardless of trend direction', () => {
    const { buildStatusSentence } = useStatusSentence()
    expect(buildStatusSentence(true, 'better', elevated([{ name: 'sleep', count: 2 }]))).toEqual([
      { text: 'Doing great', highlight: true },
      { text: ' this week, but it’s been a heavier one' },
      { text: ', mostly ' },
      { text: 'sleep', tag: true },
      { text: '.' }
    ])
    expect(buildStatusSentence(true, 'worse', elevated([{ name: 'sleep', count: 2 }]))![0]!.text)
      .toContain('Pick up the pace a little')
    expect(buildStatusSentence(true, 'steady', elevated([{ name: 'sleep', count: 2 }]))![0]!.text)
      .toContain('Holding steady')
  })

  it('lists up to three tags in the overwhelm clause', () => {
    const { buildStatusSentence } = useStatusSentence()
    expect(buildStatusSentence(true, 'worse', elevated([
      { name: 'work', count: 3 },
      { name: 'kids', count: 2 },
      { name: 'sleep', count: 1 }
    ]))).toEqual([
      { text: 'Pick up the pace a little', highlight: true },
      { text: ' this week, but it’s been a heavier one' },
      { text: ', mostly ' },
      { text: 'work', tag: true },
      { text: ', ' },
      { text: 'kids', tag: true },
      { text: ' and ' },
      { text: 'sleep', tag: true },
      { text: '.' }
    ])
  })
})
