interface ShippingRates {
  standard: number;
  express: number;
}

const RATES_BY_COUNTRY: Record<string, ShippingRates> = {
  US: { standard: 8, express: 25 },
  CA: { standard: 14, express: 35 },
};

const DEFAULT_RATES: ShippingRates = { standard: 30, express: 60 };

const FREE_SHIPPING_THRESHOLD = 150;
const GIFT_WRAP_FEE = 5;

export interface ShippingRequest {
  country: string;
  express: boolean;
  /** Order value after discount, which decides free standard shipping. */
  payable: number;
  giftWrap: boolean;
  freeShipping: boolean;
}

export function shippingFor(request: ShippingRequest): number {
  const wrapping = request.giftWrap ? GIFT_WRAP_FEE : 0;
  if (request.freeShipping) {
    return wrapping;
  }
  const rates = RATES_BY_COUNTRY[request.country] ?? DEFAULT_RATES;
  if (request.express) {
    return rates.express + wrapping;
  }
  const earnedFreeShipping = request.payable > FREE_SHIPPING_THRESHOLD;
  return (earnedFreeShipping ? 0 : rates.standard) + wrapping;
}
