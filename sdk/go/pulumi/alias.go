// Copyright 2016-2021, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pulumi

import (
	"errors"
	"strings"
)

// Alias is a partial description of prior named used for a resource. It can be processed in the
// context of a resource creation to determine what the full aliased URN would be.
type Alias struct {
	// Optional URN that uniquely identifies a resource. If specified, it takes preference and
	// other members of the struct are ignored.
	URN URNInput
	// The previous name of the resource.  If not provided, the current name of the resource is used.
	Name StringInput
	// The previous type of the resource.  If not provided, the current type of the resource is used.
	Type StringInput
	// The previous parent of the resource.  If not provided, the current parent of the resource is used.
	Parent Resource
	// The previous parent of the resource in URN format, mutually exclusive to 'Parent'
	ParentURN URNInput
	// The name of the previous stack of the resource.  If not provided, defaults to `context.GetStack()
	Stack StringInput
	// The previous project of the resource. If not provided, defaults to `context.GetProject()`.
	Project StringInput
	// There is no parent resource. We need to because go does not
	// allow distinguishing if no parent is passed from passing `nil` to parent.
	Unparent BoolInput
}

func (a Alias) collapseToURN(defaultName, defaultType string, defaultParent Resource,
	defaultProject, defaultStack string) (URNOutput, error) {

	if a.URN != nil {
		return a.URN.ToURNOutput(), nil
	}

	n := a.Name
	if n == nil {
		n = String(defaultName)
	}
	t := a.Type
	if t == nil {
		t = String(defaultType)
	}

	var parent StringPtrInput
	if defaultParent != nil {
		parent = defaultParent.URN().ToStringPtrOutput()
	}
	if a.Parent != nil && a.ParentURN != nil {
		return URNOutput{}, errors.New("alias can specify only one of Parent, ParentURN or Unparent")
	}
	if a.Parent != nil {
		parent = a.Parent.URN().ToStringPtrOutput()
	}
	if a.ParentURN != nil {
		parent = a.ParentURN.ToURNOutput().ToStringPtrOutput()
	}
	if a.Unparent != nil {
		parent = All(a.Unparent.ToBoolOutput(), parent).ApplyT(func(a []interface{}) *string {
			if a[0].(bool) {
				return nil
			}
			return a[1].(*string)
		}).(StringPtrOutput)
	}

	project := a.Project
	if project == nil {
		project = String(defaultProject)
	}
	stack := a.Stack
	if stack == nil {
		stack = String(defaultStack)
	}

	return CreateURN(n, t, parent, project, stack), nil
}

// CreateURN computes a URN from the combination of a resource name, resource type, and optional parent,
func CreateURN(name, t StringInput, parent StringPtrInput, project, stack StringInput) URNOutput {
	var parentPrefix StringInput
	parentless := func(stack, project string) string {
		return "urn:pulumi:" + stack + "::" + project + "::"
	}
	if parent != nil {
		parentPrefix = All(parent, stack, project).ApplyT(func(a []interface{}) string {
			parent := a[0].(*string)
			stack := a[1].(string)
			project := a[2].(string)
			if parent == nil {
				return parentless(stack, project)
			}
			p := *parent
			return p[0:strings.LastIndex(p, "::")] + "$"
		}).(StringOutput)
	} else {
		parentPrefix = All(stack, project).ApplyT(func(a []interface{}) string {
			return parentless(a[0].(string), a[1].(string))
		}).(StringOutput)

	}

	return All(parentPrefix, t, name).ApplyT(func(a []interface{}) URN {
		return URN(a[0].(string) + a[1].(string) + "::" + a[2].(string))
	}).(URNOutput)
}

// inheritedChildAlias computes the alias that should be applied to a child based on an alias applied to it's parent.
// This may involve changing the name of the resource in cases where the resource has a named derived from the name of
// the parent, and the parent name changed.
func inheritedChildAlias(childName, parentName, childType, project, stack string, parentURN URNOutput) URNOutput {
	aliasName := StringInput(String(childName))
	if strings.HasPrefix(childName, parentName) {
		aliasName = parentURN.ApplyT(func(urn URN) string {
			parentPrefix := urn[strings.LastIndex(string(urn), "::")+2:]
			return string(parentPrefix) + childName[len(parentName):]
		}).(StringOutput)
	}
	return CreateURN(aliasName, String(childType), parentURN.ToStringPtrOutput(), String(project), String(stack))
}
