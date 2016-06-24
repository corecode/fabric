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

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/op/go-logging"
)

var logger = logging.MustGetLogger("telemetry")

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
	v := reflect.ValueOf(seq)

	if v.Kind() == reflect.Ptr {
		c.RegisterFunc(key, func() int {
			return v.Elem().Len()
		})
	} else {
		c.RegisterLen(key, v)
	}
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

func (c *Collector) JSONValues(w http.ResponseWriter, r *http.Request) {
	v := c.Values()
	j, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		http.Error(w, fmt.Sprintf("json marshal error: %s", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(j)
}

// package functions

var DefaultTelemetry = newCollector()

func RegisterItem(key string, i Item) {
	DefaultTelemetry.RegisterItem(key, i)
}

func RegisterLen(key string, l Lener) {
	DefaultTelemetry.RegisterLen(key, l)
}

func RegisterSeq(key string, i interface{}) {
	DefaultTelemetry.RegisterSeq(key, i)
}

func RegisterFunc(key string, f func() int) {
	DefaultTelemetry.RegisterFunc(key, f)
}

func RegisterInt(key string, pi *int) {
	DefaultTelemetry.RegisterInt(key, pi)
}

func Init(prefix string, mux *http.ServeMux) {
	if mux == nil {
		mux = http.DefaultServeMux
	}

	mux.Handle(prefix+"/telemetry.json", http.HandlerFunc(DefaultTelemetry.JSONValues))
}

func logReport() {
	var lastVal map[string]int
	for {
		time.Sleep(1 * time.Second)
		if logger.IsEnabledFor(logging.INFO) {
			vals := DefaultTelemetry.Values()

			if reflect.DeepEqual(lastVal, vals) {
				continue
			}

			s := []string{"New telemetry values"}
			for k, v := range vals {
				s = append(s, fmt.Sprintf("%s\t%d", k, v))
			}
			sort.Sort(sort.StringSlice(s))
			logger.Info(strings.Join(s, "\n"))
			lastVal = vals
		}
	}
}

func init() {
	go logReport()
}
