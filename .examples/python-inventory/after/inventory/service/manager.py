from inventory.domain.planning import PlanRequest, plan_transfers
from inventory.persistence.store import load_levels, write_transfer
from inventory.service.reporting import render_report


def rebalance(request, dry_run=False, verbose=True):
    """Plan transfers for ``request`` and, unless dry running, apply them."""
    plan = plan_transfers(request)
    if not dry_run:
        _apply(plan)
    return {
        "transfers": plan.transfers,
        "skipped": plan.skipped,
        "warnings": plan.warnings,
        "report": render_report(plan, verbose),
    }


def _apply(plan):
    for sku, source, target, amount in plan.transfers:
        try:
            write_transfer(sku, source, target, amount)
        except ValueError as error:
            plan.warnings.append("failed transfer " + sku + ": " + str(error))


def run(warehouses, priority_skus, season):
    request = PlanRequest(
        levels=load_levels(),
        warehouses=warehouses,
        priority_skus=frozenset(priority_skus),
        season=season,
    )
    return rebalance(request)


def describe(level):
    return level.sku + "@" + level.warehouse + "=" + str(level.units)
