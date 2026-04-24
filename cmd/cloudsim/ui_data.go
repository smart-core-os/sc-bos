package main

import (
	"fmt"

	"github.com/smart-core-os/sc-bos/internal/cloud/sim/store/store/queries"
)

type baseViewData struct{}

func (b baseViewData) SitesPath() string          { return "/ui/sites" }
func (b baseViewData) NodesPath() string          { return "/ui/nodes" }
func (b baseViewData) ConfigVersionsPath() string { return "/ui/config-versions" }
func (b baseViewData) DeploymentsPath() string    { return "/ui/deployments" }

func (b baseViewData) NodeCheckInsPath(id int64) string {
	return fmt.Sprintf("/ui/nodes/%d/check-ins", id)
}
func (b baseViewData) DeleteSitePath(id int64) string {
	return fmt.Sprintf("/ui/sites/%d/delete", id)
}
func (b baseViewData) DeleteNodePath(id int64) string {
	return fmt.Sprintf("/ui/nodes/%d/delete", id)
}
func (b baseViewData) RotateSecretPath(id int64) string {
	return fmt.Sprintf("/ui/nodes/%d/rotate-secret", id)
}
func (b baseViewData) CreateEnrollmentCodePath(id int64) string {
	return fmt.Sprintf("/ui/nodes/%d/create-enrollment-code", id)
}
func (b baseViewData) DeleteConfigVersionPath(id int64) string {
	return fmt.Sprintf("/ui/config-versions/%d/delete", id)
}
func (b baseViewData) UpdateDeploymentStatusPath(id int64) string {
	return fmt.Sprintf("/ui/deployments/%d/update-status", id)
}
func (b baseViewData) DeleteDeploymentPath(id int64) string {
	return fmt.Sprintf("/ui/deployments/%d/delete", id)
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

type nodeSecretViewData struct {
	baseViewData
	Node   queries.Node
	Secret string // base64-encoded raw bytes, shown once
}

type configVersionsViewData struct {
	baseViewData
	ConfigVersions []queries.ConfigVersion
	NodeID         int64
	NextPageToken  string
	Error          string
}

type deploymentsViewData struct {
	baseViewData
	Deployments   []queries.Deployment
	NodeID        int64
	NextPageToken string
	Error         string
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
