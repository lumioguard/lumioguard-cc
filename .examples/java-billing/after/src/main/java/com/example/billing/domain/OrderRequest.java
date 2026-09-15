package com.example.billing.domain;

import java.util.List;

/** Everything needed to bill one order, replacing an eight-argument call. */
public record OrderRequest(
        String customerId,
        List<LineItem> items,
        String couponCode,
        String country,
        String region,
        String tier,
        boolean express,
        boolean giftWrap) {

    public boolean isTaxExempt() {
        return "exempt".equals(tier);
    }

    public TaxContext taxContext() {
        return new TaxContext(country, region, isTaxExempt());
    }
}
