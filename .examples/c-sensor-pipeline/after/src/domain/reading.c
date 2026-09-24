#include "reading.h"

double reading_scaled(const struct reading *reading, double factor) {
    return reading->value * factor;
}
