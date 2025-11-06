/*
Copyright 2025 The Volcano Authors.

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

package deviceplugin

import (
	"context"
	"math"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

type ListMonitor[T any] interface {
	Update(list []T)
	ListAndWatch(ctx context.Context, f func(list []T))
}

type listMonitor[T any] struct {
	list    []T
	version uint64
	lock    sync.RWMutex
}

func NewListMonitor[T any]() ListMonitor[T] {
	return &listMonitor[T]{}
}

func (m *listMonitor[T]) Update(list []T) {
	m.lock.Lock()
	defer m.lock.Unlock()

	newList := make([]T, len(list))
	copy(newList, list)
	m.list = newList
	m.version++
}

func (m *listMonitor[T]) ListAndWatch(ctx context.Context, f func(list []T)) {
	var version uint64 = math.MaxUint64
	wait.Until(func() {
		m.lock.RLock()
		if version == m.version {
			m.lock.RUnlock()
			return
		}
		version = m.version
		list := make([]T, len(m.list))
		copy(list, m.list)
		m.lock.RUnlock()

		f(list)
	}, time.Second, ctx.Done())
}
