#include "Receipt.hpp"

namespace {

constexpr const char* kHeavyRule = "==============================\n";
constexpr const char* kLightRule = "------------------------------\n";

}  // namespace

void Receipt::print(const QuoteRequest& request, double price) const {
    const Parcel& parcel = request.parcel;
    out_ << kHeavyRule << "Quote for parcel " << parcel.id << "\n" << kLightRule;
    out_ << "carrier: " << request.carrier << "\n";
    out_ << "weight:  " << parcel.weightKg << " kg\n";
    if (parcel.fragile) {
        out_ << "handling: fragile\n";
    }
    out_ << "price:   " << price << "\n" << kHeavyRule;
}
