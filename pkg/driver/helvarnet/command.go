package helvarnet

import (
	"fmt"
)

// send a scene recall across a group
// to call a constant light scene, use the Constant Light flag (with a parameter brightness of 1).
func recallGroupScene(group int, block string, scene string, constant string) string {
	return fmt.Sprintf(">V:1,C:11,G:%d,B:%s,K:%s,S:%s,A:1#", group, block, constant, scene)
}

// send a scene recall to a device
// This command should not be sent to any device other than a load (control gear),
// otherwise you will receive a diagnostic response if one was requested.
func recallDeviceScene(addr string, block string, scene string, constant string) string {
	return fmt.Sprintf(">V:1,C:12,@%s,B:%s,S:%s,K:%s,A:1#", addr, block, scene, constant)
}

// change the output level of all channels in a group.
func changeGroupLevel(group int, level int) string {
	return fmt.Sprintf(">V:1,C:13,G:%d,L:%d#", group, level)
}

// change the level of a load.
func changeDeviceLevel(addr string, level int) string {
	return fmt.Sprintf(">V:1,C:14,@%s,L:%d#", addr, level)
}

// query last scene in group
func queryLastSceneInGroup(group int) string {
	return fmt.Sprintf(">V:2,C:109,G:%d#", group)
}

// query the device state
func queryDeviceState(addr string) string {
	return fmt.Sprintf(">V:1,C:110,@%s#", addr)
}

// Returns the state or digital input(s) of: a device (e.g. for the PIR detector of a Multisensor -
// 0x01=occupied within past minute, 0x00=unoccupied); or the LEDs of a button panel; or the switch
// inputs of an input unit. If sent to the device level, summarises the digital input state. If sent to the
// subdevice level, gives the state of that subdevice’s input.
func queryInputState(addr string) string {
	return fmt.Sprintf(">V:1,C:151,@%s#", addr)
}

// A router will respond to this query with the scene descriptions that are prefixed with the corresponding
// group, block and scene.
// The scene description string contains the group, block, and scene numbers and scene description,
// and takes the form of '@G.B.S:Description'.
func querySceneNames() string {
	return ">V:2,C:166#"
}

// query the load level of a device, for a light device this returns the brightness
// from the docs:
//
//	Query load level commands may also report a level even though the device may be set to
//	'Off'. This is because the load level is set below the switch on level.
func queryLoadLevel(addr string) string {
	return fmt.Sprintf(">V:1,C:152,@%s#", addr)
}

// Emergency Function Test (Device)
// Request an Emergency Function Test to an emergency lighting ballast.
func deviceEmergencyFunctionTest(addr string) string {
	return fmt.Sprintf(">V:1,C:20,@%s#", addr)
}

// Emergency Duration Test (Device)
// Request an Emergency Duration Test to an emergency lighting ballast.
func deviceEmergencyDurationTest(addr string) string {
	return fmt.Sprintf(">V:1,C:22,@%s#", addr)
}

// Stop Emergency Tests (Device)
// Stop any Emergency Test running in an emergency ballast.
func deviceStopEmergencyTests(addr string) string {
	return fmt.Sprintf(">V:1,C:24,@%s#", addr)
}

// Query Emergency Function Test State
//
// - Emergency State Values
//
// - Pass 0
//
// - Lamp Failure 1
//
// - Battery Failure 2
//
// - Faulty 4
//
// - Failure 8
//
// - Test Pending 16
//
// - Unknown 32
func queryEmergencyFunctionTestState(addr string) string {
	return fmt.Sprintf(">V:1,C:171,@%s#", addr)
}

// Query Emergency Duration Test State
func queryEmergencyDurationTestState(addr string) string {
	return fmt.Sprintf(">V:1,C:173,@%s#", addr)
}

// Query Emergency Duration Test Time
// This is the time the test completed.
func queryEmergencyDurationTestTime(addr string) string {
	return fmt.Sprintf(">V:1,C:172,@%s#", addr)
}

// Query Emergency Function Test Time
// This is the time the test completed.
func queryEmergencyFunctionTestTime(addr string) string {
	return fmt.Sprintf(">V:1,C:170,@%s#", addr)
}
