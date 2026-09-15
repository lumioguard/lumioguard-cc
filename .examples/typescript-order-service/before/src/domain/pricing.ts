const CATALOG: Record<string, number> = {
  "BOOK-1": 12.5,
  "BOOK-2": 18.0,
  "FOOD-1": 4.25,
  "TOOL-1": 99.0,
  "TOOL-2": 149.0,
};

export function calculatePrice(sku: string, quantity: number): number {
  const unit = CATALOG[sku];
  if (unit === undefined) {
    return 0;
  }
  if (quantity >= 50) {
    return unit * quantity * 0.8;
  } else if (quantity >= 20) {
    return unit * quantity * 0.9;
  } else if (quantity >= 10) {
    return unit * quantity * 0.95;
  }
  return unit * quantity;
}
