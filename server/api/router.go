// Copyright 2016 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"net/http"
	"net/http/pprof"

	"github.com/gorilla/mux"
	"github.com/pingcap/pd/v4/server"
	"github.com/unrolled/render"
)

func createStreamingRender() *render.Render {
	return render.New(render.Options{
		StreamingJSON: true,
	})
}

func createIndentRender() *render.Render {
	return render.New(render.Options{
		IndentJSON: true,
	})
}

// The returned function is used as a lazy router to avoid the data race problem.
// @title Placement Driver Core API
// @version 1.0
// @description This is placement driver.
// @contact.name Placement Driver Support
// @contact.url https://github.com/pingcap/pd/issues
// @contact.email info@pingcap.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /pd/api/v1
func createRouter(ctx context.Context, prefix string, svr *server.Server) *mux.Router {
	rd := createIndentRender()

	rootRouter := mux.NewRouter().PathPrefix(prefix).Subrouter()
	handler := svr.GetHandler()

	apiRouter := rootRouter.PathPrefix("/api/v1").Subrouter()

	clusterRouter := apiRouter.NewRoute().Subrouter()
	clusterRouter.Use(newClusterMiddleware(svr).Middleware)

	operatorHandler := newOperatorHandler(handler, rd)
	apiRouter.HandleFunc("/operators", operatorHandler.List).Methods("GET")
	apiRouter.HandleFunc("/operators", operatorHandler.Post).Methods("POST")
	apiRouter.HandleFunc("/operators/{region_id}", operatorHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/operators/{region_id}", operatorHandler.Delete).Methods("DELETE")

	schedulerHandler := newSchedulerHandler(svr, rd)
	apiRouter.HandleFunc("/schedulers", schedulerHandler.List).Methods("GET")
	apiRouter.HandleFunc("/schedulers", schedulerHandler.Post).Methods("POST")
	apiRouter.HandleFunc("/schedulers/{name}", schedulerHandler.Delete).Methods("DELETE")
	apiRouter.HandleFunc("/schedulers/{name}", schedulerHandler.PauseOrResume).Methods("POST")
	schedulerConfigHandler := newSchedulerConfigHandler(svr, rd)
	rootRouter.PathPrefix(server.SchedulerConfigHandlerPath).Handler(schedulerConfigHandler)

	clusterHandler := newClusterHandler(svr, rd)
	apiRouter.Handle("/cluster", clusterHandler).Methods("GET")
	apiRouter.HandleFunc("/cluster/status", clusterHandler.GetClusterStatus).Methods("GET")

	confHandler := newConfHandler(svr, rd)
	apiRouter.HandleFunc("/config", confHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/config", confHandler.Post).Methods("POST")
	apiRouter.HandleFunc("/config/default", confHandler.GetDefault).Methods("GET")
	apiRouter.HandleFunc("/config/schedule", confHandler.GetSchedule).Methods("GET")
	apiRouter.HandleFunc("/config/schedule", confHandler.SetSchedule).Methods("POST")
	apiRouter.HandleFunc("/config/replicate", confHandler.GetReplication).Methods("GET")
	apiRouter.HandleFunc("/config/replicate", confHandler.SetReplication).Methods("POST")
	apiRouter.HandleFunc("/config/label-property", confHandler.GetLabelProperty).Methods("GET")
	apiRouter.HandleFunc("/config/label-property", confHandler.SetLabelProperty).Methods("POST")
	apiRouter.HandleFunc("/config/cluster-version", confHandler.GetClusterVersion).Methods("GET")
	apiRouter.HandleFunc("/config/cluster-version", confHandler.SetClusterVersion).Methods("POST")
	apiRouter.HandleFunc("/config/replication-mode", confHandler.GetReplicationMode).Methods("GET")
	apiRouter.HandleFunc("/config/replication-mode", confHandler.SetReplicationMode).Methods("POST")

	rulesHandler := newRulesHandler(svr, rd)
	clusterRouter.HandleFunc("/config/rules", rulesHandler.GetAll).Methods("GET")
	clusterRouter.HandleFunc("/config/rules/group/{group}", rulesHandler.GetAllByGroup).Methods("GET")
	clusterRouter.HandleFunc("/config/rules/region/{region}", rulesHandler.GetAllByRegion).Methods("GET")
	clusterRouter.HandleFunc("/config/rules/key/{key}", rulesHandler.GetAllByKey).Methods("GET")
	clusterRouter.HandleFunc("/config/rule/{group}/{id}", rulesHandler.Get).Methods("GET")
	clusterRouter.HandleFunc("/config/rule", rulesHandler.Set).Methods("POST")
	clusterRouter.HandleFunc("/config/rule/{group}/{id}", rulesHandler.Delete).Methods("DELETE")

	storeHandler := newStoreHandler(handler, rd)
	clusterRouter.HandleFunc("/store/{id}", storeHandler.Get).Methods("GET")
	clusterRouter.HandleFunc("/store/{id}", storeHandler.Delete).Methods("DELETE")
	clusterRouter.HandleFunc("/store/{id}/state", storeHandler.SetState).Methods("POST")
	clusterRouter.HandleFunc("/store/{id}/label", storeHandler.SetLabels).Methods("POST")
	clusterRouter.HandleFunc("/store/{id}/weight", storeHandler.SetWeight).Methods("POST")
	clusterRouter.HandleFunc("/store/{id}/limit", storeHandler.SetLimit).Methods("POST")
	storesHandler := newStoresHandler(handler, rd)
	clusterRouter.Handle("/stores", storesHandler).Methods("GET")
	clusterRouter.HandleFunc("/stores/remove-tombstone", storesHandler.RemoveTombStone).Methods("DELETE")
	clusterRouter.HandleFunc("/stores/limit", storesHandler.GetAllLimit).Methods("GET")
	clusterRouter.HandleFunc("/stores/limit", storesHandler.SetAllLimit).Methods("POST")
	clusterRouter.HandleFunc("/stores/limit/scene", storesHandler.SetStoreLimitScene).Methods("POST")
	clusterRouter.HandleFunc("/stores/limit/scene", storesHandler.GetStoreLimitScene).Methods("GET")

	labelsHandler := newLabelsHandler(svr, rd)
	clusterRouter.HandleFunc("/labels", labelsHandler.Get).Methods("GET")
	clusterRouter.HandleFunc("/labels/stores", labelsHandler.GetStores).Methods("GET")

	hotStatusHandler := newHotStatusHandler(handler, rd)
	apiRouter.HandleFunc("/hotspot/regions/write", hotStatusHandler.GetHotWriteRegions).Methods("GET")
	apiRouter.HandleFunc("/hotspot/regions/read", hotStatusHandler.GetHotReadRegions).Methods("GET")
	apiRouter.HandleFunc("/hotspot/stores", hotStatusHandler.GetHotStores).Methods("GET")

	regionHandler := newRegionHandler(svr, rd)
	clusterRouter.HandleFunc("/region/id/{id}", regionHandler.GetRegionByID).Methods("GET")
	clusterRouter.UseEncodedPath().HandleFunc("/region/key/{key}", regionHandler.GetRegionByKey).Methods("GET")

	srd := createStreamingRender()
	regionsAllHandler := newRegionsHandler(svr, srd)
	clusterRouter.HandleFunc("/regions", regionsAllHandler.GetAll).Methods("GET")

	regionsHandler := newRegionsHandler(svr, rd)
	clusterRouter.HandleFunc("/regions/key", regionsHandler.ScanRegions).Methods("GET")
	clusterRouter.HandleFunc("/regions/count", regionsHandler.GetRegionCount).Methods("GET")
	clusterRouter.HandleFunc("/regions/store/{id}", regionsHandler.GetStoreRegions).Methods("GET")
	clusterRouter.HandleFunc("/regions/writeflow", regionsHandler.GetTopWriteFlow).Methods("GET")
	clusterRouter.HandleFunc("/regions/readflow", regionsHandler.GetTopReadFlow).Methods("GET")
	clusterRouter.HandleFunc("/regions/confver", regionsHandler.GetTopConfVer).Methods("GET")
	clusterRouter.HandleFunc("/regions/version", regionsHandler.GetTopVersion).Methods("GET")
	clusterRouter.HandleFunc("/regions/size", regionsHandler.GetTopSize).Methods("GET")
	clusterRouter.HandleFunc("/regions/check/miss-peer", regionsHandler.GetMissPeerRegions).Methods("GET")
	clusterRouter.HandleFunc("/regions/check/extra-peer", regionsHandler.GetExtraPeerRegions).Methods("GET")
	clusterRouter.HandleFunc("/regions/check/pending-peer", regionsHandler.GetPendingPeerRegions).Methods("GET")
	clusterRouter.HandleFunc("/regions/check/down-peer", regionsHandler.GetDownPeerRegions).Methods("GET")
	clusterRouter.HandleFunc("/regions/check/offline-peer", regionsHandler.GetOfflinePeer).Methods("GET")
	clusterRouter.HandleFunc("/regions/check/empty-region", regionsHandler.GetEmptyRegion).Methods("GET")
	clusterRouter.HandleFunc("/regions/check/hist-size", regionsHandler.GetSizeHistogram).Methods("GET")
	clusterRouter.HandleFunc("/regions/check/hist-keys", regionsHandler.GetKeysHistogram).Methods("GET")
	clusterRouter.HandleFunc("/regions/sibling/{id}", regionsHandler.GetRegionSiblings).Methods("GET")

	apiRouter.Handle("/version", newVersionHandler(rd)).Methods("GET")
	apiRouter.Handle("/status", newStatusHandler(svr, rd)).Methods("GET")

	memberHandler := newMemberHandler(svr, rd)
	apiRouter.HandleFunc("/members", memberHandler.ListMembers).Methods("GET")
	apiRouter.HandleFunc("/members/name/{name}", memberHandler.DeleteByName).Methods("DELETE")
	apiRouter.HandleFunc("/members/id/{id}", memberHandler.DeleteByID).Methods("DELETE")
	apiRouter.HandleFunc("/members/name/{name}", memberHandler.SetMemberPropertyByName).Methods("POST")

	leaderHandler := newLeaderHandler(svr, rd)
	apiRouter.HandleFunc("/leader", leaderHandler.Get).Methods("GET")
	apiRouter.HandleFunc("/leader/resign", leaderHandler.Resign).Methods("POST")
	apiRouter.HandleFunc("/leader/transfer/{next_leader}", leaderHandler.Transfer).Methods("POST")

	statsHandler := newStatsHandler(svr, rd)
	clusterRouter.HandleFunc("/stats/region", statsHandler.Region).Methods("GET")

	trendHandler := newTrendHandler(svr, rd)
	apiRouter.HandleFunc("/trend", trendHandler.Handle).Methods("GET")

	adminHandler := newAdminHandler(svr, rd)
	clusterRouter.HandleFunc("/admin/cache/region/{id}", adminHandler.HandleDropCacheRegion).Methods("DELETE")
	clusterRouter.HandleFunc("/admin/reset-ts", adminHandler.ResetTS).Methods("POST")
	apiRouter.HandleFunc("/admin/persist-file/{file_name}", adminHandler.persistFile).Methods("POST")

	logHandler := newlogHandler(svr, rd)
	apiRouter.HandleFunc("/admin/log", logHandler.Handle).Methods("POST")

	replicationModeHandler := newReplicationModeHandler(svr, rd)
	clusterRouter.HandleFunc("/replication_mode/status", replicationModeHandler.GetStatus)

	componentHandler := newComponentHandler(svr, rd)
	clusterRouter.HandleFunc("/component", componentHandler.Register).Methods("POST")
	clusterRouter.HandleFunc("/component/{component}/{addr}", componentHandler.UnRegister).Methods("DELETE")
	clusterRouter.HandleFunc("/component", componentHandler.GetAllAddress).Methods("GET")
	clusterRouter.HandleFunc("/component/{type}", componentHandler.GetAddress).Methods("GET")

	pluginHandler := newPluginHandler(handler, rd)
	apiRouter.HandleFunc("/plugin", pluginHandler.LoadPlugin).Methods("POST")
	apiRouter.HandleFunc("/plugin", pluginHandler.UnloadPlugin).Methods("DELETE")

	apiRouter.Handle("/health", newHealthHandler(svr, rd)).Methods("GET")
	apiRouter.Handle("/diagnose", newDiagnoseHandler(svr, rd)).Methods("GET")
	apiRouter.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {}).Methods("GET")
	// metric query use to query metric data, the protocol is compatible with prometheus.
	apiRouter.Handle("/metric/query", newQueryMetric(svr)).Methods("GET", "POST")
	apiRouter.Handle("/metric/query_range", newQueryMetric(svr)).Methods("GET", "POST")

	// profile API
	apiRouter.HandleFunc("/debug/pprof/profile", pprof.Profile)
	apiRouter.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	apiRouter.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	apiRouter.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	apiRouter.Handle("/debug/pprof/block", pprof.Handler("block"))
	apiRouter.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))

	// Deprecated
	rootRouter.Handle("/health", newHealthHandler(svr, rd)).Methods("GET")
	// Deprecated
	rootRouter.Handle("/diagnose", newDiagnoseHandler(svr, rd)).Methods("GET")
	// Deprecated
	rootRouter.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {}).Methods("GET")

	return rootRouter
}
