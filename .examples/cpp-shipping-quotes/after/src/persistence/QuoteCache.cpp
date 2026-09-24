#include "QuoteCache.hpp"

std::optional<double> QuoteCache::find(const std::string& parcelId) const {
    auto found = prices_.find(parcelId);
    if (found == prices_.end()) {
        return std::nullopt;
    }
    return found->second;
}

void QuoteCache::store(const std::string& parcelId, double price) {
    prices_[parcelId] = price;
}
