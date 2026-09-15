package com.example.billing.service;

import com.example.billing.domain.Invoice;
import java.util.List;

public final class Reporting {

    private Reporting() {
    }

    public static String render(List<Invoice> invoices, boolean verbose) {
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
