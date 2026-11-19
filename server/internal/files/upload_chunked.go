package files

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ChunkUploadInfo describes a chunked upload session.
//
// This API is not used by the dashboard. Real uploads stream through the
// agent's WRITE_AT pieces; the methods are kept until the legacy code is
// removed.
type ChunkUploadInfo struct {
	UploadID    string `json:"upload_id"`
	Filename    string `json:"filename"`
	Destination string `json:"destination"`
	TotalSize   int64  `json:"total_size"`
	ChunkSize   int64  `json:"chunk_size"`
	TotalChunks int    `json:"total_chunks"`
}

// InitChunkedUpload validates a chunked upload and returns its session
// description. No files are created yet.
func (s *Service) InitChunkedUpload(destPath, filename string, totalSize, chunkSize int64) (*ChunkUploadInfo, error) {
	absPath, err := s.validatePath(destPath)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("destination must be a directory")
	}

	filename, err = cleanFilename(filename)
	if err != nil {
		return nil, err
	}

	uploadID := fmt.Sprintf("%d-%s", time.Now().UnixNano(), filename)
	totalChunks := int((totalSize + chunkSize - 1) / chunkSize)

	return &ChunkUploadInfo{
		UploadID:    uploadID,
		Filename:    filename,
		Destination: ToTildePath(absPath),
		TotalSize:   totalSize,
		ChunkSize:   chunkSize,
		TotalChunks: totalChunks,
	}, nil
}

// UploadChunk stores one chunk under the destination's .chunks directory.
func (s *Service) UploadChunk(destPath, filename, uploadID string, chunkIndex int, reader io.Reader) error {
	absPath, err := s.validatePath(destPath)
	if err != nil {
		return err
	}

	chunkDir := filepath.Join(absPath, ".chunks", uploadID)
	if err := os.MkdirAll(chunkDir, 0755); err != nil {
		return fmt.Errorf("failed to create chunk directory: %w", err)
	}

	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%06d", chunkIndex))
	f, err := os.Create(chunkPath)
	if err != nil {
		return fmt.Errorf("failed to create chunk file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return fmt.Errorf("failed to write chunk: %w", err)
	}

	return nil
}

// CompleteChunkedUpload joins all chunks into the final file and removes the
// chunk directory. A failure removes the partial target.
func (s *Service) CompleteChunkedUpload(destPath, filename, uploadID string, totalChunks int) error {
	absPath, err := s.validatePath(destPath)
	if err != nil {
		return err
	}

	chunkDir := filepath.Join(absPath, ".chunks", uploadID)

	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%06d", i))
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			return fmt.Errorf("missing chunk %d", i)
		}
	}

	finalPath := filepath.Join(absPath, filename)
	finalFile, err := os.Create(finalPath)
	if err != nil {
		return fmt.Errorf("failed to create final file: %w", err)
	}
	defer finalFile.Close()

	for i := 0; i < totalChunks; i++ {
		chunkPath := filepath.Join(chunkDir, fmt.Sprintf("chunk_%06d", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			finalFile.Close()
			os.Remove(finalPath)
			return fmt.Errorf("failed to open chunk %d: %w", i, err)
		}

		if _, err := io.Copy(finalFile, chunkFile); err != nil {
			chunkFile.Close()
			finalFile.Close()
			os.Remove(finalPath)
			return fmt.Errorf("failed to copy chunk %d: %w", i, err)
		}
		chunkFile.Close()
	}

	os.RemoveAll(chunkDir)

	return nil
}

// AbortChunkedUpload removes an incomplete upload's chunks.
func (s *Service) AbortChunkedUpload(destPath, uploadID string) error {
	absPath, err := s.validatePath(destPath)
	if err != nil {
		return err
	}

	chunkDir := filepath.Join(absPath, ".chunks", uploadID)
	return os.RemoveAll(chunkDir)
}
