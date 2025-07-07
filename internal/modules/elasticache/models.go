package elasticache

import (
	"time"
)

type CacheClusterEncyrptionConfig struct {
	AtRest    bool `json:"at_rest"`
	InTransit bool `json:"in_trasit"`
}

type ElastiCacheCluster struct {
	ARN                           string                                `json:"arn"`
	Id                            string                                `json:"cache_cluster_id"`
	NodeType                      string                                `json:"cache_node_type"`
	NumCacheNodes                 int32                                 `json:"num_cache_nodes"`
	Engine                        string                                `json:"engine"`
	EngineVersion                 string                                `json:"engine_version"`
	Status                        string                                `json:"status"`
	EncryptionConfig              CacheClusterEncyrptionConfig          `json:"encryption_config"`
	ReplicationGroup              string                                `json:"replication_group"`
	UnappliedUpdateActionsSummary ElastiCacheUpdateActionsSummary       `json:"update_action_summary"`
	UnappliedUpdateActions        []ElastiCacheCacheClusterUpdateAction `json:"update_actions"`
}

type ElastiCacheReplicationGroup struct {
	ARN                           string                                    `json:"arn"`
	Id                            string                                    `json:"replication_group_id"`
	NodeType                      string                                    `json:"cache_node_type"`
	Status                        string                                    `json:"status"`
	MemberClusters                []ElastiCacheCluster                      `json:"member_clusters"`
	MultiAZ                       string                                    `json:"multi_az"`
	ClusterEnabled                bool                                      `json:"cluster_enabled"`
	ClusterMode                   string                                    `json:"cluster_mode"`
	Engine                        string                                    `json:"engine"`
	EncryptionConfig              CacheClusterEncyrptionConfig              `json:"encryption_config"`
	UnappliedUpdateActionsSummary ElastiCacheUpdateActionsSummary           `json:"update_action_summary"`
	UnappliedUpdateActions        []ElastiCacheReplicationGroupUpdateAction `json:"update_actions"`
}

type ElastiCacheServerlessCache struct {
	ARN                string `json:"arn"`
	Name               string `json:"serverless_cache_name"`
	Status             string `json:"status"`
	Engine             string `json:"engine"`
	MajorEngineVersion string `json:"major_engine_version"`
	FullEngineVersion  string `json:"full_engine_version"`
}

type ElastiCacheUpdateActionsSummary struct {
	UnappliedUpdateCount               int `json:"total_unapplied_updates"`
	TotalUnappliedImportantUpdateCount int `json:"total_unapplied_important_updates"`
	TotalUnappliedCriticalUpdateCount  int `json:"total_unapplied_critical_updates"`
}

type ElastiCacheReplicationGroupUpdateAction struct {
	ReplicationGroupId string                  `json:"replication_group_id"`
	UpdateAction       ElastiCacheUpdateAction `json:"update_action"`
}

type ElastiCacheCacheClusterUpdateAction struct {
	CacheClusterId string                  `json:"replication_group_id"`
	UpdateAction   ElastiCacheUpdateAction `json:"update_action"`
}

type ElastiCacheUpdateAction struct {
	ServiceUpdate      ElastiCacheServiceUpdate                `json:"service_update"`
	AvailableDate      time.Time                               `json:"available_date"`
	Status             string                                  `json:"update_action_status"`
	StatusModifiedDate time.Time                               `json:"udpate_action_status_modified_date"`
	Completion         ElastiCacheUpdateActionCompletionStatus `json:"completion_status"`
	SlaMet             string                                  `json:"sla_met"`
	Engine             string                                  `json:"engine"`
}

type ElastiCacheServiceUpdate struct {
	Name                   string    `json:"name"`
	ReleaseDate            time.Time `json:"release_date"`
	Severity               string    `json:"severity"`
	Status                 string    `json:"status"`
	RecommendedApplyByDate time.Time `json:"recommended_apply_by_date"`
	Type                   string    `json:"type"`
}

type ElastiCacheUpdateActionCompletionStatus struct {
	TotalNodesToUpdate          int `json:"total_nodes_to_update"`
	TotalNodesAlreadyUpdated    int `json:"total_nodes_already_updated"`
	TotalNodesRemainingToUpdate int `json:"total_nodes_remaining_to_update"`
}

type CacheClustersSummary struct {
	TotalClusters                 int                             `json:"total_clusters"`
	TotalServerlessCaches         int                             `json:"total_serverless_caches"`
	TotalNodes                    int32                           `json:"total_nodes"`
	ValkeyCount                   int                             `json:"valkey_count"`
	ValkeyNodesCount              int32                           `json:"valkey_nodes_count"`
	RedisCount                    int                             `json:"redis_count"`
	RedisNodesCount               int32                           `json:"redis_nodes_count"`
	MemcachedCount                int                             `json:"memcached_count"`
	MemcachedNodesCount           int32                           `json:"memcached_nodes_count"`
	AllCacheClusters              []ElastiCacheCluster            `json:"all_cache_clusters"`
	ReplicationGroups             []ElastiCacheReplicationGroup   `json:"replication_groups"`
	NonReplicatedCacheClusters    []ElastiCacheCluster            `json:"non_replicated_cache_clusters"`
	ServerlessCaches              []ElastiCacheServerlessCache    `json:"serverless_caches"`
	UnappliedUpdateActionsSummary ElastiCacheUpdateActionsSummary `json:"unapplied_update_actions_summary"`
}
