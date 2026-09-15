package com.example.billing.domain;

public final class TaxRules {

    private TaxRules() {
    }

    public static double standardRate(String country, String region) {
        if (country.equals("US")) {
            if (region.equals("CA")) {
                return 0.0725;
            } else if (region.equals("NY")) {
                return 0.04;
            } else if (region.equals("OR")) {
                return 0.0;
            }
            return 0.06;
        }
        return 0.2;
    }

    public static double reducedRate(String country, String region) {
        if (country.equals("US")) {
            if (region.equals("NY")) {
                return 0.0;
            }
            return 0.02;
        }
        return 0.05;
    }
}
