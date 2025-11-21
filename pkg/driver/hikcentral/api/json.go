package api

import (
	"encoding/json"
)

type Request struct {
	PageNo   int `json:"pageNo"`
	PageSize int `json:"pageSize"`
}

type BaseResponse interface {
	getCode() string
	getMsg() string
}

type Response struct {
	Code string      `json:"code,omitempty"`
	Msg  string      `json:"msg,omitempty"`
	Data interface{} `json:"data,omitempty"`
}

func (r Response) getCode() string {
	return r.Code
}

func (r Response) getMsg() string {
	return r.Msg
}

type ResponseRaw struct {
	Response
	Data json.RawMessage `json:"data,omitempty"`
}

type PageResponse struct {
	Total    int `json:"total"`
	PageNo   int `json:"pageNo"`
	PageSize int `json:"pageSize"`
}

type CamerasRequest struct {
	Request
	SiteIndexCode  string `json:"siteIndexCode,omitempty"`
	DeviceType     string `json:"deviceType,omitempty"`
	BRecordSetting string `json:"bRecordSetting,omitempty"`
}

type CamerasResponse struct {
	PageResponse
	List []CameraInfo `json:"list,omitempty"`
}

type CameraRequest struct {
	CameraIndexCode string `json:"cameraIndexCode,omitempty"`
}

type CameraStatus int

const (
	CameraStatusUnknown CameraStatus = iota
	CameraStatusOnline
	CameraStatusOffline
)

type CameraInfo struct {
	CameraIndexCode string       `json:"cameraIndexCode,omitempty"`
	CameraName      string       `json:"cameraName,omitempty"`
	CapabilitySet   string       `json:"capabilitySet,omitempty"`
	DevResourceType string       `json:"devResourceType,omitempty"`
	DevIndexCode    string       `json:"devIndexCode,omitempty"`
	RecordType      string       `json:"recordType,omitempty"`
	RecordLocation  string       `json:"recordLocation,omitempty"`
	RegionIndexCode string       `json:"regionIndexCode,omitempty"`
	SiteIndexCode   string       `json:"siteIndexCode,omitempty"`
	Status          CameraStatus `json:"status,omitempty"`
}

func (c *CameraInfo) IsEqual(c2 *CameraInfo) bool {
	return c.CameraIndexCode == c2.CameraIndexCode &&
		c.CameraName == c2.CameraName &&
		c.CapabilitySet == c2.CapabilitySet &&
		c.DevResourceType == c2.DevResourceType &&
		c.DevIndexCode == c2.DevIndexCode &&
		c.RecordType == c2.RecordType &&
		c.RecordLocation == c2.RecordLocation &&
		c.RegionIndexCode == c2.RegionIndexCode &&
		c.SiteIndexCode == c2.SiteIndexCode &&
		c.Status == c2.Status
}

type CameraPreviewRequest struct {
	CameraRequest
	StreamType               int    `json:"streamType,omitempty"`
	Protocol                 string `json:"protocol,omitempty"`
	TransMode                int    `json:"transmode,omitempty"`
	RequestWebsocketProtocol int    `json:"requestWebsocketProtocol,omitempty"`
}

type CameraPreviewResponse struct {
	Url            string `json:"url,omitempty"`
	Authentication string `json:"authentication,omitempty"`
}

type EventsRequest struct {
	Request
	EventIndexCode string `json:"eventIndexCode,omitempty"`
	EventTypes     string `json:"eventTypes,omitempty"`
	SrcType        string `json:"srcType,omitempty"`
	SrcIndexes     string `json:"srcIndexs,omitempty"`
	SubSrcType     string `json:"subSrcType,omitempty"`
	SubSrcIndexes  string `json:"subSrcIndexs,omitempty"`
	StartTime      string `json:"startTime,omitempty"`
	EndTime        string `json:"endTime,omitempty"`
}

type EventsResponse struct {
	PageResponse
	List []EventRecord `json:"list,omitempty"`
}

type EventRecord struct {
	EventIndexCode      string `json:"eventIndexCode,omitempty"`
	EventType           string `json:"eventType,omitempty"`
	SrcType             string `json:"srcType,omitempty"`
	SrcIndex            string `json:"srcIndex,omitempty"`
	SubSrcType          string `json:"subSrcType,omitempty"`
	SubSrcIndex         string `json:"subSrcIndex,omitempty"`
	StartTime           string `json:"startTime,omitempty"`
	StopTime            string `json:"stopTime,omitempty"`
	Description         string `json:"description,omitempty"`
	EventPicUri         string `json:"eventPicUri,omitempty"`
	LinkCameraIndexCode string `json:"linkCameraIndexCode,omitempty"`
}

type StatisticsType int

