#pragma once

#include "QuoteRequest.hpp"

// Applies the express multiplier, insurance and the fragile fee to a base price.
double withSurcharges(const QuoteRequest& request, double basePrice);
