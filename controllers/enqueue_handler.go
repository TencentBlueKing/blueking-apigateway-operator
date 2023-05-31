package controllers

import (
	"micro-gateway/pkg/registry"

	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type enqueueHandler struct {
	client.Client
	handler registry.KubeEventHandler
}

func (h *enqueueHandler) Create(e event.CreateEvent, q workqueue.RateLimitingInterface) {
	rm := buildResourceMetadata(e.Object.GetName(), e.Object.GetLabels(), e.Object.GetObjectKind().GroupVersionKind())
	if rm != nil {
		h.handler.KubeEventHandler(rm)
	}
}

func (h *enqueueHandler) Update(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
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

func (h *enqueueHandler) Delete(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	rm := buildResourceMetadata(e.Object.GetName(), e.Object.GetLabels(), e.Object.GetObjectKind().GroupVersionKind())
	if rm != nil {
		h.handler.KubeEventHandler(rm)
	}
}

func (h *enqueueHandler) Generic(e event.GenericEvent, q workqueue.RateLimitingInterface) {
	rm := buildResourceMetadata(e.Object.GetName(), e.Object.GetLabels(), e.Object.GetObjectKind().GroupVersionKind())
	if rm != nil {
		h.handler.KubeEventHandler(rm)
	}
}
