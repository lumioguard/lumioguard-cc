const CATALOG: Record<string, number> = {
  "BOOK-1": 12.5,
  "BOOK-2": 18.0,
  "FOOD-1": 4.25,
  "TOOL-1": 99.0,
  "TOOL-2": 149.0,
};

/** Volume discounts, largest qualifying tier first. */
const VOLUME_TIERS: Array<{ minimum: number; factor: number }> = [
  { minimum: 50, factor: 0.8 },
  { minimum: 20, factor: 0.9 },
  { minimum: 10, factor: 0.95 },
];

export function calculatePrice(sku: string, quantity: number): number {
  const unit = CATALOG[sku];
  if (unit === undefined) {
    return 0;
  }
  const tier = VOLUME_TIERS.find((candidate) => quantity >= candidate.minimum);
  return unit * quantity * (tier === undefined ? 1 : tier.factor);
}
