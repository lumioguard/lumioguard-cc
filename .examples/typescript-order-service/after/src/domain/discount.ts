import { Customer } from "./order";

const VIP_RATES: Record<string, number> = { gold: 0.25, silver: 0.15 };

type CouponRule = (total: number, customer: Customer) => number;

const COUPONS: Record<string, CouponRule> = {
  WELCOME10: (total) => total * 0.1,
  SUMMER20: (total) => (total > 100 ? total * 0.2 : total * 0.05),
  FREESHIP: () => 0,
  VIP: (total, customer) => total * (VIP_RATES[customer.tier] ?? 0),
};

export function discountFor(couponCode: string | null, total: number, customer: Customer): number {
  const rule = couponCode === null ? undefined : COUPONS[couponCode];
  return rule === undefined ? 0 : rule(total, customer);
}
