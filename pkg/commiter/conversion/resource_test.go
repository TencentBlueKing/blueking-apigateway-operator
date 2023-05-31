/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - API 网关(BlueKing - APIGateway) available.
 * Copyright (C) 2017 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *     http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package conversion

import "testing"

func Test_calculateMatchSubPathRoutePriority(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "test",
			args: args{
				path: "a/b/c",
			},
			want: MATCH_SUB_PATH_PRIORITY + 5,
		},
		{
			name: "test_with_args",
			args: args{
				path: "a/:abc/c",
			},
			want: MATCH_SUB_PATH_PRIORITY + 5,
		},
		{
			name: "test_empty",
			args: args{
				path: "",
			},
			want: MATCH_SUB_PATH_PRIORITY,
		},
		{
			name: "test_ok",
			args: args{
				path: "a/abc/c",
			},
			want: MATCH_SUB_PATH_PRIORITY + 7,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateMatchSubPathRoutePriority(tt.args.path); got != tt.want {
				t.Errorf("calculateMatchSubPathRoutePriority() = %v, want %v", got, tt.want)
			}
		})
	}
}
