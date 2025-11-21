package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/vanti-dev/sc-bos/pkg/driver/hikcentral/config"
)

const resourcePrefix = "/artemis/api/resource/v1"

type Client struct {
	address    string
	appKey     string
	secret     string
	HTTPClient *http.Client
}

func NewClient(conf *config.API) *Client {
	return &Client{
		address: conf.Address,
		appKey:  conf.AppKey,
		secret:  conf.Secret,
		HTTPClient: &http.Client{
			Timeout: conf.Timeout.Duration,
		},
	}
}

func (c *Client) ListCameraInfo(ctx context.Context, req *CamerasRequest) (*CamerasResponse, error) {
	return makeReq[CamerasRequest, CamerasResponse](ctx, c, resourcePrefix+"/cameras", req)
}

func (c *Client) GetCameraInfo(ctx context.Context, req *CameraRequest) (*CameraInfo, error) {
	return makeReq[CameraRequest, CameraInfo](ctx, c, resourcePrefix+"/cameras/indexCode", req)
}

func (c *Client) GetCameraPreviewUrl(ctx context.Context, req *CameraPreviewRequest) (*CameraPreviewResponse, error) {
	if req.Protocol == "" {
		req.Protocol = "rtsp" // the spec says this is optional, but it's not
	}
	return makeReq[CameraPreviewRequest, CameraPreviewResponse](ctx, c, "/artemis/api/video/v1/cameras/previewURLs", req)
}

func (c *Client) GetCameraPeopleStats(ctx context.Context, req *StatsRequest) (*StatsResponse, error) {
	return makeReq[StatsRequest, StatsResponse](ctx, c, "/artemis/api/aiapplication/v1/people/statisticsTotalNumByTime", req)
}

func (c *Client) CameraPtzControl(ctx context.Context, req *PtzRequest) (*PtzResponse, error) {
	return makeReq[PtzRequest, PtzResponse](ctx, c, "/artemis/api/video/v1/ptzs/controlling", req)
}

func (c *Client) ListEvents(ctx context.Context, req *EventsRequest) (*EventsResponse, error) {
	return makeReq[EventsRequest, EventsResponse](ctx, c, "/artemis/api/eventService/v1/eventRecords/page", req)
}

func (c *Client) CheckAutoReviewFlow(ctx context.Context, req *AutoReviewFlowRequest) (*AutoReviewFlowResponse, error) {
	return makeReq[AutoReviewFlowRequest, AutoReviewFlowResponse](ctx, c, "/artemis/api/visitor/v1/visitorconfig/automaticapproval", req)
}

func (c *Client) ManuallyApproveVisitor(ctx context.Context, req *VisitorApprovalRequest) (*VisitorApprovalResponse, error) {
	return makeReq[VisitorApprovalRequest, VisitorApprovalResponse](ctx, c, "/artemis/api/visitor/v1/visitorapprovalflow/status", req)
}

func (c *Client) ListVisitorAppointments(ctx context.Context, req *ListVisitorAppointmentsRequest) (*ListVisitorAppointmentsResponse, error) {
	return makeReq[ListVisitorAppointmentsRequest, ListVisitorAppointmentsResponse](ctx, c, "/artemis/api/visitor/v1/appointment/appointmentlist", req)
}

func (c *Client) ListANPREvents(ctx context.Context, req *ANPREventsRequest) (*ANPREventsResponse, error) {
	return makeReq[ANPREventsRequest, ANPREventsResponse](ctx, c, "/artemis/api/pms/v1/crossRecords/page", req)
}

func (c *Client) CreateVisitorAppointment(ctx context.Context, req *VisitorAppointmentRequest) (*VisitorAppointmentData, error) {
	return makeReq[VisitorAppointmentRequest, VisitorAppointmentData](ctx, c, "/artemis/api/visitor/v2/appointment", req)
}

func (c *Client) UpdateVisitorAppointment(ctx context.Context, req *VisitorAppointmentRequest) (*VisitorAppointmentData, error) {
	return makeReq[VisitorAppointmentRequest, VisitorAppointmentData](ctx, c, "/artemis/api/visitor/v2/appointment/update", req)
}

func (c *Client) DeleteVisitorAppointment(ctx context.Context, req *DeleteVisitorAppointmentRequest) (*DeleteVisitorAppointmentResponse, error) {
	return makeReq[DeleteVisitorAppointmentRequest, DeleteVisitorAppointmentResponse](ctx, c, "/artemis/api/visitor/v1/appointment/single/delete", req)
}

func makeReq[R any, T any](ctx context.Context, client *Client, path string, r *R) (*T, error) {
	body, err := json.Marshal(r)
	if err != nil {
		return nil, fmt.Errorf("marshal: %w", err)
	}
	u, err := url.JoinPath(client.address, path)
	if err != nil {
		return nil, fmt.Errorf("joinPath: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("newRequest: %w", err)
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")

	err = prepareReq(req, body, client.secret, client.appKey)
	if err != nil {
		return nil, fmt.Errorf("prepareReq: %w", err)
	}

	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("req.do: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("body.read: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response: %s", resp.Status)
	}

	var respType ResponseRaw
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	if respType.getCode() != "0" {
		return nil, fmt.Errorf("api err %s: %s", respType.getCode(), respType.getMsg())
	}

	var dataType T
	err = json.Unmarshal(respType.Data, &dataType)
	if err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}
	return &dataType, nil
}
