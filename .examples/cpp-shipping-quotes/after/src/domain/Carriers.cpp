#include "Carriers.hpp"

#include <functional>
#include <map>
#include <string>

namespace {

using Pricer = std::function<std::optional<double>(const QuoteRequest&)>;

std::optional<double> postalPrice(const QuoteRequest& request) {
    const double weight = request.parcel.weightKg;
    double price = 9.0 + (weight - 5.0) * 1.2;
    if (weight <= 1.0) {
        price = 4.5;
    } else if (weight <= 5.0) {
        price = 9.0;
    }
    return request.zone == Zone::Remote ? price + 6.0 : price;
}

std::optional<double> longHaulCourierPrice(const QuoteRequest& request) {
    const double weight = request.parcel.weightKg;
    if (weight <= 20.0) {
        return 18.0 + weight * 1.5;
    }
    if (request.express) {
        return std::nullopt;
    }
    return 30.0 + weight * 2.0;
}

std::optional<double> courierPrice(const QuoteRequest& request) {
    switch (request.zone) {
    case Zone::Local:
        return 7.0 + request.parcel.weightKg * 0.8;
    case Zone::National:
        return 11.0 + request.parcel.weightKg * 1.1;
    default:
        return longHaulCourierPrice(request);
    }
}

std::optional<double> freightPrice(const QuoteRequest& request) {
    const double weight = request.parcel.weightKg;
    return weight < 50.0 ? 60.0 : weight * 1.3;
}

const std::map<std::string, Pricer>& pricers() {
    static const std::map<std::string, Pricer> table{
        {"postal", postalPrice},
        {"courier", courierPrice},
        {"freight", freightPrice},
    };
    return table;
}

}  // namespace

std::optional<double> basePrice(const QuoteRequest& request) {
    auto found = pricers().find(request.carrier);
    if (found == pricers().end()) {
        return std::nullopt;
    }
    return found->second(request);
}

bool isFreight(const QuoteRequest& request) {
    return request.carrier == "freight";
}
