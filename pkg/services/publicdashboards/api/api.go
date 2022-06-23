package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana/pkg/api/dtos"
	"github.com/grafana/grafana/pkg/api/response"
	"github.com/grafana/grafana/pkg/api/routing"
	"github.com/grafana/grafana/pkg/middleware"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/plugins/backendplugin"
	"github.com/grafana/grafana/pkg/services/accesscontrol"
	"github.com/grafana/grafana/pkg/services/dashboards"
	"github.com/grafana/grafana/pkg/services/datasources"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/services/publicdashboards"
	. "github.com/grafana/grafana/pkg/services/publicdashboards/models"
	"github.com/grafana/grafana/pkg/services/query"
	"github.com/grafana/grafana/pkg/util"
	"github.com/grafana/grafana/pkg/web"
)

type PublicDashboardAPI struct {
	PublicDashboardService publicdashboards.Service
	RouteRegister          routing.RouteRegister
	AccessControl          accesscontrol.AccessControl
	QueryDataService       *query.Service
	Features               *featuremgmt.FeatureManager
}

// Factory method for building api
func NewPublicDashboardApi(
	pd publicdashboards.Service,
	rr routing.RouteRegister,
	ac accesscontrol.AccessControl,
	qds *query.Service,
	features *featuremgmt.FeatureManager,
) *PublicDashboardAPI {
	return &PublicDashboardAPI{
		PublicDashboardService: pd,
		RouteRegister:          rr,
		AccessControl:          ac,
		QueryDataService:       qds,
		Features:               features,
	}
}

func (api *PublicDashboardAPI) RegisterAPIEndpoints() {
	// Don't mount routes if feature is not enabled
	if !api.Features.IsEnabled(featuremgmt.FlagPublicDashboards) {
		return
	}

	auth := accesscontrol.Middleware(api.AccessControl)
	reqSignedIn := middleware.ReqSignedIn

	// Anonymous access to public dashboard route is configured in pkg/api/api.go
	// because it is deeply dependent on the HTTPServer.Index() method and would result in a
	// circular dependency

	api.RouteRegister.Get("/api/public/dashboards/:accessToken", routing.Wrap(api.GetPublicDashboard))
	api.RouteRegister.Post("/api/public/dashboards/:accessToken/panels/:panelId/query", routing.Wrap(api.QueryPublicDashboard))

	// Create/Update Public Dashboard
	api.RouteRegister.Get("/dashboards/uid/:uid/public-config", auth(reqSignedIn, accesscontrol.EvalPermission(dashboards.ActionDashboardsWrite)), routing.Wrap(api.GetPublicDashboardConfig))
	api.RouteRegister.Post("/dashboards/uid/:uid/public-config", auth(reqSignedIn, accesscontrol.EvalPermission(dashboards.ActionDashboardsWrite)), routing.Wrap(api.SavePublicDashboardConfig))
}

// gets public dashboard
func (api *PublicDashboardAPI) GetPublicDashboard(c *models.ReqContext) response.Response {
	accessToken := web.Params(c.Req)[":accessToken"]

	dash, err := api.PublicDashboardService.GetPublicDashboard(c.Req.Context(), accessToken)
	if err != nil {
		return handleDashboardErr(http.StatusInternalServerError, "Failed to get public dashboard", err)
	}

	meta := dtos.DashboardMeta{
		Slug:                       dash.Slug,
		Type:                       models.DashTypeDB,
		CanStar:                    false,
		CanSave:                    false,
		CanEdit:                    false,
		CanAdmin:                   false,
		CanDelete:                  false,
		Created:                    dash.Created,
		Updated:                    dash.Updated,
		Version:                    dash.Version,
		IsFolder:                   false,
		FolderId:                   dash.FolderId,
		PublicDashboardAccessToken: accessToken,
	}

	dto := dtos.DashboardFullWithMeta{Meta: meta, Dashboard: dash.Data}

	return response.JSON(http.StatusOK, dto)
}

// gets public dashboard configuration for dashboard
func (api *PublicDashboardAPI) GetPublicDashboardConfig(c *models.ReqContext) response.Response {
	pdc, err := api.PublicDashboardService.GetPublicDashboardConfig(c.Req.Context(), c.OrgId, web.Params(c.Req)[":uid"])
	if err != nil {
		return handleDashboardErr(http.StatusInternalServerError, "Failed to get public dashboard config", err)
	}
	return response.JSON(http.StatusOK, pdc)
}

