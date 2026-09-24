#ifndef SENSOR_STORE_H
#define SENSOR_STORE_H

#include "../domain/reading.h"

#define STORE_CAPACITY 256

struct store {
    int sensor_ids[STORE_CAPACITY];
    double values[STORE_CAPACITY];
    int count;
};

int store_put(struct store *store, const struct reading *reading);
int store_find(const struct store *store, int sensor_id, double *value);

#endif
