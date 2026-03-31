# Smart Core DSR Trait (demand side response)

### Vehicle requirements

1. A vehicle comes along to a building and wants to charge.
2. Sometimes vehicles have higher than normal demand, they might:
   1. have a target level (could be 100%) and want to fill to that level the fastest OR
   2. have a fixed time period and want as much energy as possible in that time
3. We might know the vehicle is scheduled to arrive at a particular time or they might just show up.
4. Inability to receive this power could cause critical failures of systems (drones crash), or might just impair UX (i.e. your taxi is late).
   1. not just to that vehicle, there might be a queue of vehicles waiting for a delayed charge session to end
5. We need to be able to answer ahead of time with some confidence whether the building can satisfy an energy requirement before the vehicle turns up
   1. Used for route planning, scheduling, etc
   2. Diversions might need to be arranged if this building can't satisfy the requirements
6. The Vehicle is in control of the power draw, all we can do is advise and plan


### Other requirements

1. It would be nice for the building to be able to react to grid demand
2. It should be no more complicated than it needs to be
   1. Complexity should be placed on the server if there's a choice
3. Smart Core shouldn't require that this exists
4. It should be easy to reason about (or debug)


### Actors involved

Responsibilities, goals, conflicts for each of these

1. Charger/vehicle/high power demand thing
   1. goal: see [vehicle requirements](#vehicle-requirements)
   2. responsible for actual power draw levels
   3. should contribute information to the PMS to help with decision making
   4. responsible for safety of this system, circuit, vehicle, occupant
2. Power supply sensor
   1. responsible for reporting accurate (for some definition) and timely sensor readings
3. Power management system (brain)
   1. goal: make sure all devices have the power they need while maintaining operational limits
   2. responsible for what the optimal power profile should be
   3. responsible for communicating with devices what is wanted from it wrt power profiles
   4. responsible for collecting power predictions from devices (either via pull or push)
4. Grid
   1. goal: don't under/overload - use cheapest energy sources available
5. Breakers
   1. goal: protect the building from heat and damage
   2. responsible for breaking circuits if damage will be done
6. Vetoers for saying you can't have it
   1. goal: to know better than the system
   2. responsible for overriding system choices based on unknown factors (or bad decisions/logic)
7. Vehicle operators (e.g. deliveroo) - might not be the same operator as #1
   1. goal: deliver a reliable service
   2. want to know they can use building services (like chargers) so they can plan routes/etc
   3. don't want to turn up and have things not be like they thought they'd be
8. Occupants of the building (people using building services)
   1. goal: get on with their day
   2. don't want to have their experience impacted by external changes to their environment - e.g. too warm, lights too dim, etc
   3. want their car to be charged by the time they leave for home
9. FM / Ops / Maintenance
   1. goal: keep the building running, optimally
   2. responsible for repairs, replacements, scheduling down time
   3. want to know how successful they and the building are being - dashboards/reports
   4. responsible for overrides (maybe with their vetoers hat on)
   5. responsible for adjustment and data entry (party happening at 7, etc)


### Building assumptions

1. The building has a number of devices in it consuming power.
2. The power these devices consume is not constant - it increases/decreases and changes over time during device operation.
   1. e.g. Profile A is high power use then low power use, Profile B is low power use then high power use.
3. The device power use profile (when power is used and how much) can be adjusted either without noticeable effect or within acceptable margins.
4. There will be devices that consume power that are not controllable or predictable (turn kettle on, etc).
5. Controllable devices will have limitations to how much their power use can be adjusted (before the ice cream melts, etc).
6. Devices have a normal operating profile
7. Devices might need to reject or cancel/switch away from selected profiles based on:
   1. direct user intervention - a user clicks Cancel Profile button somewhere
   2. environmental changes - there's a fire, someone left the door open, etc
   3. bad predictions - we thought we'd need X power but we actually need Y
8. Building can be setup with devices connected that can collectively consume more power than the building is rated for, but where they _promise_ to not turn on at the same time


### Comments and Questions

1. How accurate is predicted power usage?
2. How granular/precise do values need to be?
3. Is energy constant? i.e. lots of power for a short time vs less power for a longer time - are they equivalent, is there a preferred (wear, power use, etc) profile?
4. Do we need to model curves, splines, gradients, rectangles, etc? Should it be more like audio signals (sample rates)?
5. Should this be modelled in Smart Core (DSR trait) or should this be covered via existing trait functionality
   - Turn lights to 70%, adjust HVAC set point by -2 degrees, etc VS
   - Use 10% less power, delay power use for 15 minutes, etc
6. Imagine a situation where we have energy storage used for quick charging.
   Normally the storage is filled from excess power but if we know a high-priority vehicle is due we might reduce consumption for non-essential devices to charge the storage quicker before the vehicle arrives.
   This use case might just look like a slightly more relaxed version of when the vehicle arrives without the storage being present?


### PAS 1878 Review - UK smart appliance DSR spec

Defines three parties, but there are really 4:
1. The device that is consuming power and can adjust use - ESA (energy smart appliance)
2. A system that converts between device commands and PAS 1878 commands (could be the same device as 1.) - CEM (customer energy manager)
3. A controller that tells the ESA (via the CEM) how to behave - DSRSP (DSR service provider)
4. (unmentioned) The grid and DSRSP communicate to formulate a strategy for adjusting power

The DSRSP collects a list of flexibility offers - predictions for how an ESA might consume power given possible modes of operation - from all connected CEMs.
It uses this information, and events/info from the grid to decide when and what action needs to be taken. Maybe more power needs to be consumed, maybe less.
The DSRSP sends a message to certain CEMs to request that they activate specific flexibility offers to adjust how the ESAs consume power vs their normal use.
CEMs and ESAs also take end user settings and overrides into account both during the flex offer computation and to cancel or adjust any active offers.

The spec also includes the concept of power reports, messages from ESAs back to the DSRSP to report on how much power was actually used at what time.
I think this is used for tracking and auditing - if an ESA says it should have a power profile but the report says otherwise, this info is useful.

This approach is good for cases where _flattening the curve_ is the goal. It should be able to reliably move demand around to match supply.

#### How this applies to us

Our use case is very similar, I'm not sure it's the same though.
In buildings there is a specified rating that we should not exceed, this is similar to the grid load that is being worked against.
The building rating might also change over time if the building is participating as an entity in DSR or even via commissioning (i.e. use less power during peak prices).
Where the PAS 1878 spec is concerned with flattening the curve, we are more about freeing up power for a future (possibly immediate) use.
The spec is reactive, we can be proactive.

#### Issues

There's no way to say _give me all available power_ or anything like that.
The flex options are a fixed shape which makes optimal use of available capacity difficult.


### Power Supply Trait Review

Defines a trait that reports on how much power is being consumed/is free/is available: rating, capacity, free, etc.
The trait also allows clients to notify the server that demand is about to be requested: notifications.
Notifications issued by clients can be rejected or adjusted by the server to avoid race conditions when there are multiple clients involved.
Notifications are represented as _X power for D duration_ - effectively a rectangle on the power vs time graph that starts at now.

To allow clients to request more power they can specify `force=true` when notifying, which causes the server to accept the notification even if
it would normally have rejected it due to lack of free capacity.

To integrate this design with a DSR system we'd need something that watched the capacity messages and decided to adjust power use of devices
based on this information. For example if that system notices that capacity + notified is greater than rating then more power should be made available.

Issues with this design include

1. Double counting: a notification represents an intent to draw power, when the power is drawn this is also noticed as a reduction in free capacity.
2. It's overly complicated wrt having to update the notification over time if we want to solve the double counting issue
3. There's no concept of future use or planning - notifications are always now+duration
4. Notifications are always rectangles, there are no for slopes/ramps.

Good things with this design

1. It includes a way for a client to be assigned _all available power_ - even if the adjustment is only in the magnitude direction
   1. for example the api doesn't allow you to request X power for D duration, and the server responds saying you can only have it for Y duration instead.
   2. ^ having said that, it can cap the duration of notifications, but this is a fair use feature not an standard use feature.


### General thoughts

I'm starting to think that we need to be more explicit about past/present vs future power demand/profiles/predictions/whatever.
We made a first stab at this introducing the draw notification concept, which has ended up being quite overloaded.
I'd really like the api to allow a conversation to happen - at least a negotiation - where an optimal solution can be found.
A simplistic approach would look like the 1878 spec where a collection of future power shapes are collected and combined together until a desired shape is made (like those games where you have to fix blocks into a shape).
This approach is non-optimal for our use case because our requirement includes phrases like "as quickly as possible".

Maybe this is a search problem: Adjust power to find the largest area, widest time period, highest magnitude, etc.

I could see a solution here falling into two possibly overlapping approaches

1. A client asks the server to choose a power profile for it given some parameters, invariants, etc. For example it might say Quickest charge possible, or I'll be here for 10 mins.
   1. Issues with this are we need to encode into the request the intent for what the client wants, what they are willing to change, what's immutable, what can we describe, etc
2. A client requests information about power from server, client assesses information to see if it'll work for it, client tells server to activate profiles.
   1. There's a race condition here that needs dealing with between the first request and the activation calls
   2. This adds complexity to the client as it needs to analyse the info from the server
   3. This is more flexible though as the client can use any algorithm to choose profiles to apply
3. We might be able to combine both approaches, have a _get the profiles_ + _apply the profiles_ and a _choose a profile for me_ API

Energy potential: charge stuff and have the potential be recorded for use later

Real power vs apparent power


### Decisions

1. Split out power level reading from anything else if possible
   1. Should allow more devices to implement it
   2. Shouldn't be focused on supply or demand, just on values


### End result first

The end result is that the BMS delays turning on the coolers, the lights switch to 75%, etc.
This reduces power used by these systems making it available for use by a higher priority consumer.
Eventually the high priority consumer goes away or the BMS can't delay the coolers anymore and things return to normal.

How does the BMS know to reduce power, and by how much?
Does the high priority consumer need to know how much will be available before it is?

#### Option 1: everybody reacts

The high priority consumer just starts to use power, maybe it ramps up, maybe it takes everything available when it becomes available.
The other devices (BMS, etc) notice the high load and reduce their power use, monitoring the status and adjusting consumption as needed.

This is very similar to frequency based DSR and could be implemented using a Smart Core trait for reporting load that all parties subscribe to.

1. The API for this is very simple, just exposing the current state of power use
2. This has issues with conflicts
   - what if multiple high priority consumers start consuming at the same time
   - what if there just isn't the power available
3. This solution might exclude dumber devices from releasing power even if they could without impacting UX.
4. UX might be more affected than needed
   - consuming devices have no way to know if they need to shed load and by how much and for how long
   - high priority devices have no way of knowing how much power they can use
5. No mechanism to plan ahead - store up some energy in advance of a known high demand use later


#### Option 2: dictatorship

All power consumers must get permission from a central system _before_ they consume power.
The central system is responsible for fair use, safe use, and for managing general power use.
Consuming devices are all treated the same, and must act within the bounds set out by the central system.
The consuming devices ask the central system for permission to use X power and the central system responds with a yes, no, or maybe a _use this instead_.
There needs to be a way for the central system to change its mind on who can have what power.

This feels similar to the USB power spec.
The draw notification spec is also like this, but without the requirement that the consuming device honour the response.
