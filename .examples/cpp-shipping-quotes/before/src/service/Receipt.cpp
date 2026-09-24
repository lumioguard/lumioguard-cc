#include "Receipt.hpp"

void Receipt::print(const std::string& carrier, const Parcel& parcel, double price) const {
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
}
