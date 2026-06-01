package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var allowedHosts = map[string]struct{}{
	"github.com":                {},
	"api.github.com":            {},
	"raw.githubusercontent.com": {},
}

var (
	ErrURLNotAllowed     = errors.New("URL not allowed")
	ErrSchemeNotAllowed  = errors.New("URL scheme not allowed: only https is allowed")
	ErrURLNil            = errors.New("URL is nil")
	ErrHTTPResponse      = errors.New("http response error")
	ErrResponseBodyClose = errors.New("failed to close response body")
	ErrFileNameEmpty     = errors.New("filename is empty")
	ErrInvalidFilePath   = errors.New("invalid file path")
	ErrInvalidPathArgs   = errors.New("invalid download path arguments")
	ErrFileClose         = errors.New("failed to close file")
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) newRequest(
	ctx context.Context,
	method,
	url string,
	body io.Reader,
) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return req, nil
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	err := validateURL(req.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	//nolint:gosec // URL is validated in validateURL function
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer closeQuietly(resp.Body)

		body, err := io.ReadAll(io.LimitReader(resp.Body, 1024))

		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}

		if len(body) > 0 {
			return nil, fmt.Errorf("%w: status %d: %s", ErrHTTPResponse, resp.StatusCode, string(body))
		}

		return nil, fmt.Errorf("%w: status %d", ErrHTTPResponse, resp.StatusCode)
	}

	return resp, nil
}

func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}

	return c.do(req)
}

func (c *Client) GetJSON(ctx context.Context, url string, target any) error {
	resp, err := c.Get(ctx, url)
	if err != nil {
		return fmt.Errorf("GET request failed: %w", err)
	}
	defer closeQuietly(resp.Body)

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("failed to decode JSON response: %w", err)
	}

	return nil
}

func (c *Client) Download(ctx context.Context, url string, pathParts ...string) (string, error) {
	dir, filename, err := resolveDownloadTarget(pathParts...)
	if err != nil {
		return "", fmt.Errorf("failed to resolve download target: %w", err)
	}

	if err = os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}

	cleanTarget, err := sanitizePath(dir, filename)
	if err != nil {
		return "", fmt.Errorf("failed to sanitize path: %w", err)
	}

	resp, err := c.Get(ctx, url)
	if err != nil {
		return "", fmt.Errorf("GET request failed: %w", err)
	}
	defer closeQuietly(resp.Body)

	tmpTarget := cleanTarget + ".part"
	//nolint:gosec // File path is sanitized and validated in sanitizePath function
	file, err := os.OpenFile(tmpTarget, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	if _, err := io.Copy(file, resp.Body); err != nil {
		closeQuietly(file)
		_ = os.Remove(tmpTarget)
		return "", fmt.Errorf("failed to write response to file: %w", err)
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tmpTarget)
		return "", fmt.Errorf("%w: %w", ErrFileClose, err)
	}

	if err := os.Rename(tmpTarget, cleanTarget); err != nil {
		_ = os.Remove(tmpTarget)
		return "", fmt.Errorf("failed to finalize downloaded file: %w", err)
	}
	return cleanTarget, nil
}

func resolveDownloadTarget(pathParts ...string) (string, string, error) {
	switch len(pathParts) {
	case 1:
		targetPath := filepath.Clean(pathParts[0])
		return filepath.Dir(targetPath), filepath.Base(targetPath), nil
	case 2:
		return pathParts[0], pathParts[1], nil
	default:
		return "", "", fmt.Errorf("%w: expected 1 or 2 path arguments, got %d", ErrInvalidPathArgs, len(pathParts))
	}
}

func closeQuietly(c io.Closer) {
	if err := c.Close(); err != nil {
		return
	}
}

func validateURL(u *url.URL) error {
	if u == nil {
		return ErrURLNil
	}

	if u.Scheme != "https" {
		return ErrSchemeNotAllowed
	}

	host := strings.ToLower(u.Hostname())

	if _, ok := allowedHosts[host]; !ok {
		return ErrURLNotAllowed
	}

	return nil
}

func sanitizePath(dir, filename string) (string, error) {
	if filename == "" {
		return "", ErrFileNameEmpty
	}

	filename = filepath.Base(filename)

	cleanDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path of directory: %w", err)
	}

	cleanTarget, err := filepath.Abs(filepath.Join(cleanDir, filename))
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path of target file: %w", err)
	}

	rel, err := filepath.Rel(cleanDir, cleanTarget)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("%w: %s", ErrInvalidFilePath, cleanTarget)
	}
	return cleanTarget, nil
}
