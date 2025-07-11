/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - API 网关(BlueKing - APIGateway) available.
 * Copyright (C) 2025 Tencent. All rights reserved.
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

// Package controllers ...
package controllers

import (
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

// EmptyFilter unused filter
type EmptyFilter struct{}

// Create implement EventFilter
func (uf *EmptyFilter) Create(event.CreateEvent, workqueue.RateLimitingInterface) {}

// Update implement EventFilter
func (uf *EmptyFilter) Update(event.UpdateEvent, workqueue.RateLimitingInterface) {}

// Delete implement EventFilter
func (uf *EmptyFilter) Delete(event.DeleteEvent, workqueue.RateLimitingInterface) {}

// Generic implement EventFilter
func (uf *EmptyFilter) Generic(event.GenericEvent, workqueue.RateLimitingInterface) {}
