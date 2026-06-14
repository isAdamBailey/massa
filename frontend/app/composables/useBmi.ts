const KG_PER_LB = 0.45359237
const CM_PER_IN = 2.54

/**
 * BMI category boundaries (WHO adult thresholds), shared between the
 * category label and the chart's reference-range bands.
 */
export const BMI_BOUNDARIES = {
  underweight: 18.5,
  overweight: 25,
  obese: 30
} as const

/**
 * useBmi provides BMI categorization and metric/imperial unit conversions
 * shared across the dashboard and settings forms.
 */
export function useBmi() {
  function category(bmi: number): string {
    if (bmi < BMI_BOUNDARIES.underweight) {
      return 'Underweight'
    }
    if (bmi < BMI_BOUNDARIES.overweight) {
      return 'Normal'
    }
    if (bmi < BMI_BOUNDARIES.obese) {
      return 'Overweight'
    }
    return 'Obese'
  }

  function kgToLb(kg: number): number {
    return kg / KG_PER_LB
  }

  function lbToKg(lb: number): number {
    return lb * KG_PER_LB
  }

  function cmToIn(cm: number): number {
    return cm / CM_PER_IN
  }

  function inToCm(inches: number): number {
    return inches * CM_PER_IN
  }

  return { category, kgToLb, lbToKg, cmToIn, inToCm }
}
