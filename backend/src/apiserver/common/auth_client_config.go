// Copyright 2026 The Kubeflow Authors
//
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

package common

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// GetAuthorizationClientRateLimit resolves the per-client TokenReview and
// SubjectAccessReview limits. Explicit general-client settings remain fallbacks.
func GetAuthorizationClientRateLimit() (qps float64, burst int, err error) {
	qpsKey := "AuthClientQPS"
	if !viper.IsSet(qpsKey) {
		qpsKey = "ClientQPS"
	}
	// Ordinary-token task writes can require two TokenReviews. Keep a bounded
	// budget for 20 such writes/s without changing general Kubernetes clients.
	qps = 40
	if viper.IsSet(qpsKey) {
		// client-go stores QPS as float32; reject values that become zero or infinity.
		qps, err = strconv.ParseFloat(strings.TrimSpace(viper.GetString(qpsKey)), 32)
		if err != nil || qps <= 0 || math.IsNaN(qps) || math.IsInf(qps, 0) {
			return 0, 0, fmt.Errorf("%s must be a positive finite float32 rate; set a bounded authorization-client QPS", qpsKey)
		}
	}
	burstKey := "AuthClientBurst"
	if !viper.IsSet(burstKey) {
		burstKey = "ClientBurst"
	}
	burst, err = getPositiveIntConfigWithDefault(burstKey, 80)
	if err != nil {
		return 0, 0, err
	}
	return qps, burst, nil
}
