#include "batch.h"

double batch_mean(const struct batch_result *result) {
    return result->accepted > 0 ? result->total / result->accepted : 0.0;
}
