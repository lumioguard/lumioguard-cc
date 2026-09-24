#ifndef SENSOR_PIPELINE_H
#define SENSOR_PIPELINE_H

#include "../domain/batch.h"
#include "../persistence/store.h"

/* Processes the readings of one kind and prints a summary. Returns 1 if any alert was raised. */
int process_batch(const struct batch_request *request, struct store *store, struct batch_result *result);

#endif
