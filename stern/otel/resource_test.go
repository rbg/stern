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
	"testing"

	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

func TestNewResource(t *testing.T) {
	ctx := context.Background()

	// Test with nil client config (should not error)
	resource, err := NewResource(ctx, nil)
	if err != nil {
		t.Fatalf("NewResource failed: %v", err)
	}

	if resource == nil {
		t.Fatal("expected non-nil resource")
	}

	// Check that service.name is set
	attrs := resource.Attributes()
	var foundServiceName bool
	for _, attr := range attrs {
		if attr.Key == semconv.ServiceNameKey {
			if attr.Value.AsString() == "stern" {
				foundServiceName = true
			}
		}
	}

	if !foundServiceName {
		t.Error("service.name attribute not found or incorrect")
	}
}
