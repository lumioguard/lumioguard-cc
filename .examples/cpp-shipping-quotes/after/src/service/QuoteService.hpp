#pragma once

#include <optional>

#include "Receipt.hpp"
#include "../domain/QuoteRequest.hpp"
#include "../persistence/QuoteCache.hpp"

class QuoteService {
public:
    QuoteService(QuoteCache& cache, const Receipt& receipt) : cache_(cache), receipt_(receipt) {}

    // The price, or nothing when the carrier cannot take the parcel.
    std::optional<double> quote(const QuoteRequest& request);

private:
    QuoteCache& cache_;
    const Receipt& receipt_;
};
