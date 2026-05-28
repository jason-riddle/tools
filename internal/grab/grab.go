// Package grab downloads a single file from a URL into the current directory.
package grab

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Options configures a download.
type Options struct {
	Output string
	Force  bool
}

// Download fetches rawURL, writes it to a temporary file in the current
// directory, and renames it into place on success.
func Download(ctx context.Context, client *http.Client, rawURL string, opts Options) (savedAs string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	name, err := selectFilename(resp, opts.Output)
	if err != nil {
		return "", err
	}

	if !opts.Force {
		if _, err := os.Stat(name); err == nil {
			return "", fmt.Errorf("refusing to overwrite existing file %q", name)
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("check destination %q: %w", name, err)
		}
	}

	tmp, err := os.CreateTemp(".", "."+name+".*")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer func() {
		if err == nil {
			return
		}
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", fmt.Errorf("write %q: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close temp file: %w", err)
	}

	if err := renameIntoPlace(tmp.Name(), name, opts.Force); err != nil {
		return "", err
	}

	return name, nil
}

func renameIntoPlace(tempName, destName string, force bool) error {
	if err := os.Rename(tempName, destName); err == nil {
		return nil
	} else if !force {
		return fmt.Errorf("save %q: %w", destName, err)
	}

	if err := os.Remove(destName); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove existing file %q: %w", destName, err)
	}
	if err := os.Rename(tempName, destName); err != nil {
		return fmt.Errorf("save %q: %w", destName, err)
	}
	return nil
}

func selectFilename(resp *http.Response, output string) (string, error) {
	if output != "" {
		return validateFilename(output)
	}

	if name := filenameFromContentDisposition(resp.Header.Get("Content-Disposition")); name != "" {
		return validateFilename(name)
	}

	if resp.Request != nil && resp.Request.URL != nil {
		if name := filenameFromURL(resp.Request.URL); name != "" {
			return validateFilename(name)
		}
	}

	return "", errors.New("could not determine output filename; use -o")
}

func filenameFromContentDisposition(value string) string {
	if value == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(value)
	if err != nil {
		return ""
	}
	return params["filename"]
}

func filenameFromURL(u *url.URL) string {
	base := path.Base(u.EscapedPath())
	if base == "." || base == "/" || base == "" {
		return ""
	}
	name, err := url.PathUnescape(base)
	if err != nil {
		return ""
	}
	return name
}

func validateFilename(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", errors.New("empty output filename")
	}
	if filepath.Base(name) != name || name == "." || name == ".." {
		return "", fmt.Errorf("output filename must not contain path separators: %q", name)
	}
	return name, nil
}
