package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
	"github.com/martinezmoises/Test1/internal/data"
)

const appVersion = "1.0.0"

type serverConfig struct {
	port        int
	environment string
	db          struct {
		dsn string
	}

	limiter struct {
        rps float64                      // requests per second
        burst int                        // initial requests possible
        enabled bool                     // enable or disable rate limiter
    }

}

type applicationDependencies struct {
	config       serverConfig
	logger       *slog.Logger
	productModel data.ProductModel
	reviewModel  data.ReviewModel
}

func main() {

	var settings serverConfig
	flag.IntVar(&settings.port, "port", 4000, "server port")
	flag.StringVar(&settings.environment, "env", "development", "Environment?(development|staging|production)")
	flag.StringVar(&settings.db.dsn, "db-dsn", "postgres://products:fishsticks@localhost/products", "PostgresSQL DSN")
	//read in the dsn
	flag.Float64Var(&settings.limiter.rps, "limiter-rps", 2,
	"Rate Limiter maximum requests per second")

	flag.IntVar(&settings.limiter.burst, "limiter-burst", 5,
	"Rate Limiter maximum burst")

	flag.BoolVar(&settings.limiter.enabled, "limiter-enabled", true,
	"Enable rate limiter")

	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB(settings)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connection pool established")

	appInstance := &applicationDependencies{
		config:       settings,
		logger:       logger,
		productModel: data.ProductModel{DB: db},
		reviewModel:  data.ReviewModel{DB: db},
	}

	err = appInstance.serve()
    if err != nil {
        logger.Error(err.Error())
        os.Exit(1)
    }

}

func openDB(settings serverConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", settings.db.dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
