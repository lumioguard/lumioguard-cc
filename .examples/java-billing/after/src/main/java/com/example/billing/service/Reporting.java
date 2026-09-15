package com.example.billing.service;

import com.example.billing.domain.Invoice;
import java.util.List;

/** The one place a list of invoices is turned into text. */
public final class Reporting {

    private static final String SEPARATOR = "========================================\n";
    private static final String DIVIDER = "----------------------------------------\n";

    private Reporting() {
    }

    public static String render(List<Invoice> invoices, boolean verbose) {
        double sum = invoices.stream().mapToDouble(Invoice::amount).sum();
        StringBuilder out = new StringBuilder();
        out.append(SEPARATOR).append("INVOICE SUMMARY\n").append(SEPARATOR);
        out.append("invoices: ").append(invoices.size()).append("\n");
        out.append("total: ").append(String.format("%.2f", sum)).append("\n");
        out.append(DIVIDER);
        if (verbose) {
            invoices.forEach(invoice -> out.append("  ").append(invoice.customerId()).append("\n"));
        }
        return out.append(SEPARATOR).toString();
    }
}
