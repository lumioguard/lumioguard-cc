package com.example.billing.service;

import com.example.billing.domain.Invoice;
import com.example.billing.domain.LineItem;
import com.example.billing.domain.TaxRules;
import com.example.billing.persistence.InvoiceStore;
import java.util.ArrayList;
import java.util.List;

public class BillingService {

    private final InvoiceStore store = new InvoiceStore();

    public Invoice calculateInvoice(
            String customerId,
            List<LineItem> items,
            String couponCode,
            String country,
            String region,
            String tier,
            boolean express,
            boolean giftWrap) {
        double total = 0;
        double tax = 0;
        double discount = 0;
        double shipping = 0;
        List<String> problems = new ArrayList<>();

        for (LineItem item : items) {
            if (item.quantity() > 0) {
                if (item.sku() != null && !item.sku().isBlank()) {
                    double line = item.unitPrice() * item.quantity();
                    if (line > 0) {
                        if (item.category().equals("book") || item.category().equals("food")) {
                            if (country.equals("US")) {
                                tax = tax + line * TaxRules.reducedRate("US", region);
                            } else if (country.equals("CA")) {
                                tax = tax + line * 0.05;
                            } else {
                                tax = tax + line * 0.07;
                            }
                        } else {
                            if (country.equals("US")) {
                                if (tier.equals("exempt")) {
                                    tax = tax + 0;
                                } else {
                                    tax = tax + line * TaxRules.standardRate("US", region);
                                }
                            } else if (country.equals("CA")) {
                                tax = tax + line * 0.13;
                            } else {
                                tax = tax + line * 0.20;
                            }
                        }
                        total = total + line;
                    } else {
                        problems.add("zero price for " + item.sku());
                    }
                } else {
                    problems.add("missing sku");
                }
            } else {
                problems.add("invalid quantity for " + item.sku());
            }
        }

        if (couponCode != null && !couponCode.isBlank()) {
            switch (couponCode) {
                case "WELCOME10":
                    discount = total * 0.10;
                    break;
                case "SUMMER20":
                    if (total > 100) {
                        discount = total * 0.20;
                    } else {
                        discount = total * 0.05;
                    }
                    break;
                case "FREESHIP":
                    shipping = 0;
                    break;
                case "VIP":
                    if (tier.equals("gold")) {
                        discount = total * 0.25;
                    } else if (tier.equals("silver")) {
                        discount = total * 0.15;
                    } else {
                        discount = 0;
                    }
                    break;
                default:
                    discount = 0;
            }
        }

        if (!"FREESHIP".equals(couponCode)) {
            if (country.equals("US")) {
                shipping = express ? 25 : 8;
            } else if (country.equals("CA")) {
                shipping = express ? 35 : 14;
            } else {
                shipping = express ? 60 : 30;
            }
            if (total - discount > 150 && !express) {
                shipping = 0;
            }
        }

        if (giftWrap) {
            shipping = shipping + 5;
        }

        Invoice invoice = new Invoice(customerId, total - discount + shipping + tax, problems);
        try {
            store.save(invoice);
        } catch (IllegalStateException error) {
            problems.add("could not save: " + error.getMessage());
        }
        return invoice;
    }

    public String summarise(List<Invoice> invoices, boolean verbose) {
        StringBuilder out = new StringBuilder();
        double sum = 0;
        for (Invoice invoice : invoices) {
            sum = sum + invoice.amount();
        }
        out.append("========================================\n");
        out.append("INVOICE SUMMARY\n");
        out.append("========================================\n");
        out.append("invoices: ").append(invoices.size()).append("\n");
        out.append("total: ").append(String.format("%.2f", sum)).append("\n");
        out.append("----------------------------------------\n");
        if (verbose) {
            for (Invoice invoice : invoices) {
                out.append("  ").append(invoice.customerId()).append("\n");
            }
        }
        out.append("========================================\n");
        return out.toString();
    }
}
