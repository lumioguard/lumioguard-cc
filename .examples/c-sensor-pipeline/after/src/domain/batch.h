#ifndef SENSOR_BATCH_H
#define SENSOR_BATCH_H

#include "limits.h"
#include "reading.h"

enum alert_mode { ALERT_NONE, ALERT_EVERY_REJECTION, ALERT_CRITICAL_ONLY };

/* What to process: replaces the nine loose arguments of process_batch. */
struct batch_request {
    const struct reading *readings;
    int count;
    enum sensor_kind kind;
    struct limits limits;
    int calibrate;
    enum alert_mode alert_mode;
};

struct batch_result {
    int accepted;
    int rejected;
    int alerts;
    double total;
};

double batch_mean(const struct batch_result *result);

#endif
