package elasticache

type CacheClusterEncyrptionConfig struct {
	AtRest    bool `json:"at_rest"`
	InTransit bool `json:"in_trasit"`
}

type ElastiCacheCluster struct {
	ARN              string                       `json:"arn"`
	Id               string                       `json:"cache_cluster_id"`
	NodeType         string                       `json:"cache_node_type"`
	NumCacheNodes    int32                        `json:"num_cache_nodes"`
	Engine           string                       `json:"engine"`
	EngineVersion    string                       `json:"engine_version"`
	Status           string                       `json:"status"`
	EncryptionConfig CacheClusterEncyrptionConfig `json:"encryption_config"`
	ReplicationGroup string                       `json:"replication_group"`
}

type ElastiCacheReplicationGroup struct {
	ARN              string                       `json:"arn"`
	Id               string                       `json:"replication_group_id"`
	NodeType         string                       `json:"cache_node_type"`
	Status           string                       `json:"status"`
	MemberClusters   []ElastiCacheCluster         `json:"member_clusters"`
	MultiAZ          string                       `json:"multi_az"`
	ClusterEnabled   bool                         `json:"cluster_enabled"`
	ClusterMode      string                       `json:"cluster_mode"`
	Engine           string                       `json:"engine"`
	EncryptionConfig CacheClusterEncyrptionConfig `json:"encryption_config"`
}

type ElastiCacheServerlessCache struct {
	ARN                string `json:"arn"`
	Name               string `json:"serverless_cache_name"`
	Status             string `json:"status"`
	Engine             string `json:"engine"`
	MajorEngineVersion string `json:"major_engine_version"`
	FullEngineVersion  string `json:"full_engine_version"`
}

type CacheClustersSummary struct {
	TotalClusters              int                           `json:"total_clusters"`
	TotalServerlessCaches      int                           `json:"total_serverless_caches"`
	TotalNodes                 int32                         `json:"total_nodes"`
	ValkeyCount                int                           `json:"valkey_count"`
	ValkeyNodesCount           int32                         `json:"valkey_nodes_count"`
	RedisCount                 int                           `json:"redis_count"`
	RedisNodesCount            int32                         `json:"redis_nodes_count"`
	MemcachedCount             int                           `json:"memcached_count"`
	MemcachedNodesCount        int32                         `json:"memcached_nodes_count"`
	AllCacheClusters           []ElastiCacheCluster          `json:"all_cache_clusters"`
	ReplicationGroups          []ElastiCacheReplicationGroup `json:"replication_groups"`
	NonReplicatedCacheClusters []ElastiCacheCluster          `json:"non_replicationed_cache_clusters"`
	ServerlessCaches           []ElastiCacheServerlessCache  `json:"serverless_caches"`
}
