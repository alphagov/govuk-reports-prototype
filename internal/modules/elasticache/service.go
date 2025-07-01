package elasticache

import (
	"context"
	"fmt"

	"govuk-reports-dashboard/internal/config"
	"govuk-reports-dashboard/pkg/logger"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/elasticache"
	"github.com/aws/aws-sdk-go-v2/service/elasticache/types"
)

type ElastiCacheService struct {
	client *elasticache.Client
	config *config.Config
	logger *logger.Logger
}

// NewElastiCacheService creates a new ElastiCache service instance
func NewElastiCacheService(awsConfig aws.Config, config *config.Config, logger *logger.Logger) *ElastiCacheService {
	client := elasticache.NewFromConfig(awsConfig)

	return &ElastiCacheService{
		client: client,
		config: config,
		logger: logger,
	}
}

func (s *ElastiCacheService) GetAllClusters(ctx context.Context) (*CacheClustersSummary, error) {
	s.logger.Info().Msg("Discovering ElastiCache clusters")

	cacheClusters, err := s.getCacheClusters(ctx)
	if err != nil {
		return nil, err
	}

	replicationGroups, err := s.getReplicationGroups(cacheClusters, ctx)
	if err != nil {
		return nil, err
	}

	serverlessCaches, err := s.getServerlessCaches(ctx)
	if err != nil {
		return nil, err
	}

	summary := s.generateCacheClustersSummary(replicationGroups, cacheClusters, serverlessCaches)

	return summary, nil
}

func (s *ElastiCacheService) getCacheClusters(ctx context.Context) ([]ElastiCacheCluster, error) {
	var cacheClusters []ElastiCacheCluster

	paginator := elasticache.NewDescribeCacheClustersPaginator(s.client, &elasticache.DescribeCacheClustersInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			s.logger.WithError(err).Error().Msg("Failed to describe ElastiCache Clusters")
			return nil, fmt.Errorf("failed to describe ElastiCache clusters: %w", err)
		}

		for _, cacheCluster := range page.CacheClusters {
			cacheClusters = append(cacheClusters, s.convertToElastiCacheCluster(cacheCluster))
		}

	}

	return cacheClusters, nil
}

func (s *ElastiCacheService) getReplicationGroups(cacheClusters []ElastiCacheCluster, ctx context.Context) ([]ElastiCacheReplicationGroup, error) {
	var replicationGroups []ElastiCacheReplicationGroup

	paginator := elasticache.NewDescribeReplicationGroupsPaginator(s.client, &elasticache.DescribeReplicationGroupsInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			s.logger.WithError(err).Error().Msg("Failed to describe ElastiCache replication groups")
			return nil, fmt.Errorf("failed to describe ElastiCache replication groups: %w", err)
		}

		for _, replicationGroup := range page.ReplicationGroups {
			replicationGroups = append(replicationGroups, s.convertToElastiCacheReplicationGroup(replicationGroup, cacheClusters))
		}
	}

	return replicationGroups, nil
}

func (s *ElastiCacheService) getServerlessCaches(ctx context.Context) ([]ElastiCacheServerlessCache, error) {
	var serverlessCaches []ElastiCacheServerlessCache

	paginator := elasticache.NewDescribeServerlessCachesPaginator(s.client, &elasticache.DescribeServerlessCachesInput{})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			s.logger.WithError(err).Error().Msg("Failed to describe ElastiCache serverless caches")
			return nil, fmt.Errorf("failed to describe ElastiCache serverless caches: %w", err)
		}

		for _, serverlessCache := range page.ServerlessCaches {
			serverlessCaches = append(serverlessCaches, s.convertToServerlessElastiCache(serverlessCache))
		}
	}

	return serverlessCaches, nil
}

func (s *ElastiCacheService) convertToElastiCacheCluster(cacheCluster types.CacheCluster) ElastiCacheCluster {
	return ElastiCacheCluster{
		ARN:           aws.ToString(cacheCluster.ARN),
		Id:            aws.ToString(cacheCluster.CacheClusterId),
		NodeType:      aws.ToString(cacheCluster.CacheNodeType),
		NumCacheNodes: aws.ToInt32(cacheCluster.NumCacheNodes),
		Engine:        aws.ToString(cacheCluster.Engine),
		EngineVersion: aws.ToString(cacheCluster.EngineVersion),
		Status:        aws.ToString(cacheCluster.CacheClusterStatus),
		EncryptionConfig: CacheClusterEncyrptionConfig{
			AtRest:    aws.ToBool(cacheCluster.AtRestEncryptionEnabled),
			InTransit: aws.ToBool(cacheCluster.TransitEncryptionEnabled),
		},
		ReplicationGroup: aws.ToString(cacheCluster.ReplicationGroupId),
	}
}

