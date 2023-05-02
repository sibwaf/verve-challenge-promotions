package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"verve-challenge-promotions/src"
	"verve-challenge-promotions/src/httpmiddleware"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "1321"
	}

	var routes map[string]http.HandlerFunc
	switch mode := os.Args[1]; mode {
	case "api":
		routes = src.GetApiRoutes()
	case "updater":
		routes = src.GetUpdaterRoutes()
	default:
		printUsage()
		os.Exit(1)
	}

	var err error
	src.DbClient, err = connectToDatabase()
	if err != nil {
		fmt.Printf("Failed to connect to database: %s\n", err)
		os.Exit(2)
	}

	registry := prometheus.NewRegistry()
	registry.MustRegister(
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		collectors.NewGoCollector(),
		collectors.NewDBStatsCollector(src.DbClient, os.Getenv("DB_NAME")),
	)

	middleware := httpmiddleware.New(registry, nil)

	mux := http.NewServeMux()
	mux.Handle("/prometheus", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	for route, handler := range routes {
		mux.Handle(route, middleware.WrapHandler(route, handler))
	}

	err = http.ListenAndServe(fmt.Sprintf(":%s", port), mux)
	if err != nil && err != http.ErrServerClosed {
		fmt.Printf("Failed to listen to connections: %s\n", err)
		os.Exit(3)
	}
}

func printUsage() {
	fmt.Printf("Usage: %s MODE\n", os.Args[0])
	fmt.Println("\t(arg) MODE - mode for the application to run in (api/updater)")
	fmt.Println("\t(env) SERVER_PORT - port for the server to listen on (ex. 1321)")
	fmt.Println("\t(env) DB_HOST - database hostname (ex. localhost)")
	fmt.Println("\t(env) DB_PORT - the port database is listening on (ex. 3306)")
	fmt.Println("\t(env) DB_NAME - the name of the database (ex. promotions)")
	fmt.Println("\t(env) DB_USERNAME - username for the database connection (ex. user)")
	fmt.Println("\t(env) DB_PASSWORD - password for the database connection (ex. qwerty)")
	fmt.Println("\t(env) DB_MAX_OPEN_CONNECTIONS - max amount of open database connections at the same time (ex. 64)")
}

func connectToDatabase() (db *sql.DB, err error) {
	db, err = sql.Open(
		"mysql",
		fmt.Sprintf(
			"%s:%s@(%s:%s)/%s?parseTime=true",
			os.Getenv("DB_USERNAME"),
			os.Getenv("DB_PASSWORD"),
			os.Getenv("DB_HOST"),
			os.Getenv("DB_PORT"),
			os.Getenv("DB_NAME"),
		),
	)
	if err == nil {
		err = db.Ping()
	}
	if err == nil {
		maxConnections, err := strconv.Atoi(os.Getenv("DB_MAX_OPEN_CONNECTIONS"))
		if err == nil {
			db.SetMaxOpenConns(maxConnections)
		}
	}

	return
}
