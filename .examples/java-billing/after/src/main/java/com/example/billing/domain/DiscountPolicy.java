package com.example.billing.domain;

import java.util.Map;

/** Coupons as a table of rules instead of a switch that grows for ever. */
public final class DiscountPolicy {

    @FunctionalInterface
    private interface Rule {
        double discount(double total, String tier);
    }

    private static final Map<String, Double> VIP_RATES = Map.of("gold", 0.25, "silver", 0.15);
    private static final double SUMMER_THRESHOLD = 100;

    private static final Map<String, Rule> RULES = Map.of(
            "WELCOME10", (total, tier) -> total * 0.10,
            "SUMMER20", (total, tier) -> total > SUMMER_THRESHOLD ? total * 0.20 : total * 0.05,
            "FREESHIP", (total, tier) -> 0,
            "VIP", (total, tier) -> total * VIP_RATES.getOrDefault(tier, 0.0));

    private DiscountPolicy() {
    }

    public static double discountFor(String couponCode, double total, String tier) {
        Rule rule = couponCode == null ? null : RULES.get(couponCode);
        return rule == null ? 0 : rule.discount(total, tier);
    }

    public static boolean waivesShipping(String couponCode) {
        return "FREESHIP".equals(couponCode);
    }
}
