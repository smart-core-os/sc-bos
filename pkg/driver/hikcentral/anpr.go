package hikcentral

import (
	"context"
	"encoding/base64"
	"math"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/types"
	"github.com/smart-core-os/sc-golang/pkg/resource"
	"github.com/vanti-dev/sc-bos/pkg/driver/hikcentral/api"
	"github.com/vanti-dev/sc-bos/pkg/driver/hikcentral/config"
	"github.com/vanti-dev/sc-bos/pkg/gen"
)

var (
	errStartTimeIsZero       = status.Error(codes.InvalidArgument, "access grant start time is zero")
	errEndTimeIsZero         = status.Error(codes.InvalidArgument, "access grant end time is zero")
	errStartTimeAfterEndTime = status.Error(codes.InvalidArgument, "access grant start time is after end time")
)

type ANPRController struct {
	gen.UnimplementedAccessApiServer

	client *api.Client
	logger *zap.Logger
	conf   *config.Root

	listAccessGrants *Ring[*gen.AccessGrant]

	// key is the camera's smart-core name
	accessAttemptsResources map[string]*resource.Value // of type *gen.AccessAttempt
	lastPollTime            time.Time
}

func NewANPRController(client *api.Client, conf *config.Root, resources map[string]*resource.Value, logger *zap.Logger) *ANPRController {
	return &ANPRController{
		client: client,
		logger: logger.Named("anpr"),
		conf:   conf,

		listAccessGrants:        NewRing[*gen.AccessGrant](conf.GrantManagement.MaxListAccessGrants),
		accessAttemptsResources: resources,
		lastPollTime:            time.Now().Add(time.Hour * -1), // start polling from 1 hour ago
	}
}

var _ gen.AccessApiServer = (*ANPRController)(nil)

func (c *ANPRController) poll(ctx context.Context) error {
	c.logger.Debug("starting anpr poll")

	if len(c.conf.ANPRCameras) == 0 {
		c.logger.Debug("no ANPR cameras configured, skipping poll")
		return nil
	}

	// get appointments for today
	today := time.Now().Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	var appointmentsList []api.Appointment

	pageNo := 1

	for {
		select {
		case <-ctx.Done():
			c.logger.Debug("polling cancelled")
			return ctx.Err()
		default:
		}
		res, err := c.client.ListVisitorAppointments(ctx, &api.ListVisitorAppointmentsRequest{
			AppointStartTime: formatTime(today),
			AppointEndTime:   formatTime(tomorrow),
			Request: api.Request{
				PageNo:   pageNo,
				PageSize: 100,
			},
		})
		if err != nil {
			return status.Error(codes.Unavailable, "failed to list visitor appointments: "+err.Error())
		}
		if len(res.List) == 0 {
			break
		}

		appointmentsList = append(appointmentsList, res.List...)

		pageNo++
	}

	appointmentsByPlate := appointmentListToMap(appointmentsList)

	pageNo = 1
	for _, cam := range c.conf.ANPRCameras {
		select {
		case <-ctx.Done():
			c.logger.Debug("polling cancelled")
			return ctx.Err()
		default:
		}
		for {
			resp, err := c.client.ListANPREvents(ctx, &api.ANPREventsRequest{
				CameraIndexCode: cam.EntranceCameraIndexCode,
				StartTime:       formatTime(c.lastPollTime),
				EndTime:         formatTime(time.Now()),
				PageNo:          pageNo,
				PageSize:        100,
			})

			if err != nil {
				return status.Error(codes.Unavailable, "failed to list ANPR events: "+err.Error())
			}

			if len(resp.List) == 0 {
				c.logger.Debug("no ANPR events found", zap.Int("pageNo", pageNo))
				break
			}

			for _, event := range resp.List {
				accessTime, err := parseTime(event.CrossTime)

				if err != nil {
					c.logger.Error("failed to parse access time", zap.Error(err), zap.String("crossTime", event.CrossTime))
					accessTime = time.Now() // fallback to current time if parsing fails
				}

				relevantAppointments, ok := appointmentsByPlate[formatPlateNo(event.PlateNo)]

				if !ok {
					c.logger.Debug("no appointments found for ANPR event", zap.String("plateNo", event.PlateNo))
					accessAttempt := &gen.AccessAttempt{
						Grant:             gen.AccessAttempt_DENIED,
						Reason:            "no appointment found",
						Actor:             &gen.Actor{},
						AccessAttemptTime: timestamppb.New(accessTime),
					}

					_, err = c.accessAttemptsResources[cam.Name].Set(accessAttempt)

					if err != nil {
						c.logger.Error("failed to set access attempt", zap.Error(err), zap.String("plateNo", event.PlateNo))
					}

					continue
				}

				appointment := getMostRelevantAppointment(accessTime, relevantAppointments)
				reason, actor := extractFieldsFromAppointment(appointment)

				accessAttempt := &gen.AccessAttempt{
					Grant:             gen.AccessAttempt_GRANTED,
					Reason:            reason,
					Actor:             actor,
					AccessAttemptTime: timestamppb.New(accessTime),
				}

				_, err = c.accessAttemptsResources[cam.Name].Set(accessAttempt)

				if err != nil {
					c.logger.Error("failed to set access attempt", zap.Error(err), zap.String("plateNo", event.PlateNo))
				}
			}

			pageNo++
		}
	}

	c.lastPollTime = time.Now()

	return nil
}

