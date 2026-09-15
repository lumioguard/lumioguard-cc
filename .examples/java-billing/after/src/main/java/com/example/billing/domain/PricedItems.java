package com.example.billing.domain;

import java.util.List;

public record PricedItems(double total, double tax, List<String> problems) {
}
