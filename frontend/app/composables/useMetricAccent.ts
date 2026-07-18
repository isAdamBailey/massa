export type MetricAccent = 'verdigris' | 'copper' | 'cobalt'

export type ChartMetricMode = 'weight' | 'bmi' | 'energy' | 'overwhelm'
export type LogTab = 'weight' | 'overwhelm'

/** Chart.js / canvas OKLCH strings for each metric accent. */
export const METRIC_ACCENT_OKLCH: Record<MetricAccent, {
  solid: string
  fill: string
  wash: string
  hover: string
}> = {
  verdigris: {
    solid: 'oklch(0.70 0.09 170)',
    fill: 'oklch(0.70 0.09 170 / 0.18)',
    wash: 'oklch(0.70 0.09 170 / 0.08)',
    hover: 'oklch(0.76 0.09 170)'
  },
  copper: {
    solid: 'oklch(0.72 0.12 55)',
    fill: 'oklch(0.72 0.12 55 / 0.22)',
    wash: 'oklch(0.72 0.12 55 / 0.09)',
    hover: 'oklch(0.78 0.12 55)'
  },
  cobalt: {
    solid: 'oklch(0.70 0.09 245)',
    fill: 'oklch(0.70 0.09 245 / 0.20)',
    wash: 'oklch(0.70 0.09 245 / 0.09)',
    hover: 'oklch(0.76 0.09 245)'
  }
}

const accentActiveClass: Record<MetricAccent, string> = {
  verdigris: 'bg-verdigris text-carbon',
  copper: 'bg-copper text-carbon',
  cobalt: 'bg-cobalt text-carbon'
}

const accentHoverClass: Record<MetricAccent, string> = {
  verdigris: 'hover:bg-verdigris-hover',
  copper: 'hover:bg-copper-hover',
  cobalt: 'hover:bg-cobalt-hover'
}

const accentFocusClass: Record<MetricAccent, string> = {
  verdigris: 'focus:outline-verdigris',
  copper: 'focus:outline-copper',
  cobalt: 'focus:outline-cobalt'
}

export function accentForChartMetric(mode: ChartMetricMode): MetricAccent {
  switch (mode) {
    case 'energy':
      return 'copper'
    case 'overwhelm':
      return 'cobalt'
    default:
      return 'verdigris'
  }
}

export function accentForLogTab(tab: LogTab): MetricAccent {
  return tab === 'overwhelm' ? 'cobalt' : 'verdigris'
}

export function accentActiveClasses(accent: MetricAccent): string {
  return accentActiveClass[accent]
}

export function accentButtonClasses(accent: MetricAccent): string {
  return `${accentActiveClass[accent]} ${accentHoverClass[accent]}`
}

export function accentFocusClasses(accent: MetricAccent): string {
  return accentFocusClass[accent]
}
