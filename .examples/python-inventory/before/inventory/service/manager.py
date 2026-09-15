from inventory.domain.rules import reorder_point
from inventory.domain.stock import StockLevel, load_levels
from inventory.persistence.store import write_transfer


def rebalance_warehouses(
    levels,
    warehouses,
    priority_skus,
    season,
    max_transfers,
    dry_run,
    verbose,
    tolerance,
):
    transfers = []
    skipped = []
    warnings = []
    moved_units = 0

    for sku in sorted({level.sku for level in levels}):
        by_warehouse = {level.warehouse: level for level in levels if level.sku == sku}
        if len(by_warehouse) > 1:
            surplus = []
            deficit = []
            for warehouse in warehouses:
                if warehouse in by_warehouse:
                    level = by_warehouse[warehouse]
                    point = reorder_point(sku, season, level.weekly_demand)
                    if level.units > point + tolerance:
                        if sku in priority_skus or level.units > point * 2:
                            surplus.append((warehouse, level.units - point))
                        else:
                            skipped.append((sku, warehouse, "surplus below priority"))
                    elif level.units < point - tolerance:
                        if level.weekly_demand > 0:
                            deficit.append((warehouse, point - level.units))
                        else:
                            skipped.append((sku, warehouse, "no demand"))
                    else:
                        skipped.append((sku, warehouse, "within tolerance"))
                else:
                    warnings.append("missing level for " + sku + " at " + warehouse)

            surplus.sort(key=lambda pair: -pair[1])
            deficit.sort(key=lambda pair: -pair[1])

            for target, needed in deficit:
                remaining = needed
                for index, (source, available) in enumerate(surplus):
                    if remaining <= 0:
                        break
                    if available <= 0:
                        continue
                    amount = available if available < remaining else remaining
                    if len(transfers) < max_transfers:
                        transfers.append((sku, source, target, amount))
                        surplus[index] = (source, available - amount)
                        remaining = remaining - amount
                        moved_units = moved_units + amount
                    else:
                        warnings.append("transfer limit reached for " + sku)
                        break
                if remaining > 0:
                    skipped.append((sku, target, "unsatisfied deficit"))
        else:
            skipped.append((sku, "*", "single warehouse"))

    if not dry_run:
        for sku, source, target, amount in transfers:
            try:
                write_transfer(sku, source, target, amount)
            except Exception as error:
                warnings.append("failed transfer " + sku + ": " + str(error))

    lines = []
    lines.append("=" * 40)
    lines.append("REBALANCE REPORT")
    lines.append("=" * 40)
    lines.append("transfers: " + str(len(transfers)))
    lines.append("units moved: " + str(moved_units))
    lines.append("skipped: " + str(len(skipped)))
    lines.append("warnings: " + str(len(warnings)))
    lines.append("-" * 40)
    if verbose:
        for entry in warnings:
            lines.append("  ! " + entry)
    lines.append("=" * 40)

    return {
        "transfers": transfers,
        "skipped": skipped,
        "warnings": warnings,
        "report": "\n".join(lines),
    }


def run(warehouses, priority_skus, season):
    levels = load_levels()
    return rebalance_warehouses(
        levels, warehouses, priority_skus, season, 100, False, True, 2
    )


def describe(level: StockLevel) -> str:
    return level.sku + "@" + level.warehouse + "=" + str(level.units)
