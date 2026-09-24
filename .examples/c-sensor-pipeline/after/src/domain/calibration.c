#include "calibration.h"

#define TEMPERATURE_MIN -40.0
#define TEMPERATURE_MAX 125.0

static int calibrate_temperature(struct reading *reading) {
    if (reading->value < TEMPERATURE_MIN) {
        reading->value = TEMPERATURE_MIN;
    } else if (reading->value > TEMPERATURE_MAX) {
        reading->value = TEMPERATURE_MAX;
    } else {
        reading->value = reading->value * 1.02 - 0.5;
    }
    return 0;
}

static int calibrate_humidity(struct reading *reading) {
    if (reading->value < 0.0 || reading->value > 100.0) {
        return -1;
    }
    reading->value = reading->value * 0.98;
    return 0;
}

static int calibrate_pressure(struct reading *reading) {
    reading->value = reading->value / 10.0;
    return 0;
}

int calibrate_reading(struct reading *reading) {
    switch (reading->kind) {
    case SENSOR_TEMPERATURE:
        return calibrate_temperature(reading);
    case SENSOR_HUMIDITY:
        return calibrate_humidity(reading);
    case SENSOR_PRESSURE:
        return calibrate_pressure(reading);
    default:
        return 0;
    }
}
