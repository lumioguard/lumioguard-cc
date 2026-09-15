import { discountFor } from "../domain/discount";
import { Customer, Order, OrderItem } from "../domain/order";
import { CustomerRepository, Mailer, OrderRepository } from "../domain/ports";
import { calculatePrice } from "../domain/pricing";
import { shippingFor } from "../domain/shipping";
import { taxFor } from "../domain/tax";
import { AddressInput, validateAddress } from "./validation";

export interface OrderRequest {
  address: AddressInput;
  items: OrderItem[];
  couponCode: string | null;
  express: boolean;
  giftWrap: boolean;
  notes: string;
}

export interface Dependencies {
  customers: CustomerRepository;
  orders: OrderRepository;
  mailer: Mailer;
}

export interface OrderResult {
  ok: boolean;
  total: number;
  message: string;
}

interface PricedItems {
  total: number;
  tax: number;
  errors: string[];
}

function failure(message: string): OrderResult {
  return { ok: false, total: 0, message };
}

function itemProblem(item: OrderItem): string | null {
  if (item.quantity <= 0) {
    return "invalid quantity for " + item.sku;
  }
  return item.sku === "" ? "missing sku" : null;
}

function priceItems(items: OrderItem[], country: string, customer: Customer): PricedItems {
  const priced: PricedItems = { total: 0, tax: 0, errors: [] };
  for (const item of items) {
    const problem = itemProblem(item);
    if (problem !== null) {
      priced.errors.push(problem);
      continue;
    }
    const price = calculatePrice(item.sku, item.quantity);
    if (price <= 0) {
      priced.errors.push("no price for " + item.sku);
      continue;
    }
    priced.total += price;
    priced.tax += taxFor(item, price, country, customer);
  }
  return priced;
}

function toOrder(request: OrderRequest, total: number): Order {
  const { street, city, postalCode, country, customerId } = request.address;
  return {
    customerId,
    items: request.items,
    total,
    address: street + ", " + city + " " + postalCode + ", " + country,
    notes: request.notes === "" ? "-" : request.notes,
    express: request.express,
    giftWrap: request.giftWrap,
  };
}

/** A confirmation that cannot be delivered must not fail a saved order. */
async function notify(mailer: Mailer, customer: Customer, result: OrderResult, express: boolean): Promise<void> {
  const suffix = express ? " (express)" : "";
  try {
    await mailer.send(customer.email, "Order confirmed", "Your total is " + result.total.toFixed(2) + suffix);
  } catch {
    return;
  }
}

export async function processOrder(request: OrderRequest, deps: Dependencies): Promise<OrderResult> {
  const addressErrors = validateAddress(request.address);
  if (addressErrors.length > 0) {
    return failure(addressErrors.join("; "));
  }

  const customer = await deps.customers.find(request.address.customerId);
  if (customer === null) {
    return failure("unknown customer");
  }

  const priced = priceItems(request.items, request.address.country, customer);
  if (priced.errors.length > 0) {
    return failure(priced.errors.join("; "));
  }

  const discount = discountFor(request.couponCode, priced.total, customer);
  const shipping = shippingFor({
    country: request.address.country,
    express: request.express,
    payable: priced.total - discount,
    giftWrap: request.giftWrap,
    freeShipping: request.couponCode === "FREESHIP",
  });
  const result = { ok: true, total: priced.total - discount + shipping + priced.tax, message: "ok" };

  try {
    await deps.orders.save(toOrder(request, result.total));
  } catch (error) {
    return failure("could not save: " + String(error));
  }
  await notify(deps.mailer, customer, result, request.express);
  return result;
}
