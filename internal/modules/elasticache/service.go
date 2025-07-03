package elasticache

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

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
	s.logger.Info().Msg("Discovering ElastiCache instances")

	cacheClusters, err := s.getCacheClusters(ctx)
	if err != nil {
		return nil, err
	}

	replicationGroups, err := s.getReplicationGroups(cacheClusters, ctx)
	if err != nil {
		return nil, err
	}

	serverlessCaches, err := s.GetServerlessCaches(ctx)
	if err != nil {
		return nil, err
	}

	summary, err := s.generateCacheClustersSummary(&replicationGroups, &cacheClusters, &serverlessCaches, ctx)
	if err != nil {
		return nil, err
	}

	return summary, nil
}

func (s *ElastiCacheService) getCacheClusters(ctx context.Context) ([]ElastiCacheCluster, error) {
	s.logger.Info().Msg("Discovering ElastiCache Cache Clusters")
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

func (s *ElastiCacheService) getUpdateActionsSummaryAndPopulateUpdates(replicationGroups *[]ElastiCacheReplicationGroup, cacheClusters *[]ElastiCacheCluster, ctx context.Context) (*ElastiCacheUpdateActionsSummary, error) {
	s.logger.Info().Msg("Discovering ElastiCache Unapplied Update Actions")

	var unappliedUpdateCount, unappliedImportantUpdateCount, unappliedCriticalUpdateCount int = 0, 0, 0

	replicationGroupUpdateActions, err := s.getReplicationGroupUpdateActions(*replicationGroups, ctx)
	if err != nil {
		return nil, err
	}

	for _, replicationGroupUpdateAction := range replicationGroupUpdateActions {
		if replicationGroupUpdateAction.UpdateAction.ServiceUpdate.Status != "available" {
			continue
		}

		if replicationGroupUpdateAction.UpdateAction.Status == "not-applicable" || replicationGroupUpdateAction.UpdateAction.Status == "complete" {
			continue
		}

		replicationGroupIndex := slices.IndexFunc(*replicationGroups, func(replicationGroup ElastiCacheReplicationGroup) bool {
			return replicationGroupUpdateAction.ReplicationGroupId == replicationGroup.Id
		})
		replicationGroup := &(*replicationGroups)[replicationGroupIndex]
		replicationGroup.UnappliedUpdateActions = append(replicationGroup.UnappliedUpdateActions, replicationGroupUpdateAction)

		unappliedUpdateCount += 1
		replicationGroup.UnappliedUpdateActionsSummary.UnappliedUpdateCount += 1
		switch replicationGroupUpdateAction.UpdateAction.ServiceUpdate.Severity {
		case "critical":
			unappliedCriticalUpdateCount += 1
			replicationGroup.UnappliedUpdateActionsSummary.TotalUnappliedCriticalUpdateCount += 1
		case "important":
			unappliedImportantUpdateCount += 1
			replicationGroup.UnappliedUpdateActionsSummary.TotalUnappliedImportantUpdateCount += 1
		}
	}

	cacheClusterUpdateActions, err := s.getCacheClusterUpdateActions(*cacheClusters, ctx)
	if err != nil {
		return nil, err
	}

	for _, cacheClusterUpdateAction := range cacheClusterUpdateActions {
		if cacheClusterUpdateAction.UpdateAction.ServiceUpdate.Status != "available" {
			continue
		}

		if !slices.Contains([]string{"available", "not-applicable"}, cacheClusterUpdateAction.UpdateAction.Status) {
			continue
		}

		cacheClusterIndex := slices.IndexFunc(*cacheClusters, func(cacheCluster ElastiCacheCluster) bool {
			return cacheClusterUpdateAction.CacheClusterId == cacheCluster.Id
		})
		cacheCluster := &(*cacheClusters)[cacheClusterIndex]
		cacheCluster.UnappliedUpdateActions = append(cacheCluster.UnappliedUpdateActions, cacheClusterUpdateAction)

		unappliedUpdateCount += 1
		cacheCluster.UnappliedUpdateActionsSummary.UnappliedUpdateCount += 1
		switch cacheClusterUpdateAction.UpdateAction.ServiceUpdate.Severity {
		case "critical":
			unappliedCriticalUpdateCount += 1
			cacheCluster.UnappliedUpdateActionsSummary.TotalUnappliedCriticalUpdateCount += 1
		case "important":
			unappliedImportantUpdateCount += 1
			cacheCluster.UnappliedUpdateActionsSummary.TotalUnappliedImportantUpdateCount += 1
		}
	}

	return &ElastiCacheUpdateActionsSummary{
		TotalUnappliedCriticalUpdateCount:  unappliedCriticalUpdateCount,
		TotalUnappliedImportantUpdateCount: unappliedImportantUpdateCount,
		UnappliedUpdateCount:               unappliedUpdateCount,
	}, nil
}

func (s *ElastiCacheService) getReplicationGroups(cacheClusters []ElastiCacheCluster, ctx context.Context) ([]ElastiCacheReplicationGroup, error) {
	s.logger.Info().Msg("Discovering ElastiCache Replication Groups")

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

func (s *ElastiCacheService) getReplicationGroupUpdateActions(replicationGroups []ElastiCacheReplicationGroup, ctx context.Context) ([]ElastiCacheReplicationGroupUpdateAction, error) {
	var replicationGroupIds []string = make([]string, len(replicationGroups))
	for i, replicationGroup := range replicationGroups {
		replicationGroupIds[i] = replicationGroup.Id
	}

	var replicationGroupUpdateActions []ElastiCacheReplicationGroupUpdateAction

	paginator := elasticache.NewDescribeUpdateActionsPaginator(s.client, &elasticache.DescribeUpdateActionsInput{
		ReplicationGroupIds: replicationGroupIds,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			s.logger.WithError(err).Error().Msg("Failed to describe update actions for replication groups")
			return nil, fmt.Errorf("failed to describe update actions for replication groups: %w", err)
		}

		for _, updateAction := range page.UpdateActions {
			replicationGroupUpdateAction, err := s.convertToReplicationGroupUpdateAction(updateAction)
			if err != nil {
				return nil, err
			}

			replicationGroupUpdateActions = append(replicationGroupUpdateActions, *replicationGroupUpdateAction)
		}
	}

	return replicationGroupUpdateActions, nil
}

func (s *ElastiCacheService) getCacheClusterUpdateActions(cacheClusters []ElastiCacheCluster, ctx context.Context) ([]ElastiCacheCacheClusterUpdateAction, error) {
	var cacheClusterIds []string = make([]string, len(cacheClusters))
	for i, cacheCluster := range cacheClusters {
		cacheClusterIds[i] = cacheCluster.Id
	}

	var cacheClusterUpdateActions []ElastiCacheCacheClusterUpdateAction

	paginator := elasticache.NewDescribeUpdateActionsPaginator(s.client, &elasticache.DescribeUpdateActionsInput{
		CacheClusterIds: cacheClusterIds,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			s.logger.WithError(err).Error().Msg("Failed to describe update actions for cache clusters")
			return nil, fmt.Errorf("failed to describe update actions for cache clusters: %w", err)
		}

		for _, updateAction := range page.UpdateActions {
			cacheClusterUpdateAction, err := s.convertToCacheClusterUpdateAction(updateAction)
			if err != nil {
				return nil, err
			}

			cacheClusterUpdateActions = append(cacheClusterUpdateActions, *cacheClusterUpdateAction)
		}
	}

	return cacheClusterUpdateActions, nil
}

func (s *ElastiCacheService) GetServerlessCaches(ctx context.Context) ([]ElastiCacheServerlessCache, error) {
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
		ReplicationGroup:              aws.ToString(cacheCluster.ReplicationGroupId),
		UnappliedUpdateActionsSummary: ElastiCacheUpdateActionsSummary{},
		UnappliedUpdateActions:        []ElastiCacheCacheClusterUpdateAction{},
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
		UnappliedUpdateActionsSummary: ElastiCacheUpdateActionsSummary{},
		UnappliedUpdateActions:        []ElastiCacheReplicationGroupUpdateAction{},
	}
}

func (s *ElastiCacheService) convertToReplicationGroupUpdateAction(updateAction types.UpdateAction) (*ElastiCacheReplicationGroupUpdateAction, error) {
	if updateAction.ReplicationGroupId == nil {
		return nil, fmt.Errorf("Replication Group Update Action missing ReplicationGroupId field")
	}

	elastiCacheUpdateAction, err := s.convertToElastiCacheUpdateAction(updateAction)
	if err != nil {
		return nil, err
	}

	return &ElastiCacheReplicationGroupUpdateAction{
		ReplicationGroupId: aws.ToString(updateAction.ReplicationGroupId),
		UpdateAction:       *elastiCacheUpdateAction,
	}, nil
}

func (s *ElastiCacheService) convertToCacheClusterUpdateAction(updateAction types.UpdateAction) (*ElastiCacheCacheClusterUpdateAction, error) {
	if updateAction.CacheClusterId == nil {
		return nil, fmt.Errorf("Cache Cluster Update Action missing CacheClusterId field")
	}

	elastiCacheUpdateAction, err := s.convertToElastiCacheUpdateAction(updateAction)
	if err != nil {
		return nil, err
	}

	return &ElastiCacheCacheClusterUpdateAction{
		CacheClusterId: aws.ToString(updateAction.CacheClusterId),
		UpdateAction:   *elastiCacheUpdateAction,
	}, nil
}

func (s *ElastiCacheService) convertToElastiCacheUpdateAction(updateAction types.UpdateAction) (*ElastiCacheUpdateAction, error) {
	nodeCompletion := strings.Split(aws.ToString(updateAction.NodesUpdated), "/")
	nodesUpdated, err := strconv.Atoi(nodeCompletion[0])
	if err != nil {
		s.logger.WithError(err).Error().Msg("Couldn't parse nodes completed from NodesUpdated in ElastiCache update action")
		return nil, fmt.Errorf("Couldn't parse nodes completed NodesUpdated in ElastiCache update action %w", err)
	}
	totalNodesToUpdate, err := strconv.Atoi(nodeCompletion[1])
	if err != nil {
		s.logger.WithError(err).Error().Msg("Couldn't parse total nodes to update from NodesUpdated in ElastiCache update action")
		return nil, fmt.Errorf("Couldn't parse total nodes to update from NodesUpdated in ElastiCache update action %w", err)
	}

	return &ElastiCacheUpdateAction{
		ServiceUpdate: ElastiCacheServiceUpdate{
			Name:                   aws.ToString(updateAction.ServiceUpdateName),
			ReleaseDate:            aws.ToTime(updateAction.ServiceUpdateReleaseDate),
			Severity:               aws.ToString((*string)(&updateAction.ServiceUpdateSeverity)),
			Status:                 aws.ToString((*string)(&updateAction.ServiceUpdateStatus)),
			RecommendedApplyByDate: aws.ToTime(updateAction.ServiceUpdateRecommendedApplyByDate),
			Type:                   aws.ToString((*string)(&updateAction.ServiceUpdateType)),
		},
		AvailableDate:      aws.ToTime(updateAction.UpdateActionAvailableDate),
		Status:             aws.ToString((*string)(&updateAction.UpdateActionStatus)),
		StatusModifiedDate: aws.ToTime(updateAction.UpdateActionStatusModifiedDate),
		Completion: ElastiCacheUpdateActionCompletionStatus{
			TotalNodesToUpdate:          totalNodesToUpdate,
			TotalNodesAlreadyUpdated:    nodesUpdated,
			TotalNodesRemainingToUpdate: totalNodesToUpdate - nodesUpdated,
		},
		SlaMet: aws.ToString((*string)(&updateAction.SlaMet)),
		Engine: aws.ToString(updateAction.Engine),
	}, nil
}

func (s *ElastiCacheService) generateCacheClustersSummary(replicationGroups *[]ElastiCacheReplicationGroup, cacheClusters *[]ElastiCacheCluster, serverlessCaches *[]ElastiCacheServerlessCache, ctx context.Context) (*CacheClustersSummary, error) {
	var valkeyCount, redisCount, memcachedCount int = 0, 0, 0
	var totalNodes, valkeyNodeCount, redisNodeCount, memcachedNodeCount int32 = 0, 0, 0, 0

	for _, cluster := range *cacheClusters {
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

	for _, serverlessCache := range *serverlessCaches {
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
	for _, cacheCluster := range *cacheClusters {
		if cacheCluster.ReplicationGroup == "" {
			nonReplicatedCacheClusters = append(nonReplicatedCacheClusters, cacheCluster)
		}
	}

	updateActionsSummary, err := s.getUpdateActionsSummaryAndPopulateUpdates(replicationGroups, cacheClusters, ctx)
	if err != nil {
		return nil, err
	}

	return &CacheClustersSummary{
		TotalClusters:                 len(*cacheClusters),
		TotalServerlessCaches:         len(*serverlessCaches),
		TotalNodes:                    totalNodes,
		MemcachedCount:                memcachedCount,
		MemcachedNodesCount:           memcachedNodeCount,
		RedisCount:                    redisCount,
		RedisNodesCount:               redisNodeCount,
		ValkeyCount:                   valkeyCount,
		ValkeyNodesCount:              valkeyNodeCount,
		AllCacheClusters:              *cacheClusters,
		ReplicationGroups:             *replicationGroups,
		NonReplicatedCacheClusters:    nonReplicatedCacheClusters,
		ServerlessCaches:              *serverlessCaches,
		UnappliedUpdateActionsSummary: *updateActionsSummary,
	}, nil
}
