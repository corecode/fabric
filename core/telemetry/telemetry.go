/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

                 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package telemetry

import "reflect"

type Item interface {
	Value() int
}

type Collector struct {
	items map[string]Item
}

func newCollector() *Collector {
	return &Collector{
		items: make(map[string]Item),
	}
}

func (c *Collector) Values() map[string]int {
	r := make(map[string]int)
	for k, i := range c.items {
		r[k] = i.Value()
	}
	return r
}

func (c *Collector) RegisterItem(key string, i Item) {
	c.items[key] = i
}

type Lener interface {
	Len() int
}

type lenItem struct {
	lener Lener
}

func (l lenItem) Value() int {
	return l.lener.Len()
}

func (c *Collector) RegisterLen(key string, l Lener) {
	c.RegisterItem(key, lenItem{l})
}

func (c *Collector) RegisterSeq(key string, seq interface{}) {
	c.RegisterLen(key, reflect.ValueOf(seq))
}

type funcItem struct {
	f func() int
}

func (f funcItem) Value() int {
	return f.f()
}

func (c *Collector) RegisterFunc(key string, f func() int) {
	c.RegisterItem(key, funcItem{f})
}

type intItem struct {
	pi *int
}

func (i intItem) Value() int {
	return *i.pi
}

func (c *Collector) RegisterInt(key string, pi *int) {
	c.RegisterItem(key, intItem{pi})
}
