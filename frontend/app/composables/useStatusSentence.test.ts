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
      { text: 'Trending down this week, with great momentum.' }
    ])
    expect(buildStatusSentence(true, 'worse', null)).toEqual([
      { text: 'Trending up a little this week, and that’s alright, weeks vary.' }
    ])
    expect(buildStatusSentence(true, 'steady', null)).toEqual([
      { text: 'Holding steady this week, a good place to be.' }
    ])
  })

  it('reports overwhelm alone with tags when there is no trend yet', () => {
    const { buildStatusSentence } = useStatusSentence()
    expect(buildStatusSentence(true, null, elevated([{ name: 'work', count: 3 }]))).toEqual([
      { text: 'This week has felt heavier than usual, mostly ' },
      { text: 'work', tag: true },
      { text: '.' }
    ])
  })

  it('reports overwhelm alone without tags when none were logged', () => {
    const { buildStatusSentence } = useStatusSentence()
    expect(buildStatusSentence(true, null, elevated([]))).toEqual([
      { text: 'This week has felt heavier than usual.' }
    ])
  })

  it('always contrasts with "but" when overwhelm is elevated, regardless of trend direction', () => {
    const { buildStatusSentence } = useStatusSentence()
    expect(buildStatusSentence(true, 'better', elevated([{ name: 'sleep', count: 2 }]))).toEqual([
      { text: 'Trending down this week, but it’s felt heavier than usual, mostly ' },
      { text: 'sleep', tag: true },
      { text: '.' }
    ])
    expect(buildStatusSentence(true, 'worse', elevated([{ name: 'sleep', count: 2 }]))![0]!.text)
      .toContain('Trending up this week, but')
    expect(buildStatusSentence(true, 'steady', elevated([{ name: 'sleep', count: 2 }]))![0]!.text)
      .toContain('Holding steady this week, but')
  })

  it('lists up to three tags in the overwhelm clause', () => {
    const { buildStatusSentence } = useStatusSentence()
    expect(buildStatusSentence(true, 'worse', elevated([
      { name: 'work', count: 3 },
      { name: 'kids', count: 2 },
      { name: 'sleep', count: 1 }
    ]))).toEqual([
      { text: 'Trending up this week, but it’s felt heavier than usual, mostly ' },
      { text: 'work', tag: true },
      { text: ', ' },
      { text: 'kids', tag: true },
      { text: ' and ' },
      { text: 'sleep', tag: true },
      { text: '.' }
    ])
  })
})