func (c *ANPRController) GetLastAccessAttempt(ctx context.Context, request *gen.GetLastAccessAttemptRequest) (*gen.AccessAttempt, error) {
	res, ok := c.accessAttemptsResources[request.GetName()]

	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "no access attempt resource found: %s", request.GetName())
	}
	return res.Get(resource.WithReadMask(request.GetReadMask())).(*gen.AccessAttempt), nil
}

func (c *ANPRController) PullAccessAttempts(request *gen.PullAccessAttemptsRequest, server gen.AccessApi_PullAccessAttemptsServer) error {
	res, ok := c.accessAttemptsResources[request.GetName()]
	if !ok {
		return status.Errorf(codes.InvalidArgument, "no access attempt resource found with name: %s", request.GetName())
	}

	changes := res.Pull(server.Context(), resource.WithReadMask(request.GetReadMask()), resource.WithUpdatesOnly(request.GetUpdatesOnly()))
	for item := range changes {
		if item == nil {
			continue
		}

		accessAttempt := item.Value.(*gen.AccessAttempt)

		if err := server.Send(&gen.PullAccessAttemptsResponse{
			Changes: []*gen.PullAccessAttemptsResponse_Change{
				{
					AccessAttempt: accessAttempt,
					ChangeTime:    timestamppb.New(item.ChangeTime),
					Name:          request.GetName(),
				},
			},
		}); err != nil {
			return status.Error(codes.Internal, "failed to send access attempt: "+err.Error())
		}
	}

	return nil
}

func (c *ANPRController) GetAccessGrant(ctx context.Context, request *gen.GetAccessGrantsRequest) (*gen.AccessGrant, error) {
	if c.conf.GrantManagement == nil || request.GetName() != c.conf.GrantManagement.Name {
		return nil, status.Error(codes.Unimplemented, "not implemented per camera")
	}
	if request.GetAccessGrantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "access grant ID is required")
	}

	res := c.listAccessGrants.Find(func(grant *gen.AccessGrant) bool {
		return grant.GetId() == request.GetAccessGrantId()
	})

	if res == nil {
		return nil, status.Error(codes.NotFound, "access grant not found")
	}

	return res, nil
}

func (c *ANPRController) ListAccessGrants(ctx context.Context, request *gen.ListAccessGrantsRequest) (*gen.ListAccessGrantsResponse, error) {
	if c.conf.GrantManagement == nil || request.GetName() != c.conf.GrantManagement.Name {
		return nil, status.Error(codes.Unimplemented, "not implemented per camera")
	}

	v := c.listAccessGrants.Values()

	if request.GetPageSize() <= 0 {
		request.PageSize = int32(c.listAccessGrants.Len())
	}

	if request.PageSize > 1000 {
		request.PageSize = 1000
	}

	if request.GetPageToken() != "" {
		for i, grant := range v {
			if grant.GetId() == request.GetPageToken() {
				v = v[i+1:]
				break
			}
		}
	}

	nextPageToken := ""

	if len(v) > int(request.GetPageSize()) {
		nextPageToken = v[request.GetPageSize()-1].GetId()
		v = v[:request.GetPageSize()]
	}

	return &gen.ListAccessGrantsResponse{
		AccessGrants:  v,
		NextPageToken: nextPageToken,
		TotalSize:     int32(c.listAccessGrants.Len()),
	}, nil
}

