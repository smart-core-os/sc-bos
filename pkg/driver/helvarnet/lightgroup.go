package helvarnet

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/smart-core-os/sc-api/go/traits"
	"github.com/smart-core-os/sc-bos/pkg/driver/helvarnet/config"
	"github.com/smart-core-os/sc-golang/pkg/resource"
)

// LightGroup represents a lighting group within the HelvarNet system.
type LightGroup struct {
	traits.UnimplementedLightApiServer
	traits.UnimplementedLightInfoServer

	brightness *resource.Value // *traits.Brightness
	client     *tcpClient
	conf       *config.Device
	number     int
	logger     *zap.Logger
	scenes     []config.Scene
}

func newLightingGroup(client *tcpClient, l *zap.Logger, conf *config.Device, n int) *LightGroup {
	return &LightGroup{
		brightness: resource.NewValue(resource.WithInitialValue(&traits.Brightness{}), resource.WithNoDuplicates()),
		client:     client,
		conf:       conf,
		logger:     l,
		number:     n,
	}
}

// sends the query commands to get the last known scene for this group
func (lg *LightGroup) getLastScene(ctx context.Context) error {

	command := queryLastSceneInGroup(lg.number)
	want := "?" + command[1:len(command)-1]

	response, err := lg.client.sendAndReceive(ctx, command, want)
	if err != nil {
		return fmt.Errorf("failed to get last scene for group %d: %v", lg.number, err)
	}

	split := strings.Split(response, "=")
	if len(split) < 2 {
		return fmt.Errorf("invalid response in getLastScene: %s", response)
	}

	sceneNumber := strings.TrimSuffix(split[1], "#")
	_, _ = lg.brightness.Set(&traits.Brightness{
		Preset: &traits.LightPreset{
			Name: sceneNumber,
		},
	})
	lg.logger.Info(fmt.Sprintf("last scene for %s was %s", lg.conf.Name, sceneNumber))
	return nil
}

// setScene sends the command to set the scene for the lighting group
// there are 8 blocks of 16 scenes, I am not sure what constant actually does, from the spec:
// "To call a constant light scene, use the Constant Light flag (with a parameter brightness of 1).
// The scene called will only be a constant light scene if configured so in Designer."
func (lg *LightGroup) setScene(ctx context.Context, block string, scene string, constant string) error {
	command := recallGroupScene(lg.number, block, scene, constant)

	_, err := lg.client.sendAndReceive(ctx, command, "")
	if err != nil {
		return err
	}
	return nil
}

// setLevel sends the command to set the level for the lighting group
func (lg *LightGroup) setLevel(ctx context.Context, level int) error {
	command := changeGroupLevel(lg.number, level)

	_, err := lg.client.sendAndReceive(ctx, command, "")
	if err != nil {
		return err
	}
	return nil
}

func (lg *LightGroup) getSceneNames(ctx context.Context) error {
	command := querySceneNames()
	want := "?" + command[1:len(command)-1] + "="

	r, err := lg.client.sendAndReceive(ctx, command, want)
	if err != nil {
		return err
	}

	r = strings.TrimPrefix(r, want)
	r = strings.TrimSuffix(r, "#")

	scenes := strings.Split(r, "@")

	for _, scene := range scenes {
		if !strings.HasPrefix(scene, strconv.Itoa(lg.number)) {
			// not this group
			// Note: when testing this I was getting responses like @0.1.3:Late Evening,@0.1.4:Cleaners
			// which would make 0 the group, which according to the spec is not valid (must be 1..16383)
			// these might be global scenes for all groups on this router
			// but I have not seen any documentation so no going to assume
			continue
		}
		// first half is address, second half is scene name
		s := strings.Split(scene, ":")
		if len(s) < 2 {
			lg.logger.Error("invalid scene name", zap.String("scene", scene), zap.Int("group", lg.number))
			continue
		}
		blockScene := strings.Split(s[0], ".")
		if len(blockScene) < 3 {
			lg.logger.Warn("invalid addr name", zap.String("addr", s[0]), zap.Int("group", lg.number))
			continue
		}
		title := s[1]
		lg.scenes = append(lg.scenes, config.Scene{
			Block: blockScene[1],
			Scene: blockScene[2],
			Title: title,
		})
	}

	if len(lg.scenes) == 0 {
		lg.logger.Info("no scene names found for group", zap.Int("group", lg.number))
	} else {
		lg.logger.Info("scene names", zap.Int("group", lg.number), zap.Any("scenes", lg.scenes))
		for _, scene := range lg.scenes {
			lg.logger.Info("scene", zap.String("block", scene.Block), zap.String("scene", scene.Scene), zap.String("title", scene.Title))
		}
	}
	return nil
}

