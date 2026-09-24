#include "Surcharges.hpp"

#include "Carriers.hpp"

namespace {

constexpr double kFragileFee = 3.5;
constexpr double kFlatInsurance = 5.0;
constexpr double kHighValue = 1000.0;
constexpr double kHighValueRate = 0.02;

double expressMultiplier(const QuoteRequest& request) {
    if (!request.express || isFreight(request)) {
        return 1.0;
    }
    return request.isWeekend() ? 2.0 : 1.5;
}

double insurance(const QuoteRequest& request) {
    if (!request.insured) {
        return 0.0;
    }
    const double declared = request.parcel.declaredValue;
    return declared > kHighValue ? declared * kHighValueRate : kFlatInsurance;
}

double fragileFee(const QuoteRequest& request) {
    return request.parcel.fragile && !isFreight(request) ? kFragileFee : 0.0;
}

}  // namespace

double withSurcharges(const QuoteRequest& request, double basePrice) {
    return basePrice * expressMultiplier(request) + insurance(request) + fragileFee(request);
}
