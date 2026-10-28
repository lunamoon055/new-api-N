package service

import (
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/url"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/gin-gonic/gin"
)

const taskSnapshotMaxBody = 128 << 10
const taskSnapshotMaxInput = 1 << 20

// CaptureTaskRequestSnapshot never modifies the forwarded request or fails a
// generation. No request headers (including Authorization/Cookie) are stored.
func CaptureTaskRequestSnapshot(c *gin.Context) *dto.TaskRequestSnapshot {
	snapshot := &dto.TaskRequestSnapshot{Format: "unavailable", Omitted: true}
	if c == nil || c.Request == nil {
		return snapshot
	}
	snapshot.Method, snapshot.Path = c.Request.Method, c.Request.URL.Path
	contentType, params, _ := mime.ParseMediaType(c.GetHeader("Content-Type"))
	snapshot.ContentType = contentType
	storage, err := common.GetBodyStorage(c)
	if err != nil {
		return snapshot
	}
	defer storage.Seek(0, io.SeekStart)
	var body json.RawMessage
	switch contentType {
	case "application/json":
		if storage.Size() > taskSnapshotMaxInput {
			return snapshot
		}
		data, err := storage.Bytes()
		if err != nil || common.Unmarshal(data, &body) != nil {
			return snapshot
		}
		snapshot.Format, snapshot.Omitted = "json", false
	case "application/x-www-form-urlencoded":
		if storage.Size() > taskSnapshotMaxInput {
			return snapshot
		}
		data, err := storage.Bytes()
		if err != nil {
			return snapshot
		}
		values, err := url.ParseQuery(string(data))
		if err != nil {
			return snapshot
		}
		body, _ = common.Marshal(values)
		snapshot.Format, snapshot.Omitted = "form", false
	case "multipart/form-data":
		if _, err := storage.Seek(0, io.SeekStart); err != nil {
			return snapshot
		}
		fields := map[string]any{}
		files := make([]map[string]any, 0)
		reader := multipart.NewReader(storage, params["boundary"])
		snapshot.Format, snapshot.Omitted = "multipart", false
		for count := 0; ; count++ {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil || count >= 1000 {
				snapshot.Omitted = true
				break
			}
			name := part.FormName()
			if filename := part.FileName(); filename != "" {
				size, readErr := io.Copy(io.Discard, part)
				if readErr != nil {
					snapshot.Omitted = true
				}
				files = append(files, map[string]any{"field": name, "filename": filename, "content_type": part.Header.Get("Content-Type"), "size_bytes": size})
				part.Close()
				continue
			}
			value, err := io.ReadAll(io.LimitReader(part, taskSnapshotMaxBody+1))
			part.Close()
			if err != nil || len(value) > taskSnapshotMaxBody {
				snapshot.Omitted = true
				value = []byte("[omitted]")
			}
			previous, _ := fields[name].([]string)
			fields[name] = append(previous, string(value))
		}
		body, _ = common.Marshal(map[string]any{"fields": fields, "files": files})
	default:
		return snapshot
	}
	snapshot.Body = sanitizeTaskSnapshotJSON(body, snapshot, 0)
	if len(snapshot.Body) > taskSnapshotMaxBody {
		snapshot.Body, snapshot.Omitted = nil, true
	}
	return snapshot
}

func taskSnapshotSecretField(key string) bool {
	key = strings.NewReplacer("_", "", "-", "", " ", "").Replace(strings.ToLower(key))
	switch key {
	case "authorization", "proxyauthorization", "cookie", "setcookie", "apikey", "xapikey", "key", "token", "accesstoken", "refreshtoken", "password", "secret", "clientsecret", "signature", "sig", "xamzsignature", "xamzcredential", "xosssignature", "xosscredential":
		return true
	}
	return false
}

// Decode each leaf as RawMessage so large integer seeds and explicit false/0
// remain exact; generic float64 decoding would corrupt some client values.
func sanitizeTaskSnapshotJSON(raw json.RawMessage, snapshot *dto.TaskRequestSnapshot, depth int) json.RawMessage {
	encode := func(value any) json.RawMessage { data, _ := common.Marshal(value); return data }
	if depth > 32 {
		snapshot.Omitted = true
		return encode("[omitted]")
	}
	switch common.GetJsonType(raw) {
	case "object":
		var object map[string]json.RawMessage
		if common.Unmarshal(raw, &object) != nil {
			return nil
		}
		for key, value := range object {
			if taskSnapshotSecretField(key) {
				object[key], snapshot.Redacted = encode("[REDACTED]"), true
			} else {
				object[key] = sanitizeTaskSnapshotJSON(value, snapshot, depth+1)
			}
		}
		return encode(object)
	case "array":
		var values []json.RawMessage
		if common.Unmarshal(raw, &values) != nil {
			return nil
		}
		for i := range values {
			values[i] = sanitizeTaskSnapshotJSON(values[i], snapshot, depth+1)
		}
		return encode(values)
	case "string":
		var value string
		_ = common.Unmarshal(raw, &value)
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "data:") {
			snapshot.Omitted = true
			return encode("[inline media omitted]")
		}
		if len(value) > 64<<10 {
			snapshot.Omitted = true
			return encode("[oversized value omitted]")
		}
		trimmed := strings.TrimSpace(value)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			var embedded json.RawMessage
			if common.Unmarshal([]byte(value), &embedded) == nil {
				nested := &dto.TaskRequestSnapshot{}
				safe := sanitizeTaskSnapshotJSON(embedded, nested, depth+1)
				if nested.Redacted || nested.Omitted {
					snapshot.Redacted = snapshot.Redacted || nested.Redacted
					snapshot.Omitted = snapshot.Omitted || nested.Omitted
					return encode(string(safe))
				}
				return raw
			}
		}
		if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") {
			parsed, err := url.Parse(value)
			if err == nil {
				changed := parsed.User != nil
				parsed.User = nil
				query := parsed.Query()
				for key := range query {
					if taskSnapshotSecretField(key) {
						query.Set(key, "[REDACTED]")
						changed = true
					}
				}
				if changed {
					snapshot.Redacted = true
					parsed.RawQuery = query.Encode()
					return encode(parsed.String())
				}
			}
		}
		return raw
	default:
		return raw
	}
}
