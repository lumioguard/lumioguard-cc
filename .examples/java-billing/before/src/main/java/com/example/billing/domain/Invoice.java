package com.example.billing.domain;

import com.example.billing.persistence.InvoiceStore;
import java.util.List;

public record Invoice(String customerId, double amount, List<String> problems) {

    public Invoice {
        if (customerId == null || customerId.isBlank()) {
            throw new IllegalArgumentException("customer id is required");
        }
    }

    public boolean isClean() {
        return problems.isEmpty();
    }

    public Invoice reload(InvoiceStore store) {
        return store.find(customerId);
    }
}
