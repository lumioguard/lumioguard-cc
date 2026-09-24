#include "QuoteCache.hpp"

bool QuoteCache::find(const std::string& parcelId, double& price) const {
    auto found = prices_.find(parcelId);
    if (found == prices_.end()) {
        return false;
    }
    price = found->second;
    return true;
}

void QuoteCache::store(const std::string& parcelId, double price) {
    prices_[parcelId] = price;
}
