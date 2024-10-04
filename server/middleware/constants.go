package middleware

const ReviewExchange string = "reviews"
const GamesExchange string = "games"
const GamesPartitionerQueue string = "games-partitioner"
const GamesFilterQueue string = "games-filter"
const ReviewsFilterQueue string = "reviews-filter"

// reviews filter
const ReviewsScoreFilterExchange string = "review-score-filter"
const NinetyPercentileReviewsQueue string = "90-percentile"
const LanguageReviewsFilterQueue string = "language-filter-reviews"
const FiftyThReviewsQueue string = "50-thousand"

// keys
const PositiveReviews string = "positive-reviews"
const NegativeReviews string = "negative-reviews"
