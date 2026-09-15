from dataclasses import dataclass


@dataclass(frozen=True)
class StockLevel:
    """One stock-keeping unit held at one warehouse."""

    sku: str
    warehouse: str
    units: int
    weekly_demand: int