const (
	StatisticsTypeByHour StatisticsType = iota
	StatisticsTypeByDay
	StatisticsTypeByMonth
	StatisticsTypeByMinute StatisticsType = 4
)

type StatsRequest struct {
	Request
	CameraIndexCodes string         `json:"cameraIndexCodes,omitempty"`
	StatisticsType   StatisticsType `json:"statisticsType"`
	StartTime        string         `json:"startTime,omitempty"`
	EndTime          string         `json:"endTime,omitempty"`
}

type StatsResponse struct {
	PageResponse
	Completeness int               `json:"completeness,omitempty"`
	List         []PeopleCountInfo `json:"list,omitempty"`
}

type PeopleCountInfo struct {
	Time            string `json:"time,omitempty"`
	CameraIndexCode string `json:"cameraIndexCode,omitempty"`
	ExitNum         int    `json:"exitNum,omitempty"`
	EnterNum        int    `json:"enterNum,omitempty"`
}

type PtzRequest struct {
	CameraIndexCode string `json:"cameraIndexCode,omitempty"`
	Action          int    `json:"action,omitempty"`
	Command         string `json:"command,omitempty"`
	Speed           int    `json:"speed,omitempty"`
	PresetIndex     int    `json:"presetIndex,omitempty"`
}

type PtzResponse struct {
}

type AutoReviewFlowRequest struct{}

type AutoReviewFlowResponse struct {
	AutomaticApproval int `json:"automaticApproval,omitempty"`
}

type VisitorApprovalRequest struct {
	OperateType int `json:"operateType,omitempty"`
	// 0: Approve, 1: Reject
	ApprovalOpinion  string           `json:"approvalOpinion,omitempty"`
	ApprovalFlowInfo ApprovalFlowInfo `json:"approvalFlowInfo,omitempty"`
}

type ApprovalFlowInfo struct {
	ApprovalFlowCode int `json:"approvalFlowCode,omitempty"`
}

type VisitorApprovalResponse struct {
	ErrorCode int `json:"errorCode,omitempty"`
	VisitorId int `json:"visitorId,omitempty"`
}

type ListVisitorAppointmentsRequest struct {
	Request
	UserId           string `json:"userId,omitempty"`
	AppointStartTime string `json:"appointStartTime,omitempty"`
	AppointEndTime   string `json:"appointEndTime,omitempty"`
	VisitorName      string `json:"visitorName,omitempty"`
	CompanyName      string `json:"companyName,omitempty"`
	InterviewName    string `json:"interviewName,omitempty"`
	AppointCode      string `json:"appointCode,omitempty"`
	IdentiCode       string `json:"identiCode,omitempty"`
	PhoneNo          string `json:"phoneNo,omitempty"`
	AppointState     string `json:"appointState,omitempty"`
	VisitorReason    string `json:"visitorReason,omitempty"`
}

type ListVisitorAppointmentsResponse struct {
	PageResponse
	List []Appointment `json:"list,omitempty"`
}

type Appointment struct {
	AppointStartTime  string      `json:"appointStartTime"`
	AppointEndTime    string      `json:"appointEndTime"`
	AppointCode       string      `json:"appointCode"`
	AppointID         string      `json:"appointID"`
	VisitReasonType   int         `json:"visitReasonType"`
	VisitorReasonName string      `json:"visitorReasonName"`
	VisitReasonDetail string      `json:"visitReasonDetail"`
	AppointStatus     int         `json:"appointStatus"`
	VisitorInfo       VisitorInfo `json:"visitorInfo"`
}

type ANPREventsRequest struct {
	CameraIndexCode string `json:"cameraIndexCode"`
	PlateNo         string `json:"plateNo,omitempty"`
	OwnerName       string `json:"ownerName,omitempty"`
	Contact         string `json:"contact,omitempty"`
	StartTime       string `json:"startTime"`
	EndTime         string `json:"endTime"`
	PageNo          int    `json:"pageNo"`
	PageSize        int    `json:"pageSize"`
}

type ANPREventsResponse struct {
	PageResponse
	List []VehiclePassRecord `json:"list,omitempty"`
}

type VehiclePassRecord struct {
	CrossRecordSyscode string `json:"crossRecordSyscode"`
	CameraIndexCode    string `json:"cameraIndexCode"`
	PlateNo            string `json:"plateNo"`
	OwnerName          string `json:"ownerName"`
	Contact            string `json:"contact"`
	VehicleType        int    `json:"vehicleType"`
	VehiclePicUri      string `json:"vehiclePicUri"`
	CrossTime          string `json:"crossTime"`  // ISO 8601 format
	CreateTime         string `json:"createTime"` // ISO 8601 format
}

