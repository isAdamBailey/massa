const KG_PER_LB = 0.45359237
const CM_PER_IN = 2.54

/**
 * useBmi provides BMI categorization and metric/imperial unit conversions
 * shared across the dashboard and settings forms.
 */
export function useBmi() {
  function category(bmi: number): string {
    if (bmi < 18.5) {
      return 'Underweight'
    }
    if (bmi < 25) {
      return 'Normal'
    }
    if (bmi < 30) {
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
