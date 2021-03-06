package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/darkphnx/vehiclemanager/cmd/api"
	"github.com/darkphnx/vehiclemanager/cmd/background"
	"github.com/darkphnx/vehiclemanager/internal/authservice"
	"github.com/darkphnx/vehiclemanager/internal/models"
	"github.com/darkphnx/vehiclemanager/internal/mothistoryapi"
	"github.com/darkphnx/vehiclemanager/internal/vesapi"
)

func main() {
	vesapiKey := flag.String("vesapi-key", "", "Vehicle Enquiry Service API Key")
	mothistoryapiKey := flag.String("mothistoryapi-key", "", "MOT History API Key")
	jwtSigningSecret := flag.String("jwt-signing-secret", "", "JWT Signing Secret")
	flag.Parse()

	database, err := models.InitDB("mongodb://localhost:27017")
	if err != nil {
		log.Fatal(err)
	}

	vesapiClient := vesapi.NewClient(*vesapiKey, "")
	mothistoryClient := mothistoryapi.NewClient(*mothistoryapiKey, "")
	authService := authservice.NewAuthService(*jwtSigningSecret, 24, "mot.ninja")

	backgroundTasks := background.Task{
		Database:                 database,
		VehicleEnquiryServiceAPI: vesapiClient,
		MotHistoryAPI:            mothistoryClient,
	}
	go backgroundTasks.Begin()

	apiServer := api.Server{
		Database:                 database,
		VehicleEnquiryServiceAPI: vesapiClient,
		MotHistoryAPI:            mothistoryClient,
		AuthService:              authService,
	}

	mux := mux.NewRouter()

	mux.Use(api.LoggingMiddleware)

	mux.HandleFunc("/signup", apiServer.Signup).Methods("POST")
	mux.HandleFunc("/login", apiServer.Login).Methods("POST")
	mux.HandleFunc("/logout", apiServer.Logout).Methods("GET")

	authMux := mux.PathPrefix("").Subrouter()
	authMux.Use(apiServer.AuthJwtTokenMiddleware)
	authMux.HandleFunc("/vehicles/{registration}", apiServer.VehicleShow).Methods("GET")
	authMux.HandleFunc("/vehicles/{registration}", apiServer.VehicleDelete).Methods("DELETE")
	authMux.HandleFunc("/vehicles", apiServer.VehicleList).Methods("GET")
	authMux.HandleFunc("/vehicles", apiServer.VehicleCreate).Methods("POST")

	mux.Handle("/", http.FileServer(http.Dir("./ui/build")))

	err = http.ListenAndServe(":4000", mux)
	log.Fatal(err)
}
