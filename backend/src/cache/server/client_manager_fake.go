// Copyright 2020 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/cache/client"
	"github.com/kubeflow/pipelines/backend/src/cache/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type FakeClientManager struct {
	db                *storage.DB
	cacheStore        storage.ExecutionCacheStoreInterface
	k8sCoreClientFake *client.FakeKuberneteCoreClient
	time              util.TimeInterface
}

func NewFakeClientManager(time util.TimeInterface) (*FakeClientManager, error) {
	if time == nil {
		glog.Fatalf("The time parameter must not be null.") // Must never happen
	}
	// Initialize GORM
	db, err := storage.NewFakeDB()
	if err != nil {
		return nil, err
	}

	return &FakeClientManager{
		db:                db,
		cacheStore:        storage.NewExecutionCacheStore(db, time),
		k8sCoreClientFake: client.NewFakeKuberneteCoresClient(),
		time:              time,
	}, nil
}

func NewFakeClientManagerOrFatal(time util.TimeInterface) *FakeClientManager {
	fakeStore, err := NewFakeClientManager(time)
	if err != nil {
		glog.Fatal("The fake store doesn't create successfully. Fail fast.")
	}
	return fakeStore
}

func (f *FakeClientManager) CacheStore() storage.ExecutionCacheStoreInterface {
	return f.cacheStore
}

func (f *FakeClientManager) Time() util.TimeInterface {
	return f.time
}

func (f *FakeClientManager) DB() *storage.DB {
	return f.db
}

func (f *FakeClientManager) Close() error {
	sqlDB, err := f.db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func (f *FakeClientManager) KubernetesCoreClient() client.KubernetesCoreInterface {
	return f.k8sCoreClientFake
}