func (c *ANPRController) CreateAccessGrant(ctx context.Context, request *gen.CreateAccessGrantRequest) (*gen.AccessGrant, error) {
	if c.conf.GrantManagement == nil || request.GetName() != c.conf.GrantManagement.Name {
		return nil, status.Error(codes.Unimplemented, "not implemented per camera")
	}

	if err := validAccessGrantTiming(request.GetAccessGrant()); err != nil {
		return nil, err
	}

	autoApproval, err := c.client.CheckAutoReviewFlow(ctx, &api.AutoReviewFlowRequest{})

	if err != nil {
		c.logger.Warn("failed to check auto review flow", zap.Error(err))
		autoApproval = &api.AutoReviewFlowResponse{AutomaticApproval: 1} // default to automatic approval if the check fails
	}

	manuallyApprove := c.conf.GrantManagement.EnableSmartCoreApproval && autoApproval.AutomaticApproval == 0

	req := createAppointmentRequest(request)

	resp, err := c.client.CreateVisitorAppointment(ctx, req)

	if err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	if manuallyApprove {
		c.logger.Debug("manually approving access grant reservation")

		appointmentRecordID, err := strconv.ParseInt(resp.AppointRecordID, 10, 64)

		if err != nil {
			return nil, status.Error(codes.Internal, "failed to parse appointment record ID: "+err.Error())
		}

		resp, err := c.client.ManuallyApproveVisitor(ctx, &api.VisitorApprovalRequest{
			OperateType:     0,
			ApprovalOpinion: "approved via SmartCore",
			ApprovalFlowInfo: api.ApprovalFlowInfo{
				ApprovalFlowCode: int(appointmentRecordID),
			},
		})

		if err != nil {
			return nil, status.Error(codes.Unavailable, "failed to manually approve visitor: "+err.Error())
		}

		c.logger.Info("manually approving visitor results", zap.Int("errorCode", resp.ErrorCode), zap.Int("visitorId", resp.VisitorId))
	}

	grantee := request.GetAccessGrant().GetGrantee()
	grantee.Name = resp.VisitorID

	qrCodeBytes, err := base64.StdEncoding.DecodeString(resp.QRCodeImage)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to decode QR code image: "+err.Error())
	}

	grant := &gen.AccessGrant{
		Id:        resp.AppointRecordID,
		StartTime: request.GetAccessGrant().GetStartTime(),
		EndTime:   request.GetAccessGrant().GetEndTime(),
		Purpose:   ptr(request.GetAccessGrant().GetPurpose()),
		EntryCode: ptr(resp.AppointCode),
		QrCode:    &gen.AccessGrant_QrCodeImage{QrCodeImage: qrCodeBytes},
		Grantee:   grantee,
		Granter:   request.GetAccessGrant().GetGranter(),
	}

	c.listAccessGrants.Add(grant)

	return grant, nil
}

