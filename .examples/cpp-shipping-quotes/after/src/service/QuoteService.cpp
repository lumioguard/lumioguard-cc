#include "QuoteService.hpp"

#include "../domain/Carriers.hpp"
#include "../domain/Surcharges.hpp"

std::optional<double> QuoteService::quote(const QuoteRequest& request) {
    if (auto cached = cache_.find(request.parcel.id)) {
        return cached;
    }
    auto base = basePrice(request);
    if (!base) {
        return std::nullopt;
    }
    const double price = withSurcharges(request, *base);
    cache_.store(request.parcel.id, price);
    receipt_.print(request, price);
    return price;
}
