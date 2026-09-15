package com.example.billing.domain;

import java.util.Map;
import java.util.Set;

/**
 * Tax rates as lookup tables. Adding a country or a region is a data change,
 * not another branch in the billing code.
 */
public final class TaxRules {

    private static final Set<String> REDUCED_CATEGORIES = Set.of("book", "food");

    private static final Map<String, Double> US_STANDARD_BY_REGION =
            Map.of("CA", 0.0725, "NY", 0.04, "OR", 0.0);
    private static final Map<String, Double> US_REDUCED_BY_REGION = Map.of("NY", 0.0);
    private static final double US_STANDARD_DEFAULT = 0.06;
    private static final double US_REDUCED_DEFAULT = 0.02;

    private static final Map<String, Double> STANDARD_BY_COUNTRY = Map.of("CA", 0.13);
    private static final Map<String, Double> REDUCED_BY_COUNTRY = Map.of("CA", 0.05);
    private static final double STANDARD_DEFAULT = 0.20;
    private static final double REDUCED_DEFAULT = 0.07;

    private TaxRules() {
    }

    public static double rateFor(String category, TaxContext context) {
        if (REDUCED_CATEGORIES.contains(category)) {
            return reducedRate(context);
        }
        boolean exemptHere = context.exempt() && context.isUnitedStates();
        return exemptHere ? 0 : standardRate(context);
    }

    private static double standardRate(TaxContext context) {
        if (context.isUnitedStates()) {
            return US_STANDARD_BY_REGION.getOrDefault(context.region(), US_STANDARD_DEFAULT);
        }
        return STANDARD_BY_COUNTRY.getOrDefault(context.country(), STANDARD_DEFAULT);
    }

    private static double reducedRate(TaxContext context) {
        if (context.isUnitedStates()) {
            return US_REDUCED_BY_REGION.getOrDefault(context.region(), US_REDUCED_DEFAULT);
        }
        return REDUCED_BY_COUNTRY.getOrDefault(context.country(), REDUCED_DEFAULT);
    }
}
