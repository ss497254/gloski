package worker

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ss497254/gloski/proto/agentv1"
)

// maxFilePiece caps one READ_AT reply so a request cannot make the agent
// allocate an arbitrary buffer. It stays well under the gRPC message limit.
const maxFilePiece = 4 << 20

// statFile describes a file without reading it.
func (w *Runtime) statFile(path string) (*agentv1.FileResult, error) {
	_, info, err := w.files.GetFileInfo(path)
	if err != nil {
		return nil, err
	}
	return &agentv1.FileResult{Size: info.Size(), ModTimeUnix: info.ModTime().Unix(), IsDir: info.IsDir()}, nil
}

// readFileAt returns one piece of a file. Unlike the editor's whole-file
// read it has no size limit and does not care whether the file is text.
func (w *Runtime) readFileAt(path string, offset, length int64) (*agentv1.FileResult, error) {
	absPath, info, err := w.files.GetFileInfo(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, errors.New("cannot read directory")
	}
	if offset < 0 {
		return nil, errors.New("invalid offset")
	}
	if length <= 0 || length > maxFilePiece {
		length = maxFilePiece
	}

	file, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buffer := make([]byte, length)
	n, err := file.ReadAt(buffer, offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	return &agentv1.FileResult{Content: buffer[:n], Size: info.Size(), ModTimeUnix: info.ModTime().Unix()}, nil
}

// uploadTempPath is where an upload accumulates before it replaces the
// target. It sits next to the target so the final rename stays on one
// filesystem and is atomic.
func uploadTempPath(absPath string) string {
	return filepath.Join(filepath.Dir(absPath), "."+filepath.Base(absPath)+".gloski-upload")
}

// writeFileAt appends one piece of an upload. Pieces arrive in order: the
// piece at offset 0 starts a fresh temporary file, later ones must land
// exactly at its end, and the final piece moves it into place. A partial
// upload therefore never replaces the existing file.
func (w *Runtime) writeFileAt(path string, offset int64, content []byte, final bool) error {
	absPath, err := w.files.ResolvePath(path)
	if err != nil {
		return err
	}
	if info, err := os.Stat(absPath); err == nil && info.IsDir() {
		return errors.New("destination is a directory")
	}
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return err
	}
	temp := uploadTempPath(absPath)

	flags := os.O_WRONLY
	if offset == 0 {
		flags |= os.O_CREATE | os.O_TRUNC
	}
	file, err := os.OpenFile(temp, flags, 0644)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return err
	}
	if info.Size() != offset {
		file.Close()
		return fmt.Errorf("upload out of order: expected offset %d, got %d", info.Size(), offset)
	}
	if _, err := file.WriteAt(content, offset); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if !final {
		return nil
	}
	return os.Rename(temp, absPath)
}
