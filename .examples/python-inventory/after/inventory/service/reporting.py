SEPARATOR = "=" * 40
DIVIDER = "-" * 40


def render_report(plan, verbose=False):
    """The one place a rebalance plan is turned into text."""
    lines = [
        SEPARATOR,
        "REBALANCE REPORT",
        SEPARATOR,
        "transfers: " + str(len(plan.transfers)),
        "units moved: " + str(plan.moved_units),
        "skipped: " + str(len(plan.skipped)),
        "warnings: " + str(len(plan.warnings)),
        DIVIDER,
    ]
    if verbose:
        lines.extend("  ! " + entry for entry in plan.warnings)
    lines.append(SEPARATOR)
    return "\n".join(lines)


def low_stock(levels, threshold):
    return [level for level in levels if level.units < threshold]
