#ifndef SENSOR_CALIBRATION_H
#define SENSOR_CALIBRATION_H

#include "reading.h"

/* Corrects a reading in place. Returns 0, or -1 when the raw value is impossible. */
int calibrate_reading(struct reading *reading);

#endif
