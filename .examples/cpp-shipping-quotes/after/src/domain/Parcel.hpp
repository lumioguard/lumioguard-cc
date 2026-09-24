#pragma once

#include <string>

struct Parcel {
    std::string id;
    double weightKg = 0.0;
    double declaredValue = 0.0;
    bool fragile = false;
};
