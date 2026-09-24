#include <stdio.h>

#include "pipeline.h"

int process_batch(const struct reading *readings, int count, enum sensor_kind kind, double low,
                  double high, int calibrate, int alert_mode, struct store *store,
                  struct batch_result *result) {
    double total = 0.0;
    int i;
    result->accepted = 0;
    result->rejected = 0;
    result->alerts = 0;
    for (i = 0; i < count; i++) {
        struct reading r = readings[i];
        if (r.kind == kind) {
            if (calibrate) {
                switch (r.kind) {
                case SENSOR_TEMPERATURE:
                    if (r.value < -40.0) {
                        r.value = -40.0;
                    } else if (r.value > 125.0) {
                        r.value = 125.0;
                    } else {
                        r.value = r.value * 1.02 - 0.5;
                    }
                    break;
                case SENSOR_HUMIDITY:
                    if (r.value < 0.0 || r.value > 100.0) {
                        result->rejected++;
                        continue;
                    }
                    r.value = r.value * 0.98;
                    break;
                case SENSOR_PRESSURE:
                    r.value = r.value / 10.0;
                    break;
                default:
                    break;
                }
            }
            if (r.value >= low && r.value <= high) {
                if (store != NULL) {
                    if (store_put(store, &r) != 0) {
                        result->rejected++;
                        continue;
                    }
                }
                result->accepted++;
                total += r.value;
            } else {
                result->rejected++;
                if (alert_mode == 1) {
                    result->alerts++;
                } else if (alert_mode == 2) {
                    if (r.value > high * 1.5 || r.value < low - (high - low)) {
                        result->alerts++;
                        fprintf(stderr, "sensor %d critical: %.2f\n", r.sensor_id, r.value);
                    }
                }
            }
        }
    }
    result->mean = result->accepted > 0 ? total / result->accepted : 0.0;

    printf("==============================\n");
    printf("Batch summary\n");
    printf("------------------------------\n");
    printf("accepted: %d\n", result->accepted);
    printf("rejected: %d\n", result->rejected);
    printf("alerts:   %d\n", result->alerts);
    if (result->accepted > 0) {
        printf("mean:     %.2f\n", result->mean);
    } else {
        printf("mean:     n/a\n");
    }
    printf("==============================\n");
    return result->alerts > 0 ? 1 : 0;
}