func (c *ANPRController) UpdateAccessGrant(ctx context.Context, request *gen.UpdateAccessGrantRequest) (*gen.AccessGrant, error) {
	if c.conf.GrantManagement == nil || request.GetName() != c.conf.GrantManagement.Name {
		return nil, status.Error(codes.Unimplemented, "not implemented per camera")
	}

	if request.GetAccessGrant().GetId() == "" {
		return nil, status.Error(codes.InvalidArgument, "access Grant ID is required")
	}

	if err := validAccessGrantTiming(request.GetAccessGrant()); err != nil {
		return nil, err
	}

	req := createAppointmentRequest(request)

	req.AppointRecordID = request.GetAccessGrant().GetId()

	resp, err := c.client.UpdateVisitorAppointment(ctx, req)

	if err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	grantee := request.GetAccessGrant().GetGrantee()
	grantee.Name = resp.VisitorID

	qrCodeBytes, err := base64.StdEncoding.DecodeString(resp.QRCodeImage)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to decode QR code image: "+err.Error())
	}

	updatedGrant := &gen.AccessGrant{
		Id:        resp.AppointRecordID,
		StartTime: request.GetAccessGrant().GetStartTime(),
		EndTime:   request.GetAccessGrant().GetEndTime(),
		Purpose:   ptr(request.GetAccessGrant().GetPurpose()),
		EntryCode: ptr(resp.AppointCode),
		QrCode:    &gen.AccessGrant_QrCodeImage{QrCodeImage: qrCodeBytes},
		Grantee:   grantee,
		Granter:   request.GetAccessGrant().GetGranter(),
	}

	c.listAccessGrants.Update(func(grant *gen.AccessGrant) bool {
		return grant.GetId() == request.GetAccessGrant().GetId()
	}, func(old *gen.AccessGrant) {
		proto.Merge(old, updatedGrant)
	})

	return updatedGrant, nil
}

func (c *ANPRController) DeleteAccessGrant(ctx context.Context, request *gen.DeleteAccessGrantRequest) (*gen.DeleteAccessGrantResponse, error) {
	if c.conf.GrantManagement == nil || request.GetName() != c.conf.GrantManagement.Name {
		return nil, status.Error(codes.Unimplemented, "not implemented per camera")
	}

	if request.GetAccessGrantId() == "" {
		return nil, status.Error(codes.InvalidArgument, "access Grant ID is required")
	}

	_, err := c.client.DeleteVisitorAppointment(ctx, &api.DeleteVisitorAppointmentRequest{AppointRecordID: request.GetAccessGrantId()})

	if err != nil {
		return nil, status.Error(codes.Unavailable, err.Error())
	}

	return &gen.DeleteAccessGrantResponse{}, nil
}

type accessGrantRequest interface {
	GetAccessGrant() *gen.AccessGrant
}

func validAccessGrantTiming(grant *gen.AccessGrant) error {
	if grant.GetStartTime().AsTime().IsZero() {
		return errStartTimeIsZero
	}

	if grant.GetEndTime().AsTime().IsZero() {
		return errEndTimeIsZero
	}

	if grant.GetStartTime().AsTime().After(grant.GetEndTime().AsTime()) {
		return errStartTimeAfterEndTime
	}

	return nil
}

func createAppointmentRequest(request accessGrantRequest) *api.VisitorAppointmentRequest {
	var visitReasonType int
	purpose := strings.ToLower(request.GetAccessGrant().GetPurpose())
	if strings.Contains(purpose, "business") {
		visitReasonType = 0
	} else if strings.Contains(purpose, "training") {
		visitReasonType = 1
	} else if strings.Contains(purpose, "visit") {
		visitReasonType = 2
	} else if strings.Contains(purpose, "meeting") {
		visitReasonType = 3
	} else {
		visitReasonType = 4
	}
	if len(purpose) > 128 {
		purpose = purpose[:128]
	}

	var pictureUrls []api.VisitorPhoto
	for _, source := range request.GetAccessGrant().GetGrantee().GetPicture().GetSources() {
		for _, content := range source.GetSrc() {
			if content.GetUrl() != "" {
				pictureUrls = append(pictureUrls, api.VisitorPhoto{
					ImageURL: content.GetUrl(),
				})
			}
		}
	}

	receptionistId := request.GetAccessGrant().GetGranter().GetName()
	if len(receptionistId) > 64 {
		receptionistId = receptionistId[:64]
	}

	return &api.VisitorAppointmentRequest{
		ReceptionistID:    receptionistId,
		AppointStartTime:  formatTime(request.GetAccessGrant().GetStartTime().AsTime()),
		AppointEndTime:    formatTime(request.GetAccessGrant().GetEndTime().AsTime()),
		VisitReasonType:   visitReasonType,
		VisitReasonDetail: purpose,
		VisitorInfoList: []api.VisitorInfo{
			{
				VisitorFamilyName: request.GetAccessGrant().GetGrantee().GetName(),
				VisitorGivenName:  request.GetAccessGrant().GetGrantee().GetDisplayName(),
				Gender:            0, // unknown
				Email:             request.GetAccessGrant().GetGrantee().GetEmail(),
				PlateNo:           request.GetAccessGrant().GetGrantee().GetVehicleRegistration(),
				CompanyName:       request.GetAccessGrant().GetGrantee().GetCompany(),
				VisitorPhoto:      pictureUrls,
			},
		},
	}
}

