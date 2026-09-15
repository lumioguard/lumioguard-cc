package com.example.billing.domain;

public record TaxContext(String country, String region, boolean exempt) {

    public boolean isUnitedStates() {
        return "US".equals(country);
    }
}
