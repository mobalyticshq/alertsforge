package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/mobalyticshq/alertsforge/alertsource"
	"github.com/mobalyticshq/alertsforge/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.Level.SetLevel(zapcore.InfoLevel)

	logger, err := loggerConfig.Build()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)
	log := logger.Sugar()

	if err != nil {
		log.Fatal(err)
	}

	configLoader := config.Config{}
	runbooks, err := configLoader.LoadRunbooksConfig("./config/runbooks.yaml")
	if err != nil {
		log.Fatalf("error during structure loading: %v", err)
	}
	am := alertsource.NewAlertManager(runbooks)
	http.HandleFunc("/healthz", healthz)
	http.HandleFunc("/alertWebhook/api/v2/alerts", am.AlertWebhook)
	http.HandleFunc("/processAlertBuffer", am.ProcessAlertsBufferWebhook)
	http.HandleFunc("/showAlertBuffer", am.ShowAlertsBufferWebhook)

	go am.AlertsProcessor()
	listenAddress := ":8080"
	if os.Getenv("PORT") != "" {
		listenAddress = ":" + os.Getenv("PORT")
	}

	log.Info("listening on: ", listenAddress)
	log.Fatal(http.ListenAndServe(listenAddress, nil))
}

func healthz(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Ok!")
}
