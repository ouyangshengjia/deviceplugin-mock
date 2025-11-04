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

package util

import (
	"strconv"

	dpmockv1alpha1 "volcano.sh/deviceplugin-mock/api/dpmock/v1alpha1"
)

func BuildDeviceID(format *dpmockv1alpha1.DeviceIDFormat) []string {
	var ids []string
	for prefix, amount := range format.Prefix {
		for i := range amount {
			ordinal := strconv.Itoa(int(format.OrdinalStart + i))
			ids = append(ids, prefix+format.Delimiter+ordinal)
		}
	}
	return ids
}

func IsNodeResourceReference(ref *dpmockv1alpha1.ResourceReference) bool {
	if ref == nil {
		return false
	}
	return ref.APIGroup == dpmockv1alpha1.SchemeGroupVersion.Group && ref.Kind == "NodeResource"
}
