#pragma once

#include <map>
#include <optional>
#include <string>

class QuoteCache {
public:
    std::optional<double> find(const std::string& parcelId) const;
    void store(const std::string& parcelId, double price);

private:
    std::map<std::string, double> prices_;
};
