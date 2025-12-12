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

func (r Response) GetCode() string {
	return r.Code
}

func (r Response) GetMsg() string {
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
