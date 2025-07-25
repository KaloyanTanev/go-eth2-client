// Copyright © 2020 - 2025 Attestant Limited.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package http

import (
	"bytes"
	"context"
	"encoding/hex"
	"strconv"
	"strings"
	"time"

	client "github.com/attestantio/go-eth2-client"
	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec/phase0"
)

// Spec provides the spec information of the chain.
func (s *Service) Spec(ctx context.Context,
	opts *api.SpecOpts,
) (
	*api.Response[map[string]any],
	error,
) {
	if err := s.assertIsActive(ctx); err != nil {
		return nil, err
	}
	if opts == nil {
		return nil, client.ErrNoOptions
	}

	s.specMutex.RLock()
	if s.spec != nil {
		defer s.specMutex.RUnlock()

		return &api.Response[map[string]any]{
			Data:     s.spec,
			Metadata: make(map[string]any),
		}, nil
	}
	s.specMutex.RUnlock()

	s.specMutex.Lock()
	defer s.specMutex.Unlock()
	if s.spec != nil {
		// Someone else fetched this whilst we were waiting for the lock.
		return &api.Response[map[string]any]{
			Data:     s.spec,
			Metadata: make(map[string]any),
		}, nil
	}

	// Up to us to fetch the information.
	endpoint := "/eth/v1/config/spec"
	httpResponse, err := s.get(ctx, endpoint, "", &opts.Common, false)
	if err != nil {
		return nil, err
	}

	data, metadata, err := decodeJSONResponse(bytes.NewReader(httpResponse.body), map[string]any{})
	if err != nil {
		return nil, err
	}

	config := parseSpecMap(data)

	// The application mask domain type is not provided by all nodes, so add it here if not present.
	if _, exists := config["DOMAIN_APPLICATION_MASK"]; !exists {
		config["DOMAIN_APPLICATION_MASK"] = phase0.DomainType{0x00, 0x00, 0x00, 0x01}
	}
	// The BLS to execution change domain type is not provided by all nodes, so add it here if not present.
	if _, exists := config["DOMAIN_BLS_TO_EXECUTION_CHANGE"]; !exists {
		config["DOMAIN_BLS_TO_EXECUTION_CHANGE"] = phase0.DomainType{0x0a, 0x00, 0x00, 0x00}
	}
	// The builder application domain type is not officially part of the spec, so add it here if not present.
	if _, exists := config["DOMAIN_APPLICATION_BUILDER"]; !exists {
		config["DOMAIN_APPLICATION_BUILDER"] = phase0.DomainType{0x00, 0x00, 0x00, 0x01}
	}

	s.spec = config

	return &api.Response[map[string]any]{
		Data:     s.spec,
		Metadata: metadata,
	}, nil
}

func parseSpecMap(data map[string]any) map[string]any {
	config := make(map[string]any)
	for k, v := range data {
		switch value := v.(type) {
		case string:
			config[k] = parseSpecString(k, value)
		case []any:
			config[k] = parseSpecArray(value)
		case map[string]any:
			config[k] = parseSpecMap(value)
		default:
			config[k] = v
		}
	}

	return config
}

func parseSpecArray(array []any) []any {
	result := make([]any, len(array))
	for i, element := range array {
		switch value := element.(type) {
		case string:
			result[i] = parseSpecString("", value)
		case []any:
			result[i] = parseSpecArray(value)
		case map[string]any:
			result[i] = parseSpecMap(value)
		default:
			result[i] = element
		}
	}

	return result
}

func parseSpecString(k, v string) any {
	// Handle domains.
	if strings.HasPrefix(k, "DOMAIN_") {
		byteVal, err := hex.DecodeString(strings.TrimPrefix(v, "0x"))
		if err == nil {
			var domainType phase0.DomainType
			copy(domainType[:], byteVal)

			return domainType
		}
	}

	// Handle fork versions.
	if strings.HasSuffix(k, "_FORK_VERSION") {
		byteVal, err := hex.DecodeString(strings.TrimPrefix(v, "0x"))
		if err == nil {
			var version phase0.Version
			copy(version[:], byteVal)

			return version
		}
	}

	// Handle hex strings.
	if strings.HasPrefix(v, "0x") {
		byteVal, err := hex.DecodeString(strings.TrimPrefix(v, "0x"))
		if err == nil {
			return byteVal
		}
	}

	// Handle times.
	if strings.HasSuffix(k, "_TIME") {
		intVal, err := strconv.ParseInt(v, 10, 64)
		if err == nil && intVal != 0 {
			return time.Unix(intVal, 0)
		}
	}

	// Handle durations.
	if strings.HasPrefix(k, "SECONDS_PER_") || k == "GENESIS_DELAY" {
		intVal, err := strconv.ParseInt(v, 10, 64)
		if err == nil && intVal >= 0 {
			return time.Duration(intVal) * time.Second
		}
	}

	// Handle integers.
	if v == "0" {
		return uint64(0)
	}
	intVal, err := strconv.ParseUint(v, 10, 64)
	if err == nil && intVal != 0 {
		return intVal
	}

	// Assume string.
	return v
}
