/*
Copyright 2025 The Volcano Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/rest"
)

type KubeletInterface interface {
	ListAllPods(ctx context.Context) (*v1.PodList, error)
}

type KubeletClient struct {
	client  *http.Client
	address string
	port    int
}

func NewKubeletClientForConfig(config *rest.Config, address string, port int) (KubeletInterface, error) {
	configShallowCopy := *config
	configShallowCopy.Timeout = 30 * time.Second // default request timeout
	client, err := rest.HTTPClientFor(&configShallowCopy)
	if err != nil {
		return nil, err
	}

	return &KubeletClient{
		client:  client,
		address: address,
		port:    port,
	}, nil
}

func (k KubeletClient) ListAllPods(ctx context.Context) (*v1.PodList, error) {
	path := url.URL{
		Scheme: "https",
		Host:   net.JoinHostPort(k.address, strconv.Itoa(k.port)),
		Path:   "/pods",
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}
	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error executing http request: %w", err)
	}
	defer resp.Body.Close()

	bb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response with code: %d: %w", resp.StatusCode, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error executing http request with code: %d: %s", resp.StatusCode, string(bb))
	}

	var pods v1.PodList
	if err = json.Unmarshal(bb, &pods); err != nil {
		return nil, fmt.Errorf("error unmarshalling response with code: %d: %w", resp.StatusCode, err)
	}

	return &pods, nil
}
