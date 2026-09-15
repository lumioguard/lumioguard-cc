import { OrderItem, loadCustomer, saveOrder } from "../domain/order";
import { calculatePrice } from "../domain/pricing";
import { sendEmail } from "../notification/email";

export async function processOrder(
  customerId: string,
  items: OrderItem[],
  couponCode: string | null,
  country: string,
  postalCode: string,
  street: string,
  city: string,
  express: boolean,
  giftWrap: boolean,
  notes: string,
): Promise<{ ok: boolean; total: number; message: string }> {
  let total = 0;
  let discount = 0;
  let shipping = 0;
  let tax = 0;
  const errors: string[] = [];

  if (!customerId || customerId.trim() === "") {
    errors.push("customer id is required");
  }
  if (!street || street.trim() === "") {
    errors.push("street is required");
  }
  if (!city || city.trim() === "") {
    errors.push("city is required");
  }
  if (!postalCode || postalCode.trim() === "") {
    errors.push("postal code is required");
  } else if (country === "US" && !/^\d{5}(-\d{4})?$/.test(postalCode)) {
    errors.push("invalid US postal code");
  } else if (country === "CA" && !/^[A-Z]\d[A-Z] ?\d[A-Z]\d$/.test(postalCode)) {
    errors.push("invalid CA postal code");
  } else if (country !== "US" && country !== "CA" && postalCode.length < 3) {
    errors.push("invalid postal code");
  }

  if (errors.length > 0) {
    return { ok: false, total: 0, message: errors.join("; ") };
  }

  const customer = await loadCustomer(customerId);
  if (customer === null) {
    return { ok: false, total: 0, message: "unknown customer" };
  }

  for (const item of items) {
    if (item.quantity > 0) {
      if (item.sku !== "") {
        const price = calculatePrice(item.sku, item.quantity);
        if (price > 0) {
          if (item.category === "book" || item.category === "food") {
            if (country === "US") {
              tax = tax + price * 0.0;
            } else if (country === "CA") {
              tax = tax + price * 0.05;
            } else {
              tax = tax + price * 0.07;
            }
          } else {
            if (country === "US") {
              if (customer.taxExempt) {
                tax = tax + 0;
              } else {
                tax = tax + price * 0.08;
              }
            } else if (country === "CA") {
              tax = tax + price * 0.13;
            } else {
              tax = tax + price * 0.2;
            }
          }
          total = total + price;
        } else {
          errors.push("no price for " + item.sku);
        }
      } else {
        errors.push("missing sku");
      }
    } else {
      errors.push("invalid quantity for " + item.sku);
    }
  }

  if (errors.length > 0) {
    return { ok: false, total: 0, message: errors.join("; ") };
  }

  if (couponCode !== null && couponCode !== "") {
    switch (couponCode) {
      case "WELCOME10":
        discount = total * 0.1;
        break;
      case "SUMMER20":
        if (total > 100) {
          discount = total * 0.2;
        } else {
          discount = total * 0.05;
        }
        break;
      case "FREESHIP":
        shipping = 0;
        discount = 0;
        break;
      case "VIP":
        if (customer.tier === "gold") {
          discount = total * 0.25;
        } else if (customer.tier === "silver") {
          discount = total * 0.15;
        } else {
          discount = 0;
        }
        break;
      default:
        discount = 0;
    }
  }

  if (couponCode !== "FREESHIP") {
    if (country === "US") {
      shipping = express ? 25 : 8;
    } else if (country === "CA") {
      shipping = express ? 35 : 14;
    } else {
      shipping = express ? 60 : 30;
    }
    if (total - discount > 150 && !express) {
      shipping = 0;
    }
  }

  if (giftWrap) {
    shipping = shipping + 5;
  }

  const grandTotal = total - discount + shipping + tax;

  try {
    await saveOrder({
      customerId,
      items,
      total: grandTotal,
      address: street + ", " + city + " " + postalCode + ", " + country,
      notes: notes === "" ? "-" : notes,
      express,
      giftWrap,
    });
  } catch (error) {
    return { ok: false, total: 0, message: "could not save: " + String(error) };
  }

  try {
    await sendEmail(
      customer.email,
      "Order confirmed",
      "Your total is " + grandTotal.toFixed(2) + (express ? " (express)" : ""),
    );
  } catch (error) {
    // The order is saved, so a failed email is not fatal.
  }

  return { ok: true, total: grandTotal, message: "ok" };
}
