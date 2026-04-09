package hikcentral

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/api"
	"github.com/smart-core-os/sc-bos/pkg/driver/hikcentral/config"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
)

const resourcePrefix = "/artemis/api/resource/v1"

type badStatusError struct {
	statusCode int
	status     string
}

func (e *badStatusError) Error() string {
	return fmt.Sprintf("unexpected status code %d: %s", e.statusCode, e.status)
}

type client struct {
	address    string
	appKey     string
	secret     string
	httpClient *http.Client
}

func newClient(conf *config.API) *client {
	return &client{
		address: conf.Address,
		appKey:  conf.AppKey,
		secret:  conf.Secret,
		httpClient: &http.Client{
			Timeout: conf.Timeout.Duration,
		},
	}
}

func (c *client) listCameraInfo(ctx context.Context, req *api.CamerasRequest, fc *healthpb.FaultCheck) (*api.CamerasResponse, error) {
	return makeReqWrapper[api.CamerasRequest, api.CamerasResponse](ctx, c, resourcePrefix+"/cameras", req, fc)
}

func (c *client) getCameraInfo(ctx context.Context, req *api.CameraRequest, fc *healthpb.FaultCheck) (*api.CameraInfo, error) {
	return makeReqWrapper[api.CameraRequest, api.CameraInfo](ctx, c, resourcePrefix+"/cameras/indexCode", req, fc)
}

func (c *client) getCameraPreviewUrl(ctx context.Context, req *api.CameraPreviewRequest, fc *healthpb.FaultCheck) (*api.CameraPreviewResponse, error) {
	if req.Protocol == "" {
		req.Protocol = "rtsp" // the spec says this is optional, but it's not
	}
	return makeReqWrapper[api.CameraPreviewRequest, api.CameraPreviewResponse](ctx, c, "/artemis/api/video/v1/cameras/previewURLs", req, fc)
}

func (c *client) getCameraPeopleStats(ctx context.Context, req *api.StatsRequest, fc *healthpb.FaultCheck) (*api.StatsResponse, error) {
	return makeReqWrapper[api.StatsRequest, api.StatsResponse](ctx, c, "/artemis/api/aiapplication/v1/people/statisticsTotalNumByTime", req, fc)
}

func (c *client) cameraPtzControl(ctx context.Context, req *api.PtzRequest, fc *healthpb.FaultCheck) (*api.PtzResponse, error) {
	return makeReqWrapper[api.PtzRequest, api.PtzResponse](ctx, c, "/artemis/api/video/v1/ptzs/controlling", req, fc)
}

func (c *client) listEvents(ctx context.Context, req *api.EventsRequest, fc *healthpb.FaultCheck) (*api.EventsResponse, error) {
	return makeReqWrapper[api.EventsRequest, api.EventsResponse](ctx, c, "/artemis/api/eventService/v1/eventRecords/page", req, fc)
}

func (c *client) checkAutoReviewFlow(ctx context.Context, req *api.AutoReviewFlowRequest) (*api.AutoReviewFlowResponse, error) {
	return makeReq[api.AutoReviewFlowRequest, api.AutoReviewFlowResponse](ctx, c, "/artemis/api/visitor/v1/visitorconfig/automaticapproval", req)
}

func (c *client) manuallyApproveVisitor(ctx context.Context, req *api.VisitorApprovalRequest) (*api.VisitorApprovalResponse, error) {
	return makeReq[api.VisitorApprovalRequest, api.VisitorApprovalResponse](ctx, c, "/artemis/api/visitor/v1/visitorapprovalflow/status", req)
}

func (c *client) listVisitorAppointments(ctx context.Context, req *api.ListVisitorAppointmentsRequest) (*api.ListVisitorAppointmentsResponse, error) {
	return makeReq[api.ListVisitorAppointmentsRequest, api.ListVisitorAppointmentsResponse](ctx, c, "/artemis/api/visitor/v1/appointment/appointmentlist", req)
}

func (c *client) listANPREvents(ctx context.Context, req *api.ANPREventsRequest) (*api.ANPREventsResponse, error) {
	return makeReq[api.ANPREventsRequest, api.ANPREventsResponse](ctx, c, "/artemis/api/pms/v1/crossRecords/page", req)
}

func (c *client) createVisitorAppointment(ctx context.Context, req *api.VisitorAppointmentRequest) (*api.VisitorAppointmentData, error) {
	return makeReq[api.VisitorAppointmentRequest, api.VisitorAppointmentData](ctx, c, "/artemis/api/visitor/v2/appointment", req)
}

func (c *client) updateVisitorAppointment(ctx context.Context, req *api.VisitorAppointmentRequest) (*api.VisitorAppointmentData, error) {
	return makeReq[api.VisitorAppointmentRequest, api.VisitorAppointmentData](ctx, c, "/artemis/api/visitor/v2/appointment/update", req)
}

func (c *client) deleteVisitorAppointment(ctx context.Context, req *api.DeleteVisitorAppointmentRequest) (*api.DeleteVisitorAppointmentResponse, error) {
	return makeReq[api.DeleteVisitorAppointmentRequest, api.DeleteVisitorAppointmentResponse](ctx, c, "/artemis/api/visitor/v1/appointment/single/delete", req)
}

func makeReqWrapper[R any, T any](ctx context.Context, client *client, path string, r *R, fc *healthpb.FaultCheck) (*T, error) {
	t, err := makeReq[R, T](ctx, client, path, r)
	updateReliability(ctx, err, fc)
	return t, err
}

func makeReq[R any, T any](ctx context.Context, client *client, path string, r *R) (*T, error) {
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

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("req.do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, &badStatusError{statusCode: resp.StatusCode, status: resp.Status}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("body.read: %w", err)
	}

	var respType api.ResponseRaw
	err = json.Unmarshal(respBody, &respType)
	if err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}
	if respType.GetCode() != "0" {
		return nil, fmt.Errorf("api err %s: %s", respType.GetCode(), respType.GetMsg())
	}

	var dataType T
	err = json.Unmarshal(respType.Data, &dataType)
	if err != nil {
		return nil, fmt.Errorf("unmarshal data: %w", err)
	}
	return &dataType, nil
}