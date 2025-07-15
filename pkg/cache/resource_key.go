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

// NewKey creates a new Key from a metav1.Object.
func NewKey(obj metav1.Object) Key {
	return Key{
		UID:  obj.GetUID(),
		Name: obj.GetName(),
	}
}

// GetUID returns the UID of the Key as a string.
func (pk Key) GetUID() string {
	return string(pk.UID)
}

// GetName returns the name of the Key.
func (pk Key) GetName() string {
	return pk.Name
}
