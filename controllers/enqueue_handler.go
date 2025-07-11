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
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/TencentBlueKing/blueking-apigateway-operator/pkg/registry"
)

type enqueueHandler struct {
	client.Client
	handler registry.KubeEventHandler
}

// Create ...
func (h *enqueueHandler) Create(e event.CreateEvent, _ workqueue.RateLimitingInterface) {
	rm := buildResourceMetadata(e.Object.GetName(), e.Object.GetLabels(), e.Object.GetObjectKind().GroupVersionKind())
	if rm != nil {
		h.handler.KubeEventHandler(rm)
	}
}

// Update ...
func (h *enqueueHandler) Update(e event.UpdateEvent, _ workqueue.RateLimitingInterface) {
	rmOld := buildResourceMetadata(
		e.ObjectOld.GetName(),
		e.ObjectOld.GetLabels(),
		e.ObjectOld.GetObjectKind().GroupVersionKind(),
	)
	rmNew := buildResourceMetadata(
		e.ObjectNew.GetName(),
		e.ObjectNew.GetLabels(),
		e.ObjectNew.GetObjectKind().GroupVersionKind(),
	)
	if rmNew != nil {
		h.handler.KubeEventHandler(rmNew)
	}
	if rmOld != nil &&
		(rmNew == nil || rmOld.StageInfo != rmNew.StageInfo) {
		h.handler.KubeEventHandler(rmOld)
	}
}

// Delete ...
func (h *enqueueHandler) Delete(e event.DeleteEvent, _ workqueue.RateLimitingInterface) {
	rm := buildResourceMetadata(e.Object.GetName(), e.Object.GetLabels(), e.Object.GetObjectKind().GroupVersionKind())
	if rm != nil {
		h.handler.KubeEventHandler(rm)
	}
}

// Generic ...
func (h *enqueueHandler) Generic(e event.GenericEvent, _ workqueue.RateLimitingInterface) {
	rm := buildResourceMetadata(e.Object.GetName(), e.Object.GetLabels(), e.Object.GetObjectKind().GroupVersionKind())
	if rm != nil {
		h.handler.KubeEventHandler(rm)
	}
}
