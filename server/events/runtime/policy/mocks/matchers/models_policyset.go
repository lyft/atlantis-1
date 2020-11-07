// Code generated by pegomock. DO NOT EDIT.
package matchers

import (
	"reflect"
	"github.com/petergtz/pegomock"
	models "github.com/runatlantis/atlantis/server/events/models"
)

func AnyModelsPolicySet() models.PolicySet {
	pegomock.RegisterMatcher(pegomock.NewAnyMatcher(reflect.TypeOf((*(models.PolicySet))(nil)).Elem()))
	var nullValue models.PolicySet
	return nullValue
}

func EqModelsPolicySet(value models.PolicySet) models.PolicySet {
	pegomock.RegisterMatcher(&pegomock.EqMatcher{Value: value})
	var nullValue models.PolicySet
	return nullValue
}
