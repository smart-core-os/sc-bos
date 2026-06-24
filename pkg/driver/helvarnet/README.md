# HelvarNet lighting driver

Reference: https://aca.im/driver_docs/Helvar/HelvarNet-Overview.pdf

HelvarNet is an Ethernet I/O protocol which allows third party devices (e.g. AV equipment) to query
and control a 905/910/920 router system and perform some basic system configuration, over an
Ethernet TCP connection. It is a
published standard which provides a set of rules for communicating with a Helvar lighting system.

The third party device may communicate with one or more routers in the system, provided it knows the IP address
of each router, in order to communicate with the lighting system.

To establish a TCP connection and therefore communicate with the router, the third party
device is required to connect to listener port number 50000.

Messages from the third party device can be targeted at any router in the system.


### Message Format

Any message sent to, or received from, a router can be in either ASCII or raw binary form (see
Command Format for more information).
Messages must not exceed the maximum length of 1500 bytes.
The format of the data contained within messages is defined by the protocol.
A query reply message from the router will be in the same format as the query command message
sent i.e. if a query message is sent in ASCII form then the reply will also be in ASCII.

Comparing ASCII & raw binary, the binary format requires you to always send 40 bytes but the 
ASCII can be less than 40 bytes (26 bytes for a standard scene recall message). 
Also, it will be easier to log & debug ASCII so we will use ASCII. 

### Device clocks and timezones

Routers report times (e.g. emergency test completion times) as epoch seconds derived from their
local wall-clock. If the routers' clocks are set to local time rather than UTC, the reported
values are offset from the true epoch time (e.g. an hour ahead during BST).

Set the `timezone` config option to the IANA timezone the routers' clocks follow
(e.g. `"timezone": "Europe/London"`) and the driver will re-interpret reported times in that
zone so the API exposes the correct UTC instant. When unset, reported times are taken as true
epoch seconds (UTC).
