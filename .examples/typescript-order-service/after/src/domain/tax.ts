import { Customer, OrderItem } from "./order";

interface TaxRates {
  standard: number;
  reduced: number;
}

const RATES_BY_COUNTRY: Record<string, TaxRates> = {
  US: { standard: 0.08, reduced: 0 },
  CA: { standard: 0.13, reduced: 0.05 },
};

const DEFAULT_RATES: TaxRates = { standard: 0.2, reduced: 0.07 };

const REDUCED_CATEGORIES = new Set(["book", "food"]);

/** Tax exemption applies to the standard rate in the United States only. */
function isExempt(customer: Customer, country: string): boolean {
  return customer.taxExempt && country === "US";
}

export function taxFor(item: OrderItem, amount: number, country: string, customer: Customer): number {
  const rates = RATES_BY_COUNTRY[country] ?? DEFAULT_RATES;
  if (REDUCED_CATEGORIES.has(item.category)) {
    return amount * rates.reduced;
  }
  return isExempt(customer, country) ? 0 : amount * rates.standard;
}
