package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"grpcImage/pkg/api"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	serverAddr := flag.String("server", "localhost:50051", "gRPC server address")
	uploadFile := flag.String("upload", "", "Path to file for upload")
	downloadFile := flag.String("download", "", "File name for download")
	savePath := flag.String("save", "", "Path to save downloaded file")
	listFiles := flag.Bool("list", false, "Get list of files")
	flag.Parse()

	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := api.NewFileServiceClient(conn)

	if *uploadFile != "" {
		if _, err := os.Stat(*uploadFile); os.IsNotExist(err) {
			log.Fatalf("File not found: %s", *uploadFile)
		}

		filename := filepath.Base(*uploadFile)
		uploadFileToServer(client, *uploadFile, filename)
	} else if *downloadFile != "" {
		if *savePath == "" {
			*savePath = *downloadFile
		}
		downloadFileFromServer(client, *downloadFile, *savePath)
	} else if *listFiles {
		listFilesOnServer(client)
	} else {
		fmt.Println("Usage:")
		fmt.Println("  -server=address:port  gRPC server address (default: localhost:50051)")
		fmt.Println("  -upload=path         Upload file to server")
		fmt.Println("  -download=name       Download file from server")
		fmt.Println("  -save=path          Path to save downloaded file")
		fmt.Println("  -list               Get list of files on server")
		fmt.Println("\nExamples:")
		fmt.Println("  Upload file:       client -upload=./image.jpg")
		fmt.Println("  Download file:     client -download=image.jpg -save=./downloaded.jpg")
		fmt.Println("  List files:       client -list")
	}
}

func uploadFileToServer(client api.FileServiceClient, filePath, fileName string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := client.UploadFile(ctx)
	if err != nil {
		log.Fatalf("Failed to create stream: %v", err)
	}

	err = stream.Send(&api.UploadFileRequest{
		Data: &api.UploadFileRequest_Info{
			Info: &api.FileInfo{
				Filename: fileName,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to send file info: %v", err)
	}

	buffer := make([]byte, 64*1024) // 64KB buffer
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Failed to read file: %v", err)
		}

		err = stream.Send(&api.UploadFileRequest{
			Data: &api.UploadFileRequest_Chunk{
				Chunk: buffer[:n],
			},
		})
		if err != nil {
			log.Fatalf("Failed to send chunk: %v", err)
		}
	}

	response, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("Failed to finish upload: %v", err)
	}

	log.Printf("File uploaded successfully: %s, size: %d bytes", response.Filename, response.Size)
}

func downloadFileFromServer(client api.FileServiceClient, filename, savePath string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := client.DownloadFile(ctx, &api.DownloadFileRequest{
		Filename: filename,
	})
	if err != nil {
		log.Fatalf("Failed to create download stream: %v", err)
	}

	file, err := os.Create(savePath)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	var totalSize int64
	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("Failed to receive data: %v", err)
		}

		n, err := file.Write(resp.Chunk)
		if err != nil {
			log.Fatalf("Failed to write to file: %v", err)
		}
		totalSize += int64(n)
	}

	log.Printf("File downloaded successfully: %s, size: %d bytes", filename, totalSize)
}

func listFilesOnServer(client api.FileServiceClient) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	response, err := client.ListFiles(ctx, &api.ListFilesRequest{})
	if err != nil {
		log.Fatalf("Failed to get list of files: %v", err)
	}

	if len(response.Files) == 0 {
		fmt.Println("No files on server")
		return
	}

	fmt.Println("List of files on server:")
	fmt.Println("File name | Creation date | Update date")
	fmt.Println("---------|--------------|----------------")
	for _, file := range response.Files {
		fmt.Printf("%s | %s | %s\n", file.Filename, file.CreatedAt, file.UpdatedAt)
	}
}
