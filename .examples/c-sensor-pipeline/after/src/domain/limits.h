#ifndef SENSOR_LIMITS_H
#define SENSOR_LIMITS_H

struct limits {
    double low;
    double high;
};

int limits_contains(const struct limits *limits, double value);

/* A value far outside the band: half as high again, or a whole band width too low. */
int limits_is_critical(const struct limits *limits, double value);

#endif
