package com.example.billing.domain;

import java.util.Map;

public final class ShippingPolicy {

    private record Rates(double standard, double express) {
    }

    private static final Map<String, Rates> BY_COUNTRY =
            Map.of("US", new Rates(8, 25), "CA", new Rates(14, 35));
    private static final Rates DEFAULT_RATES = new Rates(30, 60);
    private static final double FREE_SHIPPING_THRESHOLD = 150;
    private static final double GIFT_WRAP_FEE = 5;

    private ShippingPolicy() {
    }

    public static double shippingFor(ShippingRequest request) {
        double wrapping = request.giftWrap() ? GIFT_WRAP_FEE : 0;
        if (request.freeShipping()) {
            return wrapping;
        }
        Rates rates = BY_COUNTRY.getOrDefault(request.country(), DEFAULT_RATES);
        if (request.express()) {
            return rates.express() + wrapping;
        }
        boolean earnedFreeShipping = request.payable() > FREE_SHIPPING_THRESHOLD;
        return (earnedFreeShipping ? 0 : rates.standard()) + wrapping;
    }
}
