package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"grpcImage/pkg/api"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type FileService struct {
	api.UnimplementedFileServiceServer
	storageDir string
	mu         sync.RWMutex
	filesMeta  map[string]*api.FileMetadata
}

func NewFileService(storageDir string) (*FileService, error) {
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	service := &FileService{
		storageDir: storageDir,
		filesMeta:  make(map[string]*api.FileMetadata),
	}

	if err := service.loadExistingFiles(); err != nil {
		return nil, err
	}

	return service, nil
}

func (s *FileService) loadExistingFiles() error {
	files, err := os.ReadDir(s.storageDir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		s.mu.Lock()
		s.filesMeta[file.Name()] = &api.FileMetadata{
			Id:        file.Name(),
			Filename:  file.Name(),
			CreatedAt: info.ModTime().Format(time.RFC3339),
			UpdatedAt: info.ModTime().Format(time.RFC3339),
		}
		s.mu.Unlock()
	}

	return nil
}

func (s *FileService) UploadFile(stream api.FileService_UploadFileServer) error {
	var filename string
	var file *os.File
	var fileSize int64

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive data: %v", err)
		}

		switch data := req.Data.(type) {
		case *api.UploadFileRequest_Info:
			if file != nil {
				return status.Error(codes.InvalidArgument, "file info already sent")
			}

			filename = data.Info.Filename
			if filename == "" {
				return status.Error(codes.InvalidArgument, "file name cannot be empty")
			}

			filePath := filepath.Join(s.storageDir, filename)
			file, err = os.Create(filePath)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to create file: %v", err)
			}
			defer file.Close()

		case *api.UploadFileRequest_Chunk:
			if file == nil {
				return status.Error(codes.InvalidArgument, "file info not sent")
			}

			n, err := file.Write(data.Chunk)
			if err != nil {
				return status.Errorf(codes.Internal, "failed to write data: %v", err)
			}
			fileSize += int64(n)
		}
	}

	if file == nil {
		return status.Error(codes.InvalidArgument, "file not sent")
	}

	now := time.Now().Format(time.RFC3339)
	s.mu.Lock()
	s.filesMeta[filename] = &api.FileMetadata{
		Id:        filename,
		Filename:  filename,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.mu.Unlock()

	return stream.SendAndClose(&api.UploadFileResponse{
		Id:        filename,
		Filename:  filename,
		Size:      fileSize,
		CreatedAt: now,
	})
}

func (s *FileService) ListFiles(ctx context.Context, req *api.ListFilesRequest) (*api.ListFilesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := &api.ListFilesResponse{
		Files: make([]*api.FileMetadata, 0, len(s.filesMeta)),
	}

	for _, meta := range s.filesMeta {
		response.Files = append(response.Files, meta)
	}

	return response, nil
}

func (s *FileService) DownloadFile(req *api.DownloadFileRequest, stream api.FileService_DownloadFileServer) error {
	filename := req.Filename
	if filename == "" {
		return status.Error(codes.InvalidArgument, "file name cannot be empty")
	}

	s.mu.RLock()
	_, exists := s.filesMeta[filename]
	s.mu.RUnlock()
	if !exists {
		return status.Errorf(codes.NotFound, "file %s not found", filename)
	}

	filePath := filepath.Join(s.storageDir, filename)
	file, err := os.Open(filePath)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to open file: %v", err)
	}
	defer file.Close()

	buffer := make([]byte, 64*1024) // 64KB

	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to read file: %v", err)
		}

		if err := stream.Send(&api.DownloadFileResponse{
			Chunk: buffer[:n],
		}); err != nil {
			return status.Errorf(codes.Internal, "failed to send data: %v", err)
		}
	}

	return nil
}
