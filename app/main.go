package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.25.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/gorilla/mux"
)

func instrument() error {
	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("demo-app"),
	))
	if err != nil {
		return err
	}

	client, err := grpc.NewClient("localhost:4317", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	{
		exp, err := otlpmetricgrpc.New(context.Background(), otlpmetricgrpc.WithGRPCConn(client))
		if err != nil {
			return err
		}

		mp := sdkmetric.NewMeterProvider(
			sdkmetric.WithResource(res),
			sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exp, sdkmetric.WithInterval(10*time.Second))),
		)

		otel.SetMeterProvider(mp)
	}

	{
		exp, err := otlptracegrpc.New(context.Background(), otlptracegrpc.WithGRPCConn(client))
		if err != nil {
			return err
		}

		tp := sdktrace.NewTracerProvider(
			sdktrace.WithResource(res),
			sdktrace.WithBatcher(exp, sdktrace.WithBatchTimeout(10*time.Second)),
			sdktrace.WithSampler(sdktrace.AlwaysSample()),
		)

		otel.SetTracerProvider(tp)
	}

	return nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})))

	err := instrument()
	if err != nil {
		slog.Error("could not create instruments", slog.String("err", err.Error()))
		os.Exit(1)
	}

	router := mux.NewRouter()

	router.Use(
		otelhttp.NewMiddleware(
			"server",
			otelhttp.WithMeterProvider(otel.GetMeterProvider()),
			otelhttp.WithTracerProvider(otel.GetTracerProvider()),
		),
	)

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "healthy")
	})

	srv := &http.Server{
		Addr:    "localhost:8080",
		Handler: router,
	}

	l, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		slog.Error("could not listen", slog.String("err", err.Error()))
		os.Exit(1)
	}

	slog.Info("listening on address: " + srv.Addr)
	log.Fatal(srv.Serve(l))
}
