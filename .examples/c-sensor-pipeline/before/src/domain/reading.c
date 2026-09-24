#include "reading.h"

int reading_reload(struct reading *reading, struct store *store) {
    double value;
    if (store_find(store, reading->sensor_id, &value) != 0) {
        return -1;
    }
    reading->value = value;
    return 0;
}

double reading_scaled(const struct reading *reading, double factor) {
    return reading->value * factor;
}
