package com.example.billing.domain;

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
}
