package com.example.billing.persistence;

import com.example.billing.domain.Invoice;
import java.util.ArrayList;
import java.util.List;

public class InvoiceStore {

    private final List<Invoice> invoices = new ArrayList<>();

    public void save(Invoice invoice) {
        if (invoice.amount() < 0) {
            throw new IllegalStateException("negative amount");
        }
        invoices.add(invoice);
    }

    public Invoice find(String customerId) {
        return invoices.stream()
                .filter(invoice -> invoice.customerId().equals(customerId))
                .findFirst()
                .orElse(null);
    }

    public int size() {
        return invoices.size();
    }
}
