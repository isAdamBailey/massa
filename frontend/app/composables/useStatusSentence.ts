import type { WeekVerdict } from '~/composables/useWeekVerdict'
import type { WeekOverwhelmSummary } from '~/composables/useOverwhelm'

export interface StatusSentenceSegment {
  text: string
  tag?: boolean
  highlight?: boolean
}

/**
 * useStatusSentence distills the week's weight/energy trend and its
 * overwhelm state into the dashboard's single status line. No values ever
 * appear in the sentence itself - the charts below are where a user goes
 * for the exact numbers.
 *
 * Massa doesn't grade weeks: every trend gets an affirming read, because
 * steady is a good place to be and a week trending up is just data, not a
 * setback. The one moment of real contrast is when the week still felt
 * heavier than usual despite that - that's where "but" earns its keep. The
 * lead phrase of the weight verdict is flagged `highlight: true` so the
 * template can color it in the weight accent (verdigris).
 */
export function useStatusSentence() {
  function trendLead(verdict: Exclude<WeekVerdict, null>): string {
    switch (verdict) {
      case 'better':
        return 'Doing great'
      case 'worse':
        return 'Pick up the pace a little'
      case 'steady':
        return 'Holding steady'
    }
  }

  function trendTail(verdict: Exclude<WeekVerdict, null>): string {
    switch (verdict) {
      case 'better':
        return ' this week — nice momentum!'
      case 'worse':
        return ' this week, no worries, it happens.'
      case 'steady':
        return ' this week, a good place to be.'
    }
  }

  function tagSegments(tagNames: string[]): StatusSentenceSegment[] {
    return tagNames.flatMap((name, index) => {
      const separator = index === 0 ? [] : [{ text: index === tagNames.length - 1 ? ' and ' : ', ' }]
      return [...separator, { text: name, tag: true }]
    })
  }

  function buildStatusSentence(
    hasWeightData: boolean,
    weekVerdict: WeekVerdict,
    overwhelm: WeekOverwhelmSummary | null
  ): StatusSentenceSegment[] | null {
    const elevated = overwhelm?.elevated ? overwhelm : null
    const tagNames = elevated?.topTags.map(t => t.name) ?? []

    if (weekVerdict === null && !elevated) {
      return hasWeightData ? [{ text: 'Not enough history yet to call a trend this week.' }] : null
    }

    if (weekVerdict === null) {
      return tagNames.length
        ? [{ text: 'This week’s been a lot, mostly ' }, ...tagSegments(tagNames), { text: '.' }]
        : [{ text: 'This week’s been a lot.' }]
    }

    const lead: StatusSentenceSegment = { text: trendLead(weekVerdict), highlight: true }

    if (!elevated) {
      return [lead, { text: trendTail(weekVerdict) }]
    }

    const but: StatusSentenceSegment = { text: ' this week, but it’s been a heavier one' }
    return tagNames.length
      ? [lead, but, { text: ', mostly ' }, ...tagSegments(tagNames), { text: '.' }]
      : [lead, but, { text: '.' }]
  }

  return { buildStatusSentence }
}
