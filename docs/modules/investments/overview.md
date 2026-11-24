# Investments

## Overview

The investments module is a personal portfolio tracker. There is no market
sync: every value is a number typed into the CSV files ([csv.md](csv.md)). It
answers three questions: how much money went into each holding and in total,
what each holding is worth now, and how value has moved against money-in over
time. The module is read-only through the [JSON API](../../architecture/openapi.yaml).
Currency is EUR only.

## Structure

```text
entity            where money is held
  └─ instrument   a single holding at that entity
       ├─ contributions   money added, monthly
       └─ snapshots       observed value, monthly
```

- **Entity**: a name, a kind: `bank`, `broker`, `investment_firm` (for
  example a PPR manager), `state` (for example Certificados de Aforro) or
  `other`, and an optional owner: the user the account belongs to, or none
  when it belongs to the household. Its instruments take that owner. Everyone
  can see every account either way; the owner decides who can add money to it
  from a phone ([ADR 021](../../decisions/021-details-of-money-moved.md)).
  Names are unique.
- **Instrument**: a name, a type (`etf`, `fund`, `ppr`,
  `certificados_aforro` or `other`) and optional notes. Names are unique
  within an entity. Clients show friendly labels for the stored values.
- **Contribution**: an instrument, a month and a positive amount. Several
  contributions in one month are allowed and all count.
- **Snapshot**: an instrument, a month and a value (zero or more). The full
  history is kept and several per month are allowed.

## Figures

All figures are derived from the rows and never stored:

```text
allocated     = sum of contributions
current value = value of the latest snapshot (by month, then id)
gain          = current value - allocated
gain %        = gain * 10000 / allocated, as an integer; 0 when allocated is 0
```

`gainPct` is an integer scaled by 100: `734` means +7.34% and `-400` means
-4.00%. Clients divide by 100 to display it. This differs from the whole-number
`progressPct` of credits on purpose. Entity figures add up their instruments.

## API shape

Instruments are only listed under their entity
(`/investments/entities/{id}/instruments`); a single instrument can be fetched
by id. There is no flat instrument list and no bare `/investments` route. The
portfolio overview is built from `/investments/entities`. List endpoints return
summaries; the full contribution and snapshot history is fetched per
instrument.

## Access

Web sessions and device tokens can read every entity and instrument
([ADRs 010](../../decisions/010-household-data-is-shared.md) and
[022](../../decisions/022-devices-read-savings-investments-credits.md)).
Enrolled phones can create, replace or remove contributions only as the detail
of one of their cashflow movements; other changes use the CSV files
([ADR 021](../../decisions/021-details-of-money-moved.md)).
