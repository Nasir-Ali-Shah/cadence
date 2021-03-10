// This file was automatically generated by genny.
// Any changes will be lost if this file is regenerated.
// see https://github.com/cheekybits/genny

/*
 * Cadence - The resource-oriented smart contract programming language
 *
 * Copyright 2019-2021 Dapper Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Based on https://github.com/wk8/go-ordered-map, Copyright Jean Rougé
 *
 */

package orderedmap

import "container/list"

// StringStructOrderedMap
//
type StringStructOrderedMap struct {
	pairs map[string]*StringStructPair
	list  *list.List
}

// NewStringStructOrderedMap creates a new StringStructOrderedMap.
func NewStringStructOrderedMap() *StringStructOrderedMap {
	return &StringStructOrderedMap{
		pairs: make(map[string]*StringStructPair),
		list:  list.New(),
	}
}

// Clear removes all entries from this ordered map.
func (om *StringStructOrderedMap) Clear() {
	om.list.Init()
	for key := range om.pairs {
		delete(om.pairs, key)
	}
}

// Get returns the value associated with the given key.
// Returns nil if not found.
// The second return value indicates if the key is present in the map.
func (om *StringStructOrderedMap) Get(key string) (result struct{}, present bool) {
	var pair *StringStructPair
	if pair, present = om.pairs[key]; present {
		return pair.Value, present
	}
	return
}

// GetPair returns the key-value pair associated with the given key.
// Returns nil if not found.
func (om *StringStructOrderedMap) GetPair(key string) *StringStructPair {
	return om.pairs[key]
}

// Set sets the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Set`.
func (om *StringStructOrderedMap) Set(key string, value struct{}) (oldValue struct{}, present bool) {
	var pair *StringStructPair
	if pair, present = om.pairs[key]; present {
		oldValue = pair.Value
		pair.Value = value
		return
	}

	pair = &StringStructPair{
		Key:   key,
		Value: value,
	}
	pair.element = om.list.PushBack(pair)
	om.pairs[key] = pair

	return
}

// Delete removes the key-value pair, and returns what `Get` would have returned
// on that key prior to the call to `Delete`.
func (om *StringStructOrderedMap) Delete(key string) (oldValue struct{}, present bool) {
	var pair *StringStructPair
	pair, present = om.pairs[key]
	if !present {
		return
	}

	om.list.Remove(pair.element)
	delete(om.pairs, key)
	oldValue = pair.Value

	return
}

// Len returns the length of the ordered map.
func (om *StringStructOrderedMap) Len() int {
	return len(om.pairs)
}

// Oldest returns a pointer to the oldest pair.
func (om *StringStructOrderedMap) Oldest() *StringStructPair {
	return listElementToStringStructPair(om.list.Front())
}

// Newest returns a pointer to the newest pair.
func (om *StringStructOrderedMap) Newest() *StringStructPair {
	return listElementToStringStructPair(om.list.Back())
}

// Foreach iterates over the entries of the map in the insertion order, and invokes
// the provided function for each key-value pair.
func (om *StringStructOrderedMap) Foreach(f func(key string, value struct{})) {
	for pair := om.Oldest(); pair != nil; pair = pair.Next() {
		f(pair.Key, pair.Value)
	}
}

// StringStructPair
//
type StringStructPair struct {
	Key   string
	Value struct{}

	element *list.Element
}

// Next returns a pointer to the next pair.
func (p *StringStructPair) Next() *StringStructPair {
	return listElementToStringStructPair(p.element.Next())
}

// Prev returns a pointer to the previous pair.
func (p *StringStructPair) Prev() *StringStructPair {
	return listElementToStringStructPair(p.element.Prev())
}

func listElementToStringStructPair(element *list.Element) *StringStructPair {
	if element == nil {
		return nil
	}
	return element.Value.(*StringStructPair)
}
