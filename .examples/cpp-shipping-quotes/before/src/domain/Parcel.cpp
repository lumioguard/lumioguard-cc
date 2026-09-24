#include "Parcel.hpp"

bool Parcel::lastQuote(const QuoteCache& cache, double& price) const {
    return cache.find(id, price);
}
