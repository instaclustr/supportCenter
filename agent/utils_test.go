package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestJoinToSet(t *testing.T) {

	var testCases = []struct {
		a        []string
		b        []string
		expected []string
	}{
		{[]string{"a", "b"}, []string{"c"}, []string{"a", "b", "c"}},
		{[]string{"a", "a"}, []string{"c"}, []string{"a", "c"}},
		{[]string{"a", "a"}, []string{"c", "c"}, []string{"a", "c"}},
		{[]string{}, []string{"c", "c", "d"}, []string{"c", "d"}},
		{[]string{"a", "b", "a"}, []string{}, []string{"a", "b"}},
		{[]string{}, []string{}, []string{}},
	}

	for _, test := range testCases {
		assert.ElementsMatch(t, JoinToSet(test.a, test.b), test.expected)
	}
}
