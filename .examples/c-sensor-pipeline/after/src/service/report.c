#include <stdio.h>

#include "report.h"

static const char *const RULE_HEAVY = "==============================\n";
static const char *const RULE_LIGHT = "------------------------------\n";

void report_print(const struct batch_result *result) {
    printf("%sBatch summary\n%s", RULE_HEAVY, RULE_LIGHT);
    printf("accepted: %d\n", result->accepted);
    printf("rejected: %d\n", result->rejected);
    printf("alerts:   %d\n", result->alerts);
    if (result->accepted > 0) {
        printf("mean:     %.2f\n", batch_mean(result));
    } else {
        printf("mean:     n/a\n");
    }
    printf("%s", RULE_HEAVY);
}
