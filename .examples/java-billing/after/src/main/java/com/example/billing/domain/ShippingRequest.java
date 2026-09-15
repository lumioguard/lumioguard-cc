package com.example.billing.domain;

/** ``payable`` is the order value after discount, which earns free shipping. */
public record ShippingRequest(
        String country, boolean express, double payable, boolean giftWrap, boolean freeShipping) {
}
