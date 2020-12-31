package api

import (
	smem "github.com/ulule/limiter/v3/drivers/store/memory"
	"log"

	libredis "github.com/go-redis/redis/v8"

	limiter "github.com/ulule/limiter/v3"
	mhttp "github.com/ulule/limiter/v3/drivers/middleware/stdlib"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"
)

func CreateRedisClient(url string) *libredis.Client {
	option, err := libredis.ParseURL(url)
	if err != nil {
		log.Fatal(err)
	}
	return libredis.NewClient(option)
}

func createNewRate(rate string) limiter.Rate {
	lRate, err := limiter.NewRateFromFormatted(rate)
	if err != nil {
		log.Fatal(err)
	}
	return lRate
}

func CreateMemRateLimiter(rate string) *mhttp.Middleware {
	lRate := createNewRate(rate)
	// Or use a in-memory store with a goroutine which clears expired keys.
	store := smem.NewStore()
	return mhttp.NewMiddleware(limiter.New(store, lRate, limiter.WithTrustForwardHeader(true)))
}

func CreateRedisRateLimiter(rate string, prefix string, rClient *libredis.Client) *mhttp.Middleware {
	lRate := createNewRate(rate)

	// Create a store with the redis client.
	store, err := sredis.NewStoreWithOptions(rClient, limiter.StoreOptions{
		Prefix:   prefix,
		MaxRetry: 3,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Create a new middleware with the limiter instance.
	return mhttp.NewMiddleware(limiter.New(store, lRate, limiter.WithTrustForwardHeader(true)))
}
