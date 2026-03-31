package merge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/gobacnet"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/comm"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/config"
	"github.com/smart-core-os/sc-bos/pkg/driver/bacnet/known"
	"github.com/smart-core-os/sc-bos/pkg/node"
	"github.com/smart-core-os/sc-bos/pkg/proto/accesspb"
	"github.com/smart-core-os/sc-bos/pkg/proto/actorpb"
	"github.com/smart-core-os/sc-bos/pkg/proto/healthpb"
	"github.com/smart-core-os/sc-bos/pkg/task"
	"github.com/smart-core-os/sc-bos/pkg/trait"
)

type accessConfig struct {
	config.Trait

	IngressPermitted     *config.ValueSource `json:"ingressPermitted,omitempty"`
	IngressPermittedType *string             `json:"ingressPermittedType,omitempty"`
	IngressDenied        *config.ValueSource `json:"ingressDenied,omitempty"`
	IngressDeniedType    *string             `json:"ingressDeniedType,omitempty"`
}

func readAccessConfig(raw []byte) (cfg accessConfig, err error) {
	err = json.Unmarshal(raw, &cfg)
	return
}

type access struct {
	client     *gobacnet.Client
	known      known.Context
	faultCheck *healthpb.FaultCheck
	logger     *zap.Logger

	model *accesspb.Model
	*accesspb.ModelServer
	cfg      accessConfig
	pollTask *task.Intermittent
}

func newAccess(client *gobacnet.Client, devices known.Context, faultCheck *healthpb.FaultCheck, config config.RawTrait, logger *zap.Logger) (*access, error) {
	cfg, err := readAccessConfig(config.Raw)
	if err != nil {
		return nil, err
	}
	model := accesspb.NewModel()
	a := &access{
		client:      client,
		known:       devices,
		faultCheck:  faultCheck,
		logger:      logger,
		model:       model,
		ModelServer: accesspb.NewModelServer(model),
		cfg:         cfg,
	}
	a.pollTask = task.NewIntermittent(a.startPoll)
	return a, nil
}

func (a *access) startPoll(init context.Context) (stop task.StopFn, err error) {
	return startPoll(init, "access", a.cfg.PollPeriodDuration(), a.cfg.PollTimeoutDuration(), a.logger, func(ctx context.Context) error {
		_, err := a.pollPeer(ctx)
		return err
	})
}

func (a *access) AnnounceSelf(ann node.Announcer) node.Undo {
	return ann.Announce(a.cfg.Name, node.HasTrait(accesspb.TraitName, node.WithClients(accesspb.WrapApi(a))))
}

func (a *access) GetLastAccessAttempt(ctx context.Context, request *accesspb.GetLastAccessAttemptRequest) (*accesspb.AccessAttempt, error) {
	_, err := a.pollPeer(ctx)
	if err != nil {
		return nil, err
	}
	return a.ModelServer.GetLastAccessAttempt(ctx, request)
}

func (a *access) PullAccessAttempts(request *accesspb.PullAccessAttemptsRequest, server accesspb.AccessApi_PullAccessAttemptsServer) error {
	_ = a.pollTask.Attach(server.Context())
	return a.ModelServer.PullAccessAttempts(request, server)
}

func (a *access) pollPeer(ctx context.Context) (*accesspb.AccessAttempt, error) {
	data := &accesspb.AccessAttempt{}

	var resProcessors []func(response any, data *accesspb.AccessAttempt, cfg accessConfig) error
	var readValues []config.ValueSource
	var requestNames []string

	if a.cfg.IngressPermitted != nil {
		requestNames = append(requestNames, "ingressPermitted")
		readValues = append(readValues, *a.cfg.IngressPermitted)
		resProcessors = append(resProcessors, processIngressPermitted)
	}

	if a.cfg.IngressDenied != nil {
		requestNames = append(requestNames, "ingressDenied")
		readValues = append(readValues, *a.cfg.IngressDenied)
		resProcessors = append(resProcessors, processIngressDenied)
	}

	responses := comm.ReadProperties(ctx, a.client, a.known, readValues...)
	var errs []error
	for i, response := range responses {
		err := resProcessors[i](response, data, a.cfg)
		if err != nil {
			errs = append(errs, err)
		}
	}

	updateTraitFaultCheck(ctx, a.faultCheck, a.cfg.Name, trait.Access, errs)
	if len(errs) > 0 {
		return nil, multierr.Combine(errs...)
	}

	return a.model.UpdateLastAccessAttempt(data)
}

func processIngressPermitted(response any, data *accesspb.AccessAttempt, cfg accessConfig) error {
	value, ok := response.(string)
	if !ok {
		return comm.ErrReadProperty{Prop: "ingressPermitted", Cause: fmt.Errorf("converting to string")}
	}
	data.Grant = accesspb.AccessAttempt_GRANTED
	data.AccessAttemptTime = timestamppb.Now()
	data.Actor = &actorpb.Actor{
		LastGrantTime: timestamppb.Now(),
	}
	if cfg.IngressPermittedType != nil {
		data.Actor.Ids = map[string]string{
			*cfg.IngressPermittedType: value,
		}
	}
	return nil
}

func processIngressDenied(response any, data *accesspb.AccessAttempt, cfg accessConfig) error {
	value, ok := response.(string)
	if !ok {
		return comm.ErrReadProperty{Prop: "ingressDenied", Cause: fmt.Errorf("converting to string")}
	}
	data.Grant = accesspb.AccessAttempt_DENIED
	data.AccessAttemptTime = timestamppb.Now()

	if cfg.IngressDeniedType != nil {
		data.Actor = &actorpb.Actor{
			Ids: map[string]string{
				*cfg.IngressDeniedType: value,
			},
		}
	}
	return nil
}

func (a *access) CreateAccessGrant(context.Context, *accesspb.CreateAccessGrantRequest) (*accesspb.AccessGrant, error) {
	return nil, errors.New("method CreateAccessGrant not implemented")
}

func (a *access) UpdateAccessGrant(context.Context, *accesspb.UpdateAccessGrantRequest) (*accesspb.AccessGrant, error) {
	return nil, errors.New("method UpdateAccessGrant not implemented")
}

func (a *access) DeleteAccessGrant(context.Context, *accesspb.DeleteAccessGrantRequest) (*accesspb.DeleteAccessGrantResponse, error) {
	return nil, errors.New("method DeleteAccessGrant not implemented")
}

func (a *access) GetAccessGrant(context.Context, *accesspb.GetAccessGrantsRequest) (*accesspb.AccessGrant, error) {
	return nil, errors.New("method GetAccessGrant not implemented")
}

func (a *access) ListAccessGrants(context.Context, *accesspb.ListAccessGrantsRequest) (*accesspb.ListAccessGrantsResponse, error) {
	return nil, errors.New("method ListAccessGrants not implemented")
}
