#ifndef SENSOR_PIPELINE_H
#define SENSOR_PIPELINE_H

#include "../domain/reading.h"
#include "../persistence/store.h"

struct batch_result {
    int accepted;
    int rejected;
    int alerts;
    double mean;
};

int process_batch(const struct reading *readings, int count, enum sensor_kind kind, double low,
                  double high, int calibrate, int alert_mode, struct store *store,
                  struct batch_result *result);

#endif
