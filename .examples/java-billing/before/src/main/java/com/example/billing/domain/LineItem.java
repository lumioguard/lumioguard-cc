package com.example.billing.domain;

public record LineItem(String sku, int quantity, double unitPrice, String category) {
}
