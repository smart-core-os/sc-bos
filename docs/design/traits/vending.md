# Design notes for modelling devices that have an inventory and can dispense substances

## Example devices that should have this trait

1. Vending machine - "list stock", "dispense 1 Kitkat"
2. Coffee machine - "One flat white please", "there are 7 pods remaining", "the machine has used 47 pods to date"
3. Tap - "fill one cup of water", "the tap is not running"
4. Fridge - "6 apples remaining"
5. Vacuum - "the filter has been running for 4 months", "please replace the filter" (unlikely to be factored in)


## Common themes

### Consumables

A list of supported consumables the device is aware of, monitors, can dispense.

1. Dust filter (unlikely)
2. Coffee cup
3. Coffee pod
4. Water
5. Cat food
6. Apples
7. Kitkats
8. etc

### Quantity

A quantity of a consumable. This is associated with the device and with consumables in multiple ways:

1. As stock - the device has 2 Kitkats in stock, there are 4 coffee pods remaining
   1. Can stock also model "the water filter has 12 litres remaining until it needs to be changed"?
2. As a (capped?) total - the device has dispensed 217 litres of coffee in total, the filter has run for 18 hours
3. To dispense - ask the device for a cup of water, ask for two kitkats

Some quantities are discreet: kitkats, cat food bowls, glasses of water.
Some quantities are continuous: 18 hours of use, 12 litres remaining.
The same consumable can be both discreet and continuous: 560g of cat food, or 6 bowls remaining. Same for water in litres or cups

### Transfers

1. Dispense
2. Reset - i.e. restock, replace filter, add 2 apples
