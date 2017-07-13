package healthcheck

import (
	"net/http"

	"github.com/Sirupsen/logrus"

	"github.com/gorilla/mux"
)

var (
	router          = mux.NewRouter()
	healthcheckPort = ":9777"
)

func StartHealthcheck() {
	router.HandleFunc("/ping", healthCheck).Methods("GET", "HEAD").Name("Healthcheck")
	logrus.Info("Healthcheck handler is listening on ", healthcheckPort)
	logrus.Fatal(http.ListenAndServe(healthcheckPort, router))
}

func healthCheck(w http.ResponseWriter, req *http.Request) {
	// 1) test controller
	w.Write([]byte("pong"))
}
