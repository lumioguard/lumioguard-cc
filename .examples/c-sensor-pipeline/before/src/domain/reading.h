#ifndef SENSOR_READING_H
#define SENSOR_READING_H

#include "../persistence/store.h"

struct store;

enum sensor_kind { SENSOR_TEMPERATURE, SENSOR_HUMIDITY, SENSOR_PRESSURE, SENSOR_VIBRATION };

struct reading {
    int sensor_id;
    enum sensor_kind kind;
    double value;
    long timestamp;
};

/* Reloads a reading from storage, so the domain reaches into persistence. */
int reading_reload(struct reading *reading, struct store *store);
double reading_scaled(const struct reading *reading, double factor);

#endif
