from inventory.domain.stock import StockLevel

_LEVELS = [
    ("BULK-1", "north", 900, 120),
    ("BULK-1", "south", 40, 110),
    ("FRAGILE-2", "north", 12, 30),
    ("FRAGILE-2", "south", 300, 25),
    ("TOOL-9", "north", 55, 20),
]

_TRANSFERS = []


def load_levels():
    return [StockLevel(*row) for row in _LEVELS]


def write_transfer(sku, source, target, amount):
    if amount <= 0:
        raise ValueError("amount must be positive")
    _TRANSFERS.append((sku, source, target, amount))


def transfer_count():
    return len(_TRANSFERS)