type VisitorAppointmentRequest struct {
	// Only valid on update calls
	AppointRecordID   string          `json:"appointRecordId,omitempty"` // Optional, used for updates + deletes
	UserID            string          `json:"userId,omitempty"`
	ReceptionistID    string          `json:"receptionistId,omitempty"`    // Person to be visited
	AppointStartTime  string          `json:"appointStartTime"`            // Expected arrival time (ISO 8601)
	AppointEndTime    string          `json:"appointEndTime"`              // Expected leave time (ISO 8601)
	VisitReasonType   int             `json:"visitReasonType"`             // 0-business, 1-training, 2-visit, 3-meeting, 4-others.
	VisitReasonDetail string          `json:"visitReasonDetail,omitempty"` // Required if VisitReasonType == 4. Maximum 128 characters length.
	VisitorInfoList   []VisitorInfo   `json:"visitorInfoList"`             // Currently supports 1 visitor max
	AccessInfo        *AccessInfo     `json:"accessInfo,omitempty"`        // Optional access rights
	WatchListInfo     []WatchListInfo `json:"watchListInfo,omitempty"`     // Optional watch list info
}

type VisitorInfo struct {
	VisitorID         string `json:"visitorId,omitempty"`       // Optional visitor ID (max 64 chars)
	VisitorFamilyName string `json:"visitorFamilyName"`         // Required, max 256 chars, no special chars
	VisitorGivenName  string `json:"visitorGivenName"`          // Required, max 256 chars, no special chars
	Gender            int    `json:"gender"`                    // Required: 0=unknown, 1=male, 2=female
	Email             string `json:"email,omitempty"`           // Optional email
	PhoneNo           string `json:"phoneNo,omitempty"`         // Optional phone, max 20 chars
	PlateNo           string `json:"plateNo,omitempty"`         // Optional license plate, max 16 chars, no special chars
	CompanyName       string `json:"companyName,omitempty"`     // Optional company name, no special chars
	CertificateType   int    `json:"certificateType,omitempty"` // Optional: 111=ID, 414=Passport, 335=License
	CertificateNo     string `json:"certificateNo,omitempty"`   // Optional, required if certificateType is set
	Remark            string `json:"remark,omitempty"`          // Optional remark, max 128 chars

	Faces         []FaceInfo     `json:"faces,omitempty"`         // Optional list of face info (e.g., Base64 or URL)
	Fingerprints  []FingerPrint  `json:"fingerPrint,omitempty"`   // Optional fingerprint info
	Cards         []CardInfo     `json:"cards,omitempty"`         // Optional card numbers
	IdentityPhoto []string       `json:"identityPhoto,omitempty"` // Optional ID photos (Base64 or URL)
	VisitorPhoto  []VisitorPhoto `json:"visitorPhoto,omitempty"`  // Optional profile photos
	CustomField   *CustomField   `json:"customField,omitempty"`   // Optional custom fields
	AccessInfo    *AccessInfo    `json:"accessInfo,omitempty"`    // Optional access permission info
}

// Sub-objects

type FaceInfo struct {
	// Either Base64 image data or URL; field names based on API spec
	ImageData string `json:"imageData,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
}

type FingerPrint struct {
	// Define fields based on your fingerprint schema
	Data string `json:"data,omitempty"`
	Type string `json:"type,omitempty"`
}

type CardInfo struct {
	CardNo string `json:"cardNo"`
}

type VisitorPhoto struct {
	ImageData string `json:"imageData,omitempty"`
	ImageURL  string `json:"imageUrl,omitempty"`
}

type CustomField struct {
	// Define key-value structure depending on your system
	Fields map[string]string `json:"fields"`
}

type AccessInfo struct {
	// Example structure — adjust based on actual access control schema
	PersonID        string   `json:"personId,omitempty"`
	DeviceCodes     []string `json:"deviceCodes,omitempty"`
	PlanTemplateIDs []string `json:"planTemplateIds,omitempty"`
}

type VisitorAppointmentData struct {
	AppointRecordID string          `json:"appointRecordId"` // Reservation record ID
	VisitorID       string          `json:"visitorId"`       // Unique visitor ID
	AppointCode     string          `json:"appointCode"`     // Reservation code
	QRCodeImage     string          `json:"qrCodeImage"`     // Base64-encoded PNG image
	WatchListInfo   []WatchListInfo `json:"watchListInfo"`   // Optional watch list info
}

type WatchListInfo struct {
	ListType int    `json:"listType"`         // e.g., 0: None, 1: Blacklist, 2: Whitelist
	Remark   string `json:"remark,omitempty"` // Additional notes or explanation
}

type DeleteVisitorAppointmentRequest struct {
	AppointRecordID string `json:"appointRecordId"` // Required, ID of the appointment to delete
}
type DeleteVisitorAppointmentResponse struct{}
