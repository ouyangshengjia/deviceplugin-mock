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

package ascend

import "testing"

func Test_getDeviceNetworkIP(t *testing.T) {
	type args struct {
		subnet int
		idx    int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test1",
			args: args{
				subnet: 100,
				idx:    0,
			},
			want: "100.0.0.0",
		},
		{
			name: "test2",
			args: args{
				subnet: 100,
				idx:    256,
			},
			want: "100.0.1.0",
		},
		{
			name: "test3",
			args: args{
				subnet: 100,
				idx:    256 * 256,
			},
			want: "100.1.0.0",
		},
		{
			name: "test4",
			args: args{
				subnet: 100,
				idx:    256 * 256 * 256,
			},
			want: "100.0.0.0",
		},
		{
			name: "test5",
			args: args{
				subnet: 100,
				idx:    256*256*256 - 1,
			},
			want: "100.255.255.255",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getDeviceNetworkIP(tt.args.subnet, tt.args.idx); got != tt.want {
				t.Errorf("getDeviceNetworkIP() = %v, want %v", got, tt.want)
			}
		})
	}
}
