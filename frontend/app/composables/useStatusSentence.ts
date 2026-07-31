import type { WeekVerdict } from '~/composables/useWeekVerdict'
import type { WeekOverwhelmSummary } from '~/composables/useOverwhelm'

export interface StatusSentenceSegment {
  text: string
  tag?: boolean
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
 * heavier than usual despite that - that's where "but" earns its keep.
 */
export function useStatusSentence() {
  function trendClause(verdict: Exclude<WeekVerdict, null>): string {
    switch (verdict) {
      case 'better':
        return 'Trending down this week, with great momentum'
      case 'worse':
        return 'Trending up a little this week, and that’s alright, weeks vary'
      case 'steady':
        return 'Holding steady this week, a good place to be'
    }
  }

  function trendLead(verdict: Exclude<WeekVerdict, null>): string {
    switch (verdict) {
      case 'better':
        return 'Trending down'
      case 'worse':
        return 'Trending up'
      case 'steady':
        return 'Holding steady'
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
        ? [{ text: 'This week has felt heavier than usual, mostly ' }, ...tagSegments(tagNames), { text: '.' }]
        : [{ text: 'This week has felt heavier than usual.' }]
    }

    if (!elevated) {
      return [{ text: `${trendClause(weekVerdict)}.` }]
    }

    const lead = `${trendLead(weekVerdict)} this week, but it’s felt heavier than usual`
    return tagNames.length
      ? [{ text: `${lead}, mostly ` }, ...tagSegments(tagNames), { text: '.' }]
      : [{ text: `${lead}.` }]
  }

  return { buildStatusSentence }
}
