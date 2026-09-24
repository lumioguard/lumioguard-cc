#include "limits.h"

int limits_contains(const struct limits *limits, double value) {
    return value >= limits->low && value <= limits->high;
}

int limits_is_critical(const struct limits *limits, double value) {
    double width = limits->high - limits->low;
    return value > limits->high * 1.5 || value < limits->low - width;
}