// UpdateBrightness update the brightness level or preset (scene) of the lighting group
// if the request has a present included, this takes precedence and the level percent is ignored
func (lg *LightGroup) UpdateBrightness(ctx context.Context, req *traits.UpdateBrightnessRequest) (*traits.Brightness, error) {
	if req.Brightness == nil {
		return nil, status.Error(codes.InvalidArgument, "no brightness in request")
	}

	if req.Brightness.Preset != nil {
		// helvarnet scenes are in 8 blocks of 16 scenes, preset name is described in info as <block>:<scene>
		sceneSplit := strings.Split(req.Brightness.Preset.Name, ":")
		if len(sceneSplit) < 2 {
			return nil, status.Error(codes.InvalidArgument, "invalid scene format, must be <block>:<scene>")
		}
		block := sceneSplit[0]
		scene := sceneSplit[1]
		constant := "0"

		if len(sceneSplit) == 3 {
			constant = sceneSplit[2]
		}
		err := lg.setScene(ctx, block, scene, constant)
		if err != nil {
			return nil, status.Error(codes.DeadlineExceeded, "failed to set scene")
		}
		_, _ = lg.brightness.Set(&traits.Brightness{
			Preset: &traits.LightPreset{
				Name: req.Brightness.Preset.Name,
			},
		})
	} else {
		lg.logger.Debug(fmt.Sprintf("setting level %f for device %s", req.Brightness.LevelPercent, lg.conf.Name))
		level := req.Brightness.LevelPercent
		err := lg.setLevel(ctx, int(level))
		if err != nil {
			return nil, status.Error(codes.DeadlineExceeded, "failed to set scene")
		}
		_, _ = lg.brightness.Set(&traits.Brightness{
			LevelPercent: level,
		})
	}

	return nil, nil
}

func (lg *LightGroup) GetBrightness(_ context.Context, _ *traits.GetBrightnessRequest) (*traits.Brightness, error) {
	value := lg.brightness.Get()
	brightness := value.(*traits.Brightness)
	return brightness, nil
}

func (lg *LightGroup) PullBrightness(_ *traits.PullBrightnessRequest, server traits.LightApi_PullBrightnessServer) error {
	for value := range lg.brightness.Pull(server.Context()) {
		brightness := value.Value.(*traits.Brightness)
		err := server.Send(&traits.PullBrightnessResponse{Changes: []*traits.PullBrightnessResponse_Change{
			{
				Name:       lg.conf.Name,
				ChangeTime: timestamppb.New(value.ChangeTime),
				Brightness: brightness,
			},
		}})
		if err != nil {
			return err
		}
	}
	return nil
}

func (lg *LightGroup) DescribeBrightness(context.Context, *traits.DescribeBrightnessRequest) (*traits.BrightnessSupport, error) {
	result := &traits.BrightnessSupport{}
	for _, scene := range lg.scenes {
		result.Presets = append(result.Presets, &traits.LightPreset{
			Title: scene.Title,
			Name:  scene.Block + ":" + scene.Scene,
		})
	}
	return result, nil
}
