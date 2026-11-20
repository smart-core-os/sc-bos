## NABERS dashboard

Shows air quality data and metrics for NABERS reporting. The rating inputs — net
lettable area, postcode, rated hours, meter names, design targets and off-axis
scenarios — all come from `dashboards-config.json`, so the same build serves any
site. `dashboards-config.json` in this directory is a template: replace the
placeholder values with the ones from the building's Design for Performance
assessment.

### Credentials, and what that means

The dashboard authenticates itself. It reads a username and password from
`VITE_DASHBOARD_USERNAME`/`VITE_DASHBOARD_PASSWORD`, falling back to
`config.username`/`config.password` in `dashboards-config.json`, and exchanges
them for a token via the OAuth2 password grant (`src/stores/auth.js`).

**Both of those places are readable by anyone who can load the page.** Vite bakes
env vars into the bundle at build time, and `dashboards-config.json` is fetched
over plain HTTP by the browser. There is no way to hide a credential in a
single-page app, so this is not a bug to fix in the client — but it does mean:

- Give the dashboard its **own service account**, never a person's login and
  never an administrator's. It only ever issues reads, so scope it to read.
- Treat that account as public. Anyone who can reach the display, or the URL, can
  extract it and call the API directly with the same rights.
- Do not commit real credentials. The `dashboards-config.json` in this directory
  is a template and the values in it are placeholders.

If the deployment can mint a short-lived token server-side and serve it to the
page instead, prefer that; the client already treats the token as opaque and
re-fetches it on `UNAUTHENTICATED`, so only `fetchToken` would change.

### Linking out to the ops UI

The base building breakdown widget can carry a link to a fuller energy view, so
detail lives in the ops UI rather than being crammed onto this screen. Set
`nabersBaseBuilding.opsUrl` (and optionally `opsLinkLabel`) to the ops page to
link to; leave them out and no link is drawn.

## Display

Built as a fullscreen dashboard: the pages size to the viewport (`100vh`) and the
rows are flex, so the layout fills whatever display it is put on. There are no
breakpoints and type sizes are fixed, so it is intended for a large screen rather
than a phone, but no particular resolution is assumed.

## How the rating is calculated

