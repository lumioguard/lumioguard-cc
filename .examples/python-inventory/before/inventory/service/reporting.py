from inventory.domain.stock import load_levels


def stock_report(transfers, skipped, warnings, moved_units, verbose):
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
    return "\n".join(lines)


def low_stock(threshold):
    return [level for level in load_levels() if level.units < threshold]
