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

package dpmock

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"

	dpmockv1alpha1 "volcano.sh/deviceplugin-mock/api/dpmock/v1alpha1"
)

func Test_parseNodePatch(t *testing.T) {
	type args struct {
		text    string
		context *dpmockv1alpha1.ResourceBasicDescription
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test1",
			args: args{
				text: `
metadata:
  annotations:
    device-status.volcano.sh/dpmock: '{{ toJson .deviceIDFormat.prefix }}'
`,
				context: &dpmockv1alpha1.ResourceBasicDescription{
					DeviceIDFormat: dpmockv1alpha1.DeviceIDFormat{
						Prefix: map[string]int32{"dev0": 2, "dev1": 4},
					},
				},
			},
			want:    `{"metadata":{"annotations":{"device-status.volcano.sh/dpmock":"{\"dev0\":2,\"dev1\":4}"}}}`,
			wantErr: false,
		},
		{
			name: "test2",
			args: args{
				text: `
metadata:
  annotations:
    {{- $num_list := atoi .capacity | until }}
    {{- $hccs_list := join "," $num_list | list }}
    {{- $sio_list := list }}
    {{- range $i, $e := chunk 2 $num_list }}
    {{- $sio_list = join "," $e | append $sio_list }}
    {{- end }}
    volcano.sh/device-topologys: '{"topologys":[{"type":"HCCS","groups":{{ toJson $hccs_list }}},{"type":"SIO","groups":{{ toJson $sio_list }}}]}'
`,
				context: &dpmockv1alpha1.ResourceBasicDescription{
					Capacity: resource.MustParse("4"),
				},
			},
			want:    `{"metadata":{"annotations":{"volcano.sh/device-topologys":"{\"topologys\":[{\"type\":\"HCCS\",\"groups\":[\"0,1,2,3\"]},{\"type\":\"SIO\",\"groups\":[\"0,1\",\"2,3\"]}]}"}}}`,
			wantErr: false,
		},
		{
			name: "test3",
			args: args{
				text: `{"metadata":{"annotations":{"volcano.sh/device-topologys":null}}}`,
				context: &dpmockv1alpha1.ResourceBasicDescription{
					DeviceIDFormat: dpmockv1alpha1.DeviceIDFormat{
						Prefix: map[string]int32{"dev0": 2, "dev1": 4},
					},
				},
			},
			want:    `{"metadata":{"annotations":{"volcano.sh/device-topologys":null}}}`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNodePatch(tt.args.text, tt.args.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseNodePatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseNodePatch() got = %v, want %v", got, tt.want)
			}
		})
	}
}
