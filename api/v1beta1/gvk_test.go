package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGetGVK(t *testing.T) {
	gvk, ok := GetGVK(&BkGatewayResource{})
	assert.Equal(t, schema.GroupVersionKind{
		Group:   GroupVersion.Group,
		Version: GroupVersion.Version,
		Kind:    "BkGatewayResource",
	}, gvk)
	assert.Equal(t, true, ok)

	gvk, ok = GetGVK(&v1.Secret{})
	assert.Equal(t, schema.GroupVersionKind{
		Group:   v1.SchemeGroupVersion.Group,
		Version: v1.SchemeGroupVersion.Version,
		Kind:    "Secret",
	}, gvk)
	assert.Equal(t, true, ok)

	gvk, ok = GetGVK(&v1.Pod{})
	assert.Equal(t, false, ok)
}
