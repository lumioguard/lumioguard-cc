export interface AddressInput {
  customerId: string;
  street: string;
  city: string;
  postalCode: string;
  country: string;
}

const REQUIRED_FIELDS: Array<[keyof AddressInput, string]> = [
  ["customerId", "customer id"],
  ["street", "street"],
  ["city", "city"],
  ["postalCode", "postal code"],
];

const POSTAL_PATTERNS: Record<string, RegExp> = {
  US: /^\d{5}(-\d{4})?$/,
  CA: /^[A-Z]\d[A-Z] ?\d[A-Z]\d$/,
};

const MINIMUM_POSTAL_LENGTH = 3;

function missingFields(input: AddressInput): string[] {
  return REQUIRED_FIELDS.filter(([field]) => input[field].trim() === "").map(
    ([, label]) => label + " is required",
  );
}

/** Countries without a known pattern only need a plausible length. */
function postalCodeError(postalCode: string, country: string): string | null {
  const pattern = POSTAL_PATTERNS[country];
  if (pattern === undefined) {
    return postalCode.length < MINIMUM_POSTAL_LENGTH ? "invalid postal code" : null;
  }
  return pattern.test(postalCode) ? null : "invalid " + country + " postal code";
}

export function validateAddress(input: AddressInput): string[] {
  const errors = missingFields(input);
  if (input.postalCode.trim() === "") {
    return errors;
  }
  const postal = postalCodeError(input.postalCode, input.country);
  return postal === null ? errors : [...errors, postal];
}
