// Code generated by pegomock. DO NOT EDIT.
package matchers

import (
	"reflect"
	"github.com/petergtz/pegomock"
	valid "github.com/runatlantis/atlantis/server/events/yaml/valid"
)

func AnyValidPolicySet() valid.PolicySet {
	pegomock.RegisterMatcher(pegomock.NewAnyMatcher(reflect.TypeOf((*(valid.PolicySet))(nil)).Elem()))
	var nullValue valid.PolicySet
	return nullValue
}

func EqValidPolicySet(value valid.PolicySet) valid.PolicySet {
	pegomock.RegisterMatcher(&pegomock.EqMatcher{Value: value})
	var nullValue valid.PolicySet
	return nullValue
}
