import { Customer, Order } from "../domain/order";
import { CustomerRepository, OrderRepository } from "../domain/ports";

const CUSTOMERS: Customer[] = [
  { id: "c-1", email: "ada@example.invalid", tier: "gold", taxExempt: false },
  { id: "c-2", email: "grace@example.invalid", tier: "silver", taxExempt: true },
];

export class InMemoryCustomers implements CustomerRepository {
  async find(id: string): Promise<Customer | null> {
    const found = CUSTOMERS.find((candidate) => candidate.id === id);
    return found === undefined ? null : found;
  }
}

export class InMemoryOrders implements OrderRepository {
  private readonly stored: Order[] = [];

  async save(order: Order): Promise<void> {
    this.stored.push(order);
  }

  count(): number {
    return this.stored.length;
  }
}
