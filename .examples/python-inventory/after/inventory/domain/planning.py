from dataclasses import dataclass, field

from inventory.domain.rules import classify


@dataclass(frozen=True)
class PlanRequest:
    levels: list
    warehouses: list
    priority_skus: frozenset
    season: str
    max_transfers: int = 100
    tolerance: int = 2


@dataclass(frozen=True)
class Demand:
    sku: str
    warehouse: str
    units: int


@dataclass
class Plan:
    transfers: list = field(default_factory=list)
    skipped: list = field(default_factory=list)
    warnings: list = field(default_factory=list)
    moved_units: int = 0


def plan_transfers(request):
    """Move stock from warehouses above their reorder point to those below."""
    plan = Plan()
    for sku in sorted({level.sku for level in request.levels}):
        _plan_sku(plan, sku, request)
    return plan


def _plan_sku(plan, sku, request):
    warehouses = {level.warehouse for level in request.levels if level.sku == sku}
    if len(warehouses) < 2:
        plan.skipped.append((sku, "*", "single warehouse"))
        return
    surplus, deficit = _positions(plan, sku, request)
    for warehouse, needed in deficit:
        _fill_deficit(plan, Demand(sku, warehouse, needed), surplus, request.max_transfers)


def _positions(plan, sku, request):
    """Split the warehouses holding ``sku`` into givers and takers."""
    by_warehouse = {level.warehouse: level for level in request.levels if level.sku == sku}
    prioritised = sku in request.priority_skus
    surplus = []
    deficit = []
    for warehouse in request.warehouses:
        level = by_warehouse.get(warehouse)
        if level is None:
            plan.warnings.append("missing level for " + sku + " at " + warehouse)
            continue
        position = classify(level, request.season, request.tolerance, prioritised)
        if position.reason is not None:
            plan.skipped.append((sku, warehouse, position.reason))
        elif position.surplus > 0:
            surplus.append((warehouse, position.surplus))
        else:
            deficit.append((warehouse, -position.surplus))
    surplus.sort(key=lambda pair: -pair[1])
    deficit.sort(key=lambda pair: -pair[1])
    return surplus, deficit


def _fill_deficit(plan, demand, surplus, max_transfers):
    remaining = demand.units
    for index, (source, available) in enumerate(surplus):
        if remaining <= 0:
            break
        if len(plan.transfers) >= max_transfers:
            plan.warnings.append("transfer limit reached for " + demand.sku)
            break
        if available <= 0:
            continue
        amount = min(available, remaining)
        plan.transfers.append((demand.sku, source, demand.warehouse, amount))
        surplus[index] = (source, available - amount)
        remaining = remaining - amount
        plan.moved_units = plan.moved_units + amount
    if remaining > 0:
        plan.skipped.append((demand.sku, demand.warehouse, "unsatisfied deficit"))
