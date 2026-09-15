SEASONAL_FACTOR = {"spring": 1.0, "summer": 1.4, "autumn": 1.1, "winter": 0.8}


def reorder_point(sku, season, weekly_demand):
    factor = SEASONAL_FACTOR.get(season, 1.0)
    base = weekly_demand * 2
    if sku.startswith("FRAGILE-"):
        base = base * 1.5
    elif sku.startswith("BULK-"):
        base = base * 0.7
    return int(base * factor)
