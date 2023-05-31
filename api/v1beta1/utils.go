package v1beta1

import (
	"reflect"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	gvkMap   = make(map[reflect.Type]schema.GroupVersionKind)
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	RegisterGVK(&BkGatewayResource{}, &BkGatewayResourceList{}, nil)
	RegisterGVK(&BkGatewayService{}, &BkGatewayServiceList{}, nil)
	RegisterGVK(&BkGatewayStage{}, &BkGatewayStageList{}, nil)
	RegisterGVK(&BkGatewayTLS{}, &BkGatewayTLSList{}, nil)
	RegisterGVK(&BkGatewayPluginMetadata{}, &BkGatewayPluginMetadataList{}, nil)
	RegisterGVK(&BkGatewayConfig{}, &BkGatewayConfigList{}, nil)
	RegisterGVK(&BkGatewayEndpoints{}, &BkGatewayEndpointsList{}, nil)
	RegisterGVK(&BkGatewayInstance{}, &BkGatewayInstanceList{}, nil)
	RegisterGVK(&v1.Secret{}, &v1.SecretList{}, &v1.SchemeGroupVersion)
}

// RegisterGVK ...
func RegisterGVK(obj interface{}, objList interface{}, gv *schema.GroupVersion) {
	gvk := schema.GroupVersionKind{}
	if gv != nil {
		gvk.Group = gv.Group
		gvk.Version = gv.Version
	} else {
		gvk.Group = GroupVersion.Group
		gvk.Version = GroupVersion.Version
	}
	typ := reflect.TypeOf(obj)
	if typ.Kind() != reflect.Ptr {
		panic("All types must be pointers to structs.")
	}
	typ = typ.Elem()
	gvk.Kind = typ.Name()
	gvkMap[typ] = gvk
	listTyp := reflect.TypeOf(objList)
	if listTyp.Kind() != reflect.Ptr {
		panic("All types must be pointers to structs.")
	}
	listTyp = listTyp.Elem()
	gvkMap[listTyp] = gvk
}

// GetGVK ...
func GetGVK(obj interface{}) (schema.GroupVersionKind, bool) {
	typ := reflect.TypeOf(obj)
	if typ.Kind() != reflect.Ptr {
		return schema.GroupVersionKind{}, false
	}

	gvk, ok := gvkMap[typ.Elem()]
	return gvk, ok
}
