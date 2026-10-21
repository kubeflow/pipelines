package integration

import (
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestCacheRecurringRunListParams(t *testing.T) {
	params := cacheRecurringRunListParams("schedule-1")
	require.Equal(t, "created_at asc", *params.SortBy,
		"the cache assertion must inspect the second chronological run, not the cold run")
	var filter api.Filter
	require.NoError(t, protojson.Unmarshal([]byte(*params.Filter), &filter))
	require.Len(t, filter.Predicates, 1)
	require.Equal(t, "recurring_run_id", filter.Predicates[0].Key)
	require.Equal(t, api.Predicate_EQUALS, filter.Predicates[0].Operation)
	require.Equal(t, "schedule-1", filter.Predicates[0].GetStringValue())
}
