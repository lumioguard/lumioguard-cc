import { Customer, Order } from "../domain/order";

const CUSTOMERS: Customer[] = [
  { id: "c-1", email: "ada@example.invalid", tier: "gold", taxExempt: false },
  { id: "c-2", email: "grace@example.invalid", tier: "silver", taxExempt: true },
];

const ORDERS: Order[] = [];

export async function readCustomer(id: string): Promise<Customer | null> {
  const found = CUSTOMERS.find((candidate) => candidate.id === id);
  return found === undefined ? null : found;
}

export async function writeOrder(order: Order): Promise<void> {
  ORDERS.push(order);
}

export async function countOrders(): Promise<number> {
  return ORDERS.length;
}
