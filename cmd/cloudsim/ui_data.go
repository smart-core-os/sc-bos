package main

import (
	"fmt"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store"
	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

type baseViewData struct{}

func (b baseViewData) SitesPath() string             { return "/ui/sites" }
func (b baseViewData) NodesPath() string             { return "/ui/nodes" }
func (b baseViewData) ConfigVersionsPath() string    { return "/ui/config-versions" }
func (b baseViewData) ConfigDeploymentsPath() string { return "/ui/config-deployments" }

func (b baseViewData) NodeCheckInsPath(id int64) string {
	return fmt.Sprintf("/ui/nodes/%d/check-ins", id)
}
func (b baseViewData) DeleteSitePath(id int64) string {
	return fmt.Sprintf("/ui/sites/%d/delete", id)
}
func (b baseViewData) DeleteNodePath(id int64) string {
	return fmt.Sprintf("/ui/nodes/%d/delete", id)
}
func (b baseViewData) CreateEnrollmentCodePath(id int64) string {
	return fmt.Sprintf("/ui/nodes/%d/create-enrollment-code", id)
}
func (b baseViewData) DeleteConfigVersionPath(id int64) string {
	return fmt.Sprintf("/ui/config-versions/%d/delete", id)
}
func (b baseViewData) UpdateConfigDeploymentStatusPath(id int64) string {
	return fmt.Sprintf("/ui/config-deployments/%d/update-status", id)
}
func (b baseViewData) DeleteConfigDeploymentPath(id int64) string {
	return fmt.Sprintf("/ui/config-deployments/%d/delete", id)
}
func (b baseViewData) BinaryArtefactsPath() string   { return "/ui/binary-artefacts" }
func (b baseViewData) BinaryDeploymentsPath() string { return "/ui/binary-deployments" }
func (b baseViewData) DeleteBinaryArtefactPath(id int64) string {
	return fmt.Sprintf("/ui/binary-artefacts/%d/delete", id)
}
func (b baseViewData) UpdateBinaryDeploymentStatusPath(id int64) string {
	return fmt.Sprintf("/ui/binary-deployments/%d/update-status", id)
}
func (b baseViewData) DeleteBinaryDeploymentPath(id int64) string {
	return fmt.Sprintf("/ui/binary-deployments/%d/delete", id)
}

type indexViewData struct {
	baseViewData
}

type sitesViewData struct {
	baseViewData
	Sites         []queries.Site
	NextPageToken string
	Error         string
}

type nodesViewData struct {
	baseViewData
	Nodes         []queries.Node
	SiteID        int64
	NextPageToken string
	Error         string
}

type configVersionsViewData struct {
	baseViewData
	ConfigVersions []queries.ConfigVersion
	NodeID         int64
	NextPageToken  string
	Error          string
}

type configDeploymentsViewData struct {
	baseViewData
	ConfigDeployments []queries.ConfigDeployment
	NodeID            int64
	NextPageToken     string
	Error             string
}

type enrollmentCodeViewData struct {
	baseViewData
	Node queries.Node
	Code queries.EnrollmentCode
}

type checkInsViewData struct {
	baseViewData
	NodeID        int64
	Hostname      string
	CheckIns      []queries.NodeCheckIn
	NextPageToken string
}

type binaryArtefactsViewData struct {
	baseViewData
	BinaryArtefacts []store.BinaryArtefact
	SiteID          int64
	OS              string
	Arch            string
	// DefaultArch prepopulates the upload form's arch with the one cloudsim runs on (the OS default
	// is always linux, hardcoded in the template).
	DefaultArch   string
	NextPageToken string
	Error         string
}

type binaryDeploymentsViewData struct {
	baseViewData
	BinaryDeployments []queries.BinaryDeployment
	NodeID            int64
	NextPageToken     string
	Error             string
}