func appointmentListToMap(list []api.Appointment) map[string][]*api.Appointment {
	appointments := make(map[string][]*api.Appointment)
	for _, appointment := range list {
		key := formatPlateNo(appointment.VisitorInfo.PlateNo)
		appointments[key] = append(appointments[key], &appointment)
	}
	return appointments
}

// getMostRelevantAppointment finds the most relevant appointment based on the current time.
// It prefers appointments that are closer to the current time,
// and if two appointments have the same start time,
// it prefers the one with the longer duration (i.e. the one that ends later).
func getMostRelevantAppointment(now time.Time, appointments []*api.Appointment) *api.Appointment {
	if len(appointments) == 0 {
		return nil
	}

	var relevantAppointment *api.Appointment

	for _, appointment := range appointments {
		if relevantAppointment == nil {
			relevantAppointment = appointment
			continue
		}

		start, err := parseTime(appointment.AppointStartTime)
		if err != nil {
			continue
		}

		end, err := parseTime(appointment.AppointEndTime)
		if err != nil {
			continue
		}

		startDelta := now.Sub(start)
		endDelta := now.Sub(end)

		relevantStart, err := parseTime(relevantAppointment.AppointStartTime)
		if err != nil {
			continue
		}

		relevantEnd, err := parseTime(relevantAppointment.AppointEndTime)
		if err != nil {
			continue
		}

		relevantStartDelta := now.Sub(relevantStart)
		relevantEndDelta := now.Sub(relevantEnd)

		// the difference between the start times and now is smaller for the current appointment,
		// i.e. it is more relevant to current time
		if math.Abs(float64(startDelta)) < math.Abs(float64(relevantStartDelta)) {
			relevantAppointment = appointment
		}

		// we still prefer longer appointments, so if the end time is further away from now, we prefer it
		// unless the start time is in a different hour, then we prefer the one with the closer start time to now
		if relevantStart.Truncate(time.Hour) == start.Truncate(time.Hour) && math.Abs(float64(endDelta)) > math.Abs(float64(relevantEndDelta)) {
			relevantAppointment = appointment
		}
	}

	return relevantAppointment
}

func extractFieldsFromAppointment(appointment *api.Appointment) (string, *gen.Actor) {
	var reason string
	switch appointment.VisitReasonType {
	case 0:
		reason = "business"
	case 1:
		reason = "training"
	case 2:
		reason = "visit"
	case 3:
		reason = "meeting"
	}

	var pictures []*types.Image_Content
	for _, pic := range appointment.VisitorInfo.VisitorPhoto {
		pictures = append(pictures, &types.Image_Content{
			Content: &types.Image_Content_Url{
				Url: pic.ImageURL,
			},
		})
	}

	actor := &gen.Actor{
		Name:        appointment.VisitorInfo.VisitorID,
		Title:       appointment.VisitorInfo.VisitorFamilyName,
		DisplayName: appointment.VisitorInfo.VisitorGivenName,
		Picture: &types.Image{
			Sources: []*types.Image_Source{
				{
					Src: pictures,
				},
			},
		},
		Email:               appointment.VisitorInfo.Email,
		Company:             appointment.VisitorInfo.CompanyName,
		VehicleRegistration: appointment.VisitorInfo.PlateNo,
	}

	return reason, actor
}

func formatPlateNo(plateNo string) string {
	// remove all non-alphanumeric characters and convert to uppercase
	plateNo = strings.ToUpper(plateNo)
	var sb strings.Builder
	for _, r := range plateNo {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

func ptr[T any](v T) *T {
	return &v
}
