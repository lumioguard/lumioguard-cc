#pragma once

#include <ostream>
#include <string>

#include "../domain/Parcel.hpp"

class Receipt {
public:
    explicit Receipt(std::ostream& out) : out_(out) {}

    void print(const std::string& carrier, const Parcel& parcel, double price) const;

private:
    std::ostream& out_;
};
