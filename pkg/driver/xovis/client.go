package xovis

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/smart-core-os/sc-bos/pkg/gentrait/healthpb"
)

type client struct {
	// BaseURL is the root of the API.
	// e.g. https://1.2.3.4/api/v5
	BaseURL  url.URL
	Client   *http.Client
	Username string
	Password string
}

// newInsecureClient creates a client that connects over HTTPS but does not verify the server certificate.
func newInsecureClient(host string, username string, password string) *client {
	return &client{
		BaseURL: url.URL{
			Scheme: "https",
			Host:   host,
			Path:   "/api/v5",
		},
		Client: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		Username: username,
		Password: password,
	}
}

func (c *client) newRequest(method string, endpoint string) *http.Request {
	req := &http.Request{
		Method: method,
		URL:    c.BaseURL.JoinPath(endpoint),
		Header: make(http.Header),
	}
	req.SetBasicAuth(c.Username, c.Password)
	return req
}

func handleResponse(res *http.Response, destPtr any) error {
	defer func() {
		_ = res.Body.Close()
	}()
	switch res.StatusCode {
	case 200: // continue
	case 401:
		return status.Error(codes.FailedPrecondition, "server credentials are invalid")
	default:
		return readError(res.Body)
	}
	rawJSON, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if len(rawJSON) == 0 {
		// an empty response is not a valid json payload,
		// so we ignore it to avoid incorrect errors being reported
		return nil
	}
	return json.Unmarshal(rawJSON, destPtr)
}

type DeviceInfo struct {
	HWBomRev  string `json:"hw_bom_rev"`
	HWPcbRev  string `json:"hw_pcb_rev"`
	Serial    string `json:"serial"`
	FWVersion string `json:"fw_version"`
	ProdCode  string `json:"prod_code"`
	Type      string `json:"type"`
	Variant   string `json:"variant"`
	HWProdRev string `json:"hw_prod_rev"`
	HWID      string `json:"hw_id"`
}

func getDeviceInfo(conn *client) (DeviceInfo, error) {
	req := conn.newRequest("GET", "/device/info")
	res, err := conn.Client.Do(req)
	if err != nil {
		return DeviceInfo{}, err
	}
	var deviceInfo DeviceInfo
	err = handleResponse(res, &deviceInfo)
	return deviceInfo, err
}

type LiveLogicsResponse struct {
	Time   time.Time       `json:"time"`
	Logics []LiveLogicData `json:"logics"`
}

type LiveLogicResponse struct {
	Time  time.Time     `json:"time"`
	Logic LiveLogicData `json:"logic"`
}

type LiveLogicData struct {
	ID         int        `json:"id"`
	Name       string     `json:"name"`
	Info       string     `json:"info"`
	Geometries []Geometry `json:"geometries"`
	Counts     []Count    `json:"counts"`
}

type Geometry struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

type Count struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Value int    `json:"value"`
}

func doGet(conn *client, target any, endpoint string) error {
	req := conn.newRequest("GET", endpoint)
	res, err := conn.Client.Do(req)
	if err != nil {
		return err
	}
	err = handleResponse(res, &target)
	return err
}

func doPost(conn *client, target any, endpoint string, body any) error {
	req := conn.newRequest("POST", endpoint)
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
		bs, err := json.Marshal(body)
		if err != nil {
			return err
		}
		req.Body = io.NopCloser(bytes.NewReader(bs))
	}
	res, err := conn.Client.Do(req)
	if err != nil {
		return err
	}
	err = handleResponse(res, &target)
	return err
}

func getLiveLogics(ctx context.Context, conn *client, multiSensor bool, fc *healthpb.FaultCheck) (res LiveLogicsResponse, err error) {
	if multiSensor {
		err = doGet(conn, &res, "/multisensor/data/live/logics")
	} else {
		err = doGet(conn, &res, "/singlesensor/data/live/logics")
	}

	updateReliability(ctx, fc, err)
	return
}

func getLiveLogic(ctx context.Context, conn *client, multiSensor bool, id int, fc *healthpb.FaultCheck) (res LiveLogicResponse, err error) {

	if multiSensor {
		err = doGet(conn, &res, fmt.Sprintf("/multisensor/data/live/logics/%d", id))
	} else {
		err = doGet(conn, &res, fmt.Sprintf("/singlesensor/data/live/logics/%d", id))
	}

	updateReliability(ctx, fc, err)
	return
}

func resetLiveLogic(ctx context.Context, conn *client, multiSensor bool, id int, fc *healthpb.FaultCheck) error {
	var res []byte
	var err error
	if multiSensor {
		err = doPost(conn, &res, fmt.Sprintf("/multisensor/data/live/logics/%d/reset", id), nil)
	} else {
		err = doPost(conn, &res, fmt.Sprintf("/singlesensor/data/live/logics/%d/reset", id), nil)
	}

	updateReliability(ctx, fc, err)
	return err
}

func updateReliability(ctx context.Context, fc *healthpb.FaultCheck, err error) {
	if err != nil {
		h := noResponse
		var unsupportedTypeErr *json.UnmarshalTypeError
		if errors.Is(err, unsupportedTypeErr) {
			h = badResponse
		}
		fc.UpdateReliability(ctx, h)
	} else {
		fc.UpdateReliability(ctx, reliable)
	}
}
