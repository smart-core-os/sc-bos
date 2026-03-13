package trait

import "fmt"

func ExampleName_Local() {
	fmt.Println(Booking.Local())
	fmt.Println(OnOff.Local())
	// Output:
	// Booking
	// OnOff
}
