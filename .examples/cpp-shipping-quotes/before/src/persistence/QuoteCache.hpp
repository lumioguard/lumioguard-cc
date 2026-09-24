#pragma once

#include <map>
#include <string>

#include "../domain/Parcel.hpp"

class QuoteCache {
public:
    bool find(const std::string& parcelId, double& price) const;
    void store(const std::string& parcelId, double price);

private:
    std::map<std::string, double> prices_;
};
