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

import "testing"

func TestInt(t *testing.T) {
	i := 1
	c := newCollector()
	c.RegisterInt("i", &i)
	i = 2
	r := c.Values()
	if len(r) != 1 {
		t.Fatalf("should have one item")
	}
	if _, ok := r["i"]; !ok {
		t.Fatalf("should have i")
	}
	if r["i"] != 2 {
		t.Errorf("wrong value")
	}
}

type testItem struct{}

func (ti testItem) Value() int {
	return 2
}

func TestItem(t *testing.T) {
	c := newCollector()
	c.RegisterItem("i", testItem{})
	r := c.Values()
	if len(r) != 1 {
		t.Fatalf("should have one item")
	}
	if _, ok := r["i"]; !ok {
		t.Fatalf("should have i")
	}
	if r["i"] != 2 {
		t.Errorf("wrong value")
	}
}

type testSlice []struct{}

func (ts *testSlice) Len() int {
	return len(*ts)
}

func TestLen(t *testing.T) {
	sl := testSlice{struct{}{}}
	c := newCollector()
	c.RegisterLen("i", &sl)
	sl = append(sl, struct{}{})
	r := c.Values()
	if len(r) != 1 {
		t.Fatalf("should have one item")
	}
	if _, ok := r["i"]; !ok {
		t.Fatalf("should have i")
	}
	if r["i"] != 2 {
		t.Errorf("wrong value")
	}
}

func TestSeqLen(t *testing.T) {
	sl := make(map[int]struct{})
	sl[1] = struct{}{}
	c := newCollector()
	c.RegisterSeq("i", &sl)
	sl = make(map[int]struct{})
	sl[1] = struct{}{}
	sl[2] = struct{}{}
	r := c.Values()
	if len(r) != 1 {
		t.Fatalf("should have one item")
	}
	if _, ok := r["i"]; !ok {
		t.Fatalf("should have i")
	}
	if r["i"] != 2 {
		t.Errorf("wrong value")
	}
}

func TestSliceLen(t *testing.T) {
	sl := testSlice{struct{}{}}
	c := newCollector()
	c.RegisterSeq("i", &sl)
	sl = append(sl, struct{}{})
	r := c.Values()
	if len(r) != 1 {
		t.Fatalf("should have one item")
	}
	if _, ok := r["i"]; !ok {
		t.Fatalf("should have i")
	}
	if r["i"] != 2 {
		t.Errorf("wrong value")
	}
}

func TestFunc(t *testing.T) {
	c := newCollector()
	c.RegisterFunc("i", func() int {
		return 2
	})
	r := c.Values()
	if len(r) != 1 {
		t.Fatalf("should have one item")
	}
	if _, ok := r["i"]; !ok {
		t.Fatalf("should have i")
	}
	if r["i"] != 2 {
		t.Errorf("wrong value")
	}
}
