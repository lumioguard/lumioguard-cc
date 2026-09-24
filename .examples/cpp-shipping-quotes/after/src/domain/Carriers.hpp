#pragma once

#include <optional>

#include "QuoteRequest.hpp"

// The carrier's price before surcharges, or nothing when it cannot take the parcel.
std::optional<double> basePrice(const QuoteRequest& request);

// Freight prices already include express handling, fragile care and insurance terms.
bool isFreight(const QuoteRequest& request);