> This documents what `src/util/nabersRating.js` computes, worked through against
> 3 Chamberlain Square — the first deployment, and so far the only one with a full
> twelve months of measured data. The mechanics are general; the meter names,
> months and figures used to illustrate them are that site's.
>
> The inputs themselves — rated area, rated hours, postcode, design targets and
> the meter-to-end-use assignments — are site configuration rather than app
> behaviour, and are documented per site. For 3CS that is
> [mepc-3cs `config/common/nabersdashboard/README.md`](https://github.com/vanti-dev/mepc-3cs/blob/main/config/common/nabersdashboard/README.md), which is also
> where the cross-references below point.

Three stages: sum the metered energy, build the benchmark this building is rated
against, then compare.

### 1. Metered energy

Each end use points at one aggregate zone (see [What is metered](https://github.com/vanti-dev/mepc-3cs/blob/main/config/common/nabersdashboard/README.md#what-is-metered)).
Meters are cumulative accumulators, so consumption is always a **delta between
two `usage` readings**, never a single value. Two windows are read:

- **Period to date**, from the rating period start to now, giving a
  straight-line annualised projection. Suppressed below 28 elapsed days, because
  on day two a projection multiplies two days by 182.
- **Trailing 12 months**, from 13 month-boundary readings. Once twelve complete
  months exist this becomes the headline figure and the projection is dropped.
  Boundary readings are sought within 2 days of each month start.

Null handling is deliberate and matters:

- A category with **no meters configured** contributes **0**. It is not an error.
  This is how the unmetered `other` end use still draws its target bar.
- Monthly consumption is differenced **per meter** and then summed, not summed at
  each boundary and then differenced. A sum across meters at one instant is only
  comparable to a sum at another if exactly the same meters contributed to both;
  a meter absent at the earlier boundary and present at the later one would drop
  its entire lifetime cumulative total into a single month.
- A reading below its own earlier value is graded by **how far it fell**, as a
  share of what it fell from. A **reset** — essentially the whole register — takes
  the meter's history with it, so consumption across the affected span cannot be
  recovered from two cumulative points and stays `null` rather than becoming a
  clamped zero. A small **correction** reports zero for that meter, disclosed as
  estimated. See [When a register goes backwards](#1b-when-a-register-goes-backwards).
- A category with any **still-unreadable** meter, once gap filling has had its
  go, is **unknown**, and makes the whole rating `null`. Not "all unreadable":
  dropping one dead board of Terminal Fans' seventeen and summing the rest would
  be a quiet undercount and a flatteringly low rating. NABERS requires
  substituted assumptions to be worse than actual, never better, so an
  undercount is the one failure mode ruled out by construction.
- The gauge names the **individual meters** affected, not just the end use, and
  distinguishes estimated from missing. The Meter data quality panel gives the
  per-meter reason and the gap duration.

### 1a. Gap filling, and how it is disclosed

A missing boundary reading used to propagate straight to the rating, so a single
dead distribution board blanked the headline figure, broke the trend line and
dropped bars out of the breakdown chart. That was the on-site symptom. NABERS
permits estimating missing data where the estimation is **disclosed**, so the
dashboard now fills the gaps and says so throughout. Settings live in
`nabersBaseBuilding.estimation`; see the `_estimationNote` beside them.

Because the readings are cumulative, the arithmetic divides in two:

- **A gap bounded by a real reading either side is interpolated linearly.** The
  total across the gap is already fixed by the two real endpoints, so
  interpolating a boundary inside it only apportions energy between the adjacent
  months. Their sum is unchanged, which makes this exact rather than an
  assumption, and there is nothing to be conservative about.
- **A gap open at one end is a genuine extrapolation.** It runs at that meter's
  own mean rate, then adds `extrapolationUpliftPct` (10%) in whichever direction
  **increases** reported consumption, so a substituted value can never flatter
  the rating.

Estimation has a reach. Beyond `searchWindowDays` there is no nearby reading to
anchor a projection, and a meter silent for longer stays unreadable rather than
having months of consumption invented for it. `gapThresholdHours` (3) is the
widest bracket still treated as ordinary reporting jitter.

> **`searchWindowDays` is back at the 45-day default**, from 730. It was at 730
> only so the rating would compute despite the ten dark meters on floors 06 to 09,
> and those floors are out of `meterNames` entirely, so the reason went with them.
> That reason has since survived a reinstatement attempt rather than been superseded
> by one — the 13 meters went back in during June 2026 and came out again in August
> 2026. See [item 7](https://github.com/vanti-dev/mepc-3cs/blob/main/config/common/nabersdashboard/README.md#7-the-fitout-floors-are-excluded-from-the-rating).
>
> Leaving it wide would have been a hazard rather than untidiness: **one value gates
> both probes**, so a wide window licenses extrapolation from a stale reading as
> well as exact interpolation across a bracketed gap. Any *other* meter that went
> dark would have had up to two years of consumption projected for it silently,
> instead of blanking its end use and naming the board. At 45 days a long gap
> withholds the figure and tells you, which is what you want from the 62 meters
> that remain.
>
> Historical note, for anyone reading an older commit: while the window was at 730,
> a material share of the figure was projected for those ten meters at mean rate
> plus 10%, for the whole fitout period rather than for a few weeks. It was
> disclosed in amber throughout, but a screenshot of the star figure alone would
> have misled. That is no longer the shape of the problem: with the meters removed
> the figure is measured, and the caveat moved from "part of this is invented" to
> "this covers five of the nine office floors". See
> [Requires attention](https://github.com/vanti-dev/mepc-3cs/blob/main/config/common/nabersdashboard/README.md#requires-attention) item 7.

Estimated values feed everything, including the settled rating, each flagged.
Disclosure is amber throughout, distinct from green "actual" and red "missing": a
banner above the fold, a chip and a caveat naming the affected boards on the
rating gauge, triangles and dashed segments on the trend charts, a "~ Estimated"
state and an `Est. kWh` column in the monthly report, gap durations in the meter
quality table, and outlined bars on the breakdown chart. The CSV export is the
artefact an assessor reads detached from all of that, so it also carries a
provenance header with the model version, an explicit disclosure paragraph and a
total row.

Setting `estimation.enabled: false` restores the previous behaviour, where any
gap withholds the figure entirely.

### 1b. When a register goes backwards

A cumulative register is not supposed to fall, so when one does, something happened
to the meter rather than to the building. **Three** different things look identical
in a pair of point readings, and they want three different answers:

| | How it is told apart | What the dashboard does |
|---|---|---|
| **Dropout** — a bad read, usually a bare `0` | the very next reading **recovers** to where the register was | **discards the reading.** The month is then exact — nothing substituted, nothing disclosed |
| **Reset** — the register started again | it does not recover; the register climbs from near zero over weeks | withholds the span and names the board |
| **Correction** — a small persistent step down | does not recover, but is under `regressionSharePct` of the register | reports **zero** for that meter, **disclosed as estimated** |

**The dropout case is by far the commonest here, and it is a filter rather than an
estimate.** `plausibleSamples` already discards a reading that is negative or above
the meter's current value. It cannot see a bare `0` in the middle of a healthy
series, because the value is neither — and that is the corruption that does the most
damage, precisely because it does not look like corruption. It looks like a reset, so
the month is withheld and a working board is reported as faulty.

What makes it impossible is **the recovery, not the size**: for the `0` to be real,
the meter would have had to consume its entire lifetime total again before the next
record. A reset does not do that — it climbs from near zero over weeks, so the reading
after it is small. `maxDropoutSamples` (2) bounds how many consecutive bad reads count
as one dropout, which is what stops a reset's slow climb being read backwards as a
long dropout.

Discards are counted per meter in **Meter data quality** as "N discarded", with the
reason on hover, and named in the boundary reason. The figures are unaffected, but a
board discarding readings repeatedly is a driver or comms fault worth chasing.

A correction is a re-registration after a device swap, a driver writing a stale
value, or a read landing mid-update. The line sits at `regressionSharePct` (1%) of
the earlier reading, floored at `regressionToleranceKwh` (**25 kWh here**, see
below). **These are orders of magnitude apart, not a fine judgement**: a reset is
~100% of the register, and the corrections seen here are ~0.2%. Elapsed time is
deliberately not part of the test — a register that fell by 150 kWh fell by 150 kWh
whether it did so over an hour or a month.

**The floor is 25 kWh rather than the code's 1 kWh default, and that matters more
than it looks.** A proportional test cannot work on a meter with a tiny register,
because every movement is then a large share of it while the absolute energy is
negligible by construction. `EM/015 Chw Pressurisation Unit` has totalled **3.0 kWh**,
unchanged from September to December 2025; 1% of that is 0.03 kWh, so it always falls
back to the floor, and at 1 kWh any wobble at all read as a reset. That withheld
**October and November** — about 68,900 and 64,300 kWh net — over a register whose
entire contents are 0.005% of either month. `EM/002 Dhws Circulation Pump` (248 kWh)
and `EM/005 Cat 5 Booster` (171 kWh) sit in the same trap at 2.5 and 1.7 kWh.

The floor only binds on meters whose register is under 2,500 kWh, since above that
the 1% share is the larger of the two. And it cannot mask a real reset: a reset
returns the register to about zero, not to 25 kWh below where it was. Raise it
further only against a named meter and a measured drop — never to make a month
appear.

**Every refusal now carries its numbers**, so the two cases can be told apart
without reading raw history: `accumulator reset near boundary (fell 2,000 kWh from
87,336; allowance 873 kWh)`. And each failed board carries its **own** reason. Before
that, `sumDeltas` short-circuited on the first failure, so September 2025 listed 17
boards against one shared sentence when the 17 had done several different things.

**Why this exists.** December 2025 read as `✗ Missing` on this dashboard while the
month-end XLSX carried figures for all 162 meters ([count unconfirmed](https://github.com/vanti-dev/mepc-3cs/blob/main/config/common/nabersdashboard/README.md#13-the-162-meter-count-is-unconfirmed)) — which reads as the dashboard
being broken. **EM/048 Ashp 02** (`lv-ashp-2`, `coolingHeating`) ended December on
87,185.5 kWh against November's 87,335.5, a fall of **150 kWh, 0.17% of its own
register**. One unknown meter nulls the pool by design, so that withheld all
~72,000 kWh metered across the other 57 boards, blanked Gross, Net and kWh/m² for
the month, stopped the 12-month rating settling, and turned the footer into an
11-month partial. The month-end automation never noticed because it prints each
month's last reading and never differences them, so a backwards step is invisible
in the spreadsheet.

**Know what it costs.** Reporting zero **understates** by whatever the meter
genuinely consumed, which is the direction the NABERS method forbids of a
substituted value. That is a considered trade for an indicative dashboard on a
building still ironing out its metering, and it was taken over the alternative on
offer — substituting the meter's own mean rate, which for a standby unit invents
thousands of kWh to satisfy the rule and is the larger error. The error is bounded
by the size of the step, the month is badged amber, and the board is named in the
tooltip and in both CSV exports, so the caveat travels with the figure.

> **A submission figure for an affected month must come from the FM provider's
> verified BMS readings, not from here.** That is already a contractual monthly
> obligation, including logbook entries covering faults and resets.

Both knobs at `0`, or `estimation.enabled: false`, restore the old
refuse-everything behaviour.

A month the dashboard still cannot report now says **why**, on hover and in the CSV:
one line per board, each with its own reason and, for a backwards step, how far it
fell and what it was judged against. That was the second half of this defect — the
figure was withheld correctly and the table gave no way to tell that from a broken
dashboard.

### 1c. What September to December 2025 actually were

Worth recording, because the diagnosis was wrong twice before the numbers arrived and
the wrong answer was plausible each time.

September, October and November 2025 all read `✗ Missing` while the month-end XLSX
carried figures for every meter. Adding the drop size to the reason settled it in one
export: **every one of the 17 affected boards had fallen to exactly zero**, and every
one recovered afterwards.

```
EM/056 Basement Landlords Db Lighting: fell 16,768 kWh from 16,768; allowance 168 kWh
EM/010 Basement Ahu South:             fell 11,252 kWh from 11,252; allowance 113 kWh
EM/007 Hot Water Calorifier Imm. No2:  fell    916 kWh from    916; allowance  25 kWh
```

`fell X from X` means it landed on 0. None of these were resets or corrections: they
were **bad reads**. `EM/007` climbed 916 → 0 → 930, and it consumed 2 kWh in October
and 14 in November, so re-accumulating 930 between two records is not physically
available to it. Same shape on all 17.

Two consequences worth keeping in mind:

- **Widening the threshold was the wrong instrument**, and would have been wrong even
  if it had worked. `regressionToleranceKwh` at 25 kWh did clear `EM/015` — whose
  entire register is 3 kWh — but clearing a 16,768 kWh drop that way would have meant
  tolerating an arbitrary amount of invented or lost energy. The right answer was to
  recognise the reading as impossible and discard it, which makes the month **exact**
  rather than estimated.
- **October and November shared one boundary.** Both listed only `EM/007`, which
  pinned the fault to 1 Nov: October is `[Oct 1, Nov 1]` and November is
  `[Nov 1, Dec 1]`. A month-end report cannot show this, because it prints one reading
  per meter per month and the dropout sat between two of them.

December was a genuine 150 kWh step on `EM/048`, not a dropout to zero — but it too
recovers, so the dropout filter now repairs it and December reports an exact figure
instead of zero-plus-disclosure. If the step was a real downward correction rather
than a bad read, that attributes 150 kWh to December, which **overstates** — the
direction NABERS permits — where the previous treatment understated.

**August 2025 is a different failure and is not recoverable.** It is withheld by one
board with a different reason entirely:

```
EM/059 Basement Landlords Db Power: no reading at or before this point
(1 Aug 25); earliest held 6 Aug 25 (1 implausible readings discarded)
```

**Earliest held is five days *after* the boundary**, and that settles it. Unlike a
dropout, which is an impossible reading that can simply be removed, this opening value
is genuinely *unknown*: `EM/059` could have sat at its 6 August value all through July
or climbed to it, and nothing in the data distinguishes the two. Guessing invents the
meter's prior consumption rather than removing a lie, which is why a gap open at the
earlier end is refused.

**Looking further back cannot help, and this is worth being precise about**, because
`searchWindowDays` is the obvious lever and it is the wrong one:

- 45 days back from 1 Aug 2025 already reaches **17 Jun 2025**, so July is fully
  covered and then some.
- Device histories only retain from **~21 Jul 2025** (table above), so at most 11 days
  before the boundary could ever have held anything.
- `EM/059` totalled ~912 kWh by end September and consumed 20 kWh in October. History
  records only changes, so a board that quiet writing nothing across 11 days is the
  expected behaviour, not a fault.
- At most **one** record existed in that window: pass 1 pages five records per probe
  and only one reading was discarded across the meter's whole pool. If that record was
  the discarded one, it was corrupt and unusable anyway.

So the answer is retention plus on-change recording, not configuration. A backward
carry bounded by `gapThresholdHours` would not rescue it either — 6 August is five
days out, not three hours. **August 2025 has to come from the FM provider's verified
readings or King & Moffatt's manual ones**, exactly as the first rating period already
does.

It also expires: the table is a rolling twelve months, so August 2025 leaves the
window on **1 September 2026** and the standing rating should settle then without
anyone doing anything. The one loose end worth raising is `EM/059`'s single discarded
reading, visible as "1 discarded" in Meter data quality — one corrupt record on a
quiet board, low priority, but it is a real driver fault.

### 2. On-site generation

Only **self-consumed** generation reduces rated electricity; generation exported
or on-sold cannot be deducted.

```
PV self-consumed = generation − export        (when export is metered)
                 = generation                 (when it is not, flagged as assumed)
```

At 3CS there is no export meter and none is needed: (1) §6.9 states the entire
27,175 kWh/yr is consumed on site, because average landlord demand of 115 kW far
exceeds peak generation. The dashboard deducts all generation and flags the
deduction as assumed, which is the correct treatment here.

### 3. Equivalent energy

Delivered kWh of different fuels are never summed 1:1. Each is weighted by its
equivalent-energy factor first:

| Fuel | EEF |
|---|---|
| Electricity | 1.0 |
| Gas, coal | 0.75 |
| District heating | 0.9 |
| District cooling | 0.4 |
| Condenser water | 0.04 |
| Diesel | 0.8 |

3CS is all-electric, so equivalent energy equals metered electricity. Coal and
diesel convert to kWh first (22.1 MJ/kg, 38.6 MJ/L, at 3.6 MJ/kWh).

### 4. The adjusted benchmark

```
benchmark = (136 + climateCorrection) × ratedHoursFactor + serverRoomAdjustment

climateCorrection = 0.011 × HDD + 0.034 × CDD − 26      (degree days base 15.5 °C)
ratedHoursFactor  = 0.0089 × h + 0.47                    (1.0 at ≈59.6 h/week)
```

Worked for 3CS, postcode area `B`, climate zone 6 (Midlands), HDD 2117.505,
CDD 241.39:

```
climateCorrection = 0.011×2117.505 + 0.034×241.39 − 26  =    5.4998
ratedHoursFactor  = 0.0089×61.1 + 0.47                  =    1.01379
benchmark         = (136 + 5.4998) × 1.01379            =  143.451 kWhe/m²·pa
```

Star ceilings follow from that benchmark:

| Rating | Max benchmarking factor | Intensity ceiling |
|---|---|---|
| 6.0 Stars | 26.5% | 38.01 kWhe/m² |
| 5.5 Stars | 39.75% | 57.02 kWhe/m² |
| **5.0 Stars** | **53%** | **76.03 kWhe/m²** |
| 4.5 Stars | 66.25% | 95.04 kWhe/m² |

### 5. Stars

```
intensity          = equivalentKwh / ratedArea
benchmarkingFactor = intensity / benchmark × 100         (100 = exactly on benchmark, lower is better)
stars              = 7 − 3.77358 × BF/100                (clamped to 0…6)
```

The dashboard shows the continuous decimal figure and the official half-star
band. Margins are stated as `(ceiling − actual) / actual`.

### The three stat cards share one dependency

**vs DfP target**, **Carbon intensity** and **Design margin** all derive from
`totalIntensity`, which is `headlineRating?.intensity`. `NabersStatCard` shows an
em dash when its value is null, so if there is no rating all three dash out
together. Seeing all three blank is one root cause, not three, and it is **not** a
missing config value: `dfpTargets.total`, `carbonFactor`, `postcode` and
`ratedHours` are all set here, and those cards need nothing else.

Each card now states *why* it has no figure, in place of its subtitle: whether a
rating input is missing from config, which board is unreadable, or how far into
the rating period it is. If a card shows a bare dash with no reason, the build
predates that change.

| Card | Formula |
|---|---|
| vs DfP target | `(totalIntensity − dfpTargets.total) / dfpTargets.total × 100` |
| Carbon intensity | `totalIntensity × carbonFactor` |
| Design margin | `(fiveStarMax − totalIntensity) / fiveStarMax × 100` |

So the fix for empty cards is always to get a rating: check the gauge's own state
first, then the Meter data quality panel. Before gap filling existed, one
unreadable meter out of the whole set was enough to blank all three.

> **Why the dashboard can read 5.5 while the certificate says 5.0.** The formal
> NABERS UK design-review rating is a *commitment* of 5.0 Stars, made with the
> scheme's recommended 25% margin for the gap between simulation and operation.
> The as-built prediction sits at 5.75 decimal, comfortably inside a 5.5 band.
> The dashboard reports what the meters imply; it is not contradicting the
> certificate.

### Validation

The implementation was checked against Cundall's own published Stage 4 figures
by feeding their inputs through `nabersRating.js`:

| | Cundall (1) Table 7-3 | This implementation |
|---|---|---|
| Decimal rating | 5.77 | **5.777** |
| Margin to 5.0 Stars | 63.6% | **63.5%** |
| Margin to 5.5 Stars | 22.7% | **22.6%** |

Inputs: 779,419 kWh electricity plus 750 L diesel, giving 785,852 kWhe over
16,903 m². Agreement to within 0.1 percentage points confirms the benchmark,
climate zone, hours factor and star formula are all correct.

### An external check on the measured figure

That validates the *arithmetic*. (5) is the first source that can validate the
*energy*, because it carries twelve complete months of measured landlord
consumption, June 2025 to May 2026:

| | kWh | kWh/m² |
|---|---|---|
| Gross, sum of (5)'s ACTUAL monthly totals | 904,610 | 53.52 |
| less PV actually generated, (5) EM/051 | −23,256 | −1.38 |
| **Net measured** | **881,354** | **52.14** |
| Revised as-built prediction, (4) §4 | 911,028 | 53.90 |

**This is NOT on the same basis as the dashboard, and the difference is quantified.**
Both exclude MCP02, but K&M counts all nine floors while the dashboard counts five, so
the dashboard should read **below** these figures by roughly the **≈66,470 kWh** of
excluded floor energy — about 7.4%, giving order **≈815,000 kWh net, ≈48.2 kWh/m²**.

That makes this a two-sided check rather than a target:

- If the dashboard reads **close to 881,354**, the exclusion in
  [item 7](https://github.com/vanti-dev/mepc-3cs/blob/main/config/common/nabersdashboard/README.md#7-the-fitout-floors-are-excluded-from-the-rating) has not taken effect.
- If it reads **well above** 881,354, MCP02 is still being counted somewhere — see
  [item 15](https://github.com/vanti-dev/mepc-3cs/blob/main/config/common/nabersdashboard/README.md#15-mcp02-was-double-counting-central-ahu-and-pumps).
- **≈815,000 is the expected landing zone**, and the gap to K&M is not an error: it is
  the five-of-nine-floors scope, and it is why quoting the dashboard figure requires
  carrying that caveat.

Two cautions on reading (5)'s tables. Its ACTUAL block prints each row's label
*below* its data row in the PDF text layer, so a naive extraction shifts every
category by one; realigned, the rows reconcile against (5)'s own commentary (May
Cooling & Heating 18,314 kWh, the January peak of 67,488 kWh, the May total of
63,279 kWh) and against (4) §2's actual pump column. And the measured figure is
*better* than the prediction largely because the building is empty, not because it
is efficient — see [item 5](https://github.com/vanti-dev/mepc-3cs/blob/main/config/common/nabersdashboard/README.md#5-rated-area-will-fall).
