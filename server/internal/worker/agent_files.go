package worker

import (
	"encoding/json"

	"github.com/ss497254/gloski/proto/agentv1"
)

// statFile describes a file without reading it.
func (w *Runtime) statFile(path string) (*agentv1.FileResult, error) {
	_, info, err := w.files.GetFileInfo(path)
	if err != nil {
		return nil, err
	}
	return &agentv1.FileResult{Size: info.Size(), ModTimeUnix: info.ModTime().Unix(), IsDir: info.IsDir()}, nil
}

// fileErrorCode marks a FileResult as a failure reported by the device.
const fileErrorCode = 1

// fileError wraps an operation error for the master.
func fileError(err error) *agentv1.FileResult {
	return &agentv1.FileResult{Code: fileErrorCode, Message: err.Error()}
}

// handleFile runs one file request from the master and sends its result back
// on the stream.
func (w *Runtime) handleFile(id string, request *agentv1.FileRequest) {
	result := &agentv1.FileResult{}
	switch request.GetOp() {
	case agentv1.FileOp_FILE_OP_LIST:
		listing, err := w.files.List(request.GetPath())
		if err != nil {
			result = fileError(err)
			break
		}
		raw, err := json.Marshal(listing)
		if err != nil {
			result = fileError(err)
			break
		}
		result.EntriesJson = string(raw)
	case agentv1.FileOp_FILE_OP_READ:
		content, err := w.files.Read(request.GetPath())
		if err != nil {
			result = fileError(err)
			break
		}
		result.Content = []byte(content)
	case agentv1.FileOp_FILE_OP_WRITE:
		if err := w.files.Write(request.GetPath(), string(request.GetContent())); err != nil {
			result = fileError(err)
		}
	case agentv1.FileOp_FILE_OP_DELETE:
		if err := w.files.Delete(request.GetPath()); err != nil {
			result = fileError(err)
		}
	case agentv1.FileOp_FILE_OP_MKDIR:
		if err := w.files.Mkdir(request.GetPath()); err != nil {
			result = fileError(err)
		}
	case agentv1.FileOp_FILE_OP_RENAME:
		if err := w.files.Rename(request.GetPath(), request.GetNewPath()); err != nil {
			result = fileError(err)
		}
	case agentv1.FileOp_FILE_OP_STAT:
		stat, err := w.statFile(request.GetPath())
		if err != nil {
			result = fileError(err)
			break
		}
		result = stat
	default:
		result.Code, result.Message = fileErrorCode, "unsupported file operation"
	}
	_ = w.send(&agentv1.AgentMessage{
		RequestId: id,
		Payload:   &agentv1.AgentMessage_FileResult{FileResult: result},
	})
}
