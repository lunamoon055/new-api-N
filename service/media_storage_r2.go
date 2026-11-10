package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

var r2BucketPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$`)
var r2ExtensionPattern = regexp.MustCompile(`^\.[a-z0-9]{1,10}$`)

func validateR2MediaStorageProvider(provider MediaStorageProvider) error {
	endpoint, err := parseR2URL(provider.R2Endpoint)
	if err != nil || (endpoint.Path != "" && endpoint.Path != "/") {
		return fmt.Errorf("R2 endpoint must be an HTTPS origin without a bucket path")
	}
	if !r2BucketPattern.MatchString(provider.R2Bucket) {
		return fmt.Errorf("R2 bucket must be a valid 3–63 character bucket name")
	}
	publicURL, err := parseR2URL(provider.R2PublicURL)
	if err != nil || strings.HasSuffix(strings.ToLower(publicURL.Hostname()), ".r2.cloudflarestorage.com") {
		return fmt.Errorf("R2 public URL must be an HTTPS public domain, not the S3 API endpoint")
	}
	for _, segment := range strings.Split(provider.R2KeyPrefix, "/") {
		if segment == "." || segment == ".." || strings.ContainsAny(segment, "\\\r\n\x00") {
			return fmt.Errorf("R2 object prefix contains an invalid path segment")
		}
	}
	if strings.ContainsAny(provider.R2AccessKeyID, "\r\n") || strings.ContainsAny(provider.R2SecretAccessKey, "\r\n") {
		return fmt.Errorf("R2 credentials contain invalid characters")
	}
	if provider.Enabled && !hasR2Credentials(provider) {
		return fmt.Errorf("R2 Access Key ID and Secret Access Key are required before enabling")
	}
	return nil
}

func parseR2URL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.ForceQuery || u.Fragment != "" {
		return nil, fmt.Errorf("invalid R2 URL")
	}
	return u, nil
}

func hasR2Credentials(provider MediaStorageProvider) bool {
	return strings.TrimSpace(provider.R2AccessKeyID) != "" && provider.R2AccessKeyID != "********" &&
		strings.TrimSpace(provider.R2SecretAccessKey) != "" && provider.R2SecretAccessKey != "********"
}

// uploadToR2MediaStorage uses a single signed PutObject (the existing media
// limit is 200 MiB). A content-addressed key makes durable-queue retries reuse
// the same object. A spooled video is hashed and streamed from disk, never
// expanded into an in-memory multipart form. No bucket ACLs are changed here.
func uploadToR2MediaStorage(ctx context.Context, provider MediaStorageProvider, body io.ReadSeeker, size int64, filename, contentType string) (string, error) {
	if err := validateR2MediaStorageProvider(provider); err != nil {
		return "", err
	}
	if !hasR2Credentials(provider) {
		return "", fmt.Errorf("R2 Access Key ID and Secret Access Key are required for upload")
	}
	if size <= 0 || size > mediaStorageMaxBytes {
		return "", fmt.Errorf("R2 upload size must be between 1 and %d bytes", mediaStorageMaxBytes)
	}
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(body, mediaStorageMaxBytes+1))
	if err != nil || n != size {
		return "", fmt.Errorf("R2 upload content size does not match the source")
	}
	if _, err := body.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("R2 upload source cannot be rewound")
	}
	payloadHash := hex.EncodeToString(hash.Sum(nil))
	if parsed, _, err := mime.ParseMediaType(contentType); err == nil {
		contentType = parsed
	} else {
		contentType = "application/octet-stream"
	}
	extension := strings.ToLower(path.Ext(filename))
	if !r2ExtensionPattern.MatchString(extension) {
		extension = mediaStorageExtension(contentType, "application/octet-stream")
	}
	key := path.Join(provider.R2KeyPrefix, payloadHash+extension)
	endpoint, _ := parseR2URL(provider.R2Endpoint)
	endpoint.Path = "/" + provider.R2Bucket + "/" + key
	endpoint.RawPath = ""
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint.String(), body)
	if err != nil {
		return "", fmt.Errorf("R2 upload request could not be created")
	}
	req.ContentLength = size
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	signer := v4.NewSigner(func(options *v4.SignerOptions) {
		// S3 signs the escaped object path once, including spaces and Unicode.
		options.DisableURIPathEscaping = true
	})
	if err := signer.SignHTTP(ctx, aws.Credentials{
		AccessKeyID: provider.R2AccessKeyID, SecretAccessKey: provider.R2SecretAccessKey,
	}, req, payloadHash, "s3", "auto", time.Now()); err != nil {
		return "", fmt.Errorf("R2 request signing failed")
	}
	client := GetHttpClient()
	if client == nil {
		client = http.DefaultClient
	}
	signedClient := *client
	// A signature is bound to this exact host/path. Never forward it to a
	// redirect target or retry a redirected upload with altered semantics.
	signedClient.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := DoSSRFProtectedRequest(&signedClient, req)
	if err != nil {
		return "", fmt.Errorf("R2 upload request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("R2 upload returned HTTP %d", resp.StatusCode)
	}
	publicURL, _ := parseR2URL(provider.R2PublicURL)
	publicURL.Path = strings.TrimRight(publicURL.Path, "/") + "/" + key
	publicURL.RawPath = ""
	return publicURL.String(), nil
}
