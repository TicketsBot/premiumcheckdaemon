package daemon

import (
	"github.com/TicketsBot/common/premium"
	"github.com/TicketsBot/common/sentry"
	"github.com/TicketsBot/database"
	"github.com/go-redis/redis"
	"github.com/rxdn/gdl/cache"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type Daemon struct {
	Logger        *log.Logger
	db            *database.Database
	cache         *cache.PgCache
	redis         *redis.Client
	patreon       *premium.PatreonClient
	premiumClient *premium.PremiumLookupClient
	forced        []uint64
}

func NewDaemon(db *database.Database, cache *cache.PgCache, redis *redis.Client, patreon *premium.PatreonClient) *Daemon {
	var forced []uint64
	for _, raw := range strings.Split(os.Getenv("FORCED"), ",") {
		if raw == "" {
			continue
		}

		userId, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			sentry.Error(err)
			continue
		}

		forced = append(forced, userId)
	}

	return &Daemon{
		Logger:        log.New(os.Stdout, "[daemon] ", log.LstdFlags),
		db:            db,
		cache:         cache,
		redis:         redis,
		patreon:       patreon,
		premiumClient: premium.NewPremiumLookupClient(patreon, redis, cache, db),
		forced:        forced,
	}
}

func (d *Daemon) Start() {
	for {
		d.sweepWhitelabel()
		d.sweepPanels()
		time.Sleep(time.Hour * 6)
	}
}
