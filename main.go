package main

import (
	"embed"
	"log"
	"net/http"

	"surveillance-client/internal/camera"
	"surveillance-client/internal/config"
	"surveillance-client/internal/db"
	"surveillance-client/internal/discovery"
	"surveillance-client/internal/dvr"
	"surveillance-client/internal/export"
	"surveillance-client/internal/face"
	"surveillance-client/internal/go2rtc"
	"surveillance-client/internal/server"
)

//go:embed web/dist/*
var webAssets embed.FS

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatal("failed to open database:", err)
	}
	defer database.Close()

	if err := db.Migrate(database); err != nil {
		log.Fatal("failed to run migrations:", err)
	}

	cameraStore := camera.NewStore(database)
	go2rtcClient := go2rtc.NewClient(cfg.Go2RTCAPI)
	go2rtcMgr := go2rtc.NewManager(cfg, cameraStore, go2rtcClient)

	cameraHandler := camera.NewHandler(cameraStore, go2rtcMgr)
	groupHandler := camera.NewGroupHandler(cameraStore)
	discoveryHandler := discovery.NewHandler(cfg, cameraStore, go2rtcMgr)
	dvrProxyHandler := dvr.NewProxyHandler(cfg, cameraStore)
	exportHandler := export.NewHandler()

	// Face recognition
	faceStore := face.NewStore(database)
	faceAlerter := face.NewAlerter(cfg.SlackWebhookURL, faceStore)
	faceSnapshot := face.NewSnapshotFetcher(cfg.Go2RTCAPI)
	faceCameras := &cameraStoreAdapter{store: cameraStore}
	faceHandler := face.NewHandler(faceStore, faceAlerter, faceSnapshot, faceCameras, cfg.FaceServiceURL, cfg.FaceDataDir)

	deps := &server.Dependencies{
		CameraHandler:    cameraHandler,
		GroupHandler:     groupHandler,
		DiscoveryHandler: discoveryHandler,
		DvrProxyHandler:  dvrProxyHandler,
		ExportHandler:    exportHandler,
		FaceHandler:      faceHandler,
	}

	srv := server.New(cfg, webAssets, deps)

	if err := go2rtcMgr.Start(); err != nil {
		log.Printf("warning: failed to start go2rtc: %v", err)
	}
	defer go2rtcMgr.Stop()

	if err := go2rtcMgr.SyncStreams(); err != nil {
		log.Printf("warning: failed to sync streams: %v", err)
	}

	log.Printf("starting server on :%s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, srv); err != nil {
		log.Fatal(err)
	}
}

// cameraStoreAdapter adapts camera.Store to face.CameraLister
type cameraStoreAdapter struct {
	store *camera.Store
}

func (a *cameraStoreAdapter) GetCamera(id string) (face.CameraInfo, bool) {
	cam, err := a.store.GetCamera(id)
	if err != nil || cam == nil {
		return face.CameraInfo{}, false
	}
	return face.CameraInfo{
		RTSPMain: cam.RTSPMain,
		RTSPSub:  cam.RTSPSub,
		Username: cam.Username,
		Password: cam.Password,
		Name:     cam.Name,
	}, true
}
