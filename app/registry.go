package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func getToken(image string) (string, error) {
	url := fmt.Sprintf("https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/%s:pull", image)
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status code: %d", response.StatusCode)
	}

	type tokenResponse struct {
		Token string `json:"token"`
	}
	token := tokenResponse{}

	err = json.NewDecoder(response.Body).Decode(&token)
	if err != nil {
		return "", err
	}

	return token.Token, nil
}

type Layer struct {
	MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
	Digest    string `json:"digest"`
}

type Manifest struct {
	Digest   string `json:"digest"`
	Platform struct {
		Architecture string `json:"architecture"`
		Os           string `json:"os"`
	} `json:"platform"`
}

func getLayers(image string, tag string, token string) ([]Layer, error) {
	url := fmt.Sprintf("https://registry.hub.docker.com/v2/library/%s/manifests/%s", image, tag)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d", response.StatusCode)
	}

	type manifestsResponse struct {
		Manifests []Manifest `json:"manifests"`
		Layers    []Layer    `json:"layers"`
	}
	manifestsResp := manifestsResponse{}
	err = json.NewDecoder(response.Body).Decode(&manifestsResp)
	if err != nil {
		return nil, err
	}

	layers := manifestsResp.Layers
	for _, manifest := range manifestsResp.Manifests {
		if isRuntimePlatformManifest(manifest) {
			l, err := getLayers(image, manifest.Digest, token)
			if err != nil {
				return nil, err
			}
			layers = append(layers, l...)
		}
	}

	return layers, nil
}

func isRuntimePlatformManifest(manifest Manifest) bool {
	return manifest.Platform.Architecture == runtime.GOARCH && manifest.Platform.Os == runtime.GOOS
}

func PullImage(image string, dir string) (string, error) {
	token, err := getToken(image)
	if err != nil {
		return "", err
	}

	img, tag := ParseImageTag(image)

	layers, err := getLayers(img, tag, token)
	if err != nil {
		return "", err
	}

	imgDir := filepath.Join(dir, image)
	for _, layer := range layers {
		err := DownloadLayer(layer, image, token, imgDir)
		if err != nil {
			return "", err
		}
	}

	return imgDir, nil
}

func ParseImageTag(image string) (string, string) {
	i := strings.Index(image, ":")
	if i < 0 {
		return image, "latest"
	}
	return image[:i], image[i+1:]
}

func DownloadLayer(layer Layer, image string, token string, dir string) error {
	url := fmt.Sprintf("https://registry.hub.docker.com/v2/library/%s/blobs/%s", image, layer.Digest)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/vnd.oci.image.layer.v1.tar+gzip")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", response.StatusCode)
	}

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, response.Body)
	if err != nil {
		return err
	}

	return ExtractTar(dir, buf)
}

func ExtractTar(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(dst, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

			f.Close()
		}
	}
}
