package api

const (
	VideoLossAlarm                = "131329"
	VideoTamperingAlarm           = "131330"
	CameraRecordingExceptionAlarm = "385052"
	CameraRecordingRecovered      = "385053"
)

var AlarmTypes = []string{
	VideoLossAlarm,
	VideoTamperingAlarm,
	CameraRecordingExceptionAlarm,
}
