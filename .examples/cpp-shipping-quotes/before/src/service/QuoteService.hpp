#pragma once

#include <ostream>
#include <string>

#include "../domain/Parcel.hpp"
#include "../persistence/QuoteCache.hpp"

class QuoteService {
public:
    explicit QuoteService(std::ostream& out);

    // Returns the price, or -1 when the carrier cannot take the parcel.
    double quote(const std::string& carrier, const Parcel& parcel, const std::string& zone, bool express,
                 bool insured, int weekday, QuoteCache* cache);

private:
    std::ostream& out_;
};
