package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"grpcImage/internal/config"
	"grpcImage/internal/limiter"
	"grpcImage/internal/service"
	"grpcImage/pkg/api"

	"google.golang.org/grpc"
)

func main() {
	cfg := config.NewDefaultConfig()

	flag.StringVar(&cfg.ServerAddress, "addr", cfg.ServerAddress, "Server address")
	flag.StringVar(&cfg.StorageDir, "storage", cfg.StorageDir, "Directory for storing files")
	flag.IntVar(&cfg.UploadDownloadLimit, "upload-limit", cfg.UploadDownloadLimit, "Limit of concurrent requests for upload/download")
	flag.IntVar(&cfg.ListLimit, "list-limit", cfg.ListLimit, "Limit of concurrent requests for list")
	flag.Parse()

	fileService, err := service.NewFileService(cfg.StorageDir)
	if err != nil {
		log.Fatalf("Failed to create service: %v", err)
	}

	concurrencyLimiter := limiter.NewConcurrencyLimiter(cfg.UploadDownloadLimit, cfg.ListLimit)

	server := grpc.NewServer(
		grpc.UnaryInterceptor(concurrencyLimiter.UnaryInterceptor),
		grpc.StreamInterceptor(concurrencyLimiter.StreamInterceptor),
	)

	api.RegisterFileServiceServer(server, fileService)

	listener, err := net.Listen("tcp", cfg.ServerAddress)
	if err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Printf("Server started on %s", cfg.ServerAddress)
	log.Printf("Files are stored in %s", cfg.StorageDir)
	log.Printf("Upload/download limit: %d, list limit: %d",
		cfg.UploadDownloadLimit, cfg.ListLimit)

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
		<-sigCh
		log.Println("Received termination signal, stopping server...")
		server.GracefulStop()
	}()

	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
