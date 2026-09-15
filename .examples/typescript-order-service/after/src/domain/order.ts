export interface OrderItem {
  sku: string;
  quantity: number;
  category: string;
}

export interface Customer {
  id: string;
  email: string;
  tier: string;
  taxExempt: boolean;
}

export interface Order {
  customerId: string;
  items: OrderItem[];
  total: number;
  address: string;
  notes: string;
  express: boolean;
  giftWrap: boolean;
}
