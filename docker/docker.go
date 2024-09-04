package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var (
	authURL      string = `https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/%s:pull`
	manifesURL   string = `https://index.docker.io/v2/library/%s/manifests/latest`
	manifestByDigestURL string = `https://index.docker.io/v2/library/%s/manifests/%s`
	blobURL      string = `https://index.docker.io/v2/library/%s/blobs/%s`
)

func Authenticate(image string) (string, error) {
	fmt.Println("🤝 Authenticating...")
	resp, err := http.Get(fmt.Sprintf(authURL, image))
	if err != nil && resp.StatusCode != 200 {
		log.Fatalf("Error authenticating on auth.docker.io: %v", err)
		return "", nil
	}
	defer resp.Body.Close()

	var authResponse AuthResponse
	json.NewDecoder(resp.Body).Decode(&authResponse)
	return authResponse.Token, nil
}

func fetchManifestByURL(url string, token string, image string) (*ManifestResponse, error) {
	client := http.Client{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return &ManifestResponse{}, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	req.Header.Add("Accept", "application/vnd.oci.image.manifest.v1+json")

	resp, err := client.Do(req)
	if err != nil {
		return &ManifestResponse{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ManifestResponse{}, err
	}

	var manifestResponse ManifestResponse
	json.Unmarshal(body, &manifestResponse)

	// If layers are populated, this is a direct image manifest
	if len(manifestResponse.Layers) > 0 {
		return &manifestResponse, nil
	}

	// Otherwise it might be a manifest list — try to resolve linux/amd64
	var manifestList ManifestList
	json.Unmarshal(body, &manifestList)

	for _, m := range manifestList.Manifests {
		if m.Platform.Os == "linux" && m.Platform.Architecture == "amd64" {
			digestURL := fmt.Sprintf(manifestByDigestURL, image, m.Digest)
			return fetchManifestByURL(digestURL, token, image)
		}
	}

	return &manifestResponse, nil
}

func FetchManifest(image string, token string) (*ManifestResponse, error) {
	fmt.Println("🧠 Fetching Manifest from DockerHub...")
	url := fmt.Sprintf(manifesURL, image)
	return fetchManifestByURL(url, token, image)
}

func FetchLayers(image string, token string, manifest ManifestResponse) error {
	fmt.Println("🤸 Fetching Layers of Image from DockerHub...")

	client := http.Client{}

	for _, layer := range manifest.Layers {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(blobURL, image, layer.Digest), nil)
		if err != nil {
			log.Fatalf("Error creating request: %v", err)
			return err
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := client.Do(req)
		if err != nil && resp.StatusCode != 200 {
			log.Fatalf("Error fetching manifest: %v", err)
			return err
		}
		defer resp.Body.Close()

		// Create the file
		out, err := os.Create("/tmp/dockerly/rootfs/layer.tar")
		if err != nil {
			return err
		}
		defer out.Close()

		// Write the body to file
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return err
		}
	}

	return nil
}

func FetchConfig(image string, token string, manifest ManifestResponse) (Config, error) {
	fmt.Println("🏄 Fetching Config...")

	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(blobURL, image, manifest.Config.Digest), nil)
	if err != nil {
		log.Fatalf("Error creating request: %v", err)
		return Config{}, err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := client.Do(req)
	if err != nil && resp.StatusCode != 200 {
		log.Fatalf("Error fetching manifest: %v", err)
		return Config{}, err
	}

	defer resp.Body.Close()
	var config Config
	json.NewDecoder(resp.Body).Decode(&config)

	return config, nil
}

func ExtractLayer(filepath string) error {
	fmt.Println("☔️ Extracting Layers...")

	cmd := exec.Command("tar", "-xzf", filepath, "-C", "/tmp/dockerly/rootfs/")
	if err := cmd.Run(); err != nil {
		log.Fatalf("Error extracting the layer:%v", err)
	}
	return nil
}