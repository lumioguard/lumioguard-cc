#ifndef SENSOR_READING_H
#define SENSOR_READING_H

enum sensor_kind { SENSOR_TEMPERATURE, SENSOR_HUMIDITY, SENSOR_PRESSURE, SENSOR_VIBRATION };

struct reading {
    int sensor_id;
    enum sensor_kind kind;
    double value;
    long timestamp;
};

double reading_scaled(const struct reading *reading, double factor);

#endif
