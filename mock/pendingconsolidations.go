// Copyright © 2025 Attestant Limited.
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

package mock

import (
	"context"

	"github.com/attestantio/go-eth2-client/api"
	"github.com/attestantio/go-eth2-client/spec/electra"
)

// PendingConsolidations provides the pending consolidations for a given state.
func (s *Service) PendingConsolidations(ctx context.Context,
	opts *api.PendingConsolidationsOpts,
) (
	*api.Response[[]*electra.PendingConsolidation],
	error,
) {
	if s.PendingConsolidationsFunc != nil {
		return s.PendingConsolidationsFunc(ctx, opts)
	}

	return &api.Response[[]*electra.PendingConsolidation]{
		Data:     []*electra.PendingConsolidation{},
		Metadata: make(map[string]any),
	}, nil
}
