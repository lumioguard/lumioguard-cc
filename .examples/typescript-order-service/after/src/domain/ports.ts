import { Customer, Order } from "./order";

/**
 * Ports the domain needs from the outside world. The persistence and
 * notification layers implement them, so the dependency points inwards and
 * the layers stay acyclic.
 */
export interface CustomerRepository {
  find(id: string): Promise<Customer | null>;
}

export interface OrderRepository {
  save(order: Order): Promise<void>;
}

export interface Mailer {
  send(to: string, subject: string, body: string): Promise<void>;
}
