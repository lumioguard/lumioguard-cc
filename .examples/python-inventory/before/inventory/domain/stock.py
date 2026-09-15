from dataclasses import dataclass

from inventory.persistence.store import read_levels


@dataclass
class StockLevel:
    sku: str
    warehouse: str
    units: int
    weekly_demand: int


def load_levels():
    return [StockLevel(*row) for row in read_levels()]
