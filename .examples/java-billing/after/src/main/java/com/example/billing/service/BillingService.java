package com.example.billing.service;

import com.example.billing.domain.DiscountPolicy;
import com.example.billing.domain.Invoice;
import com.example.billing.domain.LineItem;
import com.example.billing.domain.OrderRequest;
import com.example.billing.domain.PricedItems;
import com.example.billing.domain.ShippingPolicy;
import com.example.billing.domain.ShippingRequest;
import com.example.billing.domain.TaxContext;
import com.example.billing.domain.TaxRules;
import com.example.billing.persistence.InvoiceStore;
import java.util.ArrayList;
import java.util.List;

public class BillingService {

    private final InvoiceStore store;

    public BillingService(InvoiceStore store) {
        this.store = store;
    }

    public Invoice calculateInvoice(OrderRequest request) {
        PricedItems priced = priceItems(request);
        double discount = DiscountPolicy.discountFor(request.couponCode(), priced.total(), request.tier());
        double shipping = ShippingPolicy.shippingFor(new ShippingRequest(
                request.country(),
                request.express(),
                priced.total() - discount,
                request.giftWrap(),
                DiscountPolicy.waivesShipping(request.couponCode())));

        List<String> problems = new ArrayList<>(priced.problems());
        Invoice invoice = new Invoice(
                request.customerId(), priced.total() - discount + shipping + priced.tax(), problems);
        save(invoice, problems);
        return invoice;
    }

    private void save(Invoice invoice, List<String> problems) {
        try {
            store.save(invoice);
        } catch (IllegalStateException error) {
            problems.add("could not save: " + error.getMessage());
        }
    }

    private static PricedItems priceItems(OrderRequest request) {
        TaxContext context = request.taxContext();
        List<String> problems = new ArrayList<>();
        double total = 0;
        double tax = 0;
        for (LineItem item : request.items()) {
            String problem = problemFor(item);
            if (problem != null) {
                problems.add(problem);
                continue;
            }
            total = total + item.lineTotal();
            tax = tax + item.lineTotal() * TaxRules.rateFor(item.category(), context);
        }
        return new PricedItems(total, tax, problems);
    }

    private static String problemFor(LineItem item) {
        if (item.quantity() <= 0) {
            return "invalid quantity for " + item.sku();
        }
        if (item.sku() == null || item.sku().isBlank()) {
            return "missing sku";
        }
        return item.lineTotal() > 0 ? null : "zero price for " + item.sku();
    }

    public String summarise(List<Invoice> invoices, boolean verbose) {
        return Reporting.render(invoices, verbose);
    }
}
