from dataclasses import dataclass

SEASONAL_FACTOR = {"spring": 1.0, "summer": 1.4, "autumn": 1.1, "winter": 0.8}

SKU_FACTOR = {"FRAGILE-": 1.5, "BULK-": 0.7}

WEEKS_OF_COVER = 2


def reorder_point(sku, season, weekly_demand):
    """Units that should be on hand before the next replenishment."""
    base = weekly_demand * WEEKS_OF_COVER
    for prefix, factor in SKU_FACTOR.items():
        if sku.startswith(prefix):
            base = base * factor
            break
    return int(base * SEASONAL_FACTOR.get(season, 1.0))


@dataclass(frozen=True)
class Position:
    """How far one warehouse is from its reorder point.

    ``surplus`` is positive for stock to give away and negative for stock
    needed. ``reason`` is set only when the warehouse takes no part in a
    transfer, and then ``surplus`` is zero.
    """

    warehouse: str
    surplus: int
    reason: str | None


def classify(level, season, tolerance, prioritised):
    point = reorder_point(level.sku, season, level.weekly_demand)
    if level.units > point + tolerance:
        if prioritised or level.units > point * 2:
            return Position(level.warehouse, level.units - point, None)
        return Position(level.warehouse, 0, "surplus below priority")
    if level.units < point - tolerance:
        if level.weekly_demand > 0:
            return Position(level.warehouse, level.units - point, None)
        return Position(level.warehouse, 0, "no demand")
    return Position(level.warehouse, 0, "within tolerance")