func (s *ElastiCacheService) convertToServerlessElastiCache(serverlessCache types.ServerlessCache) ElastiCacheServerlessCache {
	return ElastiCacheServerlessCache{
		ARN:                aws.ToString(serverlessCache.ARN),
		Name:               aws.ToString(serverlessCache.ServerlessCacheName),
		Status:             aws.ToString(serverlessCache.Status),
		Engine:             aws.ToString(serverlessCache.Engine),
		MajorEngineVersion: aws.ToString(serverlessCache.MajorEngineVersion),
		FullEngineVersion:  aws.ToString(serverlessCache.FullEngineVersion),
	}
}

func (s *ElastiCacheService) convertToElastiCacheReplicationGroup(replicationGroup types.ReplicationGroup, cacheClusters []ElastiCacheCluster) ElastiCacheReplicationGroup {
	var memberClusters []ElastiCacheCluster

	replicationGroupId := aws.ToString(replicationGroup.ReplicationGroupId)

	for _, cluster := range cacheClusters {
		if cluster.ReplicationGroup == replicationGroupId {
			memberClusters = append(memberClusters, cluster)
		}
	}

	return ElastiCacheReplicationGroup{
		ARN:            aws.ToString(replicationGroup.ARN),
		Id:             replicationGroupId,
		NodeType:       aws.ToString(replicationGroup.CacheNodeType),
		Status:         aws.ToString(replicationGroup.Status),
		MemberClusters: memberClusters,
		MultiAZ:        aws.ToString((*string)(&replicationGroup.MultiAZ)),
		ClusterEnabled: aws.ToBool(replicationGroup.ClusterEnabled),
		ClusterMode:    aws.ToString((*string)(&replicationGroup.ClusterMode)),
		Engine:         aws.ToString(replicationGroup.Engine),
		EncryptionConfig: CacheClusterEncyrptionConfig{
			AtRest:    aws.ToBool(replicationGroup.AtRestEncryptionEnabled),
			InTransit: aws.ToBool(replicationGroup.TransitEncryptionEnabled),
		},
	}
}

func (s *ElastiCacheService) generateCacheClustersSummary(replicationGroups []ElastiCacheReplicationGroup, cacheClusters []ElastiCacheCluster, serverlessCaches []ElastiCacheServerlessCache) *CacheClustersSummary {
	var valkeyCount, redisCount, memcachedCount int = 0, 0, 0
	var totalNodes, valkeyNodeCount, redisNodeCount, memcachedNodeCount int32 = 0, 0, 0, 0

	for _, cluster := range cacheClusters {
		switch cluster.Engine {
		case "memcached":
			memcachedCount += 1
			memcachedNodeCount = valkeyNodeCount + cluster.NumCacheNodes
		case "redis":
			redisCount += 1
			redisNodeCount = valkeyNodeCount + cluster.NumCacheNodes
		case "valkey":
			valkeyCount += 1
			valkeyNodeCount = valkeyNodeCount + cluster.NumCacheNodes
		}
		totalNodes = totalNodes + cluster.NumCacheNodes
	}

	for _, serverlessCache := range serverlessCaches {
		switch serverlessCache.Engine {
		case "memcached":
			memcachedCount += 1
		case "redis":
			redisCount += 1
		case "valkey":
			valkeyCount += 1
		}
	}

	var nonReplicatedCacheClusters []ElastiCacheCluster
	for _, cacheCluster := range cacheClusters {
		if cacheCluster.ReplicationGroup == "" {
			nonReplicatedCacheClusters = append(nonReplicatedCacheClusters, cacheCluster)
		}
	}

	return &CacheClustersSummary{
		TotalClusters:              len(cacheClusters),
		TotalServerlessCaches:      len(serverlessCaches),
		TotalNodes:                 totalNodes,
		MemcachedCount:             memcachedCount,
		MemcachedNodesCount:        memcachedNodeCount,
		RedisCount:                 redisCount,
		RedisNodesCount:            redisNodeCount,
		ValkeyCount:                valkeyCount,
		ValkeyNodesCount:           valkeyNodeCount,
		AllCacheClusters:           cacheClusters,
		ReplicationGroups:          replicationGroups,
		NonReplicatedCacheClusters: nonReplicatedCacheClusters,
		ServerlessCaches:           serverlessCaches,
	}
}
