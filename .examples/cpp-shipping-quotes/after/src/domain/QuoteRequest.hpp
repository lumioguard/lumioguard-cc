#pragma once

#include <string>

#include "Parcel.hpp"

enum class Zone { Local, National, Remote };

// Everything a quote depends on: replaces the seven loose arguments of quote().
struct QuoteRequest {
    std::string carrier;
    Parcel parcel;
    Zone zone = Zone::Local;
    bool express = false;
    bool insured = false;
    int weekday = 0;

    bool isWeekend() const { return weekday == 5 || weekday == 6; }
};
