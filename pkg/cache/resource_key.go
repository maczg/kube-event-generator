package cache

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Key is a map of v1.Object UID to v1.Object name.
// It is used to store the UID and name of an object uniquely.
type Key struct {
	UID  types.UID
	Name string
}

func NewKey(obj metav1.Object) Key {
	return Key{
		UID:  obj.GetUID(),
		Name: obj.GetName(),
	}
}
func (pk Key) GetUID() string {
	return string(pk.UID)
}

func (pk Key) GetName() string {
	return pk.Name
}
