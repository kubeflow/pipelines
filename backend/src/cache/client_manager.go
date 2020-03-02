package cache

import (
	"database/sql"
	"fmt"
	"pipelines/backend/src/cache/storage"
)

type ClientManager struct {
	db         *storage.DB
	cacheStore storage.ExecutionCacheStoreInterface
}
