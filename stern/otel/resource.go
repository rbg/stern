//   Copyright 2025 Robert B Gordon <rbg@openrbg.com>
//
//   Licensed under the Apache License, Version 2.0 (the "License");
//   you may not use this file except in compliance with the License.
//   You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
//   Unless required by applicable law or agreed to in writing, software
//   distributed under the License is distributed on an "AS IS" BASIS,
//   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//   See the License for the specific language governing permissions and
//   limitations under the License.

package otel

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"k8s.io/client-go/tools/clientcmd"
)

// NewResource creates an OTel resource with K8s cluster information
func NewResource(ctx context.Context, clientConfig clientcmd.ClientConfig) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String("stern"),
		semconv.ServiceVersionKey.String("v1.33.0"), // TODO: Make this dynamic
	}

	// Try to get cluster name from kubeconfig context
	if clientConfig != nil {
		rawConfig, err := clientConfig.RawConfig()
		if err == nil && rawConfig.CurrentContext != "" {
			// Use context name as cluster identifier
			attrs = append(attrs, semconv.K8SClusterName(rawConfig.CurrentContext))
		}
	}

	return resource.New(ctx,
		resource.WithAttributes(attrs...),
		resource.WithProcessRuntimeDescription(),
		resource.WithHost(),
	)
}
