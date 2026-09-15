export interface AddressInput {
  customerId: string;
  street: string;
  city: string;
  postalCode: string;
  country: string;
}

export function validateAddress(input: AddressInput): string[] {
  const errors: string[] = [];
  const { customerId, street, city, postalCode, country } = input;

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

  return errors;
}
