#include <stdio.h>

#include "pipeline.h"
#include "report.h"
#include "../domain/calibration.h"

static void record_rejection(const struct batch_request *request, const struct reading *reading,
                             struct batch_result *result) {
    result->rejected++;
    if (request->alert_mode == ALERT_EVERY_REJECTION) {
        result->alerts++;
    } else if (request->alert_mode == ALERT_CRITICAL_ONLY && limits_is_critical(&request->limits, reading->value)) {
        result->alerts++;
        fprintf(stderr, "sensor %d critical: %.2f\n", reading->sensor_id, reading->value);
    }
}

static void process_reading(const struct batch_request *request, struct reading reading,
                            struct store *store, struct batch_result *result) {
    if (request->calibrate && calibrate_reading(&reading) != 0) {
        result->rejected++;
        return;
    }
    if (!limits_contains(&request->limits, reading.value)) {
        record_rejection(request, &reading, result);
        return;
    }
    if (store != NULL && store_put(store, &reading) != 0) {
        result->rejected++;
        return;
    }
    result->accepted++;
    result->total += reading.value;
}

int process_batch(const struct batch_request *request, struct store *store, struct batch_result *result) {
    int i;
    *result = (struct batch_result){0};
    for (i = 0; i < request->count; i++) {
        if (request->readings[i].kind == request->kind) {
            process_reading(request, request->readings[i], store, result);
        }
    }
    report_print(result);
    return result->alerts > 0;
}
