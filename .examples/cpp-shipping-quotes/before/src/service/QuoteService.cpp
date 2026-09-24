#include "QuoteService.hpp"

QuoteService::QuoteService(std::ostream& out) : out_(out) {}

double QuoteService::quote(const std::string& carrier, const Parcel& parcel, const std::string& zone, bool express,
                           bool insured, int weekday, QuoteCache* cache) {
    double price = 0.0;
    if (cache != nullptr) {
        double cached = 0.0;
        if (parcel.lastQuote(*cache, cached)) {
            return cached;
        }
    }
    if (carrier == "postal") {
        if (parcel.weightKg <= 1.0) {
            price = 4.5;
        } else if (parcel.weightKg <= 5.0) {
            price = 9.0;
        } else {
            price = 9.0 + (parcel.weightKg - 5.0) * 1.2;
        }
        if (zone == "remote") {
            price += 6.0;
        }
    } else if (carrier == "courier") {
        if (zone == "local") {
            price = 7.0 + parcel.weightKg * 0.8;
        } else if (zone == "national") {
            price = 11.0 + parcel.weightKg * 1.1;
        } else {
            if (parcel.weightKg > 20.0) {
                if (express) {
                    return -1.0;
                }
                price = 30.0 + parcel.weightKg * 2.0;
            } else {
                price = 18.0 + parcel.weightKg * 1.5;
            }
        }
    } else if (carrier == "freight") {
        price = parcel.weightKg < 50.0 ? 60.0 : parcel.weightKg * 1.3;
    } else {
        return -1.0;
    }
    if (express && carrier != "freight") {
        price *= weekday == 5 || weekday == 6 ? 2.0 : 1.5;
    }
    auto insurance = [&](double declared) {
        if (!insured) {
            return 0.0;
        }
        return declared > 1000.0 ? declared * 0.02 : 5.0;
    };
    price += insurance(parcel.declaredValue);
    if (parcel.fragile && carrier != "freight") {
        price += 3.5;
    }
    if (cache != nullptr) {
        cache->store(parcel.id, price);
    }

    out_ << "==============================\n";
    out_ << "Quote for parcel " << parcel.id << "\n";
    out_ << "------------------------------\n";
    out_ << "carrier: " << carrier << "\n";
    out_ << "weight:  " << parcel.weightKg << " kg\n";
    if (parcel.fragile) {
        out_ << "handling: fragile\n";
    }
    out_ << "price:   " << price << "\n";
    out_ << "==============================\n";
    return price;
}
