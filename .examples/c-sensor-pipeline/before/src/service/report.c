#include <stdio.h>

#include "report.h"

void report_print(const struct batch_result *result) {
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
}