// sets public dashboard configuration for dashboard
func (api *PublicDashboardAPI) SavePublicDashboardConfig(c *models.ReqContext) response.Response {
	pubdash := &PublicDashboard{}
	if err := web.Bind(c.Req, pubdash); err != nil {
		return response.Error(http.StatusBadRequest, "bad request data", err)
	}

	// Always set the org id to the current auth session orgId
	pubdash.OrgId = c.OrgId

	dto := SavePublicDashboardConfigDTO{
		OrgId:           c.OrgId,
		DashboardUid:    web.Params(c.Req)[":uid"],
		UserId:          c.UserId,
		PublicDashboard: pubdash,
	}

	pubdash, err := api.PublicDashboardService.SavePublicDashboardConfig(c.Req.Context(), &dto)
	if err != nil {
		return handleDashboardErr(http.StatusInternalServerError, "Failed to save public dashboard configuration", err)
	}

	return response.JSON(http.StatusOK, pubdash)
}

// QueryPublicDashboard returns all results for a given panel on a public dashboard
// POST /api/public/dashboard/:accessToken/panels/:panelId/query
func (api *PublicDashboardAPI) QueryPublicDashboard(c *models.ReqContext) response.Response {
	panelId, err := strconv.ParseInt(web.Params(c.Req)[":panelId"], 10, 64)
	if err != nil {
		return response.Error(http.StatusBadRequest, "invalid panel ID", err)
	}

	dashboard, err := api.PublicDashboardService.GetPublicDashboard(c.Req.Context(), web.Params(c.Req)[":accessToken"])
	if err != nil {
		return response.Error(http.StatusInternalServerError, "could not fetch dashboard", err)
	}

	publicDashboard, err := api.PublicDashboardService.GetPublicDashboardConfig(c.Req.Context(), dashboard.OrgId, dashboard.Uid)
	if err != nil {
		return response.Error(http.StatusInternalServerError, "could not fetch public dashboard", err)
	}

	reqDTO, err := api.PublicDashboardService.BuildPublicDashboardMetricRequest(
		c.Req.Context(),
		dashboard,
		publicDashboard,
		panelId,
	)
	if err != nil {
		return handleDashboardErr(http.StatusInternalServerError, "Failed to get queries for public dashboard", err)
	}

	// Get all needed datasource UIDs from queries
	var uids []string
	for _, query := range reqDTO.Queries {
		uids = append(uids, query.Get("datasource").Get("uid").MustString())
	}

	// Create a temp user with read-only datasource permissions
	anonymousUser := &models.SignedInUser{OrgId: dashboard.OrgId, Permissions: make(map[int64]map[string][]string)}
	permissions := make(map[string][]string)
	datasourceScope := fmt.Sprintf("datasources:uid:%s", strings.Join(uids, ","))
	permissions[datasources.ActionQuery] = []string{datasourceScope}
	permissions[datasources.ActionRead] = []string{datasourceScope}
	anonymousUser.Permissions[dashboard.OrgId] = permissions

	resp, err := api.QueryDataService.QueryDataMultipleSources(c.Req.Context(), anonymousUser, c.SkipCache, reqDTO, true)

	if err != nil {
		return handleQueryMetricsError(err)
	}
	return toJsonStreamingResponse(api.Features, resp)
}

// util to help us unpack a dashboard err or use default http code and message
func handleDashboardErr(defaultCode int, defaultMsg string, err error) response.Response {
	var dashboardErr models.DashboardErr

	if ok := errors.As(err, &dashboardErr); ok {
		return response.Error(dashboardErr.StatusCode, dashboardErr.Error(), dashboardErr)
	}

	return response.Error(defaultCode, defaultMsg, err)
}

// Copied from pkg/api/metrics.go
func handleQueryMetricsError(err error) *response.NormalResponse {
	if errors.Is(err, models.ErrDataSourceAccessDenied) {
		return response.Error(http.StatusForbidden, "Access denied to data source", err)
	}
	if errors.Is(err, models.ErrDataSourceNotFound) {
		return response.Error(http.StatusNotFound, "Data source not found", err)
	}
	var badQuery *query.ErrBadQuery
	if errors.As(err, &badQuery) {
		return response.Error(http.StatusBadRequest, util.Capitalize(badQuery.Message), err)
	}

	if errors.Is(err, backendplugin.ErrPluginNotRegistered) {
		return response.Error(http.StatusNotFound, "Plugin not found", err)
	}

	return response.Error(http.StatusInternalServerError, "Query data error", err)
}

// Copied from pkg/api/metrics.go
func toJsonStreamingResponse(features *featuremgmt.FeatureManager, qdr *backend.QueryDataResponse) response.Response {
	statusWhenError := http.StatusBadRequest
	if features.IsEnabled(featuremgmt.FlagDatasourceQueryMultiStatus) {
		statusWhenError = http.StatusMultiStatus
	}

	statusCode := http.StatusOK
	for _, res := range qdr.Responses {
		if res.Error != nil {
			statusCode = statusWhenError
		}
	}

	return response.JSONStreaming(statusCode, qdr)
}
