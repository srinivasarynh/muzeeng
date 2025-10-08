package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"feed-service/model"
	"feed-service/repository"
	"github.com/google/uuid"
)

// FeedBuilder handles feed generation and refresh operations
type FeedBuilder interface {
	// Fan-out on write: When a user creates a post, add it to all followers' feeds
	FanOutPost(ctx context.Context, postID, authorID uuid.UUID) error

	// Refresh a user's feed (can be triggered periodically or on-demand)
	RefreshUserFeed(ctx context.Context, userID uuid.UUID) error

	// Background job to refresh feeds for active users
	RefreshActiveUserFeeds(ctx context.Context) error

	// Remove post from all feeds (when post is deleted)
	RemovePostFromFeeds(ctx context.Context, postID uuid.UUID) error
}

type feedBuilder struct {
	feedRepo   repository.FeedRepository
	followRepo FollowRepository
	mu         sync.Mutex
}

// FollowRepository interface for getting followers
type FollowRepository interface {
	GetFollowerIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	GetFollowingIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

func NewFeedBuilder(feedRepo repository.FeedRepository, followRepo FollowRepository) FeedBuilder {
	return &feedBuilder{
		feedRepo:   feedRepo,
		followRepo: followRepo,
	}
}

// When a user creates a post, immediately add it to all their followers' feeds
func (fb *feedBuilder) FanOutPost(ctx context.Context, postID, authorID uuid.UUID) error {
	followerIDs, err := fb.followRepo.GetFollowerIDs(ctx, authorID)
	if err != nil {
		return fmt.Errorf("failed to get followers: %w", err)
	}

	if len(followerIDs) == 0 {
		return nil
	}

	feedItems := make([]models.FeedCache, len(followerIDs))
	now := time.Now()

	for i, followerID := range followerIDs {
		feedItems[i] = models.FeedCache{
			ID:        uuid.New(),
			UserID:    followerID,
			PostID:    postID,
			CreatedAt: now,
		}
	}

	err = fb.feedRepo.BulkInsertFeedItems(ctx, feedItems)
	if err != nil {
		return fmt.Errorf("failed to fan out post: %w", err)
	}

	go fb.invalidateFollowersCaches(context.Background(), followerIDs)

	return nil
}

// RefreshUserFeed rebuilds a user's feed from scratch
func (fb *feedBuilder) RefreshUserFeed(ctx context.Context, userID uuid.UUID) error {
	err := fb.feedRepo.InvalidateUserFeed(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to invalidate feed cache: %w", err)
	}

	posts, err := fb.feedRepo.BuildFeedForUser(ctx, userID, 100)
	if err != nil {
		return fmt.Errorf("failed to build feed: %w", err)
	}

	err = fb.feedRepo.CacheFeedItems(ctx, userID, posts)
	if err != nil {
		return fmt.Errorf("failed to cache feed: %w", err)
	}

	return nil
}

// RefreshActiveUserFeeds refreshes feeds for recently active users
func (fb *feedBuilder) RefreshActiveUserFeeds(ctx context.Context) error {
	return nil
}

// RemovePostFromFeeds removes a deleted post from all feeds
func (fb *feedBuilder) RemovePostFromFeeds(ctx context.Context, postID uuid.UUID) error {

	return nil
}

// invalidateFollowersCaches invalidates Redis cache for multiple users
func (fb *feedBuilder) invalidateFollowersCaches(ctx context.Context, userIDs []uuid.UUID) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10)

	for _, userID := range userIDs {
		wg.Add(1)
		go func(uid uuid.UUID) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := fb.feedRepo.InvalidateUserFeed(ctx, uid)
			if err != nil {
				fmt.Printf("Failed to invalidate cache for user %s: %v\n", uid, err)
			}
		}(userID)
	}

	wg.Wait()
}

// FeedRankingService handles feed ranking algorithms
type FeedRankingService struct {
	feedRepo repository.FeedRepository
}

func NewFeedRankingService(feedRepo repository.FeedRepository) *FeedRankingService {
	return &FeedRankingService{
		feedRepo: feedRepo,
	}
}

// CalculateFeedScore calculates a ranking score for a post in the feed
func (frs *FeedRankingService) CalculateFeedScore(post models.Post, viewerID uuid.UUID) float64 {
	now := time.Now()
	age := now.Sub(post.CreatedAt).Hours()

	timeDecay := 1.0 / (1.0 + age*0.1)

	engagementScore := float64(post.LikesCount)*0.6 + float64(post.CommentsCount)*0.4

	totalScore := (timeDecay * 0.6) + (engagementScore * 0.4)

	return totalScore
}

// RankPosts sorts posts by their calculated scores
func (frs *FeedRankingService) RankPosts(posts []models.Post, viewerID uuid.UUID) []models.Post {
	type scoredPost struct {
		post  models.Post
		score float64
	}

	scored := make([]scoredPost, len(posts))
	for i, post := range posts {
		scored[i] = scoredPost{
			post:  post,
			score: frs.CalculateFeedScore(post, viewerID),
		}
	}

	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	result := make([]models.Post, len(scored))
	for i, sp := range scored {
		result[i] = sp.post
	}

	return result
}
