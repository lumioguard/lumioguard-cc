#include "store.h"

int store_put(struct store *store, const struct reading *reading) {
    if (store->count >= STORE_CAPACITY) {
        return -1;
    }
    store->sensor_ids[store->count] = reading->sensor_id;
    store->values[store->count] = reading->value;
    store->count++;
    return 0;
}

int store_find(const struct store *store, int sensor_id, double *value) {
    int i;
    for (i = store->count - 1; i >= 0; i--) {
        if (store->sensor_ids[i] == sensor_id) {
            *value = store->values[i];
            return 0;
        }
    }
    return -1;
}
