#pragma once

#include <string>

#include "../persistence/QuoteCache.hpp"

class QuoteCache;

struct Parcel {
    std::string id;
    double weightKg = 0.0;
    double declaredValue = 0.0;
    bool fragile = false;

    // Asks the cache for the last quote, so the domain reaches into persistence.
    bool lastQuote(const QuoteCache& cache, double& price) const;
};
