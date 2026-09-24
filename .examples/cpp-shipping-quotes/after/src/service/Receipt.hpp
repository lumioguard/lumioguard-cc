#pragma once

#include <ostream>

#include "../domain/QuoteRequest.hpp"

class Receipt {
public:
    explicit Receipt(std::ostream& out) : out_(out) {}

    void print(const QuoteRequest& request, double price) const;

private:
    std::ostream& out_;
};
