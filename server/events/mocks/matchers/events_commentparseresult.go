// Code generated by pegomock. DO NOT EDIT.
package matchers

import (
	"github.com/petergtz/pegomock"
	parsers "github.com/runatlantis/atlantis/server/events/parsers"
	"reflect"
)

func AnyEventsCommentParseResult() parsers.CommentParseResult {
	pegomock.RegisterMatcher(pegomock.NewAnyMatcher(reflect.TypeOf((*(parsers.CommentParseResult))(nil)).Elem()))
	var nullValue parsers.CommentParseResult
	return nullValue
}

func EqEventsCommentParseResult(value parsers.CommentParseResult) parsers.CommentParseResult {
	pegomock.RegisterMatcher(&pegomock.EqMatcher{Value: value})
	var nullValue parsers.CommentParseResult
	return nullValue
}
