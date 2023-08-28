// Copyright 2023 ecodeclub
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redis

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ecodeclub/ecache/internal/errs"
	"github.com/ecodeclub/ecache/mocks"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCache_Set(t *testing.T) {
	testCases := []struct {
		name string

		mock func(*gomock.Controller) redis.Cmdable

		key        string
		value      string
		expiration time.Duration

		wantErr error
	}{
		{
			name: "set value",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				status := redis.NewStatusCmd(context.Background())
				status.SetVal("OK")
				cmd.EXPECT().
					Set(context.Background(), "name", "大明", time.Minute).
					Return(status)
				return cmd
			},
			key:        "name",
			value:      "大明",
			expiration: time.Minute,
		},
		{
			name: "timeout",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				status := redis.NewStatusCmd(context.Background())
				status.SetErr(context.DeadlineExceeded)
				cmd.EXPECT().
					Set(context.Background(), "name", "大明", time.Minute).
					Return(status)
				return cmd
			},
			key:        "name",
			value:      "大明",
			expiration: time.Minute,

			wantErr: context.DeadlineExceeded,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			c := NewCache(tc.mock(ctrl))
			err := c.Set(context.Background(), tc.key, tc.value, tc.expiration)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}

func TestCache_Get(t *testing.T) {
	testCases := []struct {
		name string

		mock func(*gomock.Controller) redis.Cmdable

		key string

		wantErr error
		wantVal string
	}{
		{
			name: "get value",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				status := redis.NewStringCmd(context.Background())
				status.SetVal("大明")
				cmd.EXPECT().
					Get(context.Background(), "name").
					Return(status)
				return cmd
			},
			key: "name",

			wantVal: "大明",
		},
		{
			name: "get error",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				status := redis.NewStringCmd(context.Background())
				status.SetErr(redis.Nil)
				cmd.EXPECT().
					Get(context.Background(), "name").
					Return(status)
				return cmd
			},
			key: "name",

			wantErr: errs.ErrKeyNotExist,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			c := NewCache(tc.mock(ctrl))
			val := c.Get(context.Background(), tc.key)
			assert.Equal(t, tc.wantErr, val.Err)
			if val.Err != nil {
				return
			}
			assert.Equal(t, tc.wantVal, val.Val.(string))
		})
	}
}

func TestCache_SetNX(t *testing.T) {
	testCase := []struct {
		name       string
		mock       func(*gomock.Controller) redis.Cmdable
		key        string
		val        string
		expiration time.Duration
		result     bool
	}{
		{
			name: "setnx",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				boolCmd := redis.NewBoolCmd(context.Background())
				boolCmd.SetVal(true)
				cmd.EXPECT().
					SetNX(context.Background(), "setnx_key", "hello ecache", time.Second*10).
					Return(boolCmd)
				return cmd
			},
			key:        "setnx_key",
			val:        "hello ecache",
			expiration: time.Second * 10,
			result:     true,
		},
		{
			name: "setnx-fail",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				boolCmd := redis.NewBoolCmd(context.Background())
				boolCmd.SetVal(false)
				cmd.EXPECT().
					SetNX(context.Background(), "setnx-key", "hello ecache", time.Second*10).
					Return(boolCmd)

				return cmd
			},
			key:        "setnx-key",
			val:        "hello ecache",
			expiration: time.Second * 10,
			result:     false,
		},
	}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			c := NewCache(tc.mock(ctrl))
			val, err := c.SetNX(context.Background(), tc.key, tc.val, tc.expiration)
			require.NoError(t, err)
			assert.Equal(t, tc.result, val)
		})
	}
}

func TestCache_GetSet(t *testing.T) {
	testCase := []struct {
		name    string
		mock    func(*gomock.Controller) redis.Cmdable
		key     string
		val     string
		wantErr error
	}{
		{
			name: "test_get_set",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				str := redis.NewStringCmd(context.Background())
				str.SetVal("hello ecache")
				cmd.EXPECT().
					GetSet(context.Background(), "test_get_set", "hello go").
					Return(str)
				return cmd
			},
			key: "test_get_set",
			val: "hello go",
		},
		{
			name: "test_get_set_err",
			mock: func(ctrl *gomock.Controller) redis.Cmdable {
				cmd := mocks.NewMockCmdable(ctrl)
				str := redis.NewStringCmd(context.Background())
				str.SetErr(errs.ErrKeyNotExist)
				cmd.EXPECT().
					GetSet(context.Background(), "test_get_set_err", "hello ecache").
					Return(str)
				return cmd
			},
			key:     "test_get_set_err",
			val:     "hello ecache",
			wantErr: errs.ErrKeyNotExist,
		},
	}

	for _, tc := range testCase {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		c := NewCache(tc.mock(ctrl))
		val := c.GetSet(context.Background(), tc.key, tc.val)
		assert.Equal(t, tc.wantErr, val.Err)
	}
}