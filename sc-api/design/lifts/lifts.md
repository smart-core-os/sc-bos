# Trait requirements for lifts/elevators

## Core requirements

We want to enable the following UX flow

1. User scans their card at a turn style device
2. User is accepted into the building
3. User is directed to lift door 2
4. User enters the lift
5. Lift transports the user to the floor they needed to go to

We are assuming that these aren't call->wait->enter->press button->wait->check floor->leave style lifts.
Instead the "call" action is replaced with a "go from floor 1 to 4" instruction.
We should still support the more traditional use case as part of the API, though I hope we get it for free.

Do we need to track the location of each transport box, aside from the actions around transporting a user?
i.e. Where are all the lifts, where's the bus, etc

### How do we know which floor the user wants to go to?

  - How do we know which user it is
  - Or where they're going
  - Or if they want to go there now

I think this problem is so intertwined it shouldn't be embedded in Smart Core, instead SC should include the APIs needed
to enable modules to implement this based on the API.

For example the EnterLeaveSensor can announce to the module that a UserA has entered the building, the module can then
check their data to see where that user is likely to want to go. If this results in a meeting room, then the module should
convert that into a floor number to then call another API to request transport.

### How does the user know which lift to use?

I'm assuming that it's the lift system that decides which transportation box to dispatch to the callers location.
This means, if we want to display the lift number the user should enter to the user on a screen that is not owned/part 
of the lift system we need an API (request or response) that tells us this information


### The passenger and the transportation may have different ideas of location

Passenger: "I'm on floor 1 and want to go to floor 4"
Transportation: "I can take you from Floor 1 Lift 2 to Floor 4 Lift 2"


### Does a journey need to be approved/committed?

If the transportation device reports a journey to the user that the user isn't happy with, can it be cancelled or not approved?
Should it default to approve, do we need an auto_approve or no_auto_approve flag for the request?
Passenger: "I'm on floor 1 and want to go to floor 2"
Transportation: "I can do that, but you have to wait 45s before you can enter the lift"
Passenger: "Never mind, I'll take the stairs" OR "Yes, that's fine with me" OR "..." no response




## Potential concepts that might cover this

### Moving from one place to another

Teleportation? Transportation? Movement? Relocation? Journey?
The concept of moving from one location to another.
This is subtly different from simply saying "I want to be There", a journey not just a destination

### Call for destination

Call a lift with the intention of going to a location.
With this concept you are interacting with the lift button, not the lift.
There's no real concept of physical location or progress.
The origin of the movement (the start point) is baked into the API call, and transparent - well implied at least - 
if I press the call button, it's implied I'm on the same floor as that button.


### Preparation actions before moving

"Go to bay 3", "Use lift 2", "Proceed to the taxi rank", etc

Something the requesting party needs to do in order for the transportation to begin.

### Sessions: request, approved, in progress, complete, etc

Transportation is not instantaneous and requires both time to setup and time to complete.
This means we can't have an API like "move me to floor 2" as we should expect the API to have completed the task by the time it returns.
I don't even think we can have an API like "Move to ground"... "Move to Floor 2" as these end creating difficulty for modules that want to chain movements.

We might need a concept of a session either for individual requests, or for movement of transport boxes related to a request.
For example: "Take me from floor 1 to 4" might result in a lift starting to move from floor 8 to floor 1 to prepare it for the move from 1 to 4.
We _need_ to associate that move from 8 to 1 with the original request somehow so the caller can know where their lift is, and which lift to use, etc.
This is different from "dumb" lifts that light up all the call buttons and show a current floor number above the lift door. While that's simpler
it doesn't allow for any personalisation of the information (i.e. "where is my lift") and critically doesn't allow the turn style to tell the user which lift to stand in front of.

### Report how far away the transport capsule is from the start location

In different units: 1 mile away, 10 minutes away, 2 floors away

### Automatic schedules, like trains

The concept that the transportation box will be moving from A to B at T whether you request it or not.

### Combining sessions

UserA requests transport from A to B, is asked to go to Bay1 and wait 30 seconds.
10s later UserB requests transport from A to B, is asked to go to Bay1 and wait 20 seconds, for the same transport box.
UserC requests transport from A to C (which goes via B), is asked to go to Bay1 and joins UserA and UserB on the first leg of their journey to C.

I don't think this concept needs to be encoded in the API, per se, but the lift/taxi/bus should be able to do this based on the information the API _does_ provide.

### Journey progress

How far along my journey am I? In different units, 2 floors to go, 1 mile, 10 minutes, 3 floors, etc.
This is like reporting the wait/setup/prep time, should have similar units.


### Location of transportation boxes

Where are they now: Lift 1 is at floor 2, Lift 2 @ Ground floor, Bus is between Upper st and Lower st.
Are they moving, and in which direction, at what speed, when will they get there, how long have they been travelling?
Are they accepting passengers, are the doors open, etc.


### Fault status

Is the lift out of order.
Should this be covered by another trait?


### Circular transport

What if it's the london eye, where you start and end at the same place. Do we want to cover this, can we?
